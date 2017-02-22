package remoteks

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/docker/notary/storage"
	"github.com/docker/notary/trustmanager"
)

type TestError struct{}

func (err TestError) Error() string {
	return "test error"
}

type ErroringStorage struct{}

func (s ErroringStorage) Set(string, []byte) error {
	return TestError{}
}

func (s ErroringStorage) Remove(string) error {
	return TestError{}
}

func (s ErroringStorage) Get(string) ([]byte, error) {
	return nil, TestError{}
}

func (s ErroringStorage) ListFiles() []string {
	return nil
}

func (s ErroringStorage) Location() string {
	return "erroringstorage"
}

func setupTestServer(t *testing.T, addr string, store trustmanager.Storage) func() {
	s := grpc.NewServer()
	st := NewGRPCStorage(store)
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
	return func() {
		s.Stop()
		l.Close()
	}
}

func TestRemoteStore(t *testing.T) {
	name := "testfile"
	bytes := []byte{'1'}
	addr := "localhost:9888"

	closer := setupTestServer(t, addr, storage.NewMemoryStore(nil))
	defer closer()

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
		timeout:  DefaultTimeout,
	}

	loc := c.Location()
	require.Equal(t, "Remote Key Store @ "+addr, loc)

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

// GRPC converts our errors into *grpc.rpcError types.
func TestErrors(t *testing.T) {
	name := "testfile"
	bytes := []byte{'1'}
	addr := "localhost:9887"

	closer := setupTestServer(t, addr, ErroringStorage{})
	defer closer()

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
		timeout:  DefaultTimeout,
	}

	err = c.Set(name, bytes)
	require.Error(t, err)
	require.Equal(t, "test error", grpc.ErrorDesc(err))

	_, err = c.Get(name)
	require.Error(t, err)
	require.Equal(t, "test error", grpc.ErrorDesc(err))

	err = c.Remove(name)
	require.Error(t, err)
	require.Equal(t, "test error", grpc.ErrorDesc(err))
}
