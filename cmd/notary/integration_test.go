// Actually start up a notary server and run through basic TUF and key
// interactions via the client.

// Note - if using Yubikey, retrieving pins/touch doesn't seem to work right
// when running in the midst of all tests.

package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
	ctxu "github.com/docker/distribution/context"
	"github.com/docker/notary"
	"github.com/docker/notary/cryptoservice"
	"github.com/docker/notary/passphrase"
	"github.com/docker/notary/server"
	"github.com/docker/notary/server/storage"
	"github.com/docker/notary/trustmanager"
	"github.com/docker/notary/tuf/data"
	"github.com/docker/notary/tuf/utils"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

var testPassphrase = "passphrase"
var NewNotaryCommand func() *cobra.Command

// run a command and return the output as a string
func runCommand(t *testing.T, tempDir string, args ...string) (string, error) {
	b := new(bytes.Buffer)

	// Create an empty config file so we don't load the default on ~/.notary/config.json
	configFile := filepath.Join(tempDir, "config.json")

	cmd := NewNotaryCommand()
	cmd.SetArgs(append([]string{"-c", configFile, "-d", tempDir}, args...))
	cmd.SetOutput(b)
	retErr := cmd.Execute()
	output, err := ioutil.ReadAll(b)
	require.NoError(t, err)

	// Clean up state to mimic running a fresh command next time
	for _, command := range cmd.Commands() {
		command.ResetFlags()
	}

	return string(output), retErr
}

func setupServerHandler(metaStore storage.MetaStore) http.Handler {
	ctx := context.WithValue(context.Background(), "metaStore", metaStore)

	ctx = context.WithValue(ctx, "keyAlgorithm", data.ECDSAKey)

	// Eat the logs instead of spewing them out
	var b bytes.Buffer
	l := logrus.New()
	l.Out = &b
	ctx = ctxu.WithLogger(ctx, logrus.NewEntry(l))

	cryptoService := cryptoservice.NewCryptoService(trustmanager.NewKeyMemoryStore(passphrase.ConstantRetriever("pass")))
	return server.RootHandler(nil, ctx, cryptoService, nil, nil, nil)
}

// makes a testing notary-server
func setupServer() *httptest.Server {
	return httptest.NewServer(setupServerHandler(storage.NewMemStorage()))
}

// Initializes a repo with existing key
func TestInitWithRootKey(t *testing.T) {
	// -- setup --
	setUp(t)

	tempDir := tempDirWithConfig(t, "{}")
	defer os.RemoveAll(tempDir)

	server := setupServer()
	defer server.Close()

	tempFile, err := ioutil.TempFile("", "targetfile")
	require.NoError(t, err)
	tempFile.Close()
	defer os.Remove(tempFile.Name())

	// -- tests --

	// create encrypted root key
	privKey, err := utils.GenerateECDSAKey(rand.Reader)
	require.NoError(t, err)
	encryptedPEMPrivKey, err := utils.EncryptPrivateKey(privKey, data.CanonicalRootRole, testPassphrase)
	require.NoError(t, err)
	encryptedPEMKeyFilename := filepath.Join(tempDir, "encrypted_key.key")
	err = ioutil.WriteFile(encryptedPEMKeyFilename, encryptedPEMPrivKey, 0644)
	require.NoError(t, err)

	// init repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "init", "gun", "--rootkey", encryptedPEMKeyFilename)
	require.NoError(t, err)

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// check that the root key used for init is the one listed as root key
	output, err := runCommand(t, tempDir, "key", "list")
	require.NoError(t, err)
	require.True(t, strings.Contains(output, data.PublicKeyFromPrivate(privKey).ID()))

	// check error if file doesn't exist
	_, err = runCommand(t, tempDir, "-s", server.URL, "init", "gun2", "--rootkey", "bad_file")
	require.Error(t, err, "Init with nonexistent key file should error")

	// check error if file is invalid format
	badKeyFilename := filepath.Join(tempDir, "bad_key.key")
	nonPEMKey := []byte("thisisnotapemkey")
	err = ioutil.WriteFile(badKeyFilename, nonPEMKey, 0644)
	require.NoError(t, err)

	_, err = runCommand(t, tempDir, "-s", server.URL, "init", "gun2", "--rootkey", badKeyFilename)
	require.Error(t, err, "Init with non-PEM key should error")

	// check error if unencrypted PEM used
	unencryptedPrivKey, err := utils.GenerateECDSAKey(rand.Reader)
	require.NoError(t, err)
	unencryptedPEMPrivKey, err := utils.KeyToPEM(unencryptedPrivKey, data.CanonicalRootRole)
	require.NoError(t, err)
	unencryptedPEMKeyFilename := filepath.Join(tempDir, "unencrypted_key.key")
	err = ioutil.WriteFile(unencryptedPEMKeyFilename, unencryptedPEMPrivKey, 0644)
	require.NoError(t, err)

	_, err = runCommand(t, tempDir, "-s", server.URL, "init", "gun2", "--rootkey", unencryptedPEMKeyFilename)
	require.Error(t, err, "Init with unencrypted PEM key should error")

	// check error if invalid password used
	// instead of using a new retriever, we create a new key with a different pass
	badPassPrivKey, err := utils.GenerateECDSAKey(rand.Reader)
	require.NoError(t, err)
	badPassPEMPrivKey, err := utils.EncryptPrivateKey(badPassPrivKey, data.CanonicalRootRole, "bad_pass")
	require.NoError(t, err)
	badPassPEMKeyFilename := filepath.Join(tempDir, "badpass_key.key")
	err = ioutil.WriteFile(badPassPEMKeyFilename, badPassPEMPrivKey, 0644)
	require.NoError(t, err)

	_, err = runCommand(t, tempDir, "-s", server.URL, "init", "gun2", "--rootkey", badPassPEMKeyFilename)
	require.Error(t, err, "Init with wrong password should error")

	// check error if wrong role specified
	snapshotPrivKey, err := utils.GenerateECDSAKey(rand.Reader)
	require.NoError(t, err)
	snapshotPEMPrivKey, err := utils.KeyToPEM(snapshotPrivKey, data.CanonicalSnapshotRole)
	require.NoError(t, err)
	snapshotPEMKeyFilename := filepath.Join(tempDir, "snapshot_key.key")
	err = ioutil.WriteFile(snapshotPEMKeyFilename, snapshotPEMPrivKey, 0644)
	require.NoError(t, err)

	_, err = runCommand(t, tempDir, "-s", server.URL, "init", "gun2", "--rootkey", snapshotPEMKeyFilename)
	require.Error(t, err, "Init with wrong role should error")
}

