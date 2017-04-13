package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"io/ioutil"
	"testing"

	"github.com/docker/distribution/registry/client/auth"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"golang.org/x/net/context"
	ctxu "github.com/docker/distribution/context"
	"github.com/docker/notary/server/storage"
	"github.com/docker/notary/server"
	"github.com/Sirupsen/logrus"
	"github.com/docker/notary/cryptoservice"
	"github.com/docker/notary"
	"github.com/docker/notary/trustmanager"



	"github.com/docker/notary/tuf/data"
	"github.com/docker/notary/passphrase"
	"bytes"
)

func TestTokenAuth(t *testing.T) {
	var (
		baseTransport          = &http.Transport{}
		gun           data.GUN = "test"
	)
	auth, err := tokenAuth("https://localhost:9999", baseTransport, gun, readOnly)
	require.NoError(t, err)
	require.Nil(t, auth)
}

func TestAdminTokenAuth(t *testing.T) {
	var (
		baseTransport          = &http.Transport{}
		gun           data.GUN = "test"
	)
	auth, err := tokenAuth("https://localhost:9999", baseTransport, gun, admin)
	require.NoError(t, err)
	require.Nil(t, auth)
}

func TestTokenAuth200Status(t *testing.T) {
	var (
		baseTransport          = &http.Transport{}
		gun           data.GUN = "test"
	)
	s := httptest.NewServer(http.HandlerFunc(NotAuthorizedTestHandler))
	defer s.Close()

	auth, err := tokenAuth(s.URL, baseTransport, gun, readOnly)
	require.NoError(t, err)
	require.NotNil(t, auth)
}

func TestAdminTokenAuth200Status(t *testing.T) {
	var (
		baseTransport          = &http.Transport{}
		gun           data.GUN = "test"
	)
	s := httptest.NewServer(http.HandlerFunc(NotAuthorizedTestHandler))
	defer s.Close()

	auth, err := tokenAuth(s.URL, baseTransport, gun, admin)
	require.NoError(t, err)
	require.NotNil(t, auth)
}

func NotAuthorizedTestHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(401)
}

func TestTokenAuth401Status(t *testing.T) {
	var (
		baseTransport          = &http.Transport{}
		gun           data.GUN = "test"
	)
	s := httptest.NewServer(http.HandlerFunc(NotAuthorizedTestHandler))
	defer s.Close()

	auth, err := tokenAuth(s.URL, baseTransport, gun, readOnly)
	require.NoError(t, err)
	require.NotNil(t, auth)
}

func TestAdminTokenAuth401Status(t *testing.T) {
	var (
		baseTransport          = &http.Transport{}
		gun           data.GUN = "test"
	)
	s := httptest.NewServer(http.HandlerFunc(NotAuthorizedTestHandler))
	defer s.Close()

	auth, err := tokenAuth(s.URL, baseTransport, gun, admin)
	require.NoError(t, err)
	require.NotNil(t, auth)
}

func NotFoundTestHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(404)
}

func TestTokenAuthNon200Non401Status(t *testing.T) {
	var (
		baseTransport          = &http.Transport{}
		gun           data.GUN = "test"
	)
	s := httptest.NewServer(http.HandlerFunc(NotFoundTestHandler))
	defer s.Close()

	auth, err := tokenAuth(s.URL, baseTransport, gun, readOnly)
	require.NoError(t, err)
	require.Nil(t, auth)
}

func TestAdminTokenAuthNon200Non401Status(t *testing.T) {
	var (
		baseTransport          = &http.Transport{}
		gun           data.GUN = "test"
	)
	s := httptest.NewServer(http.HandlerFunc(NotFoundTestHandler))
	defer s.Close()

	auth, err := tokenAuth(s.URL, baseTransport, gun, admin)
	require.NoError(t, err)
	require.Nil(t, auth)
}

