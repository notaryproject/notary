package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docker/distribution/registry/client/auth"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"github.com/theupdateframework/notary/tuf/data"
)

// TestImportRootCert does the following
// 1. write a certificate to temp file
// 2. use importRootCert to import the certificate
// 3. confirm the content of the certificate to match expected content
// 4. test reading non-existing file
// 5. test reading non-certificate file
// 6. test import non-valid certificate
func TestImportRootCert(t *testing.T) {

	certStr := `-----BEGIN CERTIFICATE-----
MIIBWDCB/6ADAgECAhBKKoVsRNJdGsGh6tPWnE4rMAoGCCqGSM49BAMCMBMxETAP
BgNVBAMTCGRvY2tlci8qMB4XDTE3MDQyODIwMTczMFoXDTI3MDQyNjIwMTczMFow
EzERMA8GA1UEAxMIZG9ja2VyLyowWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAQQ
6RhA8sX/kWedbPPFzNqOMI+AnWOQV+u0+FQfeNO+k/Uf0LBnKhHEPSwSBuuwLPon
w+nR0YTdv3lFaM7x9nOUozUwMzAOBgNVHQ8BAf8EBAMCBaAwEwYDVR0lBAwwCgYI
KwYBBQUHAwMwDAYDVR0TAQH/BAIwADAKBggqhkjOPQQDAgNIADBFAiA+eHPInhLJ
HgP8nha+UqdYgq8ZCOlhdGTJhSdHd4sCuQIhAPXqQeWhDLA3/Pf8B7G3ZwWpPbZ8
adLwkjqoeEKMaAXf
-----END CERTIFICATE-----`
	//create a temp dir
	dir := tempDirWithConfig(t, "{}")
	defer os.RemoveAll(dir)

	//1. write cert to file
	certFile := filepath.Join(dir, "cert.crt")
	err := ioutil.WriteFile(certFile, []byte(certStr), 0644)
	require.NoError(t, err)

	//2. import root cert
	pKeys, err := importRootCert(certFile)
	require.NoError(t, err)
	require.Equal(t, 1, len(pKeys), "length of the public key slice should be 1")

	pkey := pKeys[0]

	require.Equal(t, "c58a735abbb9577f87e149b2493e1039a836ca8002488c7932931d859b850d5d", pkey.ID())
	require.Equal(t, "ecdsa-x509", pkey.Algorithm())
	require.Equal(t, certStr, strings.TrimSpace(string(pkey.Public())))

	//4. test import non-existing file
	fakeFile := filepath.Join(dir, "fake.crt")
	_, err = importRootCert(fakeFile)
	require.Error(t, err)
	require.Contains(t, err.Error(), "error reading")

	//5. test import non-certificate file
	nonCert := filepath.Join(dir, "noncert.crt")
	err = ioutil.WriteFile(nonCert, []byte("fake certificate"), 0644)
	require.NoError(t, err)

	_, err = importRootCert(nonCert)
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not contain a valid PEM certificate")

	// 6. test import non-valid certificate
	errCert := `-----BEGIN CERTIFICATE-----
MIIBWDCB/6ADAgECAhBKKoVsRNJdGsGh6tPWnE4rMAoGCCqGSM49BAMCMBMxETAP
BgNVBAMTCGRvY2tlci8qMB4XDTE31111111wMTczMFoXDTI3MDQyNjIwMTczMFow
EzERMA8GA1UEAxMIZG9ja2VyLyowWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAQQ
6RhA8sX/kWedbPPFzNqOMI+AnWOQV+u0+FQfeNO+k/Uf0LBnKhHEPSwSBuuwLPon
w+nR0YTdv3lFaM7x9nOUozUwMzAOBgNVHQ8BAf8EBAMCBaAwEwYDVR0lBAwwCgYI
KwYBBQUHAwMwDAYDVR0TAQH/BAIwADAKBggqhkjOPQQDAgNIADBFAiA+eHPInhLJ
HgP8nha+UqdYgq8ZCOlhdGTJhSdHd4sCuQIhAPXqQeWhDLA3/Pf8B7G3ZwWpPbZ8
adLwkjqoeEKMaAXf
-----END CERTIFICATE-----`
	errCertFile := filepath.Join(dir, "err.crt")
	err = ioutil.WriteFile(errCertFile, []byte(errCert), 0644)
	require.NoError(t, err)

	_, err = importRootCert(errCertFile)
	require.Error(t, err)
	require.Contains(t, err.Error(), "parsing certificate PEM bytes to x509 certificate")

}

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