// Initializes a repo, adds a target, publishes the target, lists the target,
// verifies the target, and then removes the target.
func TestClientTUFInteraction(t *testing.T) {
	// -- setup --
	setUp(t)

	tempDir := tempDirWithConfig(t, "{}")
	defer os.RemoveAll(tempDir)

	server := setupServer()
	defer server.Close()

	tempFile, err := ioutil.TempFile("", "targetfile")
	require.NoError(t, err)
	tempFile.Close()
	defer os.Remove(tempFile.Name())

	var (
		output string
		target = "sdgkadga"
	)
	// -- tests --

	// init repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "init", "gun")
	require.NoError(t, err)

	// add a target
	_, err = runCommand(t, tempDir, "add", "gun", target, tempFile.Name())
	require.NoError(t, err)

	// check status - see target
	output, err = runCommand(t, tempDir, "status", "gun")
	require.NoError(t, err)
	require.True(t, strings.Contains(output, target))

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// check status - no targets
	output, err = runCommand(t, tempDir, "status", "gun")
	require.NoError(t, err)
	require.False(t, strings.Contains(string(output), target))

	// list repo - see target
	output, err = runCommand(t, tempDir, "-s", server.URL, "list", "gun")
	require.NoError(t, err)
	require.True(t, strings.Contains(string(output), target))

	// lookup target and repo - see target
	output, err = runCommand(t, tempDir, "-s", server.URL, "lookup", "gun", target)
	require.NoError(t, err)
	require.True(t, strings.Contains(string(output), target))

	// verify repo - empty file
	output, err = runCommand(t, tempDir, "-s", server.URL, "verify", "gun", target)
	require.NoError(t, err)

	// remove target
	_, err = runCommand(t, tempDir, "remove", "gun", target)
	require.NoError(t, err)

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// list repo - don't see target
	output, err = runCommand(t, tempDir, "-s", server.URL, "list", "gun")
	require.NoError(t, err)
	require.False(t, strings.Contains(string(output), target))
}

// Initializes a repo, adds a target, publishes the target by hash, lists the target,
// verifies the target, and then removes the target.
func TestClientTUFAddByHashInteraction(t *testing.T) {
	// -- setup --
	setUp(t)

	tempDir := tempDirWithConfig(t, "{}")
	defer os.RemoveAll(tempDir)

	server := setupServer()
	defer server.Close()

	targetData := []byte{'a', 'b', 'c'}
	target256Bytes := sha256.Sum256(targetData)
	targetSha256Hex := hex.EncodeToString(target256Bytes[:])
	target512Bytes := sha512.Sum512(targetData)
	targetSha512Hex := hex.EncodeToString(target512Bytes[:])

	err := ioutil.WriteFile(filepath.Join(tempDir, "tempfile"), targetData, 0644)
	require.NoError(t, err)

	var (
		output  string
		target1 = "sdgkadga"
		target2 = "asdfasdf"
		target3 = "qwerty"
	)
	// -- tests --

	// init repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "init", "gun")
	require.NoError(t, err)

	// add a target just by sha256
	_, err = runCommand(t, tempDir, "addhash", "gun", target1, "3", "--sha256", targetSha256Hex)
	require.NoError(t, err)

	// check status - see target
	output, err = runCommand(t, tempDir, "status", "gun")
	require.NoError(t, err)
	require.True(t, strings.Contains(output, target1))

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// check status - no targets
	output, err = runCommand(t, tempDir, "status", "gun")
	require.NoError(t, err)
	require.False(t, strings.Contains(string(output), target1))

	// list repo - see target
	output, err = runCommand(t, tempDir, "-s", server.URL, "list", "gun")
	require.NoError(t, err)
	require.True(t, strings.Contains(string(output), target1))

	// lookup target and repo - see target
	output, err = runCommand(t, tempDir, "-s", server.URL, "lookup", "gun", target1)
	require.NoError(t, err)
	require.True(t, strings.Contains(string(output), target1))

	// remove target
	_, err = runCommand(t, tempDir, "remove", "gun", target1)
	require.NoError(t, err)

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// list repo - don't see target
	output, err = runCommand(t, tempDir, "-s", server.URL, "list", "gun")
	require.NoError(t, err)
	require.False(t, strings.Contains(string(output), target1))

	// add a target just by sha512
	_, err = runCommand(t, tempDir, "addhash", "gun", target2, "3", "--sha512", targetSha512Hex)
	require.NoError(t, err)

	// check status - see target
	output, err = runCommand(t, tempDir, "status", "gun")
	require.NoError(t, err)
	require.True(t, strings.Contains(output, target2))

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// check status - no targets
	output, err = runCommand(t, tempDir, "status", "gun")
	require.NoError(t, err)
	require.False(t, strings.Contains(string(output), target2))

	// list repo - see target
	output, err = runCommand(t, tempDir, "-s", server.URL, "list", "gun")
	require.NoError(t, err)
	require.True(t, strings.Contains(string(output), target2))

	// lookup target and repo - see target
	output, err = runCommand(t, tempDir, "-s", server.URL, "lookup", "gun", target2)
	require.NoError(t, err)
	require.True(t, strings.Contains(string(output), target2))

	// remove target
	_, err = runCommand(t, tempDir, "remove", "gun", target2)
	require.NoError(t, err)

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// add a target by sha256 and sha512
	_, err = runCommand(t, tempDir, "addhash", "gun", target3, "3", "--sha256", targetSha256Hex, "--sha512", targetSha512Hex)
	require.NoError(t, err)

	// check status - see target
	output, err = runCommand(t, tempDir, "status", "gun")
	require.NoError(t, err)
	require.True(t, strings.Contains(output, target3))

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// check status - no targets
	output, err = runCommand(t, tempDir, "status", "gun")
	require.NoError(t, err)
	require.False(t, strings.Contains(string(output), target3))

	// list repo - see target
	output, err = runCommand(t, tempDir, "-s", server.URL, "list", "gun")
	require.NoError(t, err)
	require.True(t, strings.Contains(string(output), target3))

	// lookup target and repo - see target
	output, err = runCommand(t, tempDir, "-s", server.URL, "lookup", "gun", target3)
	require.NoError(t, err)
	require.True(t, strings.Contains(string(output), target3))

	// remove target
	_, err = runCommand(t, tempDir, "remove", "gun", target3)
	require.NoError(t, err)

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)
}