func TestStatusUnstageAndReset(t *testing.T) {
	setUp(t)
	tempBaseDir := tempDirWithConfig(t, "{}")
	defer os.RemoveAll(tempBaseDir)

	tc := &tufCommander{
		configGetter: func() (*viper.Viper, error) {
			v := viper.New()
			v.SetDefault("trust_dir", tempBaseDir)
			return v, nil
		},
	}

	// run a reset with an empty changelist and make sure it succeeds
	tc.resetAll = true
	err := tc.tufReset(&cobra.Command{}, []string{"gun"})
	require.NoError(t, err)

	// add some targets
	tc.sha256 = "88b76b34ab83a9e4d5abe3697950fb73f940aab1aa5b534f80cf9de9708942be"
	err = tc.tufAddByHash(&cobra.Command{}, []string{"gun", "test1", "100"})
	require.NoError(t, err)
	tc.sha256 = "4a7c203ce63b036a1999ea74eebd307c338368eb2b32218b722de6c5fdc7f016"
	err = tc.tufAddByHash(&cobra.Command{}, []string{"gun", "test2", "100"})
	require.NoError(t, err)
	tc.sha256 = "64bd0565907a6a55fc66fd828a71dbadd976fa875d0a3869f53d02eb8710ecb4"
	err = tc.tufAddByHash(&cobra.Command{}, []string{"gun", "test3", "100"})
	require.NoError(t, err)
	tc.sha256 = "9d9e890af64dd0f44b8a1538ff5fa0511cc31bf1ab89f3a3522a9a581a70fad8"
	err = tc.tufAddByHash(&cobra.Command{}, []string{"gun", "test4", "100"})
	require.NoError(t, err)

	out, err := runCommand(t, tempBaseDir, "status", "gun")
	require.NoError(t, err)
	require.Contains(t, out, "test1")
	require.Contains(t, out, "test2")
	require.Contains(t, out, "test3")
	require.Contains(t, out, "test4")

	_, err = runCommand(t, tempBaseDir, "reset", "gun", "-n", "-1,1,3,10")
	require.NoError(t, err)

	out, err = runCommand(t, tempBaseDir, "status", "gun")
	require.NoError(t, err)
	require.Contains(t, out, "test1")
	require.NotContains(t, out, "test2")
	require.Contains(t, out, "test3")
	require.NotContains(t, out, "test4")

	_, err = runCommand(t, tempBaseDir, "reset", "gun", "--all")
	require.NoError(t, err)

	out, err = runCommand(t, tempBaseDir, "status", "gun")
	require.NoError(t, err)
	require.NotContains(t, out, "test1")
	require.NotContains(t, out, "test2")
	require.NotContains(t, out, "test3")
	require.NotContains(t, out, "test4")

}

func TestGetTrustPinningErrors(t *testing.T) {
	setUp(t)
	invalidTrustPinConfig := tempDirWithConfig(t, `{
		"trust_pinning": {
		    "certs": {
		        "repo3": [60, "abc", [1, 2, 3]]
		    }
		 }
	}`)
	defer os.RemoveAll(invalidTrustPinConfig)

	tc := &tufCommander{
		// returns a nil pointer
		configGetter: func() (*viper.Viper, error) {
			v := viper.New()
			v.SetConfigFile(filepath.Join(invalidTrustPinConfig, "config.json"))
			v.ReadInConfig()
			return v, nil
		},
	}
	require.Error(t, tc.tufStatus(&cobra.Command{}, []string{"gun"}))
	tc.resetAll = true
	require.Error(t, tc.tufReset(&cobra.Command{}, []string{"gun"}))
	require.Error(t, tc.tufInit(&cobra.Command{}, []string{"gun"}))
	require.Error(t, tc.tufPublish(&cobra.Command{}, []string{"gun"}))
	require.Error(t, tc.tufVerify(&cobra.Command{}, []string{"gun", "target", "file"}))
	require.Error(t, tc.tufLookup(&cobra.Command{}, []string{"gun", "target"}))
	require.Error(t, tc.tufList(&cobra.Command{}, []string{"gun"}))
	require.Error(t, tc.tufAdd(&cobra.Command{}, []string{"gun", "target", "file"}))
	require.Error(t, tc.tufRemove(&cobra.Command{}, []string{"gun", "target", "file"}))
	require.Error(t, tc.tufWitness(&cobra.Command{}, []string{"gun", "targets/role"}))
	tc.sha256 = "88b76b34ab83a9e4d5abe3697950fb73f940aab1aa5b534f80cf9de9708942be"
	require.Error(t, tc.tufAddByHash(&cobra.Command{}, []string{"gun", "test1", "100"}))
}

