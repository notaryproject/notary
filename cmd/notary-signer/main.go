package main

import (
	_ "expvar"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"github.com/theupdateframework/notary/utils"
	"github.com/theupdateframework/notary/version"
)

const (
	jsonLogFormat = "json"
	debugAddr     = "localhost:8080"
)

type cmdFlags struct {
	debug       bool
	logFormat   string
	configFile  string
	doBootstrap bool
	version     bool
}

func setupFlags(flagStorage *cmdFlags) {
	// Setup flags
	flag.StringVar(&flagStorage.configFile, "config", "", "Path to configuration file")
	flag.BoolVar(&flagStorage.debug, "debug", false, "Run in debug mode, enables Go debug server")
	flag.StringVar(&flagStorage.logFormat, "logf", "json", "Set the format of the logs. Only 'json' and 'logfmt' are supported at the moment.")
	flag.BoolVar(&flagStorage.doBootstrap, "bootstrap", false, "Do any necessary setup of configured backend storage services")
	flag.BoolVar(&flagStorage.version, "version", false, "Print the version number of notary-signer")

	// this needs to be in init so that _ALL_ logs are in the correct format
	if flagStorage.logFormat == jsonLogFormat {
		logrus.SetFormatter(new(logrus.JSONFormatter))
	}

	flag.Usage = usage
}

func main() {
	flagStorage := cmdFlags{}
	setupFlags(&flagStorage)

	flag.Parse()

	if flagStorage.version {
		fmt.Println("notary-signer " + getVersion())
		os.Exit(0)
	}

	if flagStorage.debug {
		go debugServer(debugAddr)
	} else {
		// If not in debug mode, stop tracing, core dumps if supported to help protect keys.
		if err := protect(); err != nil {
			logrus.Fatal(err.Error())
		}
	}

	// when the signer starts print the version for debugging and issue logs later
	logrus.Info(getVersion())

	signerConfig, err := parseSignerConfig(flagStorage.configFile, flagStorage.doBootstrap)
	if err != nil {
		logrus.Fatal(err.Error())
	}

	grpcServer, lis, err := setupGRPCServer(signerConfig)
	if err != nil {
		logrus.Fatal(err.Error())
	}

	if flagStorage.debug {
		log.Println("RPC server listening on", signerConfig.GRPCAddr)
	}

	c := utils.SetupSignalTrap(utils.LogLevelSignalHandle)
	if c != nil {
		defer signal.Stop(c)
	}

	grpcServer.Serve(lis)
}

func usage() {
	log.Println("usage:", os.Args[0], "<config>")
	flag.PrintDefaults()
}

func getVersion() string {
	return fmt.Sprintf("Version: %s, Git commit: %s, Go version: %s", version.NotaryVersion, version.GitCommit, runtime.Version())
}

// debugServer starts the debug server with pprof, expvar among other
// endpoints. The addr should not be exposed externally. For most of these to
// work, tls cannot be enabled on the endpoint, so it is generally separate.
func debugServer(addr string) {
	logrus.Infof("Debug server listening on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		logrus.Fatalf("error listening on debug interface: %v", err)
	}
}
