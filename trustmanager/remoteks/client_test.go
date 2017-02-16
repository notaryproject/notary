package remoteks

import (
	"crypto/tls"
	"github.com/docker/notary/storage"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"net"
	"testing"
)

const addr = "127.0.0.1:9876"

var insecureTLS = tls.Config{
	InsecureSkipVerify: true,
}

func setupTestServer(t *testing.T) *grpc.Server {
	s := grpc.NewServer()
	st := NewGRPCStorage(storage.NewMemoryStore(nil))
	l, err := net.Listen(
		"tcp",
		addr,
	)
	require.NoError(t, err)
	RegisterStoreServer(s, st)
	go func() {
		err := s.Serve(l)
		t.Logf("server errored %s", err)
	}()
	return s
}

func TestRemoteStore(t *testing.T) {
	name := "testfile"
	bytes := []byte{'1'}

	s := setupTestServer(t)
	defer s.Stop()

	// can't just use NewRemoteStore because it correctly sets up tls
	// config and for testing purposes it's easier for the client to just
	// be insecure
	cc, err := grpc.Dial(
		addr,
		grpc.WithInsecure(),
		grpc.WithBlock(),
	)
	require.NoError(t, err)
	c := &RemoteStore{
		client:   NewStoreClient(cc),
		location: addr,
	}

	err = c.Set(name, bytes)
	require.NoError(t, err)

	out, err := c.Get(name)
	require.NoError(t, err)
	require.Equal(t, bytes, out)

	ls := c.ListFiles()
	require.Len(t, ls, 1)
	require.Equal(t, name, ls[0])

	err = c.Remove(name)
	require.NoError(t, err)

	ls = c.ListFiles()
	require.Len(t, ls, 0)

	_, err = c.Get(name)
	require.Error(t, err)
}
