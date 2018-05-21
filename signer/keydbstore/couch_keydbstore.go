package keydbstore

import (
	"context"
	"fmt"
	"time"

	jose "github.com/dvsekhvalnov/jose2go"
	"github.com/flimzy/kivik"
	_ "github.com/go-kivik/couchdb" //
	"github.com/theupdateframework/notary"
	"github.com/theupdateframework/notary/storage/couchdb"
	"github.com/theupdateframework/notary/trustmanager"
	"github.com/theupdateframework/notary/tuf/data"
)

// CouchDBKeyStore persists and manages private keys on a CouchDB database
type CouchDBKeyStore struct {
	client           *kivik.Client
	dbName           string
	defaultPassAlias string
	retriever        notary.PassRetriever
	user             string
	password         string
	nowFunc          func() time.Time
}

// CDBPrivateKey represents a PrivateKey in the couch database
type CDBPrivateKey struct {
	couchdb.ID
	couchdb.Timing
	KeyID           string        `json:"key_id"`
	EncryptionAlg   string        `json:"encryption_alg"`
	KeywrapAlg      string        `json:"keywrap_alg"`
	Algorithm       string        `json:"algorithm"`
	PassphraseAlias string        `json:"passphrase_alias"`
	Gun             data.GUN      `json:"gun"`
	Role            data.RoleName `json:"role"`

	// Currently our encryption method for the private key bytes
	// produces a base64-encoded string, but for future compatibility in case
	// we change how we encrypt, use a byteslace for the encrypted private key
	// too
	Public  []byte `json:"public"`
	Private []byte `json:"private"`

	// whether this key is active or not
	LastUsed time.Time `json:"last_used"`
}

// PrivateKeysCouchTable is the table definition for notary signer's key information
var PrivateKeysCouchTable = couchdb.Table{
	Name: CDBPrivateKey{}.TableName(),
	Indexes: map[string]interface{}{
		"1": []string{"key_id"},
		"2": []string{"gun", "role", "algorithm", "last_used"},
	},
}

// TableName sets a specific table name for our CDBPrivateKey
func (g CDBPrivateKey) TableName() string {
	return "private_keys"
}

// NewCouchDBKeyStore returns a new CouchDBKeyStore backed by a CouchDB database
func NewCouchDBKeyStore(dbName, username, password string, passphraseRetriever notary.PassRetriever, defaultPassAlias string, couchClient *kivik.Client) *CouchDBKeyStore {
	return &CouchDBKeyStore{
		client:           couchClient,
		defaultPassAlias: defaultPassAlias,
		dbName:           dbName,
		retriever:        passphraseRetriever,
		user:             username,
		password:         password,
		nowFunc:          time.Now,
	}
}

// Name returns a user friendly name for the storage location
func (cdb *CouchDBKeyStore) Name() string {
	return "CouchDB"
}

// AddKey stores the contents of a private key. Both role and gun are ignored,
// we always use Key IDs as name, and don't support aliases
func (cdb *CouchDBKeyStore) AddKey(role data.RoleName, gun data.GUN, privKey data.PrivateKey) error {
	passphrase, _, err := cdb.retriever(privKey.ID(), cdb.defaultPassAlias, false, 1)
	if err != nil {
		return err
	}

	encryptedKey, err := jose.Encrypt(string(privKey.Private()), KeywrapAlg, EncryptionAlg, passphrase)
	if err != nil {
		return err
	}

	now := cdb.nowFunc()
	couchPrivKey := CDBPrivateKey{
		ID: couchdb.ID{
			ID: privKey.ID(),
		},
		Timing: couchdb.Timing{
			CreatedAt: now,
			UpdatedAt: now,
		},
		KeyID:           privKey.ID(),
		EncryptionAlg:   EncryptionAlg,
		KeywrapAlg:      KeywrapAlg,
		PassphraseAlias: cdb.defaultPassAlias,
		Algorithm:       privKey.Algorithm(),
		Gun:             gun,
		Role:            role,
		Public:          privKey.Public(),
		Private:         []byte(encryptedKey),
	}

	// Add encrypted private key to the database
	db, err := couchdb.DB(cdb.client, cdb.dbName, couchPrivKey.TableName())
	if err != nil {
		return err
	}

	_, _, err = db.CreateDoc(context.TODO(), couchPrivKey)
	if err != nil {
		return fmt.Errorf("AddKey: failed to add private key %s to database: %s", privKey.ID(), err.Error())
	}

	return nil
}

