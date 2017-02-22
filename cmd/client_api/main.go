package main

import (
	"flag"
	"github.com/Sirupsen/logrus"
	"github.com/docker/notary/client_api/api"
	"github.com/docker/notary/cmd/client_api/setup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"net"
)

var (
	configPath string
)

func init() {
	flag.StringVar(
		&configPath,
		"config",
		"config.toml",
		"path to configuration file; supported formats are JSON, YAML, and TOML",
	)
	flag.Parse()
}

func main() {
	vc, err := setup.ViperConfig(configPath)
	if err != nil {
		logrus.Fatal(err)
	}

	logrus.SetLevel(logrus.DebugLevel)

	// FIXME: refactor with notary-signer/config.go:getAddrAndTLSConfig()
	// and setupGRPCServer()
	grpcAddr, tlsConfig, err := setup.GetAddrAndTLSConfig(vc)
	if err != nil {
		logrus.Fatal("unable to set up TLS: %s", err.Error())
	}

	serverConfig := setup.Config{GRPCAddr: grpcAddr, TLSConfig: tlsConfig}

	upstreamAddr := vc.GetString("upstream.addr")

	lis, err := net.Listen("tcp", serverConfig.GRPCAddr)
	if err != nil {
		logrus.Fatal("grpc server failed to listen on %s: %v",
			serverConfig.GRPCAddr, err)
	}

	creds := credentials.NewTLS(serverConfig.TLSConfig)
	opts := []grpc.ServerOption{grpc.Creds(creds)}
	s, err := api.NewServer(upstreamAddr, opts)
	if err != nil {
		logrus.Fatal(err)
	}

	//	grpcServer, lis, err := setup.SetupGRPCServer(serverConfig)

	logrus.Infof("serving on %s", grpcAddr)
	if err := s.Serve(lis); err != nil {
	}
	logrus.Info("server shutting down")
}
