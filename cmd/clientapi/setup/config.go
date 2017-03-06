package setup

import (
	"crypto/tls"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/notary/utils"
	"github.com/spf13/viper"
	"net"

	"google.golang.org/grpc"
	ghealth "google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"github.com/docker/notary/utils/grpcauth"
)

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
	return grpcauth.NewServerAuthorizer("", nil)
}
