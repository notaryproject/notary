package utils

import (
	"bytes"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"testing"
	"github.com/docker/notary"
	"os"
	"path/filepath"
)

type TestImportStore struct {
	data map[string][]byte
}

func NewTestImportStore() *TestImportStore {
	return &TestImportStore{
		data: make(map[string][]byte),
	}
}

func (s *TestImportStore) Set(name string, data []byte) error {
	s.data[name] = data
	return nil
}

type TestExportStore struct {
	data map[string][]byte
}

func NewTestExportStore() *TestExportStore {
	return &TestExportStore{
		data: make(map[string][]byte),
	}
}

func (s *TestExportStore) Get(name string) ([]byte, error) {
	if data, ok := s.data[name]; ok {
		return data, nil
	}
	return nil, errors.New("Not Found")
}

func (s *TestExportStore) ListFiles() []string {
	files := make([]string, 0, len(s.data))
	for k := range s.data {
		files = append(files, k)
	}
	return files
}

func TestExportKeys(t *testing.T) {
	s := NewTestExportStore()

	b := &pem.Block{}
	b.Bytes = make([]byte, 1000)
	rand.Read(b.Bytes)

	c := &pem.Block{}
	c.Bytes = make([]byte, 1000)
	rand.Read(c.Bytes)

	bBytes := pem.EncodeToMemory(b)
	cBytes := pem.EncodeToMemory(c)

	s.data["ankh"] = bBytes
	s.data["morpork"] = cBytes

	buf := bytes.NewBuffer(nil)

	err := ExportKeys(buf, s, "ankh")
	require.NoError(t, err)

	err = ExportKeys(buf, s, "morpork")
	require.NoError(t, err)

	out, err := ioutil.ReadAll(buf)
	require.NoError(t, err)

	bFinal, rest := pem.Decode(out)
	require.Equal(t, b.Bytes, bFinal.Bytes)
	require.Equal(t, "ankh", bFinal.Headers["path"])

	cFinal, rest := pem.Decode(rest)
	require.Equal(t, c.Bytes, cFinal.Bytes)
	require.Equal(t, "morpork", cFinal.Headers["path"])
	require.Len(t, rest, 0)
}

func TestExportKeysByGUN(t *testing.T) {
	s := NewTestExportStore()

	b := &pem.Block{}
	b.Bytes = make([]byte, 1000)
	rand.Read(b.Bytes)

	b2 := &pem.Block{}
	b2.Bytes = make([]byte, 1000)
	rand.Read(b2.Bytes)

	c := &pem.Block{}
	c.Bytes = make([]byte, 1000)
	rand.Read(c.Bytes)

	bBytes := pem.EncodeToMemory(b)
	b2Bytes := pem.EncodeToMemory(b2)
	cBytes := pem.EncodeToMemory(c)

	s.data["ankh/one"] = bBytes
	s.data["ankh/two"] = b2Bytes
	s.data["morpork/three"] = cBytes

	buf := bytes.NewBuffer(nil)

	err := ExportKeysByGUN(buf, s, "ankh")
	require.NoError(t, err)

	out, err := ioutil.ReadAll(buf)
	require.NoError(t, err)

	bFinal, rest := pem.Decode(out)
	require.Equal(t, b.Bytes, bFinal.Bytes)
	require.Equal(t, "ankh/one", bFinal.Headers["path"])

	b2Final, rest := pem.Decode(rest)
	require.Equal(t, b2.Bytes, b2Final.Bytes)
	require.Equal(t, "ankh/two", b2Final.Headers["path"])
	require.Len(t, rest, 0)
}

