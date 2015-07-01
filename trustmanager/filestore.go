package trustmanager

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"golang.org/x/crypto/scrypt"
)

const visible os.FileMode = 0755
const private os.FileMode = 0700
const encryptedExt string = "enc"

const (
	SaltSize = 32
	KeySize  = 32
	ScryptN  = 32768
	ScryptR  = 8
	ScryptP  = 1
)

// FileStore is the interface for all FileStores
type FileStore interface {
	Add(fileName string, data []byte) error
	Remove(fileName string) error
	RemoveDir(directoryName string) error
	Get(fileName string) ([]byte, error)
	GetPath(fileName string) string
	ListAll() []string
	ListDir(directoryName string) []string
}

// fileStore implements FileStore
type fileStore struct {
	baseDir string
	fileExt string
	perms   os.FileMode
}

// NewFileStore creates a directory with 755 permissions
func NewFileStore(baseDir string, fileExt string) (FileStore, error) {
	if err := CreateDirectory(baseDir); err != nil {
		return nil, err
	}

	return &fileStore{
		baseDir: baseDir,
		fileExt: fileExt,
		perms:   visible,
	}, nil
}

// NewPrivateFileStore creates a directory with 700 permissions
func NewPrivateFileStore(baseDir string, fileExt string) (FileStore, error) {
	if err := CreatePrivateDirectory(baseDir); err != nil {
		return nil, err
	}

	return &fileStore{
		baseDir: baseDir,
		fileExt: fileExt,
		perms:   private,
	}, nil
}

// Add writes data to a file with a given name
func (f *fileStore) Add(name string, data []byte) error {
	filePath := f.genFilePath(name)
	createDirectory(filepath.Dir(filePath), f.perms)
	return ioutil.WriteFile(filePath, data, f.perms)
}

// AddEncrypted writes encrypted data to a file with a given name, given a key
func (f *fileStore) AddEncrypted(name string, data []byte, passphrase string) error {
	filePath := f.genEncryptedFilePath(name)
	createDirectory(filepath.Dir(filePath), f.perms)

	// Derive a key from the passphrase
	// First generate random salt for scrypt
	salt := make([]byte, SaltSize)
	_, err := rand.Read(salt)
	if err != nil {
		return err
	}

	// With the salt, generate key derived from passphrase
	derivedKey, err := scrypt.Key([]byte(passphrase), salt, ScryptN, ScryptR, ScryptP, KeySize)
	if err != nil {
		return err
	}

	// Instantiate a new AES Cipher with the derived key
	c, err := aes.NewCipher(derivedKey)
	if err != nil {
		return err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return err
	}

	// Generate random nonce for GCM
	nonce := make([]byte, gcm.NonceSize())
	_, err = rand.Read(nonce)
	if err != nil {
		return err
	}

	// Encypts the data using the nonce
	cipherText := gcm.Seal(nonce, nonce, data, nil)

	// Append the salt to the outData so we can use it on decrypt
	outData := append(salt, cipherText...)
	return ioutil.WriteFile(filePath, outData, f.perms)
}

// Remove removes a file identified by a name
func (f *fileStore) Remove(name string) error {
	// Attempt to remove
	filePath := f.genFilePath(name)
	return os.Remove(filePath)
}

// RemoveDir removes the directory identified by name
func (f *fileStore) RemoveDir(name string) error {
	dirPath := filepath.Join(f.baseDir, name)

	// Check to see if directory exists
	fi, err := os.Stat(dirPath)
	if err != nil {
		return err
	}

	// Check to see if it is a directory
	if !fi.IsDir() {
		return fmt.Errorf("directory not found: %s", name)
	}

	return os.RemoveAll(dirPath)
}

// Get returns the data given a file name
func (f *fileStore) Get(name string) ([]byte, error) {
	filePath := f.genFilePath(name)
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// GetEncrypted decrypts and returns data given a file name
func (f *fileStore) GetEncrypted(name string, passphrase string) ([]byte, error) {
	filePath := f.genEncryptedFilePath(name)
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// If the data doesn't have at least this size, it doesn't have any data.
	if len(data) <= SaltSize+24+4 {
		return nil, fmt.Errorf("Error while decrypting, not enough data in: %s", filePath)
	}

	// Get the salt from the first SaltSize bytes in data
	salt := data[:SaltSize]
	data = data[SaltSize:]

	// With the salt, we can generate key derived from passphrase
	derivedKey, err := scrypt.Key([]byte(passphrase), salt, ScryptN, ScryptR, ScryptP, KeySize)
	if err != nil {
		return nil, err
	}

	c, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	// Get the nonce from the next NonceSize bytes in data
	nonce := data[:gcm.NonceSize()]
	data = data[gcm.NonceSize():]

	// Decrypt the data and return plaintext
	outData, err := gcm.Open(nil, nonce, data, nil)
	if err != nil {
		return nil, err
	}

	return outData, nil
}

// GetPath returns the full final path of a file with a given name
func (f *fileStore) GetPath(name string) string {
	return f.genFilePath(name)
}

// List lists all the files inside of a store
func (f *fileStore) ListAll() []string {
	return f.list(f.baseDir)
}

// List lists all the files inside of a directory identified by a name
func (f *fileStore) ListDir(name string) []string {
	fullPath := filepath.Join(f.baseDir, name)
	return f.list(fullPath)
}

// list lists all the files in a directory given a full path
func (f *fileStore) list(path string) []string {
	files := make([]string, 0, 0)
	filepath.Walk(path, func(fp string, fi os.FileInfo, err error) error {
		// If there are errors, ignore this particular file
		if err != nil {
			return nil
		}
		// Ignore if it is a directory
		if fi.IsDir() {
			return nil
		}
		// Only allow matches that end with our extensions (e.g. *.crt and *.crt.enc)
		matched, _ := filepath.Match("*"+f.fileExt, fi.Name())
		matchedEnc, _ := filepath.Match("*"+f.fileExt+encryptedExt, fi.Name())

		if matched || matchedEnc {
			files = append(files, fp)
		}

		return nil
	})
	return files
}

// genFilePath returns the full path with extension given a file name
func (f *fileStore) genFilePath(name string) string {
	fileName := fmt.Sprintf("%s.%s", name, f.fileExt)
	return filepath.Join(f.baseDir, fileName)
}

// genEncryptedFilePath returns the full path with extension given a file name and
// the added encrypted extension
func (f *fileStore) genEncryptedFilePath(name string) string {
	fileName := fmt.Sprintf("%s.%s.%s", name, f.fileExt, encryptedExt)
	return filepath.Join(f.baseDir, fileName)
}

// CreateDirectory uses createDirectory to create a chmod 755 Directory
func CreateDirectory(dir string) error {
	return createDirectory(dir, visible)
}

// CreatePrivateDirectory uses createDirectory to create a chmod 700 Directory
func CreatePrivateDirectory(dir string) error {
	return createDirectory(dir, private)
}

// createDirectory receives a string of the path to a directory.
// It does not support passing files, so the caller has to remove
// the filename by doing filepath.Dir(full_path_to_file)
func createDirectory(dir string, perms os.FileMode) error {
	// This prevents someone passing /path/to/dir and 'dir' not being created
	// If two '//' exist, MkdirAll deals it with correctly
	dir = dir + "/"
	return os.MkdirAll(dir, perms)
}
