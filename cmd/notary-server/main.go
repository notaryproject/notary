package main

import (
	_ "expvar"
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof" // #nosec G108 // false positive as it's only listening through debugServer()
	"os"
	"os/signal"
	"runtime"

	"github.com/docker/distribution/health"
	"github.com/sirupsen/logrus"
	"github.com/theupdateframework/notary/server"
	"github.com/theupdateframework/notary/utils"
	"github.com/theupdateframework/notary/version"
)

// DebugAddress is the debug server address to listen on
const (
	jsonLogFormat = "json"
	DebugAddress  = "localhost:8080"
	envPrefix     = "NOTARY_SERVER"
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
	flag.BoolVar(&flagStorage.debug, "debug", false, "Enable the debugging server on localhost:8080")
	flag.StringVar(&flagStorage.logFormat, "logf", "json", "Set the format of the logs. Only 'json' and 'logfmt' are supported at the moment.")
	flag.BoolVar(&flagStorage.doBootstrap, "bootstrap", false, "Do any necessary setup of configured backend storage services")
	flag.BoolVar(&flagStorage.version, "version", false, "Print the version number of notary-server")

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
		fmt.Println("notary-server " + getVersion())
		os.Exit(0)
	}

	if flagStorage.debug {
		go debugServer(DebugAddress)
	}

	// when the server starts print the version for debugging and issue logs later
	logrus.Info(getVersion())

	ctx, serverConfig, err := parseServerConfig(flagStorage.configFile, health.RegisterPeriodicFunc, flagStorage.doBootstrap)
	if err != nil {
		logrus.Fatal(err.Error())
	}

	c := utils.SetupSignalTrap(utils.LogLevelSignalHandle)
	if c != nil {
		defer signal.Stop(c)
	}

	if flagStorage.doBootstrap {
		err = bootstrap(ctx)
	} else {
		logrus.Info("Starting Server")
		err = server.Run(ctx, serverConfig)
	}

	if err != nil {
		logrus.Fatal(err.Error())
	}
	return
}

func usage() {
	fmt.Println("usage:", os.Args[0])
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
