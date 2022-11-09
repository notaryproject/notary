package main

import (
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/theupdateframework/notary"
	notaryclient "github.com/theupdateframework/notary/client"
	"github.com/theupdateframework/notary/cryptoservice"
	store "github.com/theupdateframework/notary/storage"
	"github.com/theupdateframework/notary/trustmanager"
	"github.com/theupdateframework/notary/tuf/data"
	tufutils "github.com/theupdateframework/notary/tuf/utils"
)

var cmdKeyTemplate = usageTemplate{
	Use:   "key",
	Short: "Operates on keys.",
	Long:  `Operations on private keys.`,
}

var cmdKeyListTemplate = usageTemplate{
	Use:   "list",
	Short: "Lists keys.",
	Long:  "Lists all keys known to notary.",
}

var cmdRotateKeyTemplate = usageTemplate{
	Use:   "rotate [ GUN ] [ key role ]",
	Short: "Rotate a signing (non-root) key of the given type for the given Globally Unique Name and role.",
	Long:  `Generates a new key for the given Globally Unique Name and role (one of "snapshot", "targets", "root", or "timestamp").  If rotating to a server-managed key, a new key is requested from the server rather than generated.  If the generation or key request is successful, the key rotation is immediately published.  No other changes, even if they are staged, will be published.`,
}

var cmdKeyGenerateKeyTemplate = usageTemplate{
	Use:   "generate [ algorithm ]",
	Short: "Generates a new key with a given algorithm.",
	Long: "Generates a new key with a given algorithm. If hardware key " +
		"storage (e.g. a Yubikey) is available, generated root keys will be stored both " +
		"on hardware and on disk (so that it can be backed up).  Please make " +
		"sure to back up and then remove this on-key disk immediately" +
		"afterwards. If a `--output` file name is provided, two files will " +
		"be written, <output>.pem and <output>-key.pem, containing the public" +
		"and private keys respectively (the key will not be stored in Notary's " +
		"key storage, including any connected hardware storage). If no `--role` " +
		"is provided, \"root\" will be assumed.",
}

var cmdKeyRemoveTemplate = usageTemplate{
	Use:   "remove [ keyID ]",
	Short: "Removes the key with the given keyID.",
	Long:  "Removes the key with the given keyID.  If the key is stored in more than one location, you will be asked which one to remove.",
}

var cmdKeyPasswdTemplate = usageTemplate{
	Use:   "passwd [ keyID ]",
	Short: "Changes the passphrase for the key with the given keyID.",
	Long:  "Changes the passphrase for the key with the given keyID.  Will require validation of the old passphrase.",
}

var cmdKeyImportTemplate = usageTemplate{
	Use:   "import pemfile [ pemfile ... ]",
	Short: "Imports all keys from all provided .pem files",
	Long:  "Imports all keys from all provided .pem files by reading each PEM block from the file and writing that block to a unique object in the local keystore. A Yubikey will be the preferred import location for root keys if present.",
}

var cmdKeyExportTemplate = usageTemplate{
	Use:   "export",
	Short: "Exports all keys from all local keystores. Can be filtered using the --key and --gun flags.",
	Long:  "Exports all keys from all local keystores. Which keys are exported can be restricted by using the --key or --gun flags. By default the result is sent to stdout, it can be directed to a file with the -o flag. Keys stored in a Yubikey cannot be exported.",
}

type keyCommander struct {
	// these need to be set
	configGetter func() (*viper.Viper, error)
	getRetriever func() notary.PassRetriever

	// these are for command line parsing - no need to set
	rotateKeyRole          string
	rotateKeyServerManaged bool
	rotateKeyFiles         []string
	legacyVersions         int
	input                  io.Reader

	importRole    string
	generateRole  string
	keysImportGUN string
	exportGUNs    []string
	exportKeyIDs  []string
	outFile       string

	autoConfirm bool
}

