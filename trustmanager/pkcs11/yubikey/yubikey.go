// +build pkcs11

package yubikey

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"errors"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/miekg/pkcs11"
	"github.com/sirupsen/logrus"
	"github.com/theupdateframework/notary"
	"github.com/theupdateframework/notary/trustmanager/pkcs11/common"
	"github.com/theupdateframework/notary/tuf/data"
	"github.com/theupdateframework/notary/tuf/utils"
)

const (
	name = "yubikey"
	// UserPin is the user pin of a yubikey (in PIV parlance, is the PIN)
	UserPin = "123456"
	// SOUserPin is the "Security Officer" user pin - this is the PIV management
	// (MGM) key, which is different than the admin pin of the Yubikey PGP interface
	// (which in PIV parlance is the PUK, and defaults to 12345678)
	SOUserPin = "010203040506070801020304050607080102030405060708"
	numSlots  = 4 // number of slots in the yubikey

	// KeymodeNone means that no touch or PIN is required to sign with the yubikey
	KeymodeNone = 0
	// KeymodeTouch means that only touch is required to sign with the yubikey
	KeymodeTouch = 1
	// KeymodePinOnce means that the pin entry is required once the first time to sign with the yubikey
	KeymodePinOnce = 2
	// KeymodePinAlways means that pin entry is required every time to sign with the yubikey
	KeymodePinAlways = 4
)

// what key mode to use when generating keys
var (
	yubikeyKeymode = KeymodeTouch | KeymodePinOnce
	// order in which to prefer token locations on the yubikey.
	// corresponds to: 9c, 9e, 9d, 9a
	slotIDs = []int{2, 1, 3, 0}
)

// SetYubikeyKeyMode - sets the mode when generating yubikey keys.
// This is to be used for testing.  It does nothing if not building with tag
// pkcs11.
func SetYubikeyKeyMode(keyMode int) error {
	// technically 7 (1 | 2 | 4) is valid, but KeymodePinOnce +
	// KeymdoePinAlways don't really make sense together
	if keyMode < 0 || keyMode > 5 {
		return errors.New("Invalid key mode")
	}
	yubikeyKeymode = keyMode
	return nil
}

// SetTouchToSignUI - allows configurable UX for notifying a user that they
// need to touch the yubikey to sign. The callback may be used to provide a
// mechanism for updating a GUI (such as removing a modal) after the touch
// has been made
func SetTouchToSignUI(notifier func(), callback func()) {
	touchToSignUI = notifier
	if callback != nil {
		touchDoneCallback = callback
	}
}

var touchToSignUI = func() {
	fmt.Println("Please touch the attached Yubikey to perform signing.")
}

var touchDoneCallback = func() {
	// noop
}
var pkcs11Lib string

type keyStore struct {
}

func NewKeyStore() *keyStore {
	if possiblePkcs11Libs != nil {
		for _, loc := range possiblePkcs11Libs {
			_, err := os.Stat(loc)
			if err == nil {
				p := pkcs11.New(loc)
				if p != nil {
					pkcs11Lib = loc
				}
			}
		}
	}
	return &keyStore{}
}

func (ks *keyStore) Name() string {
	return name
}

// addECDSAKey adds a key to the yubikey
func (ks *keyStore) AddECDSAKey(
	ctx common.IPKCS11Ctx,
	session pkcs11.SessionHandle,
	privKey data.PrivateKey,
	hwslot common.HardwareSlot,
	passRetriever notary.PassRetriever,
	role data.RoleName,
) error {
	logrus.Debugf("Attempting to add key to yubikey with ID: %s", privKey.ID())

	err := common.Login(ctx, session, passRetriever, pkcs11.CKU_SO, SOUserPin, name)
	if err != nil {
		return err
	}
	defer ctx.Logout(session)

	// Create an ecdsa.PrivateKey out of the private key bytes
	ecdsaPrivKey, err := x509.ParseECPrivateKey(privKey.Private())
	if err != nil {
		return err
	}

	ecdsaPrivKeyD := common.EnsurePrivateKeySize(ecdsaPrivKey.D.Bytes())

	// Hard-coded policy: the generated certificate expires in 10 years.
	startTime := time.Now()
	template, err := utils.NewCertificate(role.String(), startTime, startTime.AddDate(10, 0, 0))
	if err != nil {
		return fmt.Errorf("failed to create the certificate template: %v", err)
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, template, template, ecdsaPrivKey.Public(), ecdsaPrivKey)
	if err != nil {
		return fmt.Errorf("failed to create the certificate: %v", err)
	}

	certTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_CERTIFICATE),
		pkcs11.NewAttribute(pkcs11.CKA_VALUE, certBytes),
		pkcs11.NewAttribute(pkcs11.CKA_ID, hwslot.SlotID),
	}

	privateKeyTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PRIVATE_KEY),
		pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_ECDSA),
		pkcs11.NewAttribute(pkcs11.CKA_ID, hwslot.SlotID),
		pkcs11.NewAttribute(pkcs11.CKA_EC_PARAMS, []byte{0x06, 0x08, 0x2a, 0x86, 0x48, 0xce, 0x3d, 0x03, 0x01, 0x07}),
		pkcs11.NewAttribute(pkcs11.CKA_VALUE, ecdsaPrivKeyD),
		pkcs11.NewAttribute(pkcs11.CKA_VENDOR_DEFINED, yubikeyKeymode),
	}

	_, err = ctx.CreateObject(session, certTemplate)
	if err != nil {
		return fmt.Errorf("error importing: %v", err)
	}

	_, err = ctx.CreateObject(session, privateKeyTemplate)
	if err != nil {
		return fmt.Errorf("error importing: %v", err)
	}

	return nil
}

