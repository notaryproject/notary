package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution/health"
	"github.com/docker/go-connections/tlsconfig"
	"github.com/docker/notary"
	"github.com/docker/notary/cryptoservice"
	"github.com/docker/notary/passphrase"
	pb "github.com/docker/notary/proto"
	"github.com/docker/notary/signer"
	"github.com/docker/notary/signer/api"
	"github.com/docker/notary/signer/keydbstore"
	"github.com/docker/notary/storage"
	"github.com/docker/notary/storage/rethinkdb"
	"github.com/docker/notary/trustmanager"
	"github.com/docker/notary/tuf/data"
	tufutils "github.com/docker/notary/tuf/utils"
	"github.com/docker/notary/utils"
	"github.com/spf13/viper"
	"gopkg.in/dancannon/gorethink.v2"
)

const (
	envPrefix       = "NOTARY_SIGNER"
	defaultAliasEnv = "DEFAULT_ALIAS"
)

func parseSignerConfig(configFilePath string, doBootstrap bool) (signer.Config, error) {
	config := viper.New()
	utils.SetupViper(config, envPrefix)

	// parse viper config
	if err := utils.ParseViper(config, configFilePath); err != nil {
		return signer.Config{}, err
	}

	// default is error level
	lvl, err := utils.ParseLogLevel(config, logrus.ErrorLevel)
	if err != nil {
		return signer.Config{}, err
	}
	logrus.SetLevel(lvl)

	// parse bugsnag config
	bugsnagConf, err := utils.ParseBugsnag(config)
	if err != nil {
		return signer.Config{}, err
	}
	utils.SetUpBugsnag(bugsnagConf)

	// parse server config
	grpcAddr, tlsConfig, err := getAddrAndTLSConfig(config)
	if err != nil {
		return signer.Config{}, err
	}

	// setup the cryptoservices
	cryptoServices, markFunc, err := setUpCryptoservices(config, []string{notary.MySQLBackend, notary.MemoryBackend, notary.RethinkDBBackend}, doBootstrap)
	if err != nil {
		return signer.Config{}, err
	}

	return signer.Config{
		GRPCAddr:       grpcAddr,
		TLSConfig:      tlsConfig,
		CryptoServices: cryptoServices,
		MarkKeyActive:  markFunc,
	}, nil
}

func getEnv(env string) string {
	v := viper.New()
	utils.SetupViper(v, envPrefix)
	return v.GetString(strings.ToUpper(env))
}

func passphraseRetriever(keyName, alias string, createNew bool, attempts int) (passphrase string, giveup bool, err error) {
	passphrase = getEnv(alias)

	if passphrase == "" {
		return "", false, errors.New("expected env variable to not be empty: " + alias)
	}

	return passphrase, false, nil
}

type markActive func(string) error

