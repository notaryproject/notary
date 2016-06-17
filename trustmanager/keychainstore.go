package trustmanager

import (
	"github.com/docker/docker-credential-helpers/credentials"
	"github.com/docker/docker-credential-helpers/client"
	"encoding/base64"
)

// KeyChainStore is an implementation of Storage that keeps
// the contents in the keychain access.
type KeyChainStore struct {
	newProgFunc client.ProgramFunc
}

// NewKeyChainStore creates a KeyChainStore
func NewKeyChainStore(machineCredsStore string) *KeyChainStore {
	name := "docker-credential-" + machineCredsStore
	x:=client.NewShellProgramFunc(name)
return &KeyChainStore{
	newProgFunc:x,
}
}
//
//func ARGTest() {
//	machineCredsStore:="osxkeychain"
//	myKeyChainStore:=NewKeyChainStore(machineCredsStore)
//	fmt.Println(myKeyChainStore)
//	privKey, err := GenerateECDSAKey(rand.Reader)
//	fmt.Println(privKey.Private())
//	fmt.Println(err)
//	for _, genStore := range []Storage{myKeyChainStore} {
//		fmt.Println("adding")
//		genStore.Add(privKey.ID(), privKey.Private())
//		fmt.Println("getting")
//		servename := "https://notary.docker.io/" + privKey.ID()
//		gotCreds, err := genStore.Get(servename)
//		fmt.Println(gotCreds)
//		if err != nil {
//			fmt.Println(err)
//		}
//		fmt.Println("removing")
//		genStore.Remove(privKey.ID())
//	}
//}

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
	gotSecret:=gotCredentials.Secret
	gotSecretByte,err:=base64.StdEncoding.DecodeString(gotSecret)
	return gotSecretByte, err
}

//// ListFiles lists all the files inside of a store
//func (f *MemoryFileStore) ListFiles() []string {
//var list []string
//
//for name := range f.files {
//list = append(list, name)
//}
//
//return list
//}

