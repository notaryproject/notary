package keydbstore

import (
	"fmt"

	"github.com/docker/notary"
	"github.com/docker/notary/trustmanager"
	"github.com/docker/notary/tuf/data"
	jose "github.com/dvsekhvalnov/jose2go"
	"github.com/jinzhu/gorm"
)

// Constants
const (
	EncryptionAlg = jose.A256GCM
	KeywrapAlg    = jose.PBES2_HS256_A128KW
)

// SQLKeyDBStore persists and manages private keys on a SQL database
type SQLKeyDBStore struct {
	db               gorm.DB
	dbType           string
	defaultPassAlias string
	retriever        notary.PassRetriever
}

// GormPrivateKey represents a PrivateKey in the database
type GormPrivateKey struct {
	gorm.Model
	KeyID           string `sql:"type:varchar(255);not null;unique;index:key_id_idx"`
	EncryptionAlg   string `sql:"type:varchar(255);not null"`
	KeywrapAlg      string `sql:"type:varchar(255);not null"`
	Algorithm       string `sql:"type:varchar(50);not null"`
	PassphraseAlias string `sql:"type:varchar(50);not null"`
	Gun             string `sql:"type:varchar(255);not null"`
	Role            string `sql:"type:varchar(255);not null"`
	Public          string `sql:"type:blob;not null"`
	Private         string `sql:"type:blob;not null"`
}

// TableName sets a specific table name for our GormPrivateKey
func (g GormPrivateKey) TableName() string {
	return "private_keys"
}

// NewSQLKeyDBStore returns a new SQLKeyDBStore backed by a SQL database
func NewSQLKeyDBStore(passphraseRetriever notary.PassRetriever, defaultPassAlias string,
	dbDialect string, dbArgs ...interface{}) (*SQLKeyDBStore, error) {

	db, err := gorm.Open(dbDialect, dbArgs...)
	if err != nil {
		return nil, err
	}

	return &SQLKeyDBStore{
		db:               db,
		dbType:           dbDialect,
		defaultPassAlias: defaultPassAlias,
		retriever:        passphraseRetriever,
	}, nil
}

// Name returns a user friendly name for the storage location
func (s *SQLKeyDBStore) Name() string {
	return s.dbType
}

// AddKey stores the contents of a private key. Both role and gun are ignored,
// we always use Key IDs as name, and don't support aliases
func (s *SQLKeyDBStore) AddKey(keyInfo trustmanager.KeyInfo, privKey data.PrivateKey) error {
	passphrase, _, err := s.retriever(privKey.ID(), s.defaultPassAlias, false, 1)
	if err != nil {
		return err
	}

	encryptedKey, err := jose.Encrypt(string(privKey.Private()), KeywrapAlg, EncryptionAlg, passphrase)
	if err != nil {
		return err
	}

	gormPrivKey := GormPrivateKey{
		KeyID:           privKey.ID(),
		EncryptionAlg:   EncryptionAlg,
		KeywrapAlg:      KeywrapAlg,
		PassphraseAlias: s.defaultPassAlias,
		Algorithm:       privKey.Algorithm(),
		Gun:             keyInfo.Gun,
		Role:            keyInfo.Role,
		Public:          string(privKey.Public()),
		Private:         encryptedKey}

	// Add encrypted private key to the database
	s.db.Create(&gormPrivKey)
	// Value will be false if Create succeeds
	failure := s.db.NewRecord(gormPrivKey)
	if failure {
		return fmt.Errorf("failed to add private key to database: %s", privKey.ID())
	}

	return nil
}

// GetKey returns the PrivateKey given a KeyID
func (s *SQLKeyDBStore) GetKey(keyID string) (data.PrivateKey, string, error) {
	// Retrieve the GORM private key from the database
	dbPrivateKey := GormPrivateKey{}
	if s.db.Where(&GormPrivateKey{KeyID: keyID}).First(&dbPrivateKey).RecordNotFound() {
		return nil, "", trustmanager.ErrKeyNotFound{KeyID: keyID}
	}

	// Get the passphrase to use for this key
	passphrase, _, err := s.retriever(dbPrivateKey.KeyID, dbPrivateKey.PassphraseAlias, false, 1)
	if err != nil {
		return nil, "", err
	}

	// Decrypt private bytes from the gorm key
	decryptedPrivKey, _, err := jose.Decode(dbPrivateKey.Private, passphrase)
	if err != nil {
		return nil, "", err
	}

	pubKey := data.NewPublicKey(dbPrivateKey.Algorithm, []byte(dbPrivateKey.Public))
	// Create a new PrivateKey with unencrypted bytes
	privKey, err := data.NewPrivateKey(pubKey, []byte(decryptedPrivKey))
	if err != nil {
		return nil, "", err
	}

	return privKey, dbPrivateKey.Role, nil
}

// GetKeyInfo returns the PrivateKey's role and gun in a KeyInfo given a KeyID
func (s *SQLKeyDBStore) GetKeyInfo(keyID string) (trustmanager.KeyInfo, error) {
	return trustmanager.KeyInfo{}, fmt.Errorf("GetKeyInfo currently not supported for SQLKeyDBStore, as it does not track roles or GUNs")
}

// ListKeys always returns nil. This method is here to satisfy the KeyStore interface
func (s *SQLKeyDBStore) ListKeys() map[string]trustmanager.KeyInfo {
	return nil
}

// RemoveKey removes the key from the keyfilestore
func (s *SQLKeyDBStore) RemoveKey(keyID string) error {
	// Delete the key from the database
	s.db.Where(&GormPrivateKey{KeyID: keyID}).Delete(&GormPrivateKey{})

	return nil
}

// RotateKeyPassphrase rotates the key-encryption-key
func (s *SQLKeyDBStore) RotateKeyPassphrase(keyID, newPassphraseAlias string) error {
	// Retrieve the GORM private key from the database
	dbPrivateKey := GormPrivateKey{}
	if s.db.Where(&GormPrivateKey{KeyID: keyID}).First(&dbPrivateKey).RecordNotFound() {
		return trustmanager.ErrKeyNotFound{KeyID: keyID}
	}

	// Get the current passphrase to use for this key
	passphrase, _, err := s.retriever(dbPrivateKey.KeyID, dbPrivateKey.PassphraseAlias, false, 1)
	if err != nil {
		return err
	}

	// Decrypt private bytes from the gorm key
	decryptedPrivKey, _, err := jose.Decode(dbPrivateKey.Private, passphrase)
	if err != nil {
		return err
	}

	// Get the new passphrase to use for this key
	newPassphrase, _, err := s.retriever(dbPrivateKey.KeyID, newPassphraseAlias, false, 1)
	if err != nil {
		return err
	}

	// Re-encrypt the private bytes with the new passphrase
	newEncryptedKey, err := jose.Encrypt(decryptedPrivKey, KeywrapAlg, EncryptionAlg, newPassphrase)
	if err != nil {
		return err
	}

	// Update the database object
	dbPrivateKey.Private = newEncryptedKey
	dbPrivateKey.PassphraseAlias = newPassphraseAlias
	s.db.Save(dbPrivateKey)

	return nil
}

// HealthCheck verifies that DB exists and is query-able
func (s *SQLKeyDBStore) HealthCheck() error {
	dbPrivateKey := GormPrivateKey{}
	tableOk := s.db.HasTable(&dbPrivateKey)
	switch {
	case s.db.Error != nil:
		return s.db.Error
	case !tableOk:
		return fmt.Errorf(
			"Cannot access table: %s", dbPrivateKey.TableName())
	}
	return nil
}