func (ks *keyStore) GetECDSAKey(ctx common.IPKCS11Ctx, session pkcs11.SessionHandle, hwslot common.HardwareSlot, _ notary.PassRetriever) (*data.ECDSAPublicKey, data.RoleName, error) {
	findTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),
		pkcs11.NewAttribute(pkcs11.CKA_ID, hwslot.SlotID),
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PUBLIC_KEY),
	}

	attrTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, []byte{0}),
		pkcs11.NewAttribute(pkcs11.CKA_EC_POINT, []byte{0}),
		pkcs11.NewAttribute(pkcs11.CKA_EC_PARAMS, []byte{0}),
	}

	if err := ctx.FindObjectsInit(session, findTemplate); err != nil {
		logrus.Debugf("Failed to init: %s", err.Error())
		return nil, "", err
	}
	obj, _, err := ctx.FindObjects(session, 1)
	if err != nil {
		logrus.Debugf("Failed to find objects: %v", err)
		return nil, "", err
	}
	if err := ctx.FindObjectsFinal(session); err != nil {
		logrus.Debugf("Failed to finalize: %s", err.Error())
		return nil, "", err
	}
	if len(obj) != 1 {
		logrus.Debugf("should have found one object")
		return nil, "", errors.New("no matching keys found inside of yubikey")
	}

	// Retrieve the public-key material to be able to create a new ECSAKey
	attr, err := ctx.GetAttributeValue(session, obj[0], attrTemplate)
	if err != nil {
		logrus.Debugf("Failed to get Attribute for: %v", obj[0])
		return nil, "", err
	}

	// Iterate through all the attributes of this key and saves CKA_PUBLIC_EXPONENT and CKA_MODULUS. Removes ordering specific issues.
	var rawPubKey []byte
	for _, a := range attr {
		if a.Type == pkcs11.CKA_EC_POINT {
			rawPubKey = a.Value
		}

	}

	ecdsaPubKey := ecdsa.PublicKey{Curve: elliptic.P256(), X: new(big.Int).SetBytes(rawPubKey[3:35]), Y: new(big.Int).SetBytes(rawPubKey[35:])}
	pubBytes, err := x509.MarshalPKIXPublicKey(&ecdsaPubKey)
	if err != nil {
		logrus.Debugf("Failed to Marshal public key")
		return nil, "", err
	}

	return data.NewECDSAPublicKey(pubBytes), data.CanonicalRootRole, nil
}

// sign returns a signature for a given signature request
func (ks *keyStore) Sign(ctx common.IPKCS11Ctx, session pkcs11.SessionHandle, hwslot common.HardwareSlot, passRetriever notary.PassRetriever, payload []byte) ([]byte, error) {
	err := common.Login(ctx, session, passRetriever, pkcs11.CKU_USER, UserPin, name)
	if err != nil {
		return nil, fmt.Errorf("error logging in: %v", err)
	}
	defer ctx.Logout(session)

	// Define the ECDSA Private key template
	class := pkcs11.CKO_PRIVATE_KEY
	privateKeyTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, class),
		pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_ECDSA),
		pkcs11.NewAttribute(pkcs11.CKA_ID, hwslot.SlotID),
	}

	if err := ctx.FindObjectsInit(session, privateKeyTemplate); err != nil {
		logrus.Debugf("Failed to init find objects: %s", err.Error())
		return nil, err
	}
	obj, _, err := ctx.FindObjects(session, 1)
	if err != nil {
		logrus.Debugf("Failed to find objects: %v", err)
		return nil, err
	}
	if err = ctx.FindObjectsFinal(session); err != nil {
		logrus.Debugf("Failed to finalize find objects: %s", err.Error())
		return nil, err
	}
	if len(obj) != 1 {
		return nil, errors.New("length of objects found not 1")
	}

	var sig []byte
	err = ctx.SignInit(
		session, []*pkcs11.Mechanism{pkcs11.NewMechanism(pkcs11.CKM_ECDSA, nil)}, obj[0])
	if err != nil {
		return nil, err
	}

	// Get the SHA256 of the payload
	digest := sha256.Sum256(payload)

	if (yubikeyKeymode & KeymodeTouch) > 0 {
		touchToSignUI()
		defer touchDoneCallback()
	}
	// a call to Sign, whether or not Sign fails, will clear the SignInit
	sig, err = ctx.Sign(session, digest[:])
	if err != nil {
		logrus.Debugf("Error while signing: %s", err)
		return nil, err
	}

	if sig == nil {
		return nil, errors.New("Failed to create signature")
	}
	return sig[:], nil
}

