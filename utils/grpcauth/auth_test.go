package grpcauth

import (
	"testing"
	"google.golang.org/grpc"
	"net"
	"github.com/stretchr/testify/require"
	"github.com/docker/notary/client_api/api"
	"github.com/docker/notary/client"
)

func TestServerAuthorizer(t *testing.T) {
	auth, err := NewServerAuthorizer("", nil)
	require.NoError(t, err)
	srv := grpc.NewServer(
		grpc.UnaryInterceptor(auth),
	)
	lis, err := net.Listen("tcp", "localhost:6789")
	require.NoError(t, err)

	api.NewServer("", "", srv)
	go srv.Serve(lis)

	conn, err :=  grpc.Dial(
		"localhost:6789",
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(NewClientAuthorizer()),
	)
	require.NoError(t, err)
	c := api.NewClient(conn, "testRepo")
	err = c.AddTarget(
		&client.Target{},
		"targets",
	)
	require.NoError(t, err)
}