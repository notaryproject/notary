// +build pkcs11

package opencryptoki

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/miekg/pkcs11"
	"github.com/sirupsen/logrus"
	"github.com/theupdateframework/notary"
	"github.com/theupdateframework/notary/trustmanager/pkcs11/common"
	"github.com/theupdateframework/notary/tuf/data"
	"github.com/theupdateframework/notary/tuf/utils"
)

var (
	tokenSlot uint
	pkcs11Lib string
	UserPin   string
)

const (
	name          = "openCryptoki"
	numSlots      = 999
	ock_req_major = 3
	ock_req_minor = 8
)

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

// SetSlot is used for testing, not to use interactive prompts
func SetSlot(slot uint) {
	tokenSlot = slot
}

// SetPin is used for testing, not to use interactive prompts
func SetPin(pw string) {
	UserPin = pw
}

// AddECDSAKey adds a key to the opencryptoki store
func (ks *keyStore) AddECDSAKey(ctx common.IPKCS11Ctx, session pkcs11.SessionHandle, privKey data.PrivateKey, hwslot common.HardwareSlot, passRetriever notary.PassRetriever, role data.RoleName) error {
	logrus.Debugf("Attempting to add key to %s with ID: %s", name, privKey.ID())
	err := common.Login(ctx, session, passRetriever, pkcs11.CKU_USER, UserPin, fmt.Sprintf("%s UserPin", name))
	if err != nil {
		return err
	}
	defer ctx.Logout(session)
	ecdsaPrivKey, err := x509.ParseECPrivateKey(privKey.Private())
	if err != nil {
		return err
	}

	ecdsaPrivKeyD := common.EnsurePrivateKeySize(ecdsaPrivKey.D.Bytes())

	startTime := time.Now()

	template, err := utils.NewCertificate(role.String(), startTime, startTime.AddDate(data.DefaultExpires(data.CanonicalRootRole).Year(), 0, 0))
	if err != nil {
		return fmt.Errorf("failed to create the certificate template: %v", err)
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, template, template, ecdsaPrivKey.Public(), ecdsaPrivKey)
	ecdsaPrivKey = nil
	if err != nil {
		return fmt.Errorf("failed to create the certificate: %v", err)
	}
	certTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, "Notary Certificate"),
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_CERTIFICATE),
		pkcs11.NewAttribute(pkcs11.CKA_CERTIFICATE_TYPE, pkcs11.CKC_X_509),
		pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),
		pkcs11.NewAttribute(pkcs11.CKA_VALUE, certBytes),
		pkcs11.NewAttribute(pkcs11.CKA_SUBJECT, template.SubjectKeyId),
		pkcs11.NewAttribute(pkcs11.CKA_ID, hwslot.KeyID),
	}

	privateKeyTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, "Notary Private Key"),
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PRIVATE_KEY),
		pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_ECDSA),
		pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),
		pkcs11.NewAttribute(pkcs11.CKA_PRIVATE, true),
		pkcs11.NewAttribute(pkcs11.CKA_ID, hwslot.KeyID),
		pkcs11.NewAttribute(pkcs11.CKA_EC_PARAMS, []byte{0x06, 0x08, 0x2a, 0x86, 0x48, 0xce, 0x3d, 0x03, 0x01, 0x07}),
		pkcs11.NewAttribute(pkcs11.CKA_VALUE, ecdsaPrivKeyD),
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

//GetECDSAKey gets a key by id from the opencryptoki store
func (ks *keyStore) GetECDSAKey(ctx common.IPKCS11Ctx, session pkcs11.SessionHandle, hwslot common.HardwareSlot, passRetriever notary.PassRetriever) (*data.ECDSAPublicKey, data.RoleName, error) {
	err := common.Login(ctx, session, passRetriever, pkcs11.CKU_USER, UserPin, fmt.Sprintf("%s UserPin", name))
	if err != nil {
		return nil, "", err
	}
	defer ctx.Logout(session)
	findTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_CERTIFICATE),
		pkcs11.NewAttribute(pkcs11.CKA_CERTIFICATE_TYPE, pkcs11.CKC_X_509),
		pkcs11.NewAttribute(pkcs11.CKA_ID, hwslot.KeyID),
	}
	pubTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_VALUE, []byte{0}),
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
		return nil, "", errors.New(fmt.Sprintf("no matching keys found inside of %s", name))
	}
	val, err := ctx.GetAttributeValue(session, obj[0], pubTemplate)
	if err != nil {
		logrus.Debugf("Failed to get Certificate for: %v", obj[0])
		return nil, "", err
	}
	cert, err := x509.ParseCertificate(val[0].Value)
	pub := cert.PublicKey
	if err != nil {
		logrus.Debugf("Failed to parse Certificate for: %v", obj[0])
		return nil, "", err
	}
	attr := pub.(*ecdsa.PublicKey)
	ecdsaPubKey := ecdsa.PublicKey{Curve: elliptic.P256(), X: attr.X, Y: attr.Y}
	pubBytes, err := x509.MarshalPKIXPublicKey(&ecdsaPubKey)
	if err != nil {
		logrus.Debugf("Failed to Marshal public key")
		return nil, "", err
	}

	return data.NewECDSAPublicKey(pubBytes), data.CanonicalRootRole, nil
}

