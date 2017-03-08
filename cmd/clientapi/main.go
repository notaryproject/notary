package main

import (
	"flag"
	"github.com/Sirupsen/logrus"
	"github.com/docker/notary/client_api/api"
	"github.com/docker/notary/cmd/clientapi/setup"
	"github.com/docker/notary/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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

	upstreamAddr := vc.GetString("upstream.addr")
	upstreamCAPath := utils.GetPathRelativeToConfig(vc, "upstream.tls_ca_file")

	creds := credentials.NewTLS(tlsConfig)
	auth, err := setup.Authorization(vc)
	if err != nil {
		logrus.Fatal("unable to configure authorization: %s", err.Error())
	}

	opts := []grpc.ServerOption{
		grpc.Creds(creds),
		grpc.UnaryInterceptor(auth),
	}

	srv, lis, err := setup.NewGRPCServer(grpcAddr, opts)
	if err != nil {
		logrus.Fatal("grpc server failed to start on %s: %v",
			grpcAddr, err)
	}
	s, err := api.NewServer(upstreamAddr, upstreamCAPath, srv)
	if err != nil {
		logrus.Fatal(err)
	}

	//	grpcServer, lis, err := setup.SetupGRPCServer(serverConfig)

	logrus.Infof("serving on %s", grpcAddr)
	if err := s.Serve(lis); err != nil {
	}
	logrus.Info("server shutting down")
}
