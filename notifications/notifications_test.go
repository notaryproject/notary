package notifications

import (
	"bytes"
	"net/http"
	"testing"
	"time"

	"github.com/docker/distribution/context"
	dNotifications "github.com/docker/distribution/notifications"
	"github.com/stretchr/testify/require"
)

func TestNotify(t *testing.T) {
	err := Notify("NewThing", "GUN", dNotifications.SourceRecord{}, "username", &http.Request{}, dNotifications.NewBroadcaster())
	require.NoError(t, err)
}

func TestNewRequestRecord(t *testing.T) {
	req := &http.Request{
		RemoteAddr: "https://remote.com",
		Host:       "host",
		Method:     "PUT",
		Header: http.Header{
			"User-Agent": []string{"ua"},
		},
	}
	rec := newRequestRecord(req)
	require.Equal(t, rec.Addr, context.RemoteAddr(req))
	require.Equal(t, rec.Host, req.Host)
	require.Equal(t, rec.Method, req.Method)
	require.Equal(t, "ua", rec.UserAgent)
}

func TestCreateEvent(t *testing.T) {
	req, err := http.NewRequest("GET", "https://url.com", bytes.NewBufferString("hi"))
	require.Nil(t, err)

	now := time.Now()
	mockTime := func() time.Time {
		return now
	}

	evt := createEvent("NewThing", "GUN", dNotifications.SourceRecord{Addr: "localhost"}, "username", req, mockTime)
	require.NotEmpty(t, evt.ID)
	require.Equal(t, now, evt.Timestamp)
	require.Equal(t, "NewThing", evt.Action)
	require.Equal(t, "localhost", evt.Source.Addr)
	require.Equal(t, "username", evt.Actor.Name)
	require.Equal(t, newRequestRecord(req), evt.Request)
	require.Equal(t, "GUN", evt.Target.Repository)
}
