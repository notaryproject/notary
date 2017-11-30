package memory

import "fmt"

func init() {
	fmt.Printf(`!! DEPRECATION NOTICE !!
    You are importing github.com/flimzy/driver/memory which has been deprecated.
    Please use github.com/go-kivik/memorydb instead.
    See https://github.com/flimzy/kivik/issues/178 for more information.
`)
}
