package pouchdb

import "fmt"

func init() {
	fmt.Printf(`!! DEPRECATION NOTICE !!
    You are importing github.com/flimzy/driver/pouchdb which has been deprecated.
    Please use github.com/go-kivik/pouchdb instead.
    See https://github.com/flimzy/kivik/issues/178 for more information.
`)
}
