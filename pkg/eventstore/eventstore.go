package eventstore

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Config holds the configuration for the EventStore client
type Config struct {
	APIURL     string
	APIVersion string
	AuthToken  string
}

// EventStore represents the client for interacting with the EventStore API
type EventStore struct {
	config   *Config
	client   *http.Client
}

// RFC3339Time is a custom time type that properly handles RFC3339 time strings
type RFC3339Time time.Time

// UnmarshalJSON implements the json.Unmarshaler interface
func (t *RFC3339Time) UnmarshalJSON(data []byte) error {
	// Remove quotes
	s := strings.Trim(string(data), "\"")
	if s == "null" || s == "" {
		*t = RFC3339Time(time.Time{})
		return nil
	}
	// Parse the time string
	parsedTime, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return err
	}
	*t = RFC3339Time(parsedTime)
	return nil
}

// MarshalJSON implements the json.Marshaler interface
func (t RFC3339Time) MarshalJSON() ([]byte, error) {
	if time.Time(t).IsZero() {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf("\"%s\"", time.Time(t).Format(time.RFC3339))), nil
}

// Time returns the time.Time representation
func (t RFC3339Time) Time() time.Time {
	return time.Time(t)
}

// Event represents an event in the EventStore
type Event struct {
	ID              string                 `json:"id,omitempty"`
	Source          string                 `json:"source,omitempty"`
	Subject         string                 `json:"subject"`
	Type            string                 `json:"type"`
	Time            RFC3339Time            `json:"time,omitempty"`
	Data            interface{}            `json:"data"`
	DataContentType string                 `json:"datacontenttype,omitempty"`
	SpecVersion     string                 `json:"specversion,omitempty"`
}

// NewEventStore creates a new EventStore client
func NewEventStore(config *Config) (*EventStore, error) {
	if config.APIURL == "" {
		return nil, fmt.Errorf("APIURL is required")
	}
	if config.APIVersion == "" {
		return nil, fmt.Errorf("APIVersion is required")
	}
	if config.AuthToken == "" {
		return nil, fmt.Errorf("AuthToken is required")
	}

	return &EventStore{
		config: config,
		client: &http.Client{},
	}, nil
}

// StreamEvents streams events from the specified subject
func (es *EventStore) StreamEvents(subject string) ([]Event, error) {
	url := fmt.Sprintf("%s/api/%s/stream", strings.TrimRight(es.config.APIURL, "/"), es.config.APIVersion)

	requestBody, err := json.Marshal(map[string]string{"subject": subject})
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", es.config.AuthToken))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/x-ndjson")
	req.Header.Set("User-Agent", "inoovum-eventstore-sdk-go")

	resp, err := es.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(bodyBytes))
	}

	var events []Event
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var event Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, fmt.Errorf("error parsing event JSON: %w", err)
		}

		// Set default values for CloudEvents compliance if not present
		if event.ID == "" {
			event.ID = uuid.New().String()
		}
		if event.Source == "" {
			event.Source = es.config.APIURL
		}
		if event.DataContentType == "" {
			event.DataContentType = "application/json"
		}
		if event.SpecVersion == "" {
			event.SpecVersion = "1.0"
		}
		if event.Time == RFC3339Time(time.Time{}) {
			now := time.Now().UTC()
			event.Time = RFC3339Time(now)
		}

		events = append(events, event)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	return events, nil
}

// CommitEvents commits a batch of events to the EventStore
func (es *EventStore) CommitEvents(events []Event) error {
	url := fmt.Sprintf("%s/api/%s/commit", strings.TrimRight(es.config.APIURL, "/"), es.config.APIVersion)

	// Ensure CloudEvents compliance for each event
	for i := range events {
		if events[i].ID == "" {
			events[i].ID = uuid.New().String()
		}
		if events[i].Source == "" {
			events[i].Source = es.config.APIURL
		}
		if events[i].DataContentType == "" {
			events[i].DataContentType = "application/json"
		}
		if events[i].SpecVersion == "" {
			events[i].SpecVersion = "1.0"
		}
		if events[i].Time == RFC3339Time(time.Time{}) {
			now := time.Now().UTC()
			events[i].Time = RFC3339Time(now)
		}
	}

	requestBody, err := json.Marshal(map[string][]Event{"events": events})
	if err != nil {
		return fmt.Errorf("error marshaling request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", es.config.AuthToken))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "inoovum-eventstore-sdk-go")

	resp, err := es.client.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s - %s", resp.Status, string(bodyBytes))
	}

	return nil
}

// Q executes a query against the EventStore
func (es *EventStore) Q(query string) ([]interface{}, error) {
	url := fmt.Sprintf("%s/api/%s/q", strings.TrimRight(es.config.APIURL, "/"), es.config.APIVersion)

	requestBody, err := json.Marshal(map[string]string{"query": query})
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", es.config.AuthToken))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/x-ndjson")
	req.Header.Set("User-Agent", "inoovum-eventstore-sdk-go")

	resp, err := es.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(bodyBytes))
	}

	var results []interface{}
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var result interface{}
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			return nil, fmt.Errorf("error parsing result JSON: %w", err)
		}
		results = append(results, result)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	return results, nil
}

// Ping checks the health of the EventStore API
func (es *EventStore) Ping() (string, error) {
	url := fmt.Sprintf("%s/api/%s/status/ping", strings.TrimRight(es.config.APIURL, "/"), es.config.APIVersion)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", es.config.AuthToken))
	req.Header.Set("User-Agent", "inoovum-eventstore-sdk-go")

	resp, err := es.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error: %s - %s", resp.Status, string(bodyBytes))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %w", err)
	}

	return string(bodyBytes), nil
}

// Audit runs an audit check on the EventStore
func (es *EventStore) Audit() (string, error) {
	url := fmt.Sprintf("%s/api/%s/status/audit", strings.TrimRight(es.config.APIURL, "/"), es.config.APIVersion)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", es.config.AuthToken))
	req.Header.Set("User-Agent", "inoovum-eventstore-sdk-go")

	resp, err := es.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error: %s - %s", resp.Status, string(bodyBytes))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %w", err)
	}

	return string(bodyBytes), nil
}
