package api

import "google.golang.org/grpc"

type Client struct {}

func NewClient(conn *grpc.ClientConn) (NotaryClient, error) {

	return NewNotaryClient(conn), nil
}
