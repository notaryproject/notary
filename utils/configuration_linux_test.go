// +build !windows

package utils

import (
	"os"
	"os/signal"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/theupdateframework/notary"
)

func testSetSignalTrap(t *testing.T) {
	var signalsPassedOn map[string]struct{}

	signalHandler := func(s os.Signal) {
		signalsPassedOn := make(map[string]struct{})
		signalsPassedOn[s.String()] = struct{}{}
	}
	c := SetupSignalTrap(signalHandler)

	if len(notary.NotarySupportedSignals) == 0 { // currently, windows only
		require.Nil(t, c)
	} else {
		require.NotNil(t, c)
		defer signal.Stop(c)
	}

	for _, s := range notary.NotarySupportedSignals {
		syscallSignal, ok := s.(syscall.Signal)
		require.True(t, ok)
		require.NoError(t, syscall.Kill(syscall.Getpid(), syscallSignal))
		require.Len(t, signalsPassedOn, 0)
		require.NotNil(t, signalsPassedOn[s.String()])
	}
}

// TODO: undo this extra indirection, needed for mocking notary.NotarySupportedSignals being empty, when we have
// a windows CI system running
func TestSetSignalTrap(t *testing.T) {
	testSetSignalTrap(t)
}