// Reads the configuration file for storage setup, and sets up the cryptoservice
// mapping
func setUpCryptoservices(configuration *viper.Viper, allowedBackends []string, doBootstrap bool) (
	signer.CryptoServiceIndex, markActive, error) {
	backend := configuration.GetString("storage.backend")

	if !tufutils.StrSliceContains(allowedBackends, backend) {
		return nil, nil, fmt.Errorf("%s is not an allowed backend, must be one of: %s", backend, allowedBackends)
	}

	var keyStore trustmanager.KeyStore
	var markFunc = func(string) error { return nil }
	switch backend {
	case notary.MemoryBackend:
		keyStore = trustmanager.NewKeyMemoryStore(
			passphrase.ConstantRetriever("memory-db-ignore"))
	case notary.RethinkDBBackend:
		var sess *gorethink.Session
		storeConfig, err := utils.ParseRethinkDBStorage(configuration)
		if err != nil {
			return nil, nil, err
		}
		defaultAlias, err := getDefaultAlias(configuration)
		if err != nil {
			return nil, nil, err
		}
		tlsOpts := tlsconfig.Options{
			CAFile:   storeConfig.CA,
			CertFile: storeConfig.Cert,
			KeyFile:  storeConfig.Key,
		}
		if doBootstrap {
			sess, err = rethinkdb.AdminConnection(tlsOpts, storeConfig.Source)
		} else {
			sess, err = rethinkdb.UserConnection(tlsOpts, storeConfig.Source, storeConfig.Username, storeConfig.Password)
		}
		if err != nil {
			return nil, nil, fmt.Errorf("Error starting %s driver: %s", backend, err.Error())
		}
		s := keydbstore.NewRethinkDBKeyStore(storeConfig.DBName, storeConfig.Username, storeConfig.Password, passphraseRetriever, defaultAlias, sess)
		health.RegisterPeriodicFunc("DB operational", time.Minute, s.CheckHealth)
		markFunc = s.MarkActive

		if doBootstrap {
			keyStore = s
		} else {
			keyStore = keydbstore.NewCachedKeyStore(s)
		}
	case notary.MySQLBackend, notary.SQLiteBackend:
		storeConfig, err := utils.ParseSQLStorage(configuration)
		if err != nil {
			return nil, nil, err
		}
		defaultAlias, err := getDefaultAlias(configuration)
		if err != nil {
			return nil, nil, err
		}
		dbStore, err := keydbstore.NewSQLKeyDBStore(
			passphraseRetriever, defaultAlias, storeConfig.Backend, storeConfig.Source)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create a new keydbstore: %v", err)
		}

		health.RegisterPeriodicFunc(
			"DB operational", time.Minute, dbStore.HealthCheck)
		keyStore = keydbstore.NewCachedKeyStore(dbStore)
		markFunc = dbStore.MarkActive
	}

	if doBootstrap {
		err := bootstrap(keyStore)
		if err != nil {
			logrus.Fatal(err.Error())
		}
		os.Exit(0)
	}

	cryptoService := cryptoservice.NewCryptoService(keyStore)
	cryptoServices := make(signer.CryptoServiceIndex)
	cryptoServices[data.ED25519Key] = cryptoService
	cryptoServices[data.ECDSAKey] = cryptoService
	return cryptoServices, markFunc, nil
}

func getDefaultAlias(configuration *viper.Viper) (string, error) {
	defaultAlias := configuration.GetString("storage.default_alias")
	if defaultAlias == "" {
		// backwards compatibility - support this environment variable
		defaultAlias = configuration.GetString(defaultAliasEnv)
	}

	if defaultAlias == "" {
		return "", fmt.Errorf("must provide a default alias for the key DB")
	}
	logrus.Debug("Default Alias: ", defaultAlias)
	return defaultAlias, nil
}

// set up the GRPC server
func setupGRPCServer(signerConfig signer.Config) (*grpc.Server, net.Listener, error) {

	//RPC server setup
	kms := &api.KeyManagementServer{
		CryptoServices: signerConfig.CryptoServices,
		HealthChecker:  health.CheckStatus,
	}
	ss := &api.SignerServer{
		CryptoServices: signerConfig.CryptoServices,
		HealthChecker:  health.CheckStatus,
		MarkKeyActive:  signerConfig.MarkKeyActive,
	}

	lis, err := net.Listen("tcp", signerConfig.GRPCAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("grpc server failed to listen on %s: %v",
			signerConfig.GRPCAddr, err)
	}

	creds := credentials.NewTLS(signerConfig.TLSConfig)
	opts := []grpc.ServerOption{grpc.Creds(creds)}
	grpcServer := grpc.NewServer(opts...)

	pb.RegisterKeyManagementServer(grpcServer, kms)
	pb.RegisterSignerServer(grpcServer, ss)

	return grpcServer, lis, nil
}

func getAddrAndTLSConfig(configuration *viper.Viper) (string, *tls.Config, error) {
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

func bootstrap(s interface{}) error {
	store, ok := s.(storage.Bootstrapper)
	if !ok {
		return fmt.Errorf("Store does not support bootstrapping.")
	}
	return store.Bootstrap()
}