// Initialize repo and test delegations commands by adding, listing, and removing delegations
func TestClientDelegationsInteraction(t *testing.T) {
	setUp(t)

	tempDir := tempDirWithConfig(t, "{}")
	defer os.RemoveAll(tempDir)

	server := setupServer()
	defer server.Close()

	// Setup certificate
	tempFile, err := ioutil.TempFile("", "pemfile")
	require.NoError(t, err)

	privKey, err := utils.GenerateECDSAKey(rand.Reader)
	startTime := time.Now()
	endTime := startTime.AddDate(10, 0, 0)
	cert, err := cryptoservice.GenerateCertificate(privKey, "gun", startTime, endTime)
	require.NoError(t, err)

	_, err = tempFile.Write(utils.CertToPEM(cert))
	require.NoError(t, err)
	tempFile.Close()
	defer os.Remove(tempFile.Name())

	rawPubBytes, _ := ioutil.ReadFile(tempFile.Name())
	parsedPubKey, _ := utils.ParsePEMPublicKey(rawPubBytes)
	keyID, err := utils.CanonicalKeyID(parsedPubKey)
	require.NoError(t, err)

	var output string

	// -- tests --

	// init repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "init", "gun")
	require.NoError(t, err)

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// list delegations - none yet
	output, err = runCommand(t, tempDir, "-s", server.URL, "delegation", "list", "gun")
	require.NoError(t, err)
	require.Contains(t, output, "No delegations present in this repository.")

	// add new valid delegation with single new cert, and no path
	output, err = runCommand(t, tempDir, "delegation", "add", "gun", "targets/delegation", tempFile.Name())
	require.NoError(t, err)
	require.Contains(t, output, "Addition of delegation role")
	require.Contains(t, output, keyID)
	require.NotContains(t, output, "path")

	// check status - see delegation
	output, err = runCommand(t, tempDir, "status", "gun")
	require.NoError(t, err)
	require.Contains(t, output, "Unpublished changes for gun")

	// list delegations - none yet because still unpublished
	output, err = runCommand(t, tempDir, "-s", server.URL, "delegation", "list", "gun")
	require.NoError(t, err)
	require.Contains(t, output, "No delegations present in this repository.")

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// check status - no changelist
	output, err = runCommand(t, tempDir, "status", "gun")
	require.NoError(t, err)
	require.Contains(t, output, "No unpublished changes for gun")

	// list delegations - we should see our added delegation, with no paths
	output, err = runCommand(t, tempDir, "-s", server.URL, "delegation", "list", "gun")
	require.NoError(t, err)
	require.Contains(t, output, "targets/delegation")
	require.Contains(t, output, keyID)
	require.NotContains(t, output, "\"\"")

	// add all paths to this delegation
	output, err = runCommand(t, tempDir, "delegation", "add", "gun", "targets/delegation", "--all-paths")
	require.NoError(t, err)
	require.Contains(t, output, "Addition of delegation role")
	require.Contains(t, output, "\"\"")
	require.Contains(t, output, "<all paths>")

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// list delegations - we should see our added delegation, with no paths
	output, err = runCommand(t, tempDir, "-s", server.URL, "delegation", "list", "gun")
	require.NoError(t, err)
	require.Contains(t, output, "targets/delegation")
	require.Contains(t, output, "\"\"")
	require.Contains(t, output, "<all paths>")

	// Setup another certificate
	tempFile2, err := ioutil.TempFile("", "pemfile2")
	require.NoError(t, err)

	privKey, err = utils.GenerateECDSAKey(rand.Reader)
	startTime = time.Now()
	endTime = startTime.AddDate(10, 0, 0)
	cert, err = cryptoservice.GenerateCertificate(privKey, "gun", startTime, endTime)
	require.NoError(t, err)

	_, err = tempFile2.Write(utils.CertToPEM(cert))
	require.NoError(t, err)
	require.NoError(t, err)
	tempFile2.Close()
	defer os.Remove(tempFile2.Name())

	rawPubBytes2, _ := ioutil.ReadFile(tempFile2.Name())
	parsedPubKey2, _ := utils.ParsePEMPublicKey(rawPubBytes2)
	keyID2, err := utils.CanonicalKeyID(parsedPubKey2)
	require.NoError(t, err)

	// add to the delegation by specifying the same role, this time add a scoped path
	output, err = runCommand(t, tempDir, "delegation", "add", "gun", "targets/delegation", tempFile2.Name(), "--paths", "path")
	require.NoError(t, err)
	require.Contains(t, output, "Addition of delegation role")
	require.Contains(t, output, keyID2)

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// list delegations - we should see two keys
	output, err = runCommand(t, tempDir, "-s", server.URL, "delegation", "list", "gun")
	require.NoError(t, err)
	require.Contains(t, output, "path")
	require.Contains(t, output, keyID)
	require.Contains(t, output, keyID2)

	// remove the delegation's first key
	output, err = runCommand(t, tempDir, "delegation", "remove", "gun", "targets/delegation", keyID)
	require.NoError(t, err)
	require.Contains(t, output, "Removal of delegation role")

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// list delegations - we should see the delegation but with only the second key
	output, err = runCommand(t, tempDir, "-s", server.URL, "delegation", "list", "gun")
	require.NoError(t, err)
	require.NotContains(t, output, keyID)
	require.Contains(t, output, keyID2)

	// remove the delegation's second key
	output, err = runCommand(t, tempDir, "delegation", "remove", "gun", "targets/delegation", keyID2)
	require.NoError(t, err)
	require.Contains(t, output, "Removal of delegation role")

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// list delegations - we should see no delegations
	output, err = runCommand(t, tempDir, "-s", server.URL, "delegation", "list", "gun")
	require.NoError(t, err)
	require.NotContains(t, output, keyID)
	require.NotContains(t, output, keyID2)

	// add delegation with multiple certs and multiple paths
	output, err = runCommand(t, tempDir, "delegation", "add", "gun", "targets/delegation", tempFile.Name(), tempFile2.Name(), "--paths", "path1,path2")
	require.NoError(t, err)
	require.Contains(t, output, "Addition of delegation role")
	require.Contains(t, output, keyID)
	require.Contains(t, output, keyID2)

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// list delegations - we should see two keys
	output, err = runCommand(t, tempDir, "-s", server.URL, "delegation", "list", "gun")
	require.NoError(t, err)
	require.Contains(t, output, "path1")
	require.Contains(t, output, "path2")
	require.Contains(t, output, keyID)
	require.Contains(t, output, keyID2)

	// add delegation with multiple certs and multiple paths
	output, err = runCommand(t, tempDir, "delegation", "add", "gun", "targets/delegation", "--paths", "path3")
	require.NoError(t, err)
	require.Contains(t, output, "Addition of delegation role")

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// list delegations - we should see two keys
	output, err = runCommand(t, tempDir, "-s", server.URL, "delegation", "list", "gun")
	require.NoError(t, err)
	require.Contains(t, output, "path1")
	require.Contains(t, output, "path2")
	require.Contains(t, output, "path3")
	require.Contains(t, output, keyID)
	require.Contains(t, output, keyID2)

	// just remove two paths from this delegation
	output, err = runCommand(t, tempDir, "delegation", "remove", "gun", "targets/delegation", "--paths", "path2,path3")
	require.NoError(t, err)
	require.Contains(t, output, "Removal of delegation role")

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// list delegations - we should see the same two keys, and only path1
	output, err = runCommand(t, tempDir, "-s", server.URL, "delegation", "list", "gun")
	require.NoError(t, err)
	require.Contains(t, output, "path1")
	require.NotContains(t, output, "path2")
	require.NotContains(t, output, "path3")
	require.Contains(t, output, keyID)
	require.Contains(t, output, keyID2)

	// remove the remaining path, should not remove the delegation entirely
	output, err = runCommand(t, tempDir, "delegation", "remove", "gun", "targets/delegation", "--paths", "path1")
	require.NoError(t, err)
	require.Contains(t, output, "Removal of delegation role")

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// list delegations - we should see the same two keys, and no paths
	output, err = runCommand(t, tempDir, "-s", server.URL, "delegation", "list", "gun")
	require.NoError(t, err)
	require.NotContains(t, output, "path1")
	require.NotContains(t, output, "path2")
	require.NotContains(t, output, "path3")
	require.Contains(t, output, keyID)
	require.Contains(t, output, keyID2)

	// Add a bunch of individual paths so we can test a delegation remove --all-paths
	output, err = runCommand(t, tempDir, "delegation", "add", "gun", "targets/delegation", "--paths", "abcdef,123456")
	require.NoError(t, err)

	// Add more individual paths so we can test a delegation remove --all-paths
	output, err = runCommand(t, tempDir, "delegation", "add", "gun", "targets/delegation", "--paths", "banana/split,apple/crumble/pie,orange.peel,kiwi")
	require.NoError(t, err)

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// list delegations - we should see all of our paths
	output, err = runCommand(t, tempDir, "-s", server.URL, "delegation", "list", "gun")
	require.NoError(t, err)
	require.Contains(t, output, "abcdef")
	require.Contains(t, output, "123456")
	require.Contains(t, output, "banana/split")
	require.Contains(t, output, "apple/crumble/pie")
	require.Contains(t, output, "orange.peel")
	require.Contains(t, output, "kiwi")

	// Try adding "", and check that adding it with other paths clears out the others
	output, err = runCommand(t, tempDir, "delegation", "add", "gun", "targets/delegation", "--paths", "\"\",grapefruit,pomegranate")
	require.NoError(t, err)

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// list delegations - we should see all of our old paths, and ""
	output, err = runCommand(t, tempDir, "-s", server.URL, "delegation", "list", "gun")
	require.NoError(t, err)
	require.Contains(t, output, "abcdef")
	require.Contains(t, output, "123456")
	require.Contains(t, output, "banana/split")
	require.Contains(t, output, "apple/crumble/pie")
	require.Contains(t, output, "orange.peel")
	require.Contains(t, output, "kiwi")
	require.Contains(t, output, "\"\"")
	require.NotContains(t, output, "grapefruit")
	require.NotContains(t, output, "pomegranate")

	// Try removing just ""
	output, err = runCommand(t, tempDir, "delegation", "remove", "gun", "targets/delegation", "--paths", "\"\"")
	require.NoError(t, err)

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// list delegations - we should see all of our old paths without ""
	output, err = runCommand(t, tempDir, "-s", server.URL, "delegation", "list", "gun")
	require.NoError(t, err)
	require.Contains(t, output, "abcdef")
	require.Contains(t, output, "123456")
	require.Contains(t, output, "banana/split")
	require.Contains(t, output, "apple/crumble/pie")
	require.Contains(t, output, "orange.peel")
	require.Contains(t, output, "kiwi")
	require.NotContains(t, output, "\"\"")

	// Remove --all-paths to clear out all paths from this delegation
	output, err = runCommand(t, tempDir, "delegation", "remove", "gun", "targets/delegation", "--all-paths")
	require.NoError(t, err)

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// list delegations - we should see all of our paths
	output, err = runCommand(t, tempDir, "-s", server.URL, "delegation", "list", "gun")
	require.NoError(t, err)
	require.NotContains(t, output, "abcdef")
	require.NotContains(t, output, "123456")
	require.NotContains(t, output, "banana/split")
	require.NotContains(t, output, "apple/crumble/pie")
	require.NotContains(t, output, "orange.peel")
	require.NotContains(t, output, "kiwi")

	// Check that we ignore other --paths if we pass in --all-paths on an add
	output, err = runCommand(t, tempDir, "delegation", "add", "gun", "targets/delegation", "--all-paths", "--paths", "grapefruit,pomegranate")
	require.NoError(t, err)

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// list delegations - we should only see "", and not the other paths specified
	output, err = runCommand(t, tempDir, "-s", server.URL, "delegation", "list", "gun")
	require.NoError(t, err)
	require.Contains(t, output, "\"\"")
	require.NotContains(t, output, "grapefruit")
	require.NotContains(t, output, "pomegranate")

	// Add those extra paths we ignored to set up the next test
	output, err = runCommand(t, tempDir, "delegation", "add", "gun", "targets/delegation", "--paths", "grapefruit,pomegranate")
	require.NoError(t, err)

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// Check that we ignore other --paths if we pass in --all-paths on a remove
	output, err = runCommand(t, tempDir, "delegation", "remove", "gun", "targets/delegation", "--all-paths", "--paths", "pomegranate")
	require.NoError(t, err)

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// list delegations - we should see no paths
	output, err = runCommand(t, tempDir, "-s", server.URL, "delegation", "list", "gun")
	require.NoError(t, err)
	require.NotContains(t, output, "\"\"")
	require.NotContains(t, output, "grapefruit")
	require.NotContains(t, output, "pomegranate")

	// remove by force to delete the delegation entirely
	output, err = runCommand(t, tempDir, "delegation", "remove", "gun", "targets/delegation", "-y")
	require.NoError(t, err)
	require.Contains(t, output, "Forced removal (including all keys and paths) of delegation role")

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// list delegations - we should see no delegations
	output, err = runCommand(t, tempDir, "-s", server.URL, "delegation", "list", "gun")
	require.NoError(t, err)
	require.Contains(t, output, "No delegations present in this repository.")
}

