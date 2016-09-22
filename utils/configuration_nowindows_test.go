// +build !windows

package utils

import (
	"io/ioutil"
	"os"
	"syscall"
	"testing"

	"github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestLogLevelSignalHandle(t *testing.T) {
	tempdir, err := ioutil.TempDir("", "test-signal-handle")
	require.NoError(t, err)
	defer os.RemoveAll(tempdir)

	logrus.SetLevel(logrus.InfoLevel)

	// Info + SIGUSR1 -> Debug
	LogLevelSignalHandle(syscall.SIGUSR1)
	require.Equal(t, logrus.GetLevel(), logrus.DebugLevel)

	// Debug + SIGUSR1 -> Debug
	LogLevelSignalHandle(syscall.SIGUSR1)
	require.Equal(t, logrus.GetLevel(), logrus.DebugLevel)

	// Debug + SIGUSR2-> Info
	LogLevelSignalHandle(syscall.SIGUSR2)
	require.Equal(t, logrus.GetLevel(), logrus.InfoLevel)
}
