// +build pkcs11

package common

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnsurePrivateKeySizePassesThroughRightSizeArrays(t *testing.T) {
	fullByteArray := make([]byte, ecdsaPrivateKeySize)
	for i := range fullByteArray {
		fullByteArray[i] = byte(1)
	}

	result := EnsurePrivateKeySize(fullByteArray)
	require.True(t, reflect.DeepEqual(fullByteArray, result))
}

// The pad32Byte helper function left zero-pads byte arrays that are less than
// ecdsaPrivateKeySize bytes
func TestEnsurePrivateKeySizePadsLessThanRequiredSizeArrays(t *testing.T) {
	shortByteArray := make([]byte, ecdsaPrivateKeySize/2)
	for i := range shortByteArray {
		shortByteArray[i] = byte(1)
	}

	expected := append(
		make([]byte, ecdsaPrivateKeySize-ecdsaPrivateKeySize/2),
		shortByteArray...)

	result := EnsurePrivateKeySize(shortByteArray)
	require.True(t, reflect.DeepEqual(expected, result))
}