// Initialize repo and test publishing targets with delegation roles
func TestClientDelegationsPublishing(t *testing.T) {
	setUp(t)

	tempDir := tempDirWithConfig(t, "{}")
	defer os.RemoveAll(tempDir)

	server := setupServer()
	defer server.Close()

	// Setup certificate for delegation role
	tempFile, err := ioutil.TempFile("", "pemfile")
	require.NoError(t, err)

	privKey, err := utils.GenerateRSAKey(rand.Reader, 2048)
	require.NoError(t, err)
	privKeyBytesNoRole, err := utils.KeyToPEM(privKey, "")
	require.NoError(t, err)
	privKeyBytesWithRole, err := utils.KeyToPEM(privKey, "user")
	require.NoError(t, err)
	startTime := time.Now()
	endTime := startTime.AddDate(10, 0, 0)
	cert, err := cryptoservice.GenerateCertificate(privKey, "gun", startTime, endTime)
	require.NoError(t, err)

	_, err = tempFile.Write(utils.CertToPEM(cert))
	require.NoError(t, err)
	tempFile.Close()
	defer os.Remove(tempFile.Name())

	rawPubBytes, _ := ioutil.ReadFile(tempFile.Name())
	parsedPubKey, _ := utils.ParsePEMPublicKey(rawPubBytes)
	canonicalKeyID, err := utils.CanonicalKeyID(parsedPubKey)
	require.NoError(t, err)

	// Set up targets for publishing
	tempTargetFile, err := ioutil.TempFile("", "targetfile")
	require.NoError(t, err)
	tempTargetFile.Close()
	defer os.Remove(tempTargetFile.Name())

	var target = "sdgkadga"

	var output string

	// init repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "init", "gun")
	require.NoError(t, err)

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// list delegations - none yet
	output, err = runCommand(t, tempDir, "-s", server.URL, "delegation", "list", "gun")
	require.NoError(t, err)
	require.Contains(t, output, "No delegations present in this repository.")

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// validate that we have all keys, including snapshot
	assertNumKeys(t, tempDir, 1, 2, true)

	// rotate the snapshot key to server
	output, err = runCommand(t, tempDir, "-s", server.URL, "key", "rotate", "gun", "snapshot", "-r")
	require.NoError(t, err)

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// validate that we lost the snapshot signing key
	_, signingKeyIDs := assertNumKeys(t, tempDir, 1, 1, true)
	targetKeyID := signingKeyIDs[0]

	// add new valid delegation with single new cert
	output, err = runCommand(t, tempDir, "delegation", "add", "gun", "targets/releases", tempFile.Name(), "--paths", "\"\"")
	require.NoError(t, err)
	require.Contains(t, output, "Addition of delegation role")
	require.Contains(t, output, canonicalKeyID)

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// list delegations - we should see our one delegation
	output, err = runCommand(t, tempDir, "-s", server.URL, "delegation", "list", "gun")
	require.NoError(t, err)
	require.NotContains(t, output, "No delegations present in this repository.")

	// remove the targets key to demonstrate that delegates don't need this key
	keyDir := filepath.Join(tempDir, "private", "tuf_keys")
	require.NoError(t, os.Remove(filepath.Join(keyDir, "gun", targetKeyID+".key")))

	// Note that we need to use the canonical key ID, followed by the base of the role here
	err = ioutil.WriteFile(filepath.Join(keyDir, canonicalKeyID+"_releases.key"), privKeyBytesNoRole, 0700)
	require.NoError(t, err)

	// add a target using the delegation -- will only add to targets/releases
	_, err = runCommand(t, tempDir, "add", "gun", target, tempTargetFile.Name(), "--roles", "targets/releases")
	require.NoError(t, err)

	// list targets for targets/releases - we should see no targets until we publish
	output, err = runCommand(t, tempDir, "-s", server.URL, "list", "gun", "--roles", "targets/releases")
	require.NoError(t, err)
	require.Contains(t, output, "No targets")

	output, err = runCommand(t, tempDir, "-s", server.URL, "status", "gun")
	require.NoError(t, err)

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// list targets for targets/releases - we should see our target!
	output, err = runCommand(t, tempDir, "-s", server.URL, "list", "gun", "--roles", "targets/releases")
	require.NoError(t, err)
	require.Contains(t, output, "targets/releases")

	// remove the target for this role only
	_, err = runCommand(t, tempDir, "remove", "gun", target, "--roles", "targets/releases")
	require.NoError(t, err)

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// list targets for targets/releases - we should see no targets
	output, err = runCommand(t, tempDir, "-s", server.URL, "list", "gun", "--roles", "targets/releases")
	require.NoError(t, err)
	require.Contains(t, output, "No targets present")

	// Try adding a target with a different key style - private/tuf_keys/canonicalKeyID.key with "user" set as the "role" PEM header
	// First remove the old key and add the new style
	require.NoError(t, os.Remove(filepath.Join(keyDir, canonicalKeyID+"_releases.key")))
	err = ioutil.WriteFile(filepath.Join(keyDir, canonicalKeyID+".key"), privKeyBytesWithRole, 0700)
	require.NoError(t, err)

	// add a target using the delegation -- will only add to targets/releases
	_, err = runCommand(t, tempDir, "add", "gun", target, tempTargetFile.Name(), "--roles", "targets/releases")
	require.NoError(t, err)

	// list targets for targets/releases - we should see no targets until we publish
	output, err = runCommand(t, tempDir, "-s", server.URL, "list", "gun", "--roles", "targets/releases")
	require.NoError(t, err)
	require.Contains(t, output, "No targets")

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// list targets for targets/releases - we should see our target!
	output, err = runCommand(t, tempDir, "-s", server.URL, "list", "gun", "--roles", "targets/releases")
	require.NoError(t, err)
	require.Contains(t, output, "targets/releases")

	// remove the target for this role only
	_, err = runCommand(t, tempDir, "remove", "gun", target, "--roles", "targets/releases")
	require.NoError(t, err)

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// add a target using the delegation -- will only add to targets/releases
	_, err = runCommand(t, tempDir, "add", "gun", target, tempTargetFile.Name(), "--roles", "targets/releases")
	require.NoError(t, err)

	// list targets for targets/releases - we should see no targets until we publish
	output, err = runCommand(t, tempDir, "-s", server.URL, "list", "gun", "--roles", "targets/releases")
	require.NoError(t, err)
	require.Contains(t, output, "No targets")

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// list targets for targets/releases - we should see our target!
	output, err = runCommand(t, tempDir, "-s", server.URL, "list", "gun", "--roles", "targets/releases")
	require.NoError(t, err)
	require.Contains(t, output, "targets/releases")

	// Setup another certificate
	tempFile2, err := ioutil.TempFile("", "pemfile2")
	require.NoError(t, err)

	privKey, err = utils.GenerateECDSAKey(rand.Reader)
	startTime = time.Now()
	endTime = startTime.AddDate(10, 0, 0)
	cert, err = cryptoservice.GenerateCertificate(privKey, "gun", startTime, endTime)
	require.NoError(t, err)

	_, err = tempFile2.Write(utils.CertToPEM(cert))
	require.NoError(t, err)
	require.NoError(t, err)
	tempFile2.Close()
	defer os.Remove(tempFile2.Name())

	rawPubBytes2, _ := ioutil.ReadFile(tempFile2.Name())
	parsedPubKey2, _ := utils.ParsePEMPublicKey(rawPubBytes2)
	keyID2, err := utils.CanonicalKeyID(parsedPubKey2)
	require.NoError(t, err)

	// add a nested delegation under this releases role
	output, err = runCommand(t, tempDir, "delegation", "add", "gun", "targets/releases/nested", tempFile2.Name(), "--paths", "nested/path")
	require.NoError(t, err)
	require.Contains(t, output, "Addition of delegation role")
	require.Contains(t, output, keyID2)

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

	// list delegations - we should see two roles
	output, err = runCommand(t, tempDir, "-s", server.URL, "delegation", "list", "gun")
	require.NoError(t, err)
	require.Contains(t, output, "targets/releases")
	require.Contains(t, output, "targets/releases/nested")
	require.Contains(t, output, canonicalKeyID)
	require.Contains(t, output, keyID2)
	require.Contains(t, output, "nested/path")
	require.Contains(t, output, "\"\"")
	require.Contains(t, output, "<all paths>")

	// remove by force to delete the nested delegation entirely
	output, err = runCommand(t, tempDir, "delegation", "remove", "gun", "targets/releases/nested", "-y")
	require.NoError(t, err)
	require.Contains(t, output, "Forced removal (including all keys and paths) of delegation role")

	// publish repo
	_, err = runCommand(t, tempDir, "-s", server.URL, "publish", "gun")
	require.NoError(t, err)

}