// Sign signs the payload with the key of the given ID
func (ks *keyStore) Sign(ctx common.IPKCS11Ctx, session pkcs11.SessionHandle, hwslot common.HardwareSlot, passRetriever notary.PassRetriever, payload []byte) ([]byte, error) {
	err := common.Login(ctx, session, passRetriever, pkcs11.CKU_USER, UserPin, fmt.Sprintf("%s UserPin", name))
	if err != nil {
		return nil, fmt.Errorf("error logging in: %v", err)
	}
	defer ctx.Logout(session)

	privateKeyTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_PRIVATE, true),
		pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PRIVATE_KEY),
		pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_ECDSA),
		pkcs11.NewAttribute(pkcs11.CKA_ID, hwslot.KeyID),
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
		return nil, errors.New(fmt.Sprintf("should have found exactly one private key, found ", len(obj)))
	}

	var sig []byte
	err = ctx.SignInit(session, []*pkcs11.Mechanism{pkcs11.NewMechanism(pkcs11.CKM_ECDSA, nil)}, obj[0])
	if err != nil {
		return nil, err
	}

	digest := sha256.Sum256(payload)
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

// HardwareRemoveKey removes the Key with a specified ID from the opencryptoki store
func (ks *keyStore) HardwareRemoveKey(ctx common.IPKCS11Ctx, session pkcs11.SessionHandle, hwslot common.HardwareSlot, passRetriever notary.PassRetriever, keyID string) error {
	err := common.Login(ctx, session, passRetriever, pkcs11.CKU_USER, UserPin, fmt.Sprintf("%s UserPin", name))
	if err != nil {
		return err
	}
	defer ctx.Logout(session)
	certTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),
		pkcs11.NewAttribute(pkcs11.CKA_ID, hwslot.KeyID),
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_CERTIFICATE),
	}

	keyTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_PRIVATE, true),
		pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_ECDSA),
		pkcs11.NewAttribute(pkcs11.CKA_ID, hwslot.KeyID),
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PRIVATE_KEY),
	}
	templates := [][]*pkcs11.Attribute{certTemplate, keyTemplate}
	for _, template := range templates {
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

		err = ctx.DestroyObject(session, obj[0])
		if err != nil {
			logrus.Debugf("Failed to delete cert/privkey")
			return err
		}

	}

	return nil
}

//HardwareListKeys lists all available Keys stored by opencryptoki
func (ks *keyStore) HardwareListKeys(ctx common.IPKCS11Ctx, session pkcs11.SessionHandle) (map[string]common.HardwareSlot, error) {
	keys := make(map[string]common.HardwareSlot)

	attrTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_ID, []byte{0}),
		pkcs11.NewAttribute(pkcs11.CKA_VALUE, []byte{0}),
	}

	objs, err := ks.listObjects(ctx, session)
	if err != nil {
		return nil, err
	}

	if len(objs) == 0 {
		return nil, errors.New(fmt.Sprintf("no keys found in %s", name))
	}
	logrus.Debugf("Found %d objects matching list filters", len(objs))
	for _, obj := range objs {
		var (
			cert *x509.Certificate
			slot []byte
		)
		attr, err := ctx.GetAttributeValue(session, obj, attrTemplate)
		if err != nil {
			logrus.Debugf("Failed to get Attribute for: %v", obj)
			continue
		}

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
		id := data.NewECDSAPublicKey(pubBytes).ID()
		keys[id] = common.HardwareSlot{
			Role:   data.RoleName(cert.Subject.CommonName),
			SlotID: slot,
			KeyID:  id,
		}
	}
	return keys, err
}

func (ks *keyStore) listObjects(ctx common.IPKCS11Ctx, session pkcs11.SessionHandle) ([]pkcs11.ObjectHandle, error) {
	findTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_CERTIFICATE),
		pkcs11.NewAttribute(pkcs11.CKA_CERTIFICATE_TYPE, pkcs11.CKC_X_509),
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