func (k *keyCommander) GetCommand() *cobra.Command {
	cmd := cmdKeyTemplate.ToCommand(nil)
	cmd.AddCommand(cmdKeyListTemplate.ToCommand(k.keysList))
	cmdGenerate := cmdKeyGenerateKeyTemplate.ToCommand(k.keysGenerate)
	cmdGenerate.Flags().StringVarP(
		&k.outFile,
		"output",
		"o",
		"",
		"Filepath to write export output to",
	)
	cmdGenerate.Flags().StringVarP(
		&k.generateRole, "role", "r", "root", "Role to generate key with, defaulting to \"root\".",
	)
	cmd.AddCommand(cmdGenerate)
	cmd.AddCommand(cmdKeyRemoveTemplate.ToCommand(k.keyRemove))
	cmd.AddCommand(cmdKeyPasswdTemplate.ToCommand(k.keyPassphraseChange))
	cmdRotateKey := cmdRotateKeyTemplate.ToCommand(k.keysRotate)
	cmdRotateKey.Flags().BoolVarP(&k.rotateKeyServerManaged, "server-managed", "r",
		false, "Signing and key management will be handled by the remote server "+
			"(no key will be generated or stored locally). "+
			"Required for timestamp role, optional for snapshot role")
	cmdRotateKey.Flags().IntVarP(&k.legacyVersions, "legacy", "l", 0, "Number of old version's root roles to sign with to support old clients")
	cmdRotateKey.Flags().StringSliceVarP(
		&k.rotateKeyFiles,
		"key",
		"k",
		nil,
		"New key(s) to rotate to. If not specified, one will be generated.",
	)
	cmdRotateKey.Flags().BoolVarP(&k.autoConfirm, "yes", "y", false, "skip confirmation dialog when rotating root role")
	cmd.AddCommand(cmdRotateKey)

	cmdKeysImport := cmdKeyImportTemplate.ToCommand(k.importKeys)
	cmdKeysImport.Flags().StringVarP(
		&k.importRole, "role", "r", "", "Role to import key with, if a role is not already given in a PEM header")
	cmdKeysImport.Flags().StringVarP(
		&k.keysImportGUN, "gun", "g", "", "Gun to import key with, if a gun is not already given in a PEM header")
	cmd.AddCommand(cmdKeysImport)
	cmdExport := cmdKeyExportTemplate.ToCommand(k.exportKeys)
	cmdExport.Flags().StringSliceVar(
		&k.exportGUNs,
		"gun",
		nil,
		"GUNs for which to export keys",
	)
	cmdExport.Flags().StringSliceVar(
		&k.exportKeyIDs,
		"key",
		nil,
		"Key IDs to export",
	)
	cmdExport.Flags().StringVarP(
		&k.outFile,
		"output",
		"o",
		"",
		"Filepath to write export output to",
	)
	cmd.AddCommand(cmdExport)
	return cmd
}

func (k *keyCommander) keysList(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		cmd.Usage()
		return fmt.Errorf("")
	}

	config, err := k.configGetter()
	if err != nil {
		return err
	}
	ks, err := k.getKeyStores(config, true, false)
	if err != nil {
		return err
	}

	cmd.Println("")
	prettyPrintKeys(ks, cmd.OutOrStdout())
	cmd.Println("")
	return nil
}

func (k *keyCommander) keysGenerate(cmd *cobra.Command, args []string) error {
	// We require one or no arguments (since we have a default value), but if the
	// user passes in more than one argument, we error out.
	if len(args) > 1 {
		cmd.Usage()
		return fmt.Errorf(
			"please provide only one Algorithm as an argument to generate (rsa, ecdsa)")
	}

	// If no param is given to generate, generates an ecdsa key by default
	algorithm := data.ECDSAKey

	// If we were provided an argument lets attempt to use it as an algorithm
	if len(args) > 0 {
		algorithm = strings.ToLower(args[0])
	}

	allowedCiphers := map[string]bool{
		data.ECDSAKey: true,
	}

	if !allowedCiphers[algorithm] {
		return fmt.Errorf("algorithm not allowed, possible values are: ECDSA")
	}

	config, err := k.configGetter()
	if err != nil {
		return err
	}

	// if no outFile is provided, use the known key stores
	if k.outFile == "" {
		ks, err := k.getKeyStores(config, true, true)
		if err != nil {
			return err
		}
		cs := cryptoservice.NewCryptoService(ks...)

		pubKey, err := cs.Create(data.RoleName(k.generateRole), "", algorithm)
		if err != nil {
			return fmt.Errorf("failed to create a new %s key: %v", k.generateRole, err)
		}

		cmd.Printf("Generated new %s %s key with keyID: %s\n", algorithm, k.generateRole, pubKey.ID())
		return nil
	}

	// if we had an outfile set, we'll write 2 files with the given name, appending .pem and -key.pem for the
	// public and private keys respectively
	return generateKeyToFile(k.generateRole, algorithm, k.getRetriever(), k.outFile)
}

