package main

import (
	_ "expvar"

	"github.com/docker/notary/cmd/notary-signer/runner"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func main() {
	setup := runner.ReadSignerConfig()
	runner.RunSigner(setup)
}
