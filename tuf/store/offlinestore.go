package store

import (
	"io"

	"github.com/docker/notary/tuf/data"
	"github.com/docker/notary/tuf/signed"
)

// ErrOffline is used to indicate we are operating offline
type ErrOffline struct{}

func (e ErrOffline) Error() string {
	return "client is offline"
}

var err = ErrOffline{}

// OfflineStore is to be used as a placeholder for a nil store. It simply
// returns ErrOffline for every operation
type OfflineStore struct{}

// GetMeta returns ErrOffline
func (es OfflineStore) GetMeta(name string, size int64) ([]byte, error) {
	return nil, err
}

// SetMeta returns ErrOffline
func (es OfflineStore) SetMeta(name string, blob []byte) error {
	return err
}

// SetMultiMeta returns ErrOffline
func (es OfflineStore) SetMultiMeta(map[string][]byte) error {
	return err
}

// RemoveMeta returns ErrOffline
func (es OfflineStore) RemoveMeta(name string) error {
	return err
}

// GetKey returns ErrOffline
func (es OfflineStore) GetKey(role string) (data.PublicKey, error) {
	return nil, err
}

// RotateKey returns ErrOffline
func (es OfflineStore) RotateKey(string, signed.CryptoService, ...data.PublicKey) (data.PublicKey, error) {
	return nil, err
}

// GetTarget returns ErrOffline
func (es OfflineStore) GetTarget(path string) (io.ReadCloser, error) {
	return nil, err
}

// RemoveAll return ErrOffline
func (es OfflineStore) RemoveAll() error {
	return err
}
