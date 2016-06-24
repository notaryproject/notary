package trustmanager

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/docker/docker-credential-helpers/client"
	"github.com/docker/docker-credential-helpers/credentials"
	"github.com/docker/notary/tuf/data"
	"github.com/docker/notary/tuf/utils"
)

// KeyChainStore is an implementation of Storage that keeps
// the contents in the keychain access.
type KeyNativeStore struct {
	newProgFunc client.ProgramFunc
}

// NewKeyChainStore creates a KeyChainStore
func NewKeyNativeStore(machineCredsStore string) (*KeyNativeStore, error) {
	var name string
	var e error
	e = nil
	switch machineCredsStore {
	case "osxkeychain":
		name = "docker-credential-osxkeychain"
	case "wincred":
		name = "docker-credential-wincred"
	case "secretservice":
		name = "docker-credential-secretservice"
	default:
		e = errors.New("Error, the machine cred store you specified does not exist")
	}
	x := client.NewShellProgramFunc(name)
	return &KeyNativeStore{
		newProgFunc: x,
	}, e
}

//Add writes data new KeyChain in the keychain access store
func (k *KeyNativeStore) AddKey(keyInfo KeyInfo, privKey data.PrivateKey) error {
	pemPrivKey, err := utils.KeyToPEM(privKey, keyInfo.Role, keyInfo.Gun)
	if err != nil {
		return err
	}
	secretByte := base64.StdEncoding.EncodeToString(pemPrivKey)
	keyCredentials := credentials.Credentials{
		ServerURL: privKey.ID(),
		//"|" is a blacklist character to seperate the GUN and the Role and notary_key is just used to identify to the user that this is from notary
		Username: keyInfo.Gun + "<notary_key>" + keyInfo.Role,
		Secret:   secretByte,
	}
	err = client.Store(k.newProgFunc, &(keyCredentials))
	return err
}

// Get returns the credentials from the keychain access store given a server name
func (k *KeyNativeStore) GetKey(keyID string) (data.PrivateKey, string, error) {
	serverName := keyID
	gotCredentials, err := client.Get(k.newProgFunc, serverName)
	if err != nil {
		return nil, "", err
	}
	gotSecret := gotCredentials.Secret
	gotSecretByte, err := base64.StdEncoding.DecodeString(gotSecret)
	privKey, err := utils.ParsePEMPrivateKey(gotSecretByte, "")
	role := strings.SplitAfter(gotCredentials.Username, "<notary_key>")[1]
	return privKey, role, err
}

// GetKeyInfo returns the corresponding gun and role key info for a keyID
func (k *KeyNativeStore) GetKeyInfo(keyID string) (KeyInfo, error) {
	serverName := keyID
	gotCredentials, err := client.Get(k.newProgFunc, serverName)
	if err != nil {
		return KeyInfo{}, err
	}
	keyinfo := strings.SplitAfter(gotCredentials.Username, "<notary_key>")
	gun := keyinfo[0][:(len(keyinfo[0]) - 12)]
	return KeyInfo{
		Gun:  gun,
		Role: keyinfo[1],
	}, err
}

// ListFiles lists all the Keys inside of a native store
// Just a placeholder for now- returns an empty slice
func (k *KeyNativeStore) ListKeys() map[string]KeyInfo {
	return nil
}

//Remove removes a KeyChain (identified by server name- a string) from the keychain access store
func (k *KeyNativeStore) RemoveKey(keyID string) error {
	err := client.Erase(k.newProgFunc, keyID)
	return err
}

func (k *KeyNativeStore) ExportKey(keyID string) ([]byte, error) {
	//What passphrase should we encrypt it before exporting the key?
	//Just a place-holder for now
	return []byte{}, nil
}

// Name returns a user friendly name for the location this store
// keeps its data, here it is the name of the native store on this operating system
func (k *KeyNativeStore) Name() string {
	return fmt.Sprintf("Native keychain store: %s", defaultCredentialsStore)
}