func generateKeyToFile(role, algorithm string, retriever notary.PassRetriever, outFile string) error {
	privKey, err := tufutils.GenerateKey(algorithm)
	if err != nil {
		return err
	}
	pubKey := data.PublicKeyFromPrivate(privKey)

	var (
		chosenPassphrase string
		giveup           bool
		pemPrivKey       []byte
	)
	keyID := privKey.ID()
	for attempts := 0; ; attempts++ {
		chosenPassphrase, giveup, err = retriever(keyID, "", true, attempts)
		if err == nil {
			break
		}
		if giveup || attempts > 10 {
			return trustmanager.ErrAttemptsExceeded{}
		}
	}

	if chosenPassphrase != "" {
		pemPrivKey, err = tufutils.ConvertPrivateKeyToPKCS8(privKey, data.RoleName(role), "", chosenPassphrase)
		if err != nil {
			return err
		}
	} else {
		return errors.New("no password provided")
	}

	privFileName := strings.Join([]string{outFile, "key"}, "-")
	privFile := strings.Join([]string{privFileName, "pem"}, ".")
	pubFile := strings.Join([]string{outFile, "pem"}, ".")

	err = ioutil.WriteFile(privFile, pemPrivKey, notary.PrivNoExecPerms)
	if err != nil {
		return err
	}

	pubPEM := pem.Block{
		Type: "PUBLIC KEY",
		Headers: map[string]string{
			"role": role,
		},
		Bytes: pubKey.Public(),
	}
	return ioutil.WriteFile(pubFile, pem.EncodeToMemory(&pubPEM), notary.PrivNoExecPerms)
}

func (k *keyCommander) keysRotate(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		cmd.Usage()
		return fmt.Errorf("must specify a GUN and a key role to rotate")
	}

	config, err := k.configGetter()
	if err != nil {
		return err
	}

	gun := data.GUN(args[0])
	rotateKeyRole := data.RoleName(args[1])

	rt, err := getTransport(config, gun, admin)
	if err != nil {
		return err
	}

	trustPin, err := getTrustPinning(config)
	if err != nil {
		return err
	}

	nRepo, err := notaryclient.NewFileCachedRepository(
		config.GetString("trust_dir"), gun, getRemoteTrustServer(config),
		rt, k.getRetriever(), trustPin)
	if err != nil {
		return err
	}

	var keyList []string

	for _, keyFile := range k.rotateKeyFiles {
		privKey, err := readKey(rotateKeyRole, keyFile, k.getRetriever())
		if err != nil {
			return err
		}
		err = nRepo.GetCryptoService().AddKey(rotateKeyRole, gun, privKey)
		if err != nil {
			return fmt.Errorf("error importing key: %v", err)
		}
		keyList = append(keyList, privKey.ID())
	}

	if !k.autoConfirm && rotateKeyRole == data.CanonicalRootRole {
		cmd.Print("Warning: you are about to rotate your root key.\n\n" +
			"You must use your old key to sign this root rotation.\n" +
			"Are you sure you want to proceed?  (yes/no)  ")

		if !askConfirm(k.input) {
			fmt.Fprintln(cmd.OutOrStdout(), "\nAborting action.")
			return nil
		}
	}
	nRepo.SetLegacyVersions(k.legacyVersions)
	if err := nRepo.RotateKey(rotateKeyRole, k.rotateKeyServerManaged, keyList); err != nil {
		return err
	}
	cmd.Printf("Successfully rotated %s key for repository %s\n", rotateKeyRole, gun)
	return nil
}

