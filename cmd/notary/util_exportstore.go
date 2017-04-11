package main

import (
	"github.com/docker/notary/storage"
	"github.com/docker/notary/tuf/data"
)

// ExportStore is a wrapper around a Filesystem store which filters requested roles.
// It references the set of roles it contains for export and import and the cache
// from the reference repository.
type ExportStore struct {
	store    *storage.FilesystemStore
	Roles    []data.RoleName
	refCache *storage.FilesystemStore
}

// NewExportStore returns a new instance of an ExportStore
func NewExportStore(baseDir, fileExt string, roles []data.RoleName) (
	*ExportStore, error) {
	store, err := storage.NewFileStore(baseDir, fileExt)
	if err != nil {
		return nil, err
	}

	return &ExportStore{
		store: store,
		Roles: roles,
	}, nil
}

// GetSized returns the file requested by name, up to size bytes
func (e *ExportStore) GetSized(name string, size int64) ([]byte, error) {
	if len(e.Roles) == 0 {
		return e.store.GetSized(name, size)
	}

	for _, role := range e.Roles {
		if name == role.String() {
			jsonBytes, err := e.store.GetSized(name, size)
			if err != nil {
				return nil, err
			}

			return jsonBytes, nil
		}
	}

	return nil, storage.ErrMetaNotFound{Resource: name}
}

// Set sets the value for the provided name
func (e *ExportStore) Set(name string, blob []byte) error {
	if len(e.Roles) == 0 {
		return e.store.Set(name, blob)
	}

	for _, role := range e.Roles {
		if name == role.String() {
			return e.store.Set(name, blob)
		}
	}

	return nil
}

// SetMulti sets the values for all names in the metas map
func (e *ExportStore) SetMulti(metas map[string][]byte) error {
	for role, blob := range metas {
		err := e.Set(role, blob)
		if err != nil {
			return err
		}
	}

	return nil
}

// RemoveAll cleans out the store
func (e *ExportStore) RemoveAll() error {
	return e.store.RemoveAll()
}

// Remove deletes a single item, references by name, from the store
func (e *ExportStore) Remove(name string) error {
	return e.store.Remove(name)
}
