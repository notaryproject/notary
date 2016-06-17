package main

import (
	"github.com/docker/notary/trustmanager"
	"fmt"
	"crypto/rand"
)

func main() {
	//trustmanager.Hello()
	fmt.Println("-  -  - - - --== ><>")
	//trustmanager.AddTest()
	//trustmanager.ARGTest()
	machineCredsStore:="osxkeychain"
	myKeyChainStore:=trustmanager.NewKeyChainStore(machineCredsStore)
	fmt.Println(myKeyChainStore)
	genStore:=myKeyChainStore
	privKey, err := trustmanager.GenerateECDSAKey(rand.Reader)
	fmt.Println(privKey.Private())
	fmt.Println(err)
	fmt.Println("adding")
	genStore.Add(privKey.ID(), privKey.Private())
	fmt.Println("getting")
	servename := "https://notary.docker.io/" + privKey.ID()
	gotCreds, err := genStore.Get(servename)
	fmt.Println(gotCreds)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("removing")
	genStore.Remove(privKey.ID())
}
