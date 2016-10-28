package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"strconv"

	"github.com/docker/notary/tuf/data"
)

type key struct {
	algorithm string
	public    []byte
}

type ver struct {
	version      int
	data         []byte
	createupdate time.Time
}

// we want to keep these sorted by version so that it's in increasing version
// order
type verList []ver

func (k verList) Len() int      { return len(k) }
func (k verList) Swap(i, j int) { k[i], k[j] = k[j], k[i] }
func (k verList) Less(i, j int) bool {
	return k[i].version < k[j].version
}

type change struct {
	id         int
	gun        string
	ver        int
	checksum   string
	recordedAt time.Time
}

func (c change) ChangeID() string      { return fmt.Sprintf("%d", c.id) }
func (c change) GUN() string           { return c.gun }
func (c change) Version() int          { return c.ver }
func (c change) Checksum() string      { return c.checksum }
func (c change) RecordedAt() time.Time { return c.recordedAt }

// MemStorage is really just designed for dev and testing. It is very
// inefficient in many scenarios
type MemStorage struct {
	lock      sync.Mutex
	tufMeta   map[string]verList
	keys      map[string]map[string]*key
	checksums map[string]map[string]ver
	changes   []Change
}

// NewMemStorage instantiates a memStorage instance
func NewMemStorage() *MemStorage {
	return &MemStorage{
		tufMeta:   make(map[string]verList),
		keys:      make(map[string]map[string]*key),
		checksums: make(map[string]map[string]ver),
	}
}

// UpdateCurrent updates the meta data for a specific role
func (st *MemStorage) UpdateCurrent(gun string, update MetaUpdate) error {
	id := entryKey(gun, update.Role)
	st.lock.Lock()
	defer st.lock.Unlock()
	if space, ok := st.tufMeta[id]; ok {
		for _, v := range space {
			if v.version >= update.Version {
				return ErrOldVersion{}
			}
		}
	}
	version := ver{version: update.Version, data: update.Data, createupdate: time.Now()}
	st.tufMeta[id] = append(st.tufMeta[id], version)
	checksumBytes := sha256.Sum256(update.Data)
	checksum := hex.EncodeToString(checksumBytes[:])

	_, ok := st.checksums[gun]
	if !ok {
		st.checksums[gun] = make(map[string]ver)
	}
	st.checksums[gun][checksum] = version
	if update.Role == data.CanonicalTimestampRole {
		st.writeChange(gun, update.Version, checksum)
	}
	return nil
}

// writeChange must only be called by a function already holding a lock on
// the MemStorage. Behaviour is undefined otherwise
func (st *MemStorage) writeChange(gun string, version int, checksum string) {
	c := change{
		id:         len(st.changes),
		gun:        gun,
		ver:        version,
		checksum:   checksum,
		recordedAt: time.Now(),
	}
	st.changes = append(st.changes, c)
}

// UpdateMany updates multiple TUF records
func (st *MemStorage) UpdateMany(gun string, updates []MetaUpdate) error {
	st.lock.Lock()
	defer st.lock.Unlock()

	versioner := make(map[string]map[int]struct{})
	constant := struct{}{}

	// ensure that we only update in one transaction
	for _, u := range updates {
		id := entryKey(gun, u.Role)

		// prevent duplicate versions of the same role
		if _, ok := versioner[u.Role][u.Version]; ok {
			return ErrOldVersion{}
		}
		if _, ok := versioner[u.Role]; !ok {
			versioner[u.Role] = make(map[int]struct{})
		}
		versioner[u.Role][u.Version] = constant

		if space, ok := st.tufMeta[id]; ok {
			for _, v := range space {
				if v.version >= u.Version {
					return ErrOldVersion{}
				}
			}
		}
	}

	for _, u := range updates {
		id := entryKey(gun, u.Role)

		version := ver{version: u.Version, data: u.Data, createupdate: time.Now()}
		st.tufMeta[id] = append(st.tufMeta[id], version)
		sort.Sort(st.tufMeta[id]) // ensure that it's sorted
		checksumBytes := sha256.Sum256(u.Data)
		checksum := hex.EncodeToString(checksumBytes[:])

		_, ok := st.checksums[gun]
		if !ok {
			st.checksums[gun] = make(map[string]ver)
		}
		st.checksums[gun][checksum] = version
		if u.Role == data.CanonicalTimestampRole {
			st.writeChange(gun, u.Version, checksum)
		}
	}
	return nil
}

// GetCurrent returns the createupdate date metadata for a given role, under a GUN.
func (st *MemStorage) GetCurrent(gun, role string) (*time.Time, []byte, error) {
	id := entryKey(gun, role)
	st.lock.Lock()
	defer st.lock.Unlock()
	space, ok := st.tufMeta[id]
	if !ok || len(space) == 0 {
		return nil, nil, ErrNotFound{}
	}
	return &(space[len(space)-1].createupdate), space[len(space)-1].data, nil
}

// GetChecksum returns the createupdate date and metadata for a given role, under a GUN.
func (st *MemStorage) GetChecksum(gun, role, checksum string) (*time.Time, []byte, error) {
	st.lock.Lock()
	defer st.lock.Unlock()
	space, ok := st.checksums[gun][checksum]
	if !ok || len(space.data) == 0 {
		return nil, nil, ErrNotFound{}
	}
	return &(space.createupdate), space.data, nil
}

// Delete deletes all the metadata for a given GUN
func (st *MemStorage) Delete(gun string) error {
	st.lock.Lock()
	defer st.lock.Unlock()
	for k := range st.tufMeta {
		if strings.HasPrefix(k, gun) {
			delete(st.tufMeta, k)
		}
	}
	delete(st.checksums, gun)
	return nil
}

// GetChanges returns a []Change starting from but excluding the record
// identified by changeID. In the context of the memory store, changeID
// is simply an index into st.changes. The ID of a change, and its index
// are equal, therefore, we want to return results starting at index
// changeID+1 to match the exclusivity of the interface definition.
func (st *MemStorage) GetChanges(changeID string, pageSize int, filterName string, reversed bool) ([]Change, error) {
	id, err := strconv.ParseInt(changeID, 10, 32)
	size := len(st.changes)
	if err != nil || size <= int(id) {
		return nil, ErrBadChangeID{id: changeID}
	}
	start := int(id) + 1
	end := start + pageSize
	if end >= size {
		return st.changes[start:], nil
	}
	return st.changes[start:end], nil
}

func entryKey(gun, role string) string {
	return fmt.Sprintf("%s.%s", gun, role)
}
