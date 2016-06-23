package trustmanager

import (
	"testing"
	"crypto/rand"
	"github.com/stretchr/testify/require"
)

//Has been hard-coded with osx, need to make this more generic
//Testing that add is working while simultaneously testing that get is working for a keychain we know exists
func TestAddGetKeyInKeyChain(t *testing.T) {
	machineCredsStore:="osxkeychain"
	myKeyChainStore, err:=NewKeyChainStore(machineCredsStore)
	privKey, err := GenerateECDSAKey(rand.Reader)
	genStore :=[]Storage{myKeyChainStore}[0]
	err=genStore.Add(privKey.ID(), privKey.Private())
	require.NoError(t, err)
	defer genStore.Remove(privKey.ID())
	serveName := "https://notary.docker.io/" + privKey.ID()
	gotCreds, err := genStore.Get(serveName)
	require.NoError(t, err)
	require.Equal(t, privKey.Private(), gotCreds, "unexpected content in the file")
}

//Testing that remove is working for a key that we know exists, also tests that get doesn't panic if a key doesn't exist
func TestRemoveWorks(t *testing.T) {
	machineCredsStore:="osxkeychain"
	myKeyChainStore, err:=NewKeyChainStore(machineCredsStore)
	privKey, err := GenerateECDSAKey(rand.Reader)
	genStore :=[]Storage{myKeyChainStore}[0]
	err=genStore.Add(privKey.ID(), privKey.Private())
	require.NoError(t, err)
	serveName := "https://notary.docker.io/" + privKey.ID()
	gotCreds, err := genStore.Get(serveName)
	require.NoError(t, err)
	require.Equal(t, privKey.Private(), gotCreds, "issue in the add function")
	err=genStore.Remove(privKey.ID())
	require.NoError(t, err)
	gotCreds, err = genStore.Get(serveName)
	require.Error(t, err)
	require.NotEqual(t, privKey.Private(), gotCreds, "issue in the remove function")
}

//Testing that remove doesn't crash on a key that doesn't exist
func TestRemoveNoPanic(t *testing.T) {
	machineCredsStore:="osxkeychain"
	myKeyChainStore, err:=NewKeyChainStore(machineCredsStore)
	genStore :=[]Storage{myKeyChainStore}[0]
	err=genStore.Remove("notaryTestRemoveDoesntCrash")
	require.Error(t,err)
}

//Test for list files
func TestList(t *testing.T)  {
	machineCredsStore:="osxkeychain"
	myKeyChainStore, err:=NewKeyChainStore(machineCredsStore)
	require.NoError(t, err)
	lst:=myKeyChainStore.ListFiles()
	require.Equal(t, lst, []string{"a","b"}, "There's something very wrong since this is just a dummy function")
}