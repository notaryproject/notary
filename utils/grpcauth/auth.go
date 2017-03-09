package grpcauth

import (
	"github.com/Sirupsen/logrus"
	"github.com/docker/notary/utils/token"
	google_protobuf "github.com/golang/protobuf/ptypes/empty"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

type guner interface {
	GetGun() string
}

// ServerAuthorizer performs server checks for the correct authorization tokens
type ServerAuthorizer struct {
	permissions map[string][]string
	auth        *token.Auth
}

// NewServerAuthorizer instantiates a ServerAuthorizer and returns the Interceptor
// attached to it.
func NewServerAuthorizer(auth *token.Auth, permissions map[string][]string) (grpc.UnaryServerInterceptor, error) {
	s := ServerAuthorizer{
		permissions: permissions,
		auth:        auth,
	}
	return s.Interceptor, nil
}

// Interceptor checks the provided tokens and either returns an error that includes the required
// token scope and actions, or allows the request to proceed
// TODO: are the error responses the ones we want to use
func (s ServerAuthorizer) Interceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	if s.auth != nil {
		gnr, ok := req.(guner)
		if !ok {
			return &google_protobuf.Empty{}, grpc.Errorf(
				codes.Unauthenticated,
				"no authorization credentials provided",
			)
		}
		md, ok := metadata.FromContext(ctx)
		var rawToken string
		if ok {
			ts := md["Authorization"]
			if len(ts) > 0 {
				rawToken = ts[0]
			}
		}
		if _, err := s.auth.Authorize(rawToken); !ok || err != nil {
			md := s.auth.ChallengeHeaders(
				err,
				token.BuildAccessRecords(
					gnr.GetGun(),
					s.permissions[info.FullMethod]...,
				)...,
			)
			grpc.SendHeader(ctx, md)
			return &google_protobuf.Empty{}, grpc.Errorf(
				codes.Unauthenticated,
				"no authorization credentials provided",
			)
		}
	}
	return handler(ctx, req)
}

// ClientAuthorizer deals with satisfying tokens required by the server. If it receives an
// error response, it will attempt to retrieve a token the server will accept
type ClientAuthorizer struct{}

func NewClientAuthorizer() grpc.UnaryClientInterceptor {
	c := ClientAuthorizer{}
	return c.Interceptor
}

// Interceptor attempts to retrieve and attach the appropriate tokens for the request
// being made
func (c *ClientAuthorizer) Interceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	headers := metadata.MD{}
	opts = append(opts, grpc.Header(&headers))
	err := invoker(ctx, method, req, reply, cc, opts...)
	if err != nil {
		logrus.Error(err)
		//return err
	}

	md := metadata.New(map[string]string{"Authorization": "foo"})
	ctx = metadata.NewContext(ctx, md)
	err = invoker(ctx, method, req, reply, cc, opts...)
	return err
}
