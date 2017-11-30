package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/flimzy/kivik"
	_ "github.com/go-kivik/couchdb" // The CouchDB driver
	"github.com/go-kivik/kivikd"
	"github.com/go-kivik/kiviktest"
	_ "github.com/go-kivik/memorydb" // The Memory driver
)

func main() {
	var verbose bool
	pflag.BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	flagVerbose := pflag.Lookup("verbose")

	cmdServe := &cobra.Command{
		Use:   "serve",
		Short: "Start a Kivik test server",
	}
	cmdServe.Flags().AddFlag(flagVerbose)
	var listenAddr string
	cmdServe.Flags().StringVarP(&listenAddr, "http", "", ":5984", "HTTP bind address to serve")
	var driverName string
	cmdServe.Flags().StringVarP(&driverName, "driver", "d", "memory", "Backend driver to use")
	var dsn string
	cmdServe.Flags().StringVarP(&dsn, "dsn", "", "", "Data source name")
	cmdServe.Run = func(cmd *cobra.Command, args []string) {
		service := &kivikd.Service{}

		client, err := kivik.New(context.Background(), driverName, dsn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to connect: %s\n", err)
			os.Exit(1)
		}
		service.Client = client
		if listenAddr != "" {
			service.Bind(listenAddr)
		}
		fmt.Printf("Listening on %s\n", listenAddr)
		fmt.Println(service.Start())
		os.Exit(1)
	}

	cmdTest := &cobra.Command{
		Use:   "test [Remote Server DSN]",
		Short: "Run the test suite against the remote server",
	}
	cmdTest.Flags().AddFlag(flagVerbose)
	// cmdTest.Flags().StringVarP(&dsn, "dsn", "", "", "Data source name")
	var tests []string
	cmdTest.Flags().StringSliceVarP(&tests, "test", "", []string{"auto"}, "List of tests to run")
	var listTests bool
	cmdTest.Flags().BoolVarP(&listTests, "list", "l", false, "List available tests")
	var run string
	cmdTest.Flags().StringVarP(&run, "run", "", "", "Run only those tests matching the regular expression")
	var rw bool
	cmdTest.Flags().BoolVarP(&rw, "write", "w", false, "Allow tests which write to the database")
	var cleanup bool
	cmdTest.Flags().BoolVarP(&cleanup, "cleanup", "c", false, "Clean up after previous test run, then exit")
	cmdTest.Run = func(cmd *cobra.Command, args []string) {
		if listTests {
			kiviktest.ListTests()
			os.Exit(0)
		}
		if len(args) != 1 {
			cmd.Usage()
			os.Exit(1)
		}
		kiviktest.RunTests(kiviktest.Options{
			Driver:  "couch",
			DSN:     args[0],
			Verbose: verbose,
			RW:      rw,
			Suites:  tests,
			Match:   run,
			Cleanup: cleanup,
		})
	}

	rootCmd := &cobra.Command{
		Use:  "kivik",
		Long: "Kivik is a tool for hosting and testing CouchDB services",
	}
	rootCmd.AddCommand(cmdServe, cmdTest)
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(2)
	}
}
