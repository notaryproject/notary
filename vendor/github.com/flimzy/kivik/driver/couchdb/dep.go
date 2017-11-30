package couchdb

import "fmt"

func init() {
	fmt.Printf(`!! DEPRECATION NOTICE !!
    You are importing github.com/flimzy/driver/couchdb which has been deprecated.
    Please use github.com/go-kivik/couchdb instead.
    See https://github.com/flimzy/kivik/issues/178 for more information.
`)
}