// getKey returns the CDBPrivateKey given a KeyID, as well as the decrypted private bytes
func (cdb *CouchDBKeyStore) getKey(keyID string) (*CDBPrivateKey, string, error) {
	// Retrieve the CouchDB private key from the database
	dbPrivateKey := CDBPrivateKey{}

	db, err := couchdb.DB(cdb.client, cdb.dbName, dbPrivateKey.TableName())
	if err != nil {
		return nil, "", err
	}

	rows, err := db.Find(context.TODO(), map[string]interface{}{
		"selector": map[string]interface{}{
			"key_id": map[string]string{
				"$eq": keyID,
			},
		},
	})
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, "", trustmanager.ErrKeyNotFound{}
	}

	err = rows.ScanDoc(&dbPrivateKey)
	if err != nil {
		return nil, "", trustmanager.ErrKeyNotFound{}
	}

	// Get the passphrase to use for this key
	passphrase, _, err := cdb.retriever(dbPrivateKey.KeyID, dbPrivateKey.PassphraseAlias, false, 1)
	if err != nil {
		return nil, "", err
	}

	// Decrypt private bytes from the key
	decryptedPrivKey, _, err := jose.Decode(string(dbPrivateKey.Private), passphrase)
	if err != nil {
		return nil, "", err
	}

	return &dbPrivateKey, decryptedPrivKey, nil
}

// GetPrivateKey returns the PrivateKey given a KeyID
func (cdb *CouchDBKeyStore) GetPrivateKey(keyID string) (data.PrivateKey, data.RoleName, error) {
	dbPrivateKey, decryptedPrivKey, err := cdb.getKey(keyID)
	if err != nil {
		return nil, "", err
	}

	pubKey := data.NewPublicKey(dbPrivateKey.Algorithm, dbPrivateKey.Public)

	// Create a new PrivateKey with unencrypted bytes
	privKey, err := data.NewPrivateKey(pubKey, []byte(decryptedPrivKey))
	if err != nil {
		return nil, "", err
	}

	return activatingPrivateKey{PrivateKey: privKey, activationFunc: cdb.markActive}, dbPrivateKey.Role, nil
}

// GetKey returns the PublicKey given a KeyID, and does not activate the key
func (cdb *CouchDBKeyStore) GetKey(keyID string) data.PublicKey {
	dbPrivateKey, _, err := cdb.getKey(keyID)
	if err != nil {
		return nil
	}

	return data.NewPublicKey(dbPrivateKey.Algorithm, dbPrivateKey.Public)
}

// ListKeys always returns nil. This method is here to satisfy the CryptoService interface
func (cdb CouchDBKeyStore) ListKeys(role data.RoleName) []string {
	return nil
}

// ListAllKeys always returns nil. This method is here to satisfy the CryptoService interface
func (cdb CouchDBKeyStore) ListAllKeys() map[string]data.RoleName {
	return nil
}

// RemoveKey removes the key from the table
func (cdb CouchDBKeyStore) RemoveKey(keyID string) error {
	// Delete the key from the database
	dbPrivateKey := CDBPrivateKey{KeyID: keyID}

	db, err := couchdb.DB(cdb.client, cdb.dbName, dbPrivateKey.TableName())
	if err != nil {
		return err
	}

	rows, err := db.Find(context.TODO(), map[string]interface{}{
		"selector": map[string]interface{}{
			"key_id": map[string]string{
				"$eq": keyID,
			},
		},
	})
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.ScanDoc(&dbPrivateKey); err != nil {
			return fmt.Errorf("Could not ScanDoc result: %s", err)
		}
		if _, err = db.Delete(context.TODO(), keyID, dbPrivateKey.ID.Rev); err != nil {
			return fmt.Errorf("Unable to delete private key %s from database: %s", keyID, err.Error())
		}
	}

	return nil
}

