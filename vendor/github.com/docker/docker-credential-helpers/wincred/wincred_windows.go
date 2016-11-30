package wincred

import (
	winc "github.com/danieljoos/wincred"
	"github.com/docker/docker-credential-helpers/credentials"
)

// Wincred handles secrets using the Windows credential service.
type Wincred struct{}

// Add adds new credentials to the windows credentials manager.
func (h Wincred) Add(creds *credentials.Credentials) error {
	g := winc.NewGenericCredential(creds.ServerURL)
	g.UserName = creds.Username
	g.CredentialBlob = []byte(creds.Secret)
	g.Persist = winc.PersistLocalMachine
	return g.Write()
}

// Delete removes credentials from the windows credentials manager.
func (h Wincred) Delete(serverURL string) error {
	g, err := winc.GetGenericCredential(serverURL)
	if g == nil {
		return nil
	}
	if err != nil {
		return err
	}
	return g.Delete()
}

// Get retrieves credentials from the windows credentials manager.
func (h Wincred) Get(serverURL string) (string, string, error) {
	g, _ := winc.GetGenericCredential(serverURL)
	if g == nil {
		return "", "", credentials.NewErrCredentialsNotFound()
	}
	return g.UserName, string(g.CredentialBlob), nil
}

func (h Wincred) List() ([]string, []string, error) {
	creds, err := winc.List()
	paths := make([]string, len(creds))
	accts := make([]string, len(creds))
	if err != nil {
		return nil, nil, err
	}
	for i := range creds {
		paths[i] = creds[i].TargetName
		accts[i] = creds[i].UserName
	}
	return paths, accts, nil
}
