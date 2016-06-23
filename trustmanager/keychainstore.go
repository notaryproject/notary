package trustmanager

import (
	"github.com/docker/docker-credential-helpers/credentials"
	"github.com/docker/docker-credential-helpers/client"
	"encoding/base64"
	"errors"
)

// KeyChainStore is an implementation of Storage that keeps
// the contents in the keychain access.
type KeyChainStore struct {
	newProgFunc client.ProgramFunc
}

// NewKeyChainStore creates a KeyChainStore
func NewKeyChainStore(machineCredsStore string) (*KeyChainStore, error) {
	var name string
	var e error
	e=nil
	switch machineCredsStore {
	case "osxkeychain":
		name = "docker-credential-osxkeychain"
	case "wincred":
		name = "docker-credential-wincred"
	case "secretservice":
		name = "docker-credential-secretservice"
	default:
		e=errors.New("Error, the machine cred store you specified does not exist")
	}
	x:=client.NewShellProgramFunc(name)
	return &KeyChainStore{
		newProgFunc:x,
	}, e
}

//Add writes data new KeyChain in the keychain access store
func (k *KeyChainStore) Add(fileName string, data []byte) error {
	serveName:="https://notary.docker.io/"+fileName
	secretByte:=base64.StdEncoding.EncodeToString(data)
	keyCredentials:=credentials.Credentials{
		ServerURL:serveName,
		Username:"Role goes here",
		Secret:secretByte,
	}
	err:=client.Store(k.newProgFunc,&(keyCredentials))
	return err
}

//Remove removes a KeyChain (identified by server name- a string) from the keychain access store
func (k *KeyChainStore) Remove(fileName string) error {
	serverName:="https://notary.docker.io/"+fileName
	err:=client.Erase(k.newProgFunc,serverName)
	return err
}

//// Get returns the credentials from the keychain access store given a server name
func (k *KeyChainStore) Get(serverName string) ([]byte, error) {
	gotCredentials,err:=client.Get(k.newProgFunc,serverName)
	if err!=nil {
		return nil,errors.New("There is no keychain associated with the server string")
	}
	gotSecret:=gotCredentials.Secret
	gotSecretByte,err:=base64.StdEncoding.DecodeString(gotSecret)
	return gotSecretByte, err
}

// ListFiles lists all the files inside of a store
// Just a placeholder for now, need to find a way to do this
func (f *KeyChainStore) ListFiles() []string {
	list:=[]string{"a","b"}
	return list
}

