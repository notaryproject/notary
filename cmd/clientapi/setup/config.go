package setup

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/notary"
	grpcauth "github.com/docker/notary/auth/grpc"
	"github.com/docker/notary/auth/token"
	"github.com/docker/notary/client_api/api"
	"github.com/docker/notary/utils"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	ghealth "google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"net"
)

// the client api will always require push and pull permissions against
// the upstream notary server so at a minimum, all endpoints will request
// these permissions.
var requiredPermissions = []string{"push", "pull"}

func ViperConfig(path string) (*viper.Viper, error) {
	vc := viper.New()
	vc.SetConfigFile(path)
	err := vc.ReadInConfig()
	if err != nil {
		return nil, err
	}
	logrus.Info(vc.AllSettings())
	return vc, nil
}

func NewGRPCServer(addr string, opts []grpc.ServerOption) (*grpc.Server, net.Listener, error) {
	//RPC server setup
	hs := ghealth.NewServer()

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, fmt.Errorf("grpc server failed to listen on %s: %v", addr, err)
	}

	grpcServer := grpc.NewServer(opts...)

	healthpb.RegisterHealthServer(grpcServer, hs)

	return grpcServer, lis, nil
}

func GetAddrAndTLSConfig(vc *viper.Viper) (string, *tls.Config, error) {
	tlsConfig, err := utils.ParseServerTLS(vc, true)
	if err != nil {
		return "", nil, fmt.Errorf("unable to set up TLS: %s", err.Error())
	}

	grpcAddr := vc.GetString("server.grpc_addr")
	if grpcAddr == "" {
		return "", nil, fmt.Errorf("grpc listen address required for server")
	}

	return grpcAddr, tlsConfig, nil
}

func Authorization(vc *viper.Viper) (grpc.UnaryServerInterceptor, error) {
	authType := vc.GetString("auth.type")
	switch authType {
	case notary.AuthTypeToken:
		var (
			realm          = vc.GetString("auth.options.realm")
			service        = vc.GetString("auth.options.service")
			issuer         = vc.GetString("auth.options.issuer")
			rootCAPath     = vc.GetString("auth.options.rootcertbundle")
			permissions    = vc.GetStringMap("auth.options.permissions")
			tokenAuth, err = token.NewAuth(realm, issuer, service, rootCAPath)
		)
		if err != nil {
			return nil, err
		}
		basePerms, err := buildBasePermissionList("push", "pull")
		if err != nil {
			return nil, err
		}
		basePerms, err = mergePermissions(basePerms, permissions)
		if err != nil {
			return nil, err
		}
		return grpcauth.NewServerAuthorizer(tokenAuth, basePerms)

	case "":
		// no auth configured
		return nil, nil
	}
	return nil, fmt.Errorf("unrecognized authorization type: %s", authType)
}

func buildBasePermissionList(requiredPerms ...string) (map[string][]string, error) {
	srv := grpc.Server{}
	apiSrv := api.Server{}
	api.RegisterNotaryServer(&srv, &apiSrv)
	srvInfo := srv.GetServiceInfo()
	svc, ok := srvInfo["api.Notary"]
	if !ok {
		return nil, errors.New("could not find api.Notary service")
	}
	permissions := make(map[string][]string)
	for _, method := range svc.Methods {
		permissions[method.Name] = requiredPerms
	}
	return permissions, nil
}

func mergePermissions(basePerms map[string][]string, additional map[string]interface{}) (map[string][]string, error) {
	for k, v := range basePerms {
		if more, ok := additional[k]; ok {
			add, ok := more.([]string)
			if !ok {
				return basePerms, fmt.Errorf("additional permissions for %s could not be parsed as a list of strings", k)
			}
			basePerms[k] = append(v, add...)
		}
	}
	return basePerms, nil
}
