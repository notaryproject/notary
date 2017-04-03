package main

import (
	"github.com/docker/notary/tuf/data"
	"github.com/docker/notary/storage"
)

// ExportStore is a wrapper around a Filesystem store which filters requested roles.
type ExportStore struct {
	store	*storage.FilesystemStore
	Roles	[]data.RoleName
}

func NewExportStore(baseDir, fileExt string, roles []data.RoleName) (*ExportStore, error) {
	store, err := storage.NewFileStore(baseDir, fileExt)
	if err != nil {
		return nil, err
	}

	return &ExportStore{
		store: store,
		Roles: roles,
	}, nil
}

func (e *ExportStore) GetSized(name string, size int64) ([]byte, error) {
	if len(e.Roles) == 0 {
		return e.store.GetSized(name, size)
	}

	for _, role := range e.Roles {
		if name == role.String() {
			return e.store.GetSized(name, size)
		}
	}

	return nil, storage.ErrMetaNotFound{Resource: name}
}

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

func (e *ExportStore) SetMulti(metas map[string][]byte) error {
	for role, blob := range metas {
		err := e.Set(role, blob)
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *ExportStore) RemoveAll() error {
	return e.store.RemoveAll()
}

func (e *ExportStore) Remove(name string) error {
	return e.store.Remove(name)
}
