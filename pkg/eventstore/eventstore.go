package eventstore

import (
	"context"
	"fmt"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

// Config holds the configuration for the EventStore client
type Config struct {
	APIURL     string
	APIVersion string
	AuthToken  string
}

// EventStore represents the client for interacting with the EventStore API
type EventStore struct {
	config     *Config
	ceClient   cloudevents.Client
}

// Event represents an event in the EventStore
type Event struct {
	Subject string
	Type    string
	Data    interface{}
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

	c, err := cloudevents.NewClientHTTP()
	if err != nil {
		return nil, fmt.Errorf("failed to create CloudEvents client: %w", err)
	}

	return &EventStore{
		config:   config,
		ceClient: c,
	}, nil
}

// StreamEvents streams events from the specified subject
func (es *EventStore) StreamEvents(subject string) ([]Event, error) {
	// Implementation pending
	return nil, nil
}

// CommitEvents commits a batch of events to the EventStore
func (es *EventStore) CommitEvents(events []Event) error {
	ctx := context.Background()
	for _, event := range events {
		e := cloudevents.NewEvent()
		e.SetType(event.Type)
		e.SetSource(es.config.APIURL)
		e.SetSubject(event.Subject)
		if err := e.SetData(cloudevents.ApplicationJSON, event.Data); err != nil {
			return fmt.Errorf("failed to set event data: %w", err)
		}

		if result := es.ceClient.Send(ctx, e); cloudevents.IsUndelivered(result) {
			return fmt.Errorf("failed to send event: %w", result)
		}
	}
	return nil
}

// Ping checks the health of the EventStore API
func (es *EventStore) Ping() (string, error) {
	// Implementation pending
	return "OK", nil
}

// Audit runs an audit check on the EventStore
func (es *EventStore) Audit() (string, error) {
	// Implementation pending
	return "OK", nil
}
