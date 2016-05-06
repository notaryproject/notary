package store

import (
	"io"
)

// ErrOffline is used to indicate we are operating offline
type ErrOffline struct{}

func (e ErrOffline) Error() string {
	return "client is offline"
}

// OfflineStore is to be used as a placeholder for a nil store. It simply
// returns ErrOffline for every operation
type OfflineStore struct{}

// GetMeta returns ErrOffline
func (es OfflineStore) GetMeta(name string, size int64) ([]byte, error) {
	return nil, ErrOffline{}
}

// SetMeta returns ErrOffline
func (es OfflineStore) SetMeta(name string, blob []byte) error {
	return ErrOffline{}
}

// SetMultiMeta returns ErrOffline
func (es OfflineStore) SetMultiMeta(map[string][]byte) error {
	return ErrOffline{}
}

// RemoveMeta returns ErrOffline
func (es OfflineStore) RemoveMeta(name string) error {
	return ErrOffline{}
}

// GetKey returns ErrOffline
func (es OfflineStore) GetKey(role string) ([]byte, error) {
	return nil, ErrOffline{}
}

// GetTarget returns ErrOffline
func (es OfflineStore) GetTarget(path string) (io.ReadCloser, error) {
	return nil, ErrOffline{}
}

// RemoveAll return ErrOffline
func (es OfflineStore) RemoveAll() error {
	return ErrOffline{}
}