func removeKeyInteractively(keyStores []trustmanager.KeyStore, keyID string,
	in io.Reader, out io.Writer) error {

	var foundKeys [][]string
	var storesByIndex []trustmanager.KeyStore

	for _, store := range keyStores {
		for keypath, keyInfo := range store.ListKeys() {
			if filepath.Base(keypath) == keyID {
				foundKeys = append(foundKeys,
					[]string{keypath, keyInfo.Role.String(), store.Name()})
				storesByIndex = append(storesByIndex, store)
			}
		}
	}

	if len(foundKeys) == 0 {
		return fmt.Errorf("no key with ID %s found", keyID)
	}

	if len(foundKeys) > 1 {
		for {
			// ask the user for which key to delete
			fmt.Fprintf(out, "Found the following matching keys:\n")
			for i, info := range foundKeys {
				fmt.Fprintf(out, "\t%d. %s: %s (%s)\n", i+1, info[0], info[1], info[2])
			}
			fmt.Fprint(out, "Which would you like to delete?  Please enter a number:  ")
			var result string
			if _, err := fmt.Fscanln(in, &result); err != nil {
				return err
			}
			index, err := strconv.Atoi(strings.TrimSpace(result))

			if err != nil || index > len(foundKeys) || index < 1 {
				fmt.Fprintf(out, "\nInvalid choice: %s\n", string(result))
				continue
			}
			foundKeys = [][]string{foundKeys[index-1]}
			storesByIndex = []trustmanager.KeyStore{storesByIndex[index-1]}
			fmt.Fprintln(out, "")
			break
		}
	}
	// Now the length must be 1 - ask for confirmation.
	keyDescription := fmt.Sprintf("%s (role %s) from %s", foundKeys[0][0],
		foundKeys[0][1], foundKeys[0][2])

	fmt.Fprintf(out, "Are you sure you want to remove %s?  (yes/no)  ",
		keyDescription)
	if !askConfirm(in) {
		fmt.Fprintln(out, "\nAborting action.")
		return nil
	}

	if err := storesByIndex[0].RemoveKey(foundKeys[0][0]); err != nil {
		return err
	}

	fmt.Fprintf(out, "\nDeleted %s.\n", keyDescription)
	return nil
}

// keyRemove deletes a private key based on ID
func (k *keyCommander) keyRemove(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		cmd.Usage()
		return fmt.Errorf("must specify the key ID of the key to remove")
	}

	config, err := k.configGetter()
	if err != nil {
		return err
	}
	ks, err := k.getKeyStores(config, true, false)
	if err != nil {
		return err
	}
	keyID := args[0]

	// This is an invalid ID
	if len(keyID) != notary.SHA256HexSize {
		return fmt.Errorf("invalid key ID provided: %s", keyID)
	}
	cmd.Println("")
	err = removeKeyInteractively(ks, keyID, k.input, cmd.OutOrStdout())
	cmd.Println("")
	return err
}

// keyPassphraseChange changes the passphrase for a private key based on ID
func (k *keyCommander) keyPassphraseChange(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		cmd.Usage()
		return fmt.Errorf("must specify the key ID of the key to change the passphrase of")
	}

	config, err := k.configGetter()
	if err != nil {
		return err
	}
	ks, err := k.getKeyStores(config, true, false)
	if err != nil {
		return err
	}

	keyID := args[0]

	// This is an invalid ID
	if len(keyID) != notary.SHA256HexSize {
		return fmt.Errorf("invalid key ID provided: %s", keyID)
	}

	// Find which keyStore we should replace the key password in, and replace if we find it
	var foundKeyStore trustmanager.KeyStore
	var privKey data.PrivateKey
	var keyInfo trustmanager.KeyInfo
	var cs *cryptoservice.CryptoService
	for _, keyStore := range ks {
		cs = cryptoservice.NewCryptoService(keyStore)
		if privKey, _, err = cs.GetPrivateKey(keyID); err == nil {
			foundKeyStore = keyStore
			break
		}
	}
	if foundKeyStore == nil {
		return fmt.Errorf("could not retrieve local key for key ID provided: %s", keyID)
	}
	// Must use a different passphrase retriever to avoid caching the
	// unlocking passphrase and reusing that.
	passChangeRetriever := k.getRetriever()
	var addingKeyStore trustmanager.KeyStore
	switch foundKeyStore.Name() {
	case "yubikey":
		addingKeyStore, err = getYubiStore(nil, passChangeRetriever)
		keyInfo = trustmanager.KeyInfo{Role: data.CanonicalRootRole}
	default:
		addingKeyStore, err = trustmanager.NewKeyFileStore(config.GetString("trust_dir"), passChangeRetriever)
		if err != nil {
			return err
		}
		keyInfo, err = foundKeyStore.GetKeyInfo(keyID)
	}
	if err != nil {
		return err
	}
	err = addingKeyStore.AddKey(keyInfo, privKey)
	if err != nil {
		return err
	}
	cmd.Printf("\nSuccessfully updated passphrase for key ID: %s\n", keyID)
	return nil
}

