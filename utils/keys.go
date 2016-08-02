package utils

import (
	"encoding/pem"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/notary"
	"github.com/docker/notary/tuf/utils"
	"io"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"
)

// Exporter is a simple interface for the two functions we need from the Storage interface
type Exporter interface {
	Get(string) ([]byte, error)
	ListFiles() []string
}

// Importer is a simple interface for the one function we need from the Storage interface
type Importer interface {
	Set(string, []byte) error
}

// ExportKeysByGUN exports all keys filtered to a GUN
func ExportKeysByGUN(to io.Writer, s Exporter, gun string) error {
	keys := s.ListFiles()
	sort.Strings(keys) // ensure consistenct. ListFiles has no order guarantee
	for _, k := range keys {
		dir := filepath.Dir(k)
		if dir == gun { // must be full GUN match
			if err := ExportKeys(to, s, k); err != nil {
				return err
			}
		}
	}
	return nil
}

// ExportKeysByID exports all keys matching the given ID
func ExportKeysByID(to io.Writer, s Exporter, ids []string) error {
	want := make(map[string]struct{})
	for _, id := range ids {
		want[id] = struct{}{}
	}
	keys := s.ListFiles()
	for _, k := range keys {
		id := filepath.Base(k)
		if _, ok := want[id]; ok {
			if err := ExportKeys(to, s, k); err != nil {
				return err
			}
		}
	}
	return nil
}

// ExportKeys copies a key from the store to the io.Writer
func ExportKeys(to io.Writer, s Exporter, from string) error {
	// get PEM block
	k, err := s.Get(from)
	if err != nil {
		return err
	}

	gun := ""
	if strings.HasPrefix(from, notary.NonRootKeysSubdir) {
		// trim subdir
		gun = strings.TrimPrefix(from, notary.NonRootKeysSubdir)
		// trim filename
		gun = filepath.Dir(gun)
		// trim leading and trailing path separator
		gun = strings.Trim(gun, fmt.Sprintf("%c", filepath.Separator))
	}
	// parse PEM blocks if there are more than one
	for block, rest := pem.Decode(k); block != nil; block, rest = pem.Decode(rest) {
		// add from path in a header for later import
		block.Headers["path"] = from
		block.Headers["gun"] = gun
		// write serialized PEM
		err = pem.Encode(to, block)
		if err != nil {
			return err
		}
	}
	return nil
}

// ImportKeys expects an io.Reader containing one or more PEM blocks.
// It reads PEM blocks one at a time until pem.Decode returns a nil
// block.
// Each block is written to the subpath indicated in the "path" PEM
// header. If the file already exists, the file is truncated. Multiple
// adjacent PEMs with the same "path" header are appended together.
func ImportKeys(from io.Reader, to []Importer, role string, gun string) error {
	data, err := ioutil.ReadAll(from)
	if err != nil {
		return err
	}
	var (
		writeTo string
		toWrite []byte
	)
	for block, rest := pem.Decode(data); block != nil; block, rest = pem.Decode(rest) {
		if block.Headers["role"] == "" {
			// no worries about if check as for GUN here because empty roles will get a role:notary.DefaultImportRole
			block.Headers["role"] = role
		}
		if block.Headers["gun"] == "" {
			if gun!= "" {
				block.Headers["gun"] = gun
			}
		}
		loc, ok := block.Headers["path"]
		// only if the path isn't specified do we get into this parsing path logic
		if !ok || loc == "" {
			if block.Headers["role"] == "" {
				// now we have no clue where to copy this key so we skip it since we have no path or role
				logrus.Info("failed to import key to store: PEM headers did not contain import path")
				continue
			}
			// if the path isn't specified, we will try to infer the path rel to trust dir from the role (and then gun)

			decodedKey, err := utils.ParsePEMPrivateKey(data, "")
			if err!=nil {
				logrus.Info("failed to import key to store: Invalid key generated, key may be encrypted and not contains path header")
				continue
			}
			keyID := decodedKey.ID()

			if block.Headers["role"] == "root" {
				// does not make sense for root keys to have GUNs, so import it without the GUN even if specified
				loc = filepath.Join(notary.RootKeysSubdir, keyID)
			} else {
				// additional path inference from gun
				loc = filepath.Join(notary.NonRootKeysSubdir, block.Headers["gun"], keyID)
			}
		}
		// this is the path + no-role case where we assume the key is a delegation key
		if block.Headers["role"] == "" {
			block.Headers["role"] = notary.DefaultImportRole
		}

		if loc != writeTo {
			// next location is different from previous one. We've finished aggregating
			// data for the previous file. If we have data, write the previous file,
			// the clear toWrite and set writeTo to the next path we're going to write
			if toWrite != nil {
				if err = importToStores(to, writeTo, toWrite); err != nil {
					return err
				}
			}
			// set up for aggregating next file's data
			toWrite = nil
			writeTo = loc
		}
		delete(block.Headers, "path")
		toWrite = append(toWrite, pem.EncodeToMemory(block)...)
	}
	if toWrite != nil { // close out final iteration if there's data left
		return importToStores(to, writeTo, toWrite)
	}
	return nil
}

func importToStores(to []Importer, path string, bytes []byte) error {
	var err error
	for _, i := range to {
		if err = i.Set(path, bytes); err != nil {
			logrus.Errorf("failed to import key to store: %s", err.Error())
			continue
		}
		break
	}
	return err
}
