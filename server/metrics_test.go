package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/theupdateframework/notary/tuf/signed"
)

func TestMetricsEndpoint(t *testing.T) {
	handler := RootHandler(context.Background(), nil, signed.NewEd25519(),
		nil, nil, nil)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	res, err := http.Get(ts.URL + "/metrics")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)
}