// RotateKeyPassphrase rotates the key-encryption-key
func (cdb CouchDBKeyStore) RotateKeyPassphrase(keyID, newPassphraseAlias string) error {
	dbPrivateKey, decryptedPrivKey, err := cdb.getKey(keyID)
	if err != nil {
		return err
	}

	// Get the new passphrase to use for this key
	newPassphrase, _, err := cdb.retriever(dbPrivateKey.KeyID, newPassphraseAlias, false, 1)
	if err != nil {
		return err
	}

	// Re-encrypt the private bytes with the new passphrase
	newEncryptedKey, err := jose.Encrypt(decryptedPrivKey, KeywrapAlg, EncryptionAlg, newPassphrase)
	if err != nil {
		return err
	}

	// Update the database object
	dbPrivateKey.Private = []byte(newEncryptedKey)
	dbPrivateKey.PassphraseAlias = newPassphraseAlias

	db, err := couchdb.DB(cdb.client, cdb.dbName, dbPrivateKey.TableName())
	if err != nil {
		return err
	}

	rows, err := db.Find(context.TODO(), map[string]interface{}{
		"selector": map[string]interface{}{
			"key_id": map[string]string{
				"$eq": keyID,
			},
		},
	})
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var doc CDBPrivateKey
		rows.ScanDoc(&doc)
		doc.Private = []byte(newEncryptedKey)
		doc.PassphraseAlias = newPassphraseAlias
		doc.ID.ID = keyID
		if _, err := db.Put(context.TODO(), keyID, doc); err != nil {
			return err
		}
	}

	return nil
}

// markActive marks a particular key as active
func (cdb CouchDBKeyStore) markActive(keyID string) error {
	db, err := couchdb.DB(cdb.client, cdb.dbName, PrivateKeysCouchTable.Name)
	if err != nil {
		return err
	}

	rows, err := db.Find(context.TODO(), map[string]interface{}{
		"selector": map[string]interface{}{
			"key_id": map[string]string{
				"$eq": keyID,
			},
		},
	})
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var doc CDBPrivateKey
		rows.ScanDoc(&doc)
		doc.LastUsed = cdb.nowFunc()
		doc.ID.ID = keyID
		if _, err := db.Put(context.TODO(), keyID, doc); err != nil {
			return err
		}
	}

	return nil
}

// Create will attempt to first re-use an inactive key for the same role, gun, and algorithm.
// If one isn't found, it will create a private key and add it to the DB as an inactive key
func (cdb CouchDBKeyStore) Create(role data.RoleName, gun data.GUN, algorithm string) (data.PublicKey, error) {
	dbPrivateKey := RDBPrivateKey{}
	db, err := couchdb.DB(cdb.client, cdb.dbName, dbPrivateKey.TableName())
	if err != nil {
		return nil, err
	}

	rows, err := db.Find(context.TODO(), map[string]interface{}{
		"selector": map[string]interface{}{
			"gun":       gun.String(),
			"role":      role.String(),
			"algorithm": algorithm,
			"last_used": time.Time{},
		},
	})
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if rows.Next() {
		var dbPrivateKey CDBPrivateKey
		rows.ScanDoc(&dbPrivateKey)
		return data.NewPublicKey(dbPrivateKey.Algorithm, dbPrivateKey.Public), nil
	}

	privKey, err := generatePrivateKey(algorithm)
	if err != nil {
		return nil, err
	}
	if err = cdb.AddKey(role, gun, privKey); err != nil {
		return nil, fmt.Errorf("failed to store key: %v", err)
	}

	return privKey, nil
}

// Bootstrap sets up the database and tables, also creating the notary signer user with appropriate db permission
func (cdb CouchDBKeyStore) Bootstrap() error {
	if err := couchdb.SetupDB(cdb.client, cdb.dbName, []couchdb.Table{
		PrivateKeysCouchTable,
	}); err != nil {
		return err
	}
	return couchdb.CreateAndGrantDBUser(cdb.client, cdb.dbName, cdb.user, cdb.password)
}

// CheckHealth verifies that DB exists and is query-able
func (cdb CouchDBKeyStore) CheckHealth() error {
	db, err := couchdb.DB(cdb.client, cdb.dbName, PrivateKeysCouchTable.Name)
	if err != nil {
		return err
	}
	if _, err = db.Stats(context.TODO()); err != nil {
		return err
	}
	return nil
}