func (k *keyCommander) importKeys(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		cmd.Usage()
		return fmt.Errorf("must specify at least one input file to import keys from")
	}
	config, err := k.configGetter()
	if err != nil {
		return err
	}

	directory := config.GetString("trust_dir")
	importers, err := getImporters(directory, k.getRetriever())
	if err != nil {
		return err
	}
	for _, file := range args {
		from, err := os.Open(file)
		if err != nil {
			return err
		}
		defer func() {
			_ = from.Close()
		}()
		if err = trustmanager.ImportKeys(from, importers, k.importRole, k.keysImportGUN, k.getRetriever()); err != nil {
			return err
		}
	}
	return nil
}

func (k *keyCommander) exportKeys(cmd *cobra.Command, args []string) error {
	var (
		out io.Writer
		err error
	)
	if len(args) > 0 {
		cmd.Usage()
		return fmt.Errorf("export does not take any positional arguments")
	}
	config, err := k.configGetter()
	if err != nil {
		return err
	}

	if k.outFile == "" {
		out = cmd.OutOrStdout()
	} else {
		f, err := os.OpenFile(k.outFile, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, notary.PrivNoExecPerms)
		if err != nil {
			return err
		}
		defer func() {
			_ = f.Close()
		}()
		out = f
	}

	directory := config.GetString("trust_dir")
	fileStore, err := store.NewPrivateKeyFileStorage(directory, notary.KeyExtension)
	if err != nil {
		return err
	}
	if len(k.exportGUNs) > 0 {
		if len(k.exportKeyIDs) > 0 {
			return fmt.Errorf("only the --gun or --key flag may be provided, not a mix of the two flags")
		}
		for _, gun := range k.exportGUNs {
			return trustmanager.ExportKeysByGUN(out, fileStore, gun)
		}
	} else if len(k.exportKeyIDs) > 0 {
		return trustmanager.ExportKeysByID(out, fileStore, k.exportKeyIDs)
	}
	// export everything
	keys := fileStore.ListFiles()
	for _, k := range keys {
		err := trustmanager.ExportKeys(out, fileStore, k)
		if err != nil {
			return err
		}
	}
	return nil
}

func (k *keyCommander) getKeyStores(
	config *viper.Viper, withHardware, hardwareBackup bool) ([]trustmanager.KeyStore, error) {

	retriever := k.getRetriever()

	directory := config.GetString("trust_dir")
	fileKeyStore, err := trustmanager.NewKeyFileStore(directory, retriever)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create private key store in directory: %s", directory)
	}

	ks := []trustmanager.KeyStore{fileKeyStore}

	if withHardware {
		var yubiStore trustmanager.KeyStore
		if hardwareBackup {
			yubiStore, err = getYubiStore(fileKeyStore, retriever)
		} else {
			yubiStore, err = getYubiStore(nil, retriever)
		}
		if err == nil && yubiStore != nil {
			// Note that the order is important, since we want to prioritize
			// the yubikey store
			ks = []trustmanager.KeyStore{yubiStore, fileKeyStore}
		}
	}

	return ks, nil
}
