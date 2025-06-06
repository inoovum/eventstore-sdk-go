# inoovum® EventStore Go SDK

This is the official Go SDK for the inoovum® EventStore. It provides a simple interface to interact with the EventStore API.

## Requirements

* Go 1.22 or higher

## Installation

```bash
go get github.com/inoovum/eventstore-sdk-go
```

## Configuration

The SDK requires the following environment variables to be set:

* `EVENTSTORE_API_URL`: The URL of the EventStore API
* `EVENTSTORE_API_VERSION`: The version of the API to use
* `EVENTSTORE_AUTH_TOKEN`: Your authentication token

Alternatively, you can pass these values directly when creating the client:

```go
config := &eventstore.Config{
    APIURL:     "https://your-api-url",
    APIVersion: "v1",
    AuthToken:  "your-auth-token",
}

client, err := eventstore.NewEventStore(config)
if err != nil {
    log.Fatal(err)
}
```

## Usage

### Streaming Events

```go
events, err := client.StreamEvents("/customer")
if err != nil {
    log.Fatal(err)
}

for _, event := range events {
    fmt.Printf("Event Type: %s, Data: %v\n", event.Type, event.Data)
}
```

### Committing Events

```go
// Example for creating a new user
events := []eventstore.Event{
    {
        Subject: "/user",  // For new resources
        Type:    "added",
        Data: map[string]interface{}{
            "name": "John Doe",
        },
    },
    {
        Subject: "/user/42fe976a-a17d-4959-9be0-61ebd9cd499a",  // For existing resources
        Type:    "updated",
        Data: map[string]interface{}{
            "name": "John Smith",
        },
    },
}

err := client.CommitEvents(events)
if err != nil {
    log.Fatal(err)
}
```

### Querying Events

```go
query := `
FROM e IN events
WHERE e.type == 'user.created'
PROJECT INTO { id: e.id, name: e.data.name }
`

results, err := client.Q(query)
if err != nil {
    log.Fatal(err)
}

for _, result := range results {
    fmt.Printf("Result: %v\n", result)
}
```

### Health Checks

```go
// Ping the API
response, err := client.Ping()
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Ping response: %s\n", response)

// Run audit
response, err = client.Audit()
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Audit response: %s\n", response)
```

## CloudEvents Compliance

All events in the SDK are CloudEvents compliant. The following fields are automatically set if not provided:

* `id`: A new UUID
* `source`: The API URL
* `time`: Current UTC timestamp
* `datacontenttype`: "application/json"
* `specversion`: "1.0"

## Error Handling

All methods return errors when something goes wrong. Make sure to check for errors and handle them appropriately.

## License

Copyright © 2024 inoovum GmbH. All rights reserved.