func fakeAuthServerFactory(t *testing.T, expectedScope string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		require.Contains(t, r.URL.RawQuery, "scope="+url.QueryEscape(expectedScope))
		w.WriteHeader(200)
	}
}

func authChallengerFactory(URL string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Www-Authenticate", fmt.Sprintf(`Bearer realm="%s"`, URL))
		w.WriteHeader(401)
	}
}
func TestConfigureRepo(t *testing.T) {
	authserver := httptest.NewServer(http.HandlerFunc(fakeAuthServerFactory(t, "repository:yes:pull")))
	defer authserver.Close()

	s := httptest.NewServer(http.HandlerFunc(authChallengerFactory(authserver.URL)))
	defer s.Close()

	tempBaseDir := tempDirWithConfig(t, "{}")
	defer os.RemoveAll(tempBaseDir)
	v := viper.New()
	v.SetDefault("trust_dir", tempBaseDir)
	v.Set("remote_server.url", s.URL)

	repo, err := ConfigureRepo(v, nil, true, readOnly)("yes")
	require.NoError(t, err)
	//perform an arbitrary action to trigger a call to the fake auth server
	repo.ListRoles()
}

func TestConfigureRepoRW(t *testing.T) {
	authserver := httptest.NewServer(http.HandlerFunc(fakeAuthServerFactory(t, "repository:yes:push,pull")))
	defer authserver.Close()

	s := httptest.NewServer(http.HandlerFunc(authChallengerFactory(authserver.URL)))
	defer s.Close()

	tempBaseDir := tempDirWithConfig(t, "{}")
	defer os.RemoveAll(tempBaseDir)
	v := viper.New()
	v.SetDefault("trust_dir", tempBaseDir)
	v.Set("remote_server.url", s.URL)

	repo, err := ConfigureRepo(v, nil, true, readWrite)("yes")
	require.NoError(t, err)
	//perform an arbitrary action to trigger a call to the fake auth server
	repo.ListRoles()
}

func TestConfigureRepoAdmin(t *testing.T) {
	authserver := httptest.NewServer(http.HandlerFunc(fakeAuthServerFactory(t, "repository:yes:*")))
	defer authserver.Close()

	s := httptest.NewServer(http.HandlerFunc(authChallengerFactory(authserver.URL)))
	defer s.Close()

	tempBaseDir := tempDirWithConfig(t, "{}")
	defer os.RemoveAll(tempBaseDir)
	v := viper.New()
	v.SetDefault("trust_dir", tempBaseDir)
	v.Set("remote_server.url", s.URL)

	repo, err := ConfigureRepo(v, nil, true, admin)("yes")
	require.NoError(t, err)
	//perform an arbitrary action to trigger a call to the fake auth server
	repo.ListRoles()
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

func TestPasswordStoreWithEnvvar(t *testing.T) {
	myurl, err := url.Parse("https://docker.io")
	require.NoError(t, err)

	ps := passwordStore{}

	creds := base64.StdEncoding.EncodeToString([]byte("me:mypassword"))
	os.Setenv("NOTARY_AUTH", creds)

	username, passwd := ps.Basic(myurl)
	require.Equal(t, "me", username)
	require.Equal(t, "mypassword", passwd)

	creds = base64.StdEncoding.EncodeToString([]byte(":mypassword"))
	os.Setenv("NOTARY_AUTH", creds)

	username, passwd = ps.Basic(myurl)
	require.Equal(t, "", username)
	require.Equal(t, "", passwd)

	os.Setenv("NOTARY_AUTH", "not-base64-encoded")

	username, passwd = ps.Basic(myurl)
	require.Equal(t, "", username)
	require.Equal(t, "", passwd)
}
