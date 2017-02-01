package main

import (
	_ "expvar"
	_ "net/http/pprof"

	_ "github.com/docker/distribution/registry/auth/htpasswd"
	_ "github.com/docker/distribution/registry/auth/token"
	"github.com/docker/notary/cmd/notary-server/runner"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func main() {
	serverSetup := runner.ReadServerConfig()
	runner.RunServer(serverSetup)
}