func TestExportKeysByID(t *testing.T) {
	s := NewTestExportStore()

	b := &pem.Block{}
	b.Bytes = make([]byte, 1000)
	rand.Read(b.Bytes)

	c := &pem.Block{}
	c.Bytes = make([]byte, 1000)
	rand.Read(c.Bytes)

	bBytes := pem.EncodeToMemory(b)
	cBytes := pem.EncodeToMemory(c)

	s.data["ankh"] = bBytes
	s.data["morpork/identifier"] = cBytes

	buf := bytes.NewBuffer(nil)

	err := ExportKeysByID(buf, s, []string{"identifier"})
	require.NoError(t, err)

	out, err := ioutil.ReadAll(buf)
	require.NoError(t, err)

	cFinal, rest := pem.Decode(out)
	require.Equal(t, c.Bytes, cFinal.Bytes)
	require.Equal(t, "morpork/identifier", cFinal.Headers["path"])
	require.Len(t, rest, 0)
}

func TestExport2InOneFile(t *testing.T) {
	s := NewTestExportStore()

	b := &pem.Block{}
	b.Bytes = make([]byte, 1000)
	rand.Read(b.Bytes)

	b2 := &pem.Block{}
	b2.Bytes = make([]byte, 1000)
	rand.Read(b2.Bytes)

	c := &pem.Block{}
	c.Bytes = make([]byte, 1000)
	rand.Read(c.Bytes)

	bBytes := pem.EncodeToMemory(b)
	b2Bytes := pem.EncodeToMemory(b2)
	bBytes = append(bBytes, b2Bytes...)
	cBytes := pem.EncodeToMemory(c)

	s.data["ankh"] = bBytes
	s.data["morpork"] = cBytes

	buf := bytes.NewBuffer(nil)

	err := ExportKeys(buf, s, "ankh")
	require.NoError(t, err)

	err = ExportKeys(buf, s, "morpork")
	require.NoError(t, err)

	out, err := ioutil.ReadAll(buf)
	require.NoError(t, err)

	bFinal, rest := pem.Decode(out)
	require.Equal(t, b.Bytes, bFinal.Bytes)
	require.Equal(t, "ankh", bFinal.Headers["path"])

	b2Final, rest := pem.Decode(rest)
	require.Equal(t, b2.Bytes, b2Final.Bytes)
	require.Equal(t, "ankh", b2Final.Headers["path"])

	cFinal, rest := pem.Decode(rest)
	require.Equal(t, c.Bytes, cFinal.Bytes)
	require.Equal(t, "morpork", cFinal.Headers["path"])
	require.Len(t, rest, 0)
}

func TestImportKeys(t *testing.T) {
	s := NewTestImportStore()

	from, _ := os.OpenFile("test.key", os.O_RDONLY, notary.PrivKeyPerms)
	b := &pem.Block{
		Headers: make(map[string]string),
	}
	b.Bytes, _ = ioutil.ReadAll(from)
	rand.Read(b.Bytes)
	b.Headers["path"] = "ankh"

	c := &pem.Block{
		Headers: make(map[string]string),
	}
	c.Bytes, _ = ioutil.ReadAll(from)
	rand.Read(c.Bytes)
	c.Headers["path"] = "morpork"

	bBytes := pem.EncodeToMemory(b)
	cBytes := pem.EncodeToMemory(c)

	byt := append(bBytes, cBytes...)

	in := bytes.NewBuffer(byt)

	err := ImportKeys(in, []Importer{s}, "", "")
	require.NoError(t, err)

	bFinal, bRest := pem.Decode(s.data["ankh"])
	require.Equal(t, b.Bytes, bFinal.Bytes)
	require.Equal(t, "", bFinal.Headers["path"]) // path header is stripped during import
	require.Equal(t, notary.DefaultImportRole, bFinal.Headers["role"]) // if no role is specified we assume it is a delegation key
	require.Len(t, bRest, 0)

	cFinal, cRest := pem.Decode(s.data["morpork"])
	require.Equal(t, c.Bytes, cFinal.Bytes)
	require.Equal(t, "", bFinal.Headers["path"])
	require.Len(t, cRest, 0)
}