func TestPasswordStore(t *testing.T) {
	myurl, err := url.Parse("https://docker.io")
	require.NoError(t, err)

	// whether or not we're anonymous, because this isn't a terminal,
	for _, ps := range []auth.CredentialStore{passwordStore{}, passwordStore{anonymous: true}} {
		username, passwd := ps.Basic(myurl)
		require.Equal(t, "", username)
		require.Equal(t, "", passwd)

		ps.SetRefreshToken(myurl, "someService", "token") // doesn't return an error, just want to make sure no state changes
		require.Equal(t, "", ps.RefreshToken(myurl, "someService"))
	}
}

func TestExportImportAddFile(t *testing.T) {
	setUp(t)

	s := fullTestServer(t)
	defer s.Close()

	tempBaseDir := tempDirWithConfig(t, "{}")
	defer os.RemoveAll(tempBaseDir)

	tc := &tufCommander{
		configGetter: func() (*viper.Viper, error) {
			v := viper.New()
			v.SetDefault("trust_dir", tempBaseDir)
			v.SetDefault("remote_server.url", s.URL)
			return v, nil
		},

		retriever: passphrase.ConstantRetriever("pass"),
	}

	url, err := tc.configGetter()
	require.Equal(t, url.GetString("remote_server.url"), s.URL)

	require.NoError(t, tc.tufInit(&cobra.Command{}, []string{"gun"}))

	tc.sha256 = "88b76b34ab83a9e4d5abe3697950fb73f940aab1aa5b534f80cf9de9708942be"
	require.NoError(t, tc.tufAddByHash(&cobra.Command{}, []string{"gun", "target1", "100"}))
	require.NoError(t, tc.tufPublish(&cobra.Command{}, []string{"gun"}))

	var buf1 bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOutput(&buf1)
	require.NoError(t, tc.tufList(cmd, []string{"gun"}))
	require.Contains(t, buf1.String(), "target1")


	tempExportDir, err := ioutil.TempDir("", "export-repo")
	require.NoError(t, err)
	defer os.RemoveAll(tempExportDir)

	tc = &tufCommander{
		configGetter: func() (*viper.Viper, error) {
			v := viper.New()
			v.SetDefault("trust_dir", tempBaseDir)
			v.SetDefault("remote_server.url", s.URL)
			return v, nil
		},
		output: tempExportDir,

		retriever: passphrase.ConstantRetriever("pass"),
	}
	require.NoError(t, tc.tufExportGUN(&cobra.Command{}, []string{"gun", "targets"}))
	targetsJSONpath := filepath.Join(tempExportDir, "targets.json")
	_, err = os.Stat(targetsJSONpath)
	require.NoError(t, err)



	targetFilePath := filepath.Join(tempBaseDir,"file2")
	require.NoError(t, ioutil.WriteFile(targetFilePath, []byte("foo"), 0666))

	tc = &tufCommander{
		configGetter: func() (*viper.Viper, error) {
			v := viper.New()
			v.SetDefault("trust_dir", tempBaseDir)
			v.SetDefault("remote_server.url", s.URL)
			return v, nil
		},
		input: targetsJSONpath,
		output: targetsJSONpath,

		retriever: passphrase.ConstantRetriever("pass"),
	}
	require.NoError(t, tc.tufAdd(&cobra.Command{}, []string{"gun", "target2", targetFilePath}))
	require.NoError(t, tc.tufImportGUN(&cobra.Command{}, []string{"gun", "targets", targetsJSONpath}))

	var buf2 bytes.Buffer
	cmd.SetOutput(&buf2)
	require.NoError(t, tc.tufList(cmd, []string{"gun"}))
	require.Contains(t, buf2.String(), "target1")
	require.Contains(t, buf2.String(), "target2")
}

func fullTestServer(t *testing.T) *httptest.Server {
	// Set up server
	ctx := context.WithValue(
		context.Background(), notary.CtxKeyMetaStore, storage.NewMemStorage())

	// Do not pass one of the const KeyAlgorithms here as the value! Passing a
	// string is in itself good test that we are handling it correctly as we
	// will be receiving a string from the configuration.
	ctx = context.WithValue(ctx, notary.CtxKeyKeyAlgo, "ecdsa")

	// Eat the logs instead of spewing them out
	var b bytes.Buffer
	l := logrus.New()
	l.Out = &b
	ctx = ctxu.WithLogger(ctx, logrus.NewEntry(l))

	cryptoService := cryptoservice.NewCryptoService(trustmanager.NewKeyMemoryStore(passphrase.ConstantRetriever("pass")))
	return httptest.NewServer(server.RootHandler(ctx, nil, cryptoService, nil, nil, nil))
}