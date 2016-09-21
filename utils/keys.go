package utils

import (
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/notary"
	tufdata "github.com/docker/notary/tuf/data"
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
	sort.Strings(keys) // ensure consistency. ListFiles has no order guarantee
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
func ImportKeys(from io.Reader, to []Importer, fallbackRole string, fallbackGun string, passRet notary.PassRetriever) error {
	data, err := ioutil.ReadAll(from)
	if err != nil {
		return err
	}
	var (
		writeTo string
		toWrite []byte
	)
	for block, rest := pem.Decode(data); block != nil; block, rest = pem.Decode(rest) {
		// if there is a path then we set the gun header from this path
		if rawPath := block.Headers["path"]; rawPath != "" {
			pathWOFileName := strings.TrimSuffix(rawPath, filepath.Base(rawPath))
			if strings.HasPrefix(pathWOFileName, notary.NonRootKeysSubdir) {
				gunName := strings.TrimPrefix(pathWOFileName, notary.NonRootKeysSubdir)
				gunName = gunName[1:(len(gunName) - 1)] // remove the slashes
				if gunName != "" {
					block.Headers["gun"] = gunName
				}
			}
		}
		if block.Headers["gun"] == "" {
			if fallbackGun != "" {
				block.Headers["gun"] = fallbackGun
			}
		}
		if block.Headers["role"] == "" {
			if fallbackRole == "" {
				block.Headers["role"] = notary.DefaultImportRole
			} else {
				block.Headers["role"] = fallbackRole
			}
		}
		loc, ok := block.Headers["path"]
		// only if the path isn't specified do we get into this parsing path logic
		if !ok || loc == "" {
			// if the path isn't specified, we will try to infer the path rel to trust dir from the role (and then gun)
			// parse key for the keyID which we will save it by.
			// if the key is encrypted at this point, we will generate an error and continue since we don't know the ID to save it by
			decodedKey, err := utils.ParsePEMPrivateKey(pem.EncodeToMemory(block), "")
			if err != nil {
				logrus.Info("failed to import key to store: Invalid key generated, key may be encrypted and does not contain path header")
				continue
			}
			keyID := decodedKey.ID()
			switch block.Headers["role"] {
			case tufdata.CanonicalRootRole:
				// this is a root key so import it to trustDir/root_keys/
				loc = filepath.Join(notary.RootKeysSubdir, keyID)
			case tufdata.CanonicalSnapshotRole, tufdata.CanonicalTargetsRole, tufdata.CanonicalTimestampRole:
				// this is a canonical key
				loc = filepath.Join(notary.NonRootKeysSubdir, block.Headers["gun"], keyID)
			default:
				//this is a delegation key
				loc = filepath.Join(notary.NonRootKeysSubdir, keyID)
			}
		}

		// A root key or a delegations key should not have a gun
		// Note that a key that is not any of the canonical roles (except root) is a delegations key and should not have a gun
		if block.Headers["role"] != tufdata.CanonicalSnapshotRole && block.Headers["role"] != tufdata.CanonicalTargetsRole && block.Headers["role"] != tufdata.CanonicalTimestampRole {
			delete(block.Headers, "gun")
		} else {
			// check if the key is missing a gun header or has an empty gun and error out since we don't know where to import this key to
			if block.Headers["gun"] == "" {
				logrus.Info("failed to import key to store: Cannot have canonical role key without a gun, don't know where to import it")
				continue
			}
		}

		// the path header is not of any use once we've imported the key so strip it away
		delete(block.Headers, "path")

		// we are now all set for import but let's first encrypt the key
		blockBytes := pem.EncodeToMemory(block)
		// check if key is encrypted, note: if it is encrypted at this point, it will have had a path header
		if privKey, err := utils.ParsePEMPrivateKey(blockBytes, ""); err == nil {
			// Key is not encrypted- ask for a passphrase and encrypt this key
			var chosenPassphrase string
			for attempts := 0; ; attempts++ {
				var giveup bool
				chosenPassphrase, giveup, err = passRet(loc, block.Headers["role"], true, attempts)
				if err == nil {
					break
				}
				if giveup || attempts > 10 {
					return errors.New("maximum number of passphrase attempts exceeded")
				}
			}
			blockBytes, err = utils.EncryptPrivateKey(privKey, block.Headers["role"], block.Headers["gun"], chosenPassphrase)
			if err != nil {
				return errors.New("failed to encrypt key with given passphrase")
			}
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

		toWrite = append(toWrite, blockBytes...)
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