func (ks *keyStore) HardwareRemoveKey(ctx common.IPKCS11Ctx, session pkcs11.SessionHandle, hwslot common.HardwareSlot, passRetriever notary.PassRetriever, keyID string) error {
	err := common.Login(ctx, session, passRetriever, pkcs11.CKU_SO, SOUserPin, name)
	if err != nil {
		return err
	}
	defer ctx.Logout(session)

	template := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),
		pkcs11.NewAttribute(pkcs11.CKA_ID, hwslot.SlotID),
		//pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PRIVATE_KEY),
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_CERTIFICATE),
	}

	if err := ctx.FindObjectsInit(session, template); err != nil {
		logrus.Debugf("Failed to init find objects: %s", err.Error())
		return err
	}
	obj, b, err := ctx.FindObjects(session, 1)
	if err != nil {
		logrus.Debugf("Failed to find objects: %s %v", err.Error(), b)
		return err
	}
	if err := ctx.FindObjectsFinal(session); err != nil {
		logrus.Debugf("Failed to finalize find objects: %s", err.Error())
		return err
	}
	if len(obj) != 1 {
		logrus.Debugf("should have found exactly one object")
		return err
	}

	// Delete the certificate
	err = ctx.DestroyObject(session, obj[0])
	if err != nil {
		logrus.Debugf("Failed to delete cert")
		return err
	}
	return nil
}

func (ks *keyStore) HardwareListKeys(ctx common.IPKCS11Ctx, session pkcs11.SessionHandle) (keys map[string]common.HardwareSlot, err error) {
	keys = make(map[string]common.HardwareSlot)

	attrTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_ID, []byte{0}),
		pkcs11.NewAttribute(pkcs11.CKA_VALUE, []byte{0}),
	}

	objs, err := ks.listObjects(ctx, session)
	if err != nil {
		return nil, err
	}

	if len(objs) == 0 {
		return nil, errors.New("no keys found in yubikey")
	}
	logrus.Debugf("Found %d objects matching list filters", len(objs))
	for _, obj := range objs {
		var (
			cert *x509.Certificate
			slot []byte
		)
		// Retrieve the public-key material to be able to create a new ECDSA
		attr, err := ctx.GetAttributeValue(session, obj, attrTemplate)
		if err != nil {
			logrus.Debugf("Failed to get Attribute for: %v", obj)
			continue
		}

		// Iterate through all the attributes of this key and saves CKA_PUBLIC_EXPONENT and CKA_MODULUS. Removes ordering specific issues.
		for _, a := range attr {
			if a.Type == pkcs11.CKA_ID {
				slot = a.Value
			}
			if a.Type == pkcs11.CKA_VALUE {
				cert, err = x509.ParseCertificate(a.Value)
				if err != nil {
					continue
				}
				if !data.ValidRole(data.RoleName(cert.Subject.CommonName)) {
					continue
				}
			}
		}

		// we found nothing
		if cert == nil {
			continue
		}

		var ecdsaPubKey *ecdsa.PublicKey
		switch cert.PublicKeyAlgorithm {
		case x509.ECDSA:
			ecdsaPubKey = cert.PublicKey.(*ecdsa.PublicKey)
		default:
			logrus.Infof("Unsupported x509 PublicKeyAlgorithm: %d", cert.PublicKeyAlgorithm)
			continue
		}

		pubBytes, err := x509.MarshalPKIXPublicKey(ecdsaPubKey)
		if err != nil {
			logrus.Debugf("Failed to Marshal public key")
			continue
		}

		keys[data.NewECDSAPublicKey(pubBytes).ID()] = common.HardwareSlot{
			Role:   data.RoleName(cert.Subject.CommonName),
			SlotID: slot,
		}
	}
	return
}