// Splits a string into lines, and returns any lines that are not empty (
// striped of whitespace)
func splitLines(chunk string) []string {
	splitted := strings.Split(strings.TrimSpace(chunk), "\n")
	var results []string

	for _, line := range splitted {
		line := strings.TrimSpace(line)
		if line != "" {
			results = append(results, line)
		}
	}

	return results
}

// List keys, parses the output, and returns the unique key IDs as an array
// of root key IDs and an array of signing key IDs.  Output expected looks like:
//     ROLE      GUN          KEY ID                   LOCATION
// ----------------------------------------------------------------
//   root               8bd63a896398b558ac...   file (.../private)
//   snapshot   repo    e9e9425cd9a85fc7a5...   file (.../private)
//   targets    repo    f5b84e2d92708c5acb...   file (.../private)
func getUniqueKeys(t *testing.T, tempDir string) ([]string, []string) {
	output, err := runCommand(t, tempDir, "key", "list")
	require.NoError(t, err)
	lines := splitLines(output)
	if len(lines) == 1 && lines[0] == "No signing keys found." {
		return []string{}, []string{}
	}
	if len(lines) < 3 { // 2 lines of header, at least 1 line with keys
		t.Logf("This output is not what is expected by the test:\n%s", output)
	}

	var (
		rootMap    = make(map[string]bool)
		nonrootMap = make(map[string]bool)
		root       []string
		nonroot    []string
	)
	// first two lines are header
	for _, line := range lines[2:] {
		parts := strings.Fields(line)
		var (
			placeToGo map[string]bool
			keyID     string
		)
		if strings.TrimSpace(parts[0]) == "root" {
			// no gun, so there are only 3 fields
			placeToGo, keyID = rootMap, parts[1]
		} else {
			// gun comes between role and key ID
			placeToGo, keyID = nonrootMap, parts[2]
		}
		// keys are 32-chars long (32 byte shasum, hex-encoded)
		require.Len(t, keyID, 64)
		placeToGo[keyID] = true
	}
	for k := range rootMap {
		root = append(root, k)
	}
	for k := range nonrootMap {
		nonroot = append(nonroot, k)
	}

	return root, nonroot
}

