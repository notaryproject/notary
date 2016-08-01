package notifications

import (
	"net/http"
	"time"

	"github.com/docker/distribution/context"
	dNotifications "github.com/docker/distribution/notifications"
	"github.com/docker/distribution/uuid"
)

const (
	// TrustDataUpdate is a notification type, indicating that trust data has been updated
	TrustDataUpdate = "TRUST_DATA_UPDATE"
)

type timeGetter func() time.Time

// Endpoint represents an endpoint that can be notified when an event occurs
type Endpoint struct {
	Name      string        // identifies the endpoint in the registry instance.
	Disabled  bool          // disables the endpoint
	URL       string        // post url for the endpoint.
	Headers   http.Header   // static headers that should be added to all requests
	Timeout   time.Duration // HTTP timeout
	Threshold int           // circuit breaker threshold before backing off on failure
	Backoff   time.Duration // backoff duration
	CA        string        // acceptable CA to avoid x509 unknown authority error
}

// Notify notifies registered endpoints that an event has occurred
func Notify(action, gun string, source dNotifications.SourceRecord, username string, request *http.Request, broadcaster *dNotifications.Broadcaster) error {
	evt := createEvent(action, gun, source, username, request, time.Now)
	return broadcaster.Write(evt)
}

// NewRequestRecord builds a RequestRecord from an http.Request
func newRequestRecord(r *http.Request) dNotifications.RequestRecord {
	return dNotifications.RequestRecord{
		Addr:      context.RemoteAddr(r),
		Host:      r.Host,
		Method:    r.Method,
		UserAgent: r.UserAgent(),
	}
}

// createEvent creates an event with actor and source populated.
func createEvent(action, gun string, source dNotifications.SourceRecord, username string, request *http.Request, tg timeGetter) dNotifications.Event {
	evt := dNotifications.Event{
		ID:        uuid.Generate().String(),
		Timestamp: tg(),
		Action:    action,
		Source:    source,
		Actor:     dNotifications.ActorRecord{Name: username},
		Request:   newRequestRecord(request),
	}
	evt.Target.Repository = gun
	return evt
}
