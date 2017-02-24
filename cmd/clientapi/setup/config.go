package setup

import (
	"crypto/tls"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/notary/utils"
	"github.com/spf13/viper"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	ghealth "google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type Config struct {
	GRPCAddr  string
	TLSConfig *tls.Config
}

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

func SetupGRPCServer(serverConfig Config) (*grpc.Server, net.Listener, error) {

	//RPC server setup
	hs := ghealth.NewServer()

	lis, err := net.Listen("tcp", serverConfig.GRPCAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("grpc server failed to listen on %s: %v",
			serverConfig.GRPCAddr, err)
	}

	creds := credentials.NewTLS(serverConfig.TLSConfig)
	opts := []grpc.ServerOption{grpc.Creds(creds)}
	grpcServer := grpc.NewServer(opts...)

	healthpb.RegisterHealthServer(grpcServer, hs)

	return grpcServer, lis, nil
}

func GetAddrAndTLSConfig(configuration *viper.Viper) (string, *tls.Config, error) {
	tlsConfig, err := utils.ParseServerTLS(configuration, true)
	if err != nil {
		return "", nil, fmt.Errorf("unable to set up TLS: %s", err.Error())
	}

	grpcAddr := configuration.GetString("server.grpc_addr")
	if grpcAddr == "" {
		return "", nil, fmt.Errorf("grpc listen address required for server")
	}

	return grpcAddr, tlsConfig, nil
}