// List keys, parses the output, and asserts something about the number of root
// keys and number of signing keys, as well as returning them.
func assertNumKeys(t *testing.T, tempDir string, numRoot, numSigning int,
	rootOnDisk bool) ([]string, []string) {

	root, signing := getUniqueKeys(t, tempDir)
	require.Len(t, root, numRoot)
	require.Len(t, signing, numSigning)
	for _, rootKeyID := range root {
		_, err := os.Stat(filepath.Join(
			tempDir, "private", "root_keys", rootKeyID+".key"))
		// os.IsExist checks to see if the error is because a file already
		// exists, and hence it isn't actually the right function to use here
		require.Equal(t, rootOnDisk, !os.IsNotExist(err))

		// this function is declared is in the build-tagged setup files
		verifyRootKeyOnHardware(t, rootKeyID)
	}
	return root, signing
}

// Adds the given target to the gun, publishes it, and lists it to ensure that
// it appears.  Returns the listing output.
func assertSuccessfullyPublish(
	t *testing.T, tempDir, url, gun, target, fname string) string {

	_, err := runCommand(t, tempDir, "add", gun, target, fname)
	require.NoError(t, err)

	_, err = runCommand(t, tempDir, "-s", url, "publish", gun)
	require.NoError(t, err)

	output, err := runCommand(t, tempDir, "-s", url, "list", gun)
	require.NoError(t, err)
	require.True(t, strings.Contains(string(output), target))

	return output
}

