package grpcauth

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution/registry/auth"
	google_protobuf "github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"strings"
)

type guner interface {
	GetGun() string
}

// ServerAuthorizer performs server checks for the correct authorization tokens
type ServerAuthorizer struct {
	permissions map[string][]string
	realm       string
	service     string
}

// NewServerAuthorizer instantiates a ServerAuthorizer and returns the Interceptor
// attached to it.
func NewServerAuthorizer(tokenCAPath string, permissions map[string][]string) (grpc.UnaryServerInterceptor, error) {
	s := ServerAuthorizer{
		permissions: permissions,
	}
	return s.Interceptor, nil
}

// Interceptor checks the provided tokens and either returns an error that includes the required
// token scope and actions, or allows the request to proceed
// TODO: are the error responses the ones we want to use
func (s ServerAuthorizer) Interceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	gnr, ok := req.(guner)
	if !ok {
		if !ok {
			return &google_protobuf.Empty{}, grpc.Errorf(
				codes.Unauthenticated,
				"no authorization credentials provided",
			)
		}
	}
	md, ok := metadata.FromContext(ctx)
	if !ok || !s.authorized(md) {
		md, ok := s.buildAuthChallenge(gnr.GetGun(), info.FullMethod)
		if !ok {
			return &google_protobuf.Empty{}, grpc.Errorf(
				codes.Unauthenticated,
				"no authorization credentials provided",
			)
		}
		grpc.SendHeader(ctx, md)
		return &google_protobuf.Empty{}, grpc.Errorf(
			codes.Unauthenticated,
			"no authorization credentials provided",
		)
	}
	return handler(ctx, req)
}

func (s ServerAuthorizer) buildAuthChallenge(gun, method string) (metadata.MD, bool) {
	str := fmt.Sprintf("Bearer realm=%q,service=%q", s.realm, s.service)

	perms, ok := s.permissions[method]
	if !ok {
		return nil, ok
	}
	access := buildAccessRecords(gun, perms...)

	scopes := make([]string, 0, len(access))

	for resource, actions := range access {
		scopes = append(scopes, fmt.Sprintf(
			"%s:%s:%s",
			resource.Type,
			resource.Name,
			strings.Join(actions, ",")),
		)
	}

	scope := strings.Join(scopes, " ")

	str = fmt.Sprintf("%s,scope=%q", str, scope)
	return metadata.MD{
		"WWW-Authenticate": []string{str},
	}, true
}

func (s ServerAuthorizer) authorized(md metadata.MD) bool {
	_, ok := md["Authorization"]
	return ok
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

func buildAccessRecords(repo string, actions ...string) map[auth.Resource][]string {
	return map[auth.Resource][]string{
		auth.Resource{
			Type: "repository",
			Name: repo,
		}: actions,
	}
}
