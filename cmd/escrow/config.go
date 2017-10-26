package main

import (
	"fmt"
	"net"

	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/theupdateframework/notary"
	"github.com/theupdateframework/notary/storage"
	"github.com/theupdateframework/notary/trustmanager"
	"github.com/theupdateframework/notary/trustmanager/remoteks"
	"github.com/theupdateframework/notary/utils"
)

func parseConfig(path string) (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigFile(path)
	return v, v.ReadInConfig()
}

func setupGRPCServer(v *viper.Viper) (*grpc.Server, error) {
	storage, err := setupStorage(v)
	if err != nil {
		return nil, err
	}

	tlsConfig, err := utils.ParseServerTLS(v, !v.GetBool("server.insecure"))
	if err != nil {
		return nil, err
	}
	creds := credentials.NewTLS(tlsConfig)
	opts := []grpc.ServerOption{grpc.Creds(creds)}
	server := grpc.NewServer(opts...)
	keyStore := remoteks.NewGRPCStorage(storage)
	remoteks.RegisterStoreServer(server, keyStore)
	return server, nil
}

func setupStorage(v *viper.Viper) (trustmanager.Storage, error) {
	backend := v.GetString("storage.backend")
	switch backend {
	case notary.MemoryBackend:
		return storage.NewMemoryStore(nil), nil
	case notary.FileBackend:
		return storage.NewFileStore(v.GetString("storage.path"), notary.KeyExtension)
	}
	return nil, fmt.Errorf("%s is not an allowed backend for the Key Store interface", backend)
}

func setupNetListener(v *viper.Viper) (net.Listener, error) {
	return net.Listen(
		"tcp",
		v.GetString("server.addr"),
	)
}