// Tests root key generation and key rotation
func TestClientKeyGenerationRotation(t *testing.T) {
	// -- setup --
	setUp(t)

	tempDir := tempDirWithConfig(t, "{}")
	defer os.RemoveAll(tempDir)

	tempfiles := make([]string, 2)
	for i := 0; i < 2; i++ {
		tempFile, err := ioutil.TempFile("", "targetfile")
		require.NoError(t, err)
		tempFile.Close()
		tempfiles[i] = tempFile.Name()
		defer os.Remove(tempFile.Name())
	}

	server := setupServer()
	defer server.Close()

	var target = "sdgkadga"

	// -- tests --

	// starts out with no keys
	assertNumKeys(t, tempDir, 0, 0, true)

	// generate root key produces a single root key and no other keys
	_, err := runCommand(t, tempDir, "key", "generate", data.ECDSAKey)
	require.NoError(t, err)
	assertNumKeys(t, tempDir, 1, 0, true)

	// initialize a repo, should have signing keys and no new root key
	_, err = runCommand(t, tempDir, "-s", server.URL, "init", "gun")
	require.NoError(t, err)
	origRoot, origSign := assertNumKeys(t, tempDir, 1, 2, true)

	// publish using the original keys
	assertSuccessfullyPublish(t, tempDir, server.URL, "gun", target, tempfiles[0])

	// rotate the signing keys
	_, err = runCommand(t, tempDir, "-s", server.URL, "key", "rotate", "gun", data.CanonicalSnapshotRole)
	require.NoError(t, err)
	_, err = runCommand(t, tempDir, "-s", server.URL, "key", "rotate", "gun", data.CanonicalTargetsRole)
	require.NoError(t, err)
	root, sign := assertNumKeys(t, tempDir, 1, 2, true)
	require.Equal(t, origRoot[0], root[0])

	// just do a cursory rotation check that the keys aren't equal anymore
	for _, origKey := range origSign {
		for _, key := range sign {
			require.NotEqual(
				t, key, origKey, "One of the signing keys was not removed")
		}
	}

	// publish using the new keys
	output := assertSuccessfullyPublish(
		t, tempDir, server.URL, "gun", target+"2", tempfiles[1])
	// assert that the previous target is sitll there
	require.True(t, strings.Contains(string(output), target))
}

