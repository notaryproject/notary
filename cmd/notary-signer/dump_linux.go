package main

import (
	"golang.org/x/sys/unix"
)

func protect() error {
	// Make sure process is not dumpable, so will not core dump, which would
	// write keys to disk, and cannot be ptraced to read keys.
	return unix.Prctl(unix.PR_SET_DUMPABLE, 0, 0, 0, 0)
}