func (ks *keyStore) listObjects(ctx common.IPKCS11Ctx, session pkcs11.SessionHandle) ([]pkcs11.ObjectHandle, error) {
	findTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_CERTIFICATE),
	}

	if err := ctx.FindObjectsInit(session, findTemplate); err != nil {
		logrus.Debugf("Failed to init: %s", err.Error())
		return nil, err
	}

	objs, b, err := ctx.FindObjects(session, numSlots)
	for err == nil {
		var o []pkcs11.ObjectHandle
		o, b, err = ctx.FindObjects(session, numSlots)
		if err != nil {
			continue
		}
		if len(o) == 0 {
			break
		}
		objs = append(objs, o...)
	}
	if err != nil {
		logrus.Debugf("Failed to find: %s %v", err.Error(), b)
		if len(objs) == 0 {
			return nil, err
		}
	}
	if err := ctx.FindObjectsFinal(session); err != nil {
		logrus.Debugf("Failed to finalize: %s", err.Error())
		return nil, err
	}
	return objs, nil
}

func (ks *keyStore) GetNextEmptySlot(ctx common.IPKCS11Ctx, session pkcs11.SessionHandle) ([]byte, error) {
	findTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),
	}
	attrTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_ID, []byte{0}),
	}

	if err := ctx.FindObjectsInit(session, findTemplate); err != nil {
		logrus.Debugf("Failed to init: %s", err.Error())
		return nil, err
	}
	objs, b, err := ctx.FindObjects(session, numSlots)
	// if there are more objects than `numSlots`, get all of them until
	// there are no more to get
	for err == nil {
		var o []pkcs11.ObjectHandle
		o, b, err = ctx.FindObjects(session, numSlots)
		if err != nil {
			continue
		}
		if len(o) == 0 {
			break
		}
		objs = append(objs, o...)
	}
	taken := make(map[int]bool)
	if err != nil {
		logrus.Debugf("Failed to find: %s %v", err.Error(), b)
		return nil, err
	}
	if err = ctx.FindObjectsFinal(session); err != nil {
		logrus.Debugf("Failed to finalize: %s\n", err.Error())
		return nil, err
	}
	for _, obj := range objs {
		// Retrieve the slot ID
		attr, err := ctx.GetAttributeValue(session, obj, attrTemplate)
		if err != nil {
			continue
		}

		// Iterate through attributes. If an ID attr was found, mark it as taken
		for _, a := range attr {
			if a.Type == pkcs11.CKA_ID {
				if len(a.Value) < 1 {
					continue
				}
				// a byte will always be capable of representing all slot IDs
				// for the Yubikeys
				slotNum := int(a.Value[0])
				if slotNum >= numSlots {
					// defensive
					continue
				}
				taken[slotNum] = true
			}
		}
	}
	// iterate the token locations in our preferred order and use the first
	// available one. Otherwise exit the loop and return an error.
	for _, loc := range slotIDs {
		if !taken[loc] {
			return []byte{byte(loc)}, nil
		}
	}
	return nil, errors.New("yubikey has no available slots")
}

// SetupHSMEnv is a method that depends on the existences

func (ks *keyStore) SetupHSMEnv(libLoader common.Pkcs11LibLoader) (
	common.IPKCS11Ctx, pkcs11.SessionHandle, error) {

	if pkcs11Lib == "" {
		return nil, 0, common.ErrHSMNotPresent{Err: "no library found"}
	}
	p := libLoader(pkcs11Lib)

	if p == nil {
		return nil, 0, fmt.Errorf("failed to load library %s", pkcs11Lib)
	}

	if err := p.Initialize(); err != nil {
		defer common.FinalizeAndDestroy(p)
		return nil, 0, fmt.Errorf("found library %s, but initialize error %s", pkcs11Lib, err.Error())
	}

	slots, err := p.GetSlotList(true)
	if err != nil {
		defer common.FinalizeAndDestroy(p)
		return nil, 0, fmt.Errorf(
			"loaded library %s, but failed to list HSM slots %s", pkcs11Lib, err)
	}
	// Check to see if we got any slots from the HSM.
	if len(slots) < 1 {
		defer common.FinalizeAndDestroy(p)
		return nil, 0, fmt.Errorf(
			"loaded library %s, but no HSM slots found", pkcs11Lib)
	}

	// CKF_SERIAL_SESSION: TRUE if cryptographic functions are performed in serial with the application; FALSE if the functions may be performed in parallel with the application.
	// CKF_RW_SESSION: TRUE if the session is read/write; FALSE if the session is read-only
	session, err := p.OpenSession(slots[0], pkcs11.CKF_SERIAL_SESSION|pkcs11.CKF_RW_SESSION)
	if err != nil {
		defer common.Cleanup(p, session)
		return nil, 0, fmt.Errorf(
			"loaded library %s, but failed to start session with HSM %s",
			pkcs11Lib, err)
	}

	logrus.Debugf("Initialized PKCS11 library %s and started HSM session", pkcs11Lib)
	return p, session, nil
}