// Helper method to get the subdirectory for TUF keys
func getKeySubdir(role, gun string) string {
	subdir := notary.PrivDir
	switch role {
	case data.CanonicalRootRole:
		return filepath.Join(subdir, notary.RootKeysSubdir)
	case data.CanonicalTargetsRole:
		return filepath.Join(subdir, notary.NonRootKeysSubdir, gun)
	case data.CanonicalSnapshotRole:
		return filepath.Join(subdir, notary.NonRootKeysSubdir, gun)
	default:
		return filepath.Join(subdir, notary.NonRootKeysSubdir)
	}
}

// Tests default root key generation
func TestDefaultRootKeyGeneration(t *testing.T) {
	// -- setup --
	setUp(t)

	tempDir := tempDirWithConfig(t, "{}")
	defer os.RemoveAll(tempDir)

	// -- tests --

	// starts out with no keys
	assertNumKeys(t, tempDir, 0, 0, true)

	// generate root key with no algorithm produces a single ECDSA root key and no other keys
	_, err := runCommand(t, tempDir, "key", "generate")
	require.NoError(t, err)
	assertNumKeys(t, tempDir, 1, 0, true)
}

// Tests the interaction with the verbose and log-level flags
func TestLogLevelFlags(t *testing.T) {
	// Test default to fatal
	n := notaryCommander{}
	n.setVerbosityLevel()
	require.Equal(t, "fatal", logrus.GetLevel().String())

	// Test that verbose (-v) sets to error
	n.verbose = true
	n.setVerbosityLevel()
	require.Equal(t, "error", logrus.GetLevel().String())

	// Test that debug (-D) sets to debug
	n.debug = true
	n.setVerbosityLevel()
	require.Equal(t, "debug", logrus.GetLevel().String())

	// Test that unsetting verboseError still uses verboseDebug
	n.verbose = false
	n.setVerbosityLevel()
	require.Equal(t, "debug", logrus.GetLevel().String())
}

func TestClientKeyPassphraseChange(t *testing.T) {
	// -- setup --
	setUp(t)

	tempDir := tempDirWithConfig(t, "{}")
	defer os.RemoveAll(tempDir)

	server := setupServer()
	defer server.Close()

	target := "sdgkadga"
	tempFile, err := ioutil.TempFile("/tmp", "targetfile")
	require.NoError(t, err)
	tempFile.Close()
	defer os.Remove(tempFile.Name())

	// -- tests --
	_, err = runCommand(t, tempDir, "-s", server.URL, "init", "gun1")
	require.NoError(t, err)

	// we should have three keys stored locally in total: root, targets, snapshot
	rootIDs, signingIDs := assertNumKeys(t, tempDir, 1, 2, true)
	for _, keyID := range signingIDs {
		// try changing the private key passphrase
		_, err = runCommand(t, tempDir, "-s", server.URL, "key", "passwd", keyID)
		require.NoError(t, err)

		// assert that the signing keys (number and IDs) didn't change
		_, signingIDs = assertNumKeys(t, tempDir, 1, 2, true)
		require.Contains(t, signingIDs, keyID)

		// make sure we can still publish with this signing key
		assertSuccessfullyPublish(t, tempDir, server.URL, "gun1", target, tempFile.Name())
	}

	// only one rootID, try changing the private key passphrase
	rootID := rootIDs[0]
	_, err = runCommand(t, tempDir, "-s", server.URL, "key", "passwd", rootID)
	require.NoError(t, err)

	// make sure we can init a new repo with this key
	_, err = runCommand(t, tempDir, "-s", server.URL, "init", "gun2")
	require.NoError(t, err)

	// assert that the root key ID didn't change
	rootIDs, _ = assertNumKeys(t, tempDir, 1, 4, true)
	require.Equal(t, rootID, rootIDs[0])
}

func tempDirWithConfig(t *testing.T, config string) string {
	tempDir, err := ioutil.TempDir("", "repo")
	require.NoError(t, err)
	err = ioutil.WriteFile(filepath.Join(tempDir, "config.json"), []byte(config), 0644)
	require.NoError(t, err)
	return tempDir
}

func TestMain(m *testing.M) {
	if testing.Short() {
		// skip
		os.Exit(0)
	}
	os.Exit(m.Run())
}
