package main

import (
	"fmt"
	"github.com/docker/notary"
	"github.com/docker/notary/storage"
	"github.com/docker/notary/trustmanager"
	"github.com/docker/notary/trustmanager/remoteks"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"net"
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
	server := grpc.NewServer()
	keyStore := remoteks.NewGRPCStorage(storage)
	remoteks.RegisterStoreServer(server, keyStore)
	return server, nil
}

func setupStorage(v *viper.Viper) (trustmanager.Storage, error) {
	backend := v.GetString("backend")
	switch backend {
	case notary.MemoryBackend:
		return storage.NewMemoryStore(nil), nil
	case notary.FileBackend:
		return storage.NewFileStore(v.GetString("path"), notary.KeyExtension)
	}
	return nil, fmt.Errorf("%s is not an allowed backend for the Key Store interface", backend)
}

func setupNetListener(v *viper.Viper) (net.Listener, error) {
	return net.Listen(
		"tcp",
		v.GetString("addr"),
	)
}