//GetNextEmptySlot returns the first empty slot found by opencryptoki to store a key
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
		attr, err := ctx.GetAttributeValue(session, obj, attrTemplate)
		if err != nil {
			continue
		}

		for _, a := range attr {
			if a.Type == pkcs11.CKA_ID {
				if len(a.Value) < 1 {
					continue
				}
				slotNum := int(a.Value[0])
				if slotNum >= numSlots {
					continue
				}
				taken[slotNum] = true
			}
		}
	}
	for loc := 0; loc < numSlots; loc++ {
		if !taken[loc] {
			return []byte{byte(loc)}, nil
		}
	}
	return nil, errors.New("Crypto Express has no available slots")
}

//SetupHSMEnv is responsible for opening the HSM session and performing some checks before (lib available, right version, mechanism available, etc)
func (ks *keyStore) SetupHSMEnv(libLoader common.Pkcs11LibLoader) (common.IPKCS11Ctx, pkcs11.SessionHandle, error) {
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
	info, _ := p.GetInfo()
	if (info.LibraryVersion.Major >= ock_req_major && info.LibraryVersion.Minor >= ock_req_minor) == false {
		defer common.FinalizeAndDestroy(p)
		return nil, 0, fmt.Errorf("found library %s, but OpenCryptoki Version to low (3.8 Required)", pkcs11Lib)
	}
	slots, err := p.GetSlotList(true)
	if err != nil {
		defer common.FinalizeAndDestroy(p)
		return nil, 0, fmt.Errorf(
			"loaded library %s, but failed to list HSM slots %s", pkcs11Lib, err)
	}
	if len(slots) < 1 {
		defer common.FinalizeAndDestroy(p)
		return nil, 0, fmt.Errorf(
			"loaded library %s, but no HSM slots found", pkcs11Lib)
	}
	if len(slots) == 1 {
		tokenSlot = slots[0]
	}
	if tokenSlot == 0 {
		var text string
		prettyPrintTokens(slots, os.Stdout, p)
		fmt.Printf("Enter %s Token Slot to use: ", name)
		fmt.Scanln(&text)
		parsedInt, _ := strconv.Atoi(text)
		tokenSlot = uint(parsedInt)
	}
	err = hasECDSAMechanism(p, slots)
	if err != nil {
		defer common.FinalizeAndDestroy(p)
		return nil, 0, fmt.Errorf("found library %s, but %s", pkcs11Lib, err)
	}

	if tokenSlot != 0 && !contains(tokenSlot, slots) {
		defer common.FinalizeAndDestroy(p)
		return nil, 0, fmt.Errorf("Slot %d not Available. Available Slots: %d", tokenSlot, slots)
	}
	session, err := p.OpenSession(tokenSlot, pkcs11.CKF_RW_SESSION)
	if err != nil {
		defer common.Cleanup(p, session)
		return nil, 0, fmt.Errorf(
			"loaded library %s, but failed to start session with HSM %s",
			pkcs11Lib, err)
	}

	logrus.Debugf("Initialized PKCS11 library %s and started HSM session", pkcs11Lib)
	return p, session, nil
}

func contains(element uint, array []uint) bool {
	for i := 0; i < len(array); i++ {
		if element == array[i] {
			return true
		}
	}
	return false
}
func hasECDSAMechanism(p common.IPKCS11Ctx, slots []uint) error {
	for _, slot := range slots {
		mechanisms, _ := p.GetMechanismList(slot)
		for _, mechanism := range mechanisms {
			if mechanism.Mechanism == pkcs11.CKM_ECDSA {
				return nil
			}
		}
	}
	return nil
	return errors.New("available Tokens do not support ECDSA Mechanism")
}

func prettyPrintTokens(slots []uint, writer io.Writer, p common.IPKCS11Ctx) {
	fmt.Println("Available Tokens:")
	tw := initTabWriter([]string{"SLOT", "MODEL", "LABEL", "FLAGS"}, writer)

	for _, slot := range slots {
		info, _ := p.GetTokenInfo(uint(slot))
		fmt.Fprintf(
			tw,
			"%d\t%s\t%s\t%d\n",
			slot,
			info.Model,
			info.Label,
			info.Flags,
		)
	}
	tw.Flush()
}
func initTabWriter(columns []string, writer io.Writer) *tabwriter.Writer {
	tw := tabwriter.NewWriter(writer, 4, 4, 4, ' ', 0)
	fmt.Fprintln(tw, strings.Join(columns, "\t"))
	breakLine := make([]string, 0, len(columns))
	for _, h := range columns {
		breakLine = append(
			breakLine,
			strings.Repeat("-", len(h)),
		)
	}
	fmt.Fprintln(tw, strings.Join(breakLine, "\t"))
	return tw
}