func TestImportNoPath(t *testing.T) {
	s := NewTestImportStore()

	b := &pem.Block{
		Headers: make(map[string]string),
	}
	from, _ := os.OpenFile("test.key", os.O_RDONLY, notary.PrivKeyPerms)
	defer from.Close()
	b.Bytes, _ = ioutil.ReadAll(from)

	in := bytes.NewBuffer(b.Bytes)

	err := ImportKeys(in, []Importer{s}, "root", "")
	require.NoError(t, err)

	for key := range s.data {
		// no path but role included should work
		require.Equal(t, key, filepath.Join(notary.RootKeysSubdir,"7baafcc9e5100ab062d886f06468f6c76e70b54b90e5d38537dadc6299c976d9"))
	}

	err = ImportKeys(in, []Importer{s}, "", "")
	require.NoError(t, err)

	require.Len(t, s.data, 1) // no path and no role should not work
}

func TestNonRootPathInference(t *testing.T) {
	s := NewTestImportStore()

	b := &pem.Block{
		Headers: make(map[string]string),
	}
	from, _ := os.OpenFile("test.key", os.O_RDONLY, notary.PrivKeyPerms)
	defer from.Close()
	b.Bytes, _ = ioutil.ReadAll(from)

	in := bytes.NewBuffer(b.Bytes)

	err := ImportKeys(in, []Importer{s}, "somerole", "somegun")
	require.NoError(t, err)

	for key := range s.data {
		// no path but role included should work
		require.Equal(t, key, filepath.Join(notary.NonRootKeysSubdir, "somegun", "7baafcc9e5100ab062d886f06468f6c76e70b54b90e5d38537dadc6299c976d9"))
	}
}

func TestBlockHeaderPrecedence(t *testing.T) {
	s := NewTestImportStore()

	b := &pem.Block{
		Headers: make(map[string]string),
	}
	from, _ := os.OpenFile("testprecedence.key", os.O_RDONLY, notary.PrivKeyPerms)
	defer from.Close()
	b.Bytes, _ = ioutil.ReadAll(from)

	in := bytes.NewBuffer(b.Bytes)

	err := ImportKeys(in, []Importer{s}, "somerole", "somegun")
	require.NoError(t, err)

	for key := range s.data {
		// block header role should take precedence over command line flag
		require.Equal(t, key, filepath.Join(notary.RootKeysSubdir, "7baafcc9e5100ab062d886f06468f6c76e70b54b90e5d38537dadc6299c976d9"))
	}
}

func TestImportKeys2InOneFile(t *testing.T) {
	s := NewTestImportStore()

	b := &pem.Block{
		Headers: make(map[string]string),
	}
	b.Bytes = make([]byte, 1000)
	rand.Read(b.Bytes)
	b.Headers["path"] = "ankh"

	b2 := &pem.Block{
		Headers: make(map[string]string),
	}
	b2.Bytes = make([]byte, 1000)
	rand.Read(b2.Bytes)
	b2.Headers["path"] = "ankh"

	c := &pem.Block{
		Headers: make(map[string]string),
	}
	c.Bytes = make([]byte, 1000)
	rand.Read(c.Bytes)
	c.Headers["path"] = "morpork"

	bBytes := pem.EncodeToMemory(b)
	b2Bytes := pem.EncodeToMemory(b2)
	bBytes = append(bBytes, b2Bytes...)
	cBytes := pem.EncodeToMemory(c)

	byt := append(bBytes, cBytes...)

	in := bytes.NewBuffer(byt)

	err := ImportKeys(in, []Importer{s}, "", "")
	require.NoError(t, err)

	bFinal, bRest := pem.Decode(s.data["ankh"])
	require.Equal(t, b.Bytes, bFinal.Bytes)
	require.Equal(t, "", bFinal.Headers["path"]) // path header is stripped during import

	b2Final, b2Rest := pem.Decode(bRest)
	require.Equal(t, b2.Bytes, b2Final.Bytes)
	require.Equal(t, "", b2Final.Headers["path"]) // path header is stripped during import
	require.Len(t, b2Rest, 0)

	cFinal, cRest := pem.Decode(s.data["morpork"])
	require.Equal(t, c.Bytes, cFinal.Bytes)
	require.Equal(t, "", bFinal.Headers["path"])
	require.Len(t, cRest, 0)
}
