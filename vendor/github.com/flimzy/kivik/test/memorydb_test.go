package test

import (
	"testing"

	_ "github.com/go-kivik/memorydb"
	"github.com/go-kivik/memorydb/test"
)

func TestMemory(t *testing.T) {
	test.MemoryTest(t)
}
