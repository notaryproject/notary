[![Build Status](https://travis-ci.org/go-kivik/couchdb.svg?branch=master)](https://travis-ci.org/go-kivik/couchdb) [![Codecov](https://img.shields.io/codecov/c/github/go-kivik/couchdb.svg?style=flat)](https://codecov.io/gh/go-kivik/couchdb) [![GoDoc](https://godoc.org/github.com/go-kivik/couchdb?status.svg)](http://godoc.org/github.com/go-kivik/couchdb)

# Kivik CouchDB

CouchDB driver for [Kivik](https://github.com/go-kivik/couchdb).

## Usage

This package provides an implementation of the
[`github.com/flimzy/kivik/driver`](http://godoc.org/github.com/flimzy/kivik/driver)
interface. You must import the driver and can then use the full
[`Kivik`](http://godoc.org/github.com/flimzy/kivik) API. Please consult the
[Kivik wiki](https://github.com/flimzy/kivik/wiki) for complete documentation
and coding examples.

```go
package main

import (
    "context"

    "github.com/flimzy/kivik"
    _ "github.com/go-kivik/couchdb" // The CouchDB driver
)

func main() {
    client, err := kivik.New(context.TODO(), "pouch", "")
    // ...
}
```

## License

This software is released under the terms of the Apache 2.0 license. See
LICENCE.md, or read the [full license](http://www.apache.org/licenses/LICENSE-2.0).
