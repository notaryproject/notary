package p11store

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/miekg/pkcs11"
	"github.com/sirupsen/logrus"
	"github.com/theupdateframework/notary"
	"github.com/theupdateframework/notary/trustmanager"
	"github.com/theupdateframework/notary/tuf/data"
	"os"
	"strings"
	"sync"
	"time"
)

// Information about a PKCS#11 key that might be available.
// In principle this information may be invalidated during the life
// of the process; currently, things just stop working if this happens.
type pkcs11KeyInfo struct {
	// Slot containing the token in which the key was last found.
	slot uint

	// CKA_ID value for key
	id []byte

	// Notary-specific key information
	keyInfo trustmanager.KeyInfo

	// Public key
	publicKey *ecdsa.PublicKey

	// Public key in Notary format
	tufKey data.TUFKey

	// Notary key ID
	tufId string
}

// Mockable interface to PKCS#11 implementation
type iPKCS11Ctx interface {
	Initialize() error
	GetSlotList(tokenPresent bool) ([]uint, error)
	GetTokenInfo(slotID uint) (pkcs11.TokenInfo, error)
	OpenSession(slotID uint, flags uint) (pkcs11.SessionHandle, error)
	CloseSession(sh pkcs11.SessionHandle) error
	GetSessionInfo(sh pkcs11.SessionHandle) (pkcs11.SessionInfo, error)
	Login(sh pkcs11.SessionHandle, userType uint, pin string) error
	GenerateKeyPair(sh pkcs11.SessionHandle, m []*pkcs11.Mechanism, public, private []*pkcs11.Attribute) (pkcs11.ObjectHandle, pkcs11.ObjectHandle, error)
	DestroyObject(sh pkcs11.SessionHandle, oh pkcs11.ObjectHandle) error
	GetAttributeValue(sh pkcs11.SessionHandle, o pkcs11.ObjectHandle, a []*pkcs11.Attribute) ([]*pkcs11.Attribute, error)
	SetAttributeValue(sh pkcs11.SessionHandle, o pkcs11.ObjectHandle, a []*pkcs11.Attribute) error
	FindObjectsInit(sh pkcs11.SessionHandle, temp []*pkcs11.Attribute) error
	FindObjects(sh pkcs11.SessionHandle, max int) ([]pkcs11.ObjectHandle, bool, error)
	FindObjectsFinal(sh pkcs11.SessionHandle) error
	SignInit(sh pkcs11.SessionHandle, m []*pkcs11.Mechanism, o pkcs11.ObjectHandle) error
	Sign(sh pkcs11.SessionHandle, message []byte) ([]byte, error)
}

// A keystore representing a generic PKCS#11 implementation.
type Pkcs11Store struct {
	// PKCS#11 library context
	ctx iPKCS11Ctx

	// Map of key IDs to names
	keyInfos map[string]pkcs11KeyInfo

	// When keyInfos expires
	keyInfoExpires time.Time

	// Map of slots to cached RW sessions
	sessions map[uint]pkcs11.SessionHandle

	// Lock serializing access to this keystore.
	lock sync.Mutex

	// Function to acquire a passphrase
	passRetriever notary.PassRetriever
}

// Errors

// ErrNoProvider indicates no PKCS#11 provider was configured.
var ErrNoProvider = errors.New("no PKCS#11 provider found")

// ErrNotImplemented is returned by unimplemented methods.
var ErrNotImplemented = errors.New("PKCS#11 keystore operation not supported")

// ErrTokenNotFound is return when the token for key generation could not be foud.
var ErrTokenNotFound = errors.New("PKCS#11 token not found")

// Object identifiers

// secp256r1 is the OID for NIST P-256.
var secp256r1 = []byte{0x06, 0x08, 0x2a, 0x86, 0x48, 0xce, 0x3d, 0x03, 0x01, 0x07}

// Implementation of trustmanager.KeyStore methods

func (p *Pkcs11Store) AddKey(keyInfo trustmanager.KeyInfo, privKey data.PrivateKey) (err error) {
	// This key store doesn't support import, only key generation on the HSM.
	return ErrNotImplemented
}

func (p *Pkcs11Store) GetKey(keyID string) (privKey data.PrivateKey, role data.RoleName, err error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	var info pkcs11KeyInfo
	var session pkcs11.SessionHandle
	var privateObject pkcs11.ObjectHandle
	if info, session, _, privateObject, err = p.findKey(keyID); err != nil {
		return
	}
	// Extend data.TUFKey with additional information to allow use through PKCS#11
	// and crypto.Signer.
	privKey = &Pkcs11PrivateKey{
		TUFKey:    info.tufKey,
		Store:     p,
		Session:   session,
		Object:    privateObject,
		PublicKey: info.publicKey,
	}
	role = info.keyInfo.Role
	return
}

func (p *Pkcs11Store) GetKeyInfo(keyID string) (ki trustmanager.KeyInfo, err error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	if err = p.getKeyInfos(); err != nil {
		return
	}
	var ok bool
	var pki pkcs11KeyInfo
	if pki, ok = p.keyInfos[keyID]; !ok {
		err = trustmanager.ErrKeyNotFound{keyID}
		return
	}
	ki = pki.keyInfo
	return
}

func (p *Pkcs11Store) ListKeys() (keyInfos map[string]trustmanager.KeyInfo) {
	p.lock.Lock()
	defer p.lock.Unlock()
	var err error
	if err = p.getKeyInfos(); err != nil {
		// No way to return an error so just log it
		logrus.Errorf("enumerating PKCS#11 keys: %s", err)
		return
	}
	// Extract the information our caller needs
	keyInfos = map[string]trustmanager.KeyInfo{}
	for keyID, pkeyInfo := range p.keyInfos {
		keyInfos[keyID] = pkeyInfo.keyInfo
	}
	return
}

func (p *Pkcs11Store) RemoveKey(keyID string) (err error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	var session pkcs11.SessionHandle
	var publicObject, privateObject pkcs11.ObjectHandle
	if _, session, publicObject, privateObject, err = p.findKey(keyID); err != nil {
		return
	}
	if err = p.ctx.DestroyObject(session, privateObject); err != nil {
		logrus.Errorf("C_DestroyObject: %s", err)
		return
	}
	if err = p.ctx.DestroyObject(session, publicObject); err != nil {
		logrus.Errorf("C_DestroyObject: %s", err)
		return
	}
	delete(p.keyInfos, keyID)
	return
}

func (p *Pkcs11Store) Name() string {
	return "pkcs11"
}

func (p *Pkcs11Store) Generate(keyInfo trustmanager.KeyInfo, token, algorithm string) (keyID string, pubKey data.PublicKey, err error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	// Enforce restrictions on what we support
	if algorithm != data.ECDSAKey {
		err = fmt.Errorf("algorithm '%s' not supported by PKCS#11 key store", algorithm)
		return
	}
	// Find the token to generate the key on
	tokenSlot := ^uint(0)
	if err = p.forEachToken(func(session pkcs11.SessionHandle, slot uint) (err error) {
		var ti pkcs11.TokenInfo
		if ti, err = p.ctx.GetTokenInfo(slot); err != nil {
			logrus.Errorf("C_GetTokenInfo: %s", err)
			return
		}
		if tokenMatches(token, &ti) {
			tokenSlot = slot
		}
		return
	}); err != nil {
		return
	}
	if tokenSlot == ^uint(0) {
		err = ErrTokenNotFound
		return
	}
	if err = p.getKeyInfos(); err != nil {
		return
	}
	// Get a RW session on the required slot
	var session pkcs11.SessionHandle
	if session, err = p.getSession(tokenSlot); err != nil {
		return
	}
	// Generate a random ID for the key pair
	idbytes := make([]byte, 12)
	if _, err = rand.Read(idbytes); err != nil {
		return
	}
	id := base64.RawStdEncoding.EncodeToString(idbytes)
	// Generate the label
	label := fmt.Sprintf("%s:%s", keyInfo.Role, keyInfo.Gun)
	// Key generation parameters
	publicAttributes := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PUBLIC_KEY),
		pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_ECDSA),
		pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),
		pkcs11.NewAttribute(pkcs11.CKA_VERIFY, true),
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, label),
		pkcs11.NewAttribute(pkcs11.CKA_ID, id),
		pkcs11.NewAttribute(pkcs11.CKA_ECDSA_PARAMS, secp256r1),
	}
	privateAttributes := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),
		pkcs11.NewAttribute(pkcs11.CKA_SIGN, true),
		pkcs11.NewAttribute(pkcs11.CKA_SENSITIVE, true),
		pkcs11.NewAttribute(pkcs11.CKA_EXTRACTABLE, false),
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, label),
		pkcs11.NewAttribute(pkcs11.CKA_ID, id),
	}
	mechanism := []*pkcs11.Mechanism{pkcs11.NewMechanism(pkcs11.CKM_ECDSA_KEY_PAIR_GEN, nil)}
	// Generate the key
	var publicObject, privateObject pkcs11.ObjectHandle
	if publicObject, privateObject, err = p.ctx.GenerateKeyPair(session, mechanism, publicAttributes, privateAttributes); err != nil {
		logrus.Errorf("C_GenerateKeyPair: %s", err)
		return
	}
	// If something goes wrong before we finished, clean up the key
	completed := false
	defer func() {
		if !completed {
			p.ctx.DestroyObject(session, publicObject)
			p.ctx.DestroyObject(session, privateObject)
		}
	}()
	// Construct the key information
	var info pkcs11KeyInfo
	if info, err = p.getKeyInfo(session, publicObject); err != nil {
		return
	}
	info.slot = tokenSlot
	keyID = info.tufId
	pubKey = &data.ECDSAPublicKey{info.tufKey}
	// Attempt to set CKA_ID to the TUF id. This isn't completely
	// necessary but makes life easier for anyone trying to
	// understand the situation through PKCS#11-native tooling.
	idAttributes := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_ID, info.tufId),
	}
	if err = p.ctx.SetAttributeValue(session, publicObject, idAttributes); err != nil {
		logrus.Errorf("C_SetAttributeValue for public key: %s", err)
		return
	}
	if err = p.ctx.SetAttributeValue(session, privateObject, idAttributes); err != nil {
		logrus.Errorf("C_SetAttributeValue for private key: %s", err)
		return
	}
	// Keep id field in step with CKA_ID
	info.id = []byte(info.tufId)
	// Save the key information for later
	p.keyInfos[info.tufId] = info
	// Don't destroy the key
	completed = true
	return
}

// Other public functions

// Map of provider paths to context structures
var providers = map[string]iPKCS11Ctx{}
var providerLock sync.Mutex

// NewPkcs11Store creates a new Pkcs11Store using a specific PKCS#11 provider.
// If path=="" then ${NOTARY_HSM_LIB} is used.
func NewPkcs11Store(path string, passRetriever notary.PassRetriever) (p *Pkcs11Store, err error) {
	providerLock.Lock()
	defer providerLock.Unlock()
	if path == "" {
		// Environment variable name chosen by analogy with VAULT_HSM_LIB.
		if path = os.Getenv("NOTARY_HSM_LIB"); path == "" {
			err = ErrNoProvider
			return
		}
	}
	// We cache initialized provider(s), so that it's harmless to call
	// NewPkcs11Store more than once in the same process.
	var ctx iPKCS11Ctx
	var ok bool
	if ctx, ok = providers[path]; !ok {
		// pkcs11 panics if the library doesn't exist, so check that up front.
		if _, err = os.Stat(path);os.IsNotExist(err) {
			return
		}
		if ctx = pkcs11.New(path); ctx == nil {
			err = ErrNoProvider
			return
		}
		if err = ctx.Initialize(); err != nil {
			logrus.Errorf("C_Initialize: %s", err)
			return
		}
		providers[path] = ctx
	}
	p = &Pkcs11Store{
		passRetriever: passRetriever,
		sessions:      map[uint]pkcs11.SessionHandle{},
		keyInfos:      nil,
		ctx:           ctx,
	}
	return
}

// Internal utilies

// forEachToken iterates over all tokens and invokes f with a transient public session.
// If f returns an error then the iteration stops immediately.
func (p *Pkcs11Store) forEachToken(f func(session pkcs11.SessionHandle, slot uint) (err error)) (err error) {
	var slots []uint
	if slots, err = p.ctx.GetSlotList(true); err != nil {
		logrus.Errorf("C_GetSlotList: %s", err)
		return
	}
	for _, slot := range slots {
		var session pkcs11.SessionHandle
		if session, err = p.ctx.OpenSession(slot, pkcs11.CKF_SERIAL_SESSION); err != nil {
			if e, ok := err.(pkcs11.Error); ok && e == pkcs11.CKR_TOKEN_NOT_RECOGNIZED {
				err = nil
				continue
			}
			logrus.Errorf("C_OpenSession slot %v read-only session: %s", slot, err)
			return
		}
		defer p.ctx.CloseSession(session)
		if err = f(session, slot); err != nil {
			return
		}
	}
	return
}

// forEachObject iterates over the objects matching a template and invokes f on each.
// If f returns an error then the iteration stops immediately.
func (p *Pkcs11Store) forEachObject(session pkcs11.SessionHandle, attributes []*pkcs11.Attribute, f func(object pkcs11.ObjectHandle) (err error)) (err error) {
	if err = p.ctx.FindObjectsInit(session, attributes); err != nil {
		logrus.Errorf("C_FindObjectsInit: %s", err)
		return
	}
	defer p.ctx.FindObjectsFinal(session)
	for {
		var objects []pkcs11.ObjectHandle
		var more bool
		if objects, more, err = p.ctx.FindObjects(session, 64); err != nil {
			logrus.Errorf("C_FindObjects: %s", err)
			return
		}
		for _, object := range objects {
			if err = f(object); err != nil {
				return
			}
		}
		if !more {
			break
		}
	}
	return
}

// getKeyInfos initializes p.keyInfos if it is nil or if it is out of date
func (p *Pkcs11Store) getKeyInfos() (err error) {
	if p.keyInfos != nil && time.Now().Before(p.keyInfoExpires) {
		return
	}
	var keyInfos = map[string]pkcs11KeyInfo{}
	// Search template.
	// Currently only ECDSA keys are supported, so we restrict the search to that.
	searchAttributes := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PUBLIC_KEY),
		pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_ECDSA),
	}
	// Search all tokens for usable keys
	if err = p.forEachToken(func(session pkcs11.SessionHandle, slot uint) (err error) {
		err = p.forEachObject(session, searchAttributes, func(object pkcs11.ObjectHandle) (err error) {
			var info pkcs11KeyInfo
			if info, err = p.getKeyInfo(session, object); err != nil {
				err = nil // Ignore unusable keys
				return
			}
			info.slot = slot
			keyInfos[info.tufId] = info
			return
		})
		return
	}); err != nil {
		return
	}
	p.keyInfos = keyInfos
	p.keyInfoExpires = time.Now().Add(time.Second)
	return
}

// getKeyInfo recovers information about a public key that's already on a token.
func (p *Pkcs11Store) getKeyInfo(session pkcs11.SessionHandle, object pkcs11.ObjectHandle) (info pkcs11KeyInfo, err error) {
	// Object attributes that we're interested in.
	keyAttributes := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, 0),
		pkcs11.NewAttribute(pkcs11.CKA_ID, 0),
		pkcs11.NewAttribute(pkcs11.CKA_EC_POINT, 0),
		pkcs11.NewAttribute(pkcs11.CKA_EC_PARAMS, 0),
	}
	// Get the key attributes and translate them into Notary's preferred form
	var attributes []*pkcs11.Attribute
	if attributes, err = p.ctx.GetAttributeValue(session, object, keyAttributes); err != nil {
		logrus.Errorf("C_GetAttributeValue: %s", err)
		return
	}
	// Pick out the attributes we care about.
	// We assume that only the supported key type is supplied.
	var params, point []byte
	for _, attribute := range attributes {
		switch attribute.Type {
		case pkcs11.CKA_LABEL:
			bits := strings.SplitN(string(attribute.Value), ":", 2)
			info.keyInfo.Role = data.RoleName(bits[0])
			info.keyInfo.Gun = data.GUN(bits[1])
		case pkcs11.CKA_ID:
			info.id = attribute.Value
		case pkcs11.CKA_EC_PARAMS:
			params = attribute.Value
		case pkcs11.CKA_EC_POINT:
			point = attribute.Value
		}
	}
	// Skip keys with missing components
	if params == nil || point == nil || info.keyInfo.Role == "" {
		err = errors.New("missing key components")
		return
	}
	// Skip wrong curve
	if bytes.Compare(secp256r1, params) != 0 {
		err = errors.New("wrong ECC domain")
		return
	}
	// Parse the public key. PKCS#11 encodes it twice.
	pointBytes := make([]byte, 0, 65)
	if _, err = asn1.Unmarshal(point, &pointBytes); err != nil {
		return
	}
	curve := elliptic.P256()
	x, y := elliptic.Unmarshal(curve, pointBytes)
	info.publicKey = &ecdsa.PublicKey{curve, x, y}
	var publicKeyBytes []byte
	if publicKeyBytes, err = x509.MarshalPKIXPublicKey(info.publicKey); err != nil {
		return
	}
	info.tufKey = data.NewECDSAPublicKey(publicKeyBytes).TUFKey
	info.tufId = info.tufKey.ID()
	return
}

// tokenMatches returns true if tokenInfo matches the specification in token.
func tokenMatches(token string, tokenInfo *pkcs11.TokenInfo) (matches bool) {
	bits := strings.SplitN(token, ":", 2)
	if len(bits) != 2 {
		return false
	}
	var value string
	key := strings.ToLower(bits[0])
	if key == "label" {
		value = tokenInfo.Label
	} else if key == "serialnumber" {
		value = tokenInfo.SerialNumber
	} else {
		// Give the user a hint about what went wrong
		logrus.Printf("unrecognized PKCS#11 token selector '%s'", token)
		return false
	}
	// pkcs11 already trimmed trailing whitespace so we can directly compare.
	return value == bits[1]
}

// getSession returns a RW session for a slot.
func (p *Pkcs11Store) getSession(slot uint) (session pkcs11.SessionHandle, err error) {
	var ok bool
	// Use the cached session if it exists and is not stale
	if session, ok = p.sessions[slot]; ok {
		if _, err = p.ctx.GetSessionInfo(session); err == nil {
			return
		}
		// Session exists but is stale
		delete(p.sessions, slot)
		if err = p.ctx.CloseSession(session); err != nil {
			logrus.Errorf("C_CloseSession: %v (ignored)", err)
		}
	}
	if session, err = p.ctx.OpenSession(slot, pkcs11.CKF_SERIAL_SESSION|pkcs11.CKF_RW_SESSION); err != nil {
		logrus.Errorf("C_OpenSession slot %v rw session: %s", slot, err)
		return
	}
	closeSession := session
	defer func() {
		if closeSession != 0 {
			p.ctx.CloseSession(closeSession)
		}
	}()
	// Construct a 'keyName' based on the token description.
	// Some tokens return empty strings for this field or that
	// so exclude them.
	var ti pkcs11.TokenInfo
	if ti, err = p.ctx.GetTokenInfo(slot); err != nil {
		logrus.Errorf("C_GetTokenInfo: %s", err)
		return
	}
	tokenBits := make([]string, 0, 4)
	if ti.ManufacturerID != "" {
		tokenBits = append(tokenBits, ti.ManufacturerID)
	}
	if ti.Model != "" {
		tokenBits = append(tokenBits, ti.Model)
	}
	if ti.Label != "" {
		tokenBits = append(tokenBits, ti.Label)
	}
	if ti.SerialNumber != "" {
		tokenBits = append(tokenBits, fmt.Sprintf("(%s)", ti.SerialNumber))
	}
	tokenName := strings.Join(tokenBits, " ")
	if ti.Flags & pkcs11.CKF_LOGIN_REQUIRED != 0 {
		// Acquire the passphrase
		var passphrase string
		var giveup bool
		for attempts := 0; ; attempts++ {
			if passphrase, giveup, err = p.passRetriever(tokenName, "pkcs11", false, attempts); err != nil {
				return
			}
			if giveup || attempts > 10 {
				err = trustmanager.ErrPasswordInvalid{}
				return
			}
			err = p.ctx.Login(session, pkcs11.CKU_USER, passphrase)
			if err == nil { // Success
				break
			}
			if e, ok := err.(pkcs11.Error); ok {
				if e == pkcs11.CKR_PIN_INCORRECT { // Passphrase incorrect
					continue
				}
			}
			logrus.Errorf("C_Login: %s", err)
			return // Some more serious error
		}
	}
	// Cache the session for next time
	p.sessions[slot] = session
	closeSession = 0 // don't close
	return
}

// findKey finds a key and returns the object handles and a RW session for them.
func (p *Pkcs11Store) findKey(keyID string) (info pkcs11KeyInfo, session pkcs11.SessionHandle, publicObject, privateObject pkcs11.ObjectHandle, err error) {
	if err = p.getKeyInfos(); err != nil {
		return
	}
	var ok bool
	if info, ok = p.keyInfos[keyID]; !ok {
		err = trustmanager.ErrKeyNotFound{keyID}
		return
	}
	// Get a RW session on the required slot
	if session, err = p.getSession(info.slot); err != nil {
		return
	}
	// Find the private key
	searchAttributes := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),
		pkcs11.NewAttribute(pkcs11.CKA_ID, info.id),
	}
	keyAttributes := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, 0),
	}
	privateAttribute := pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PRIVATE_KEY)
	if err = p.forEachObject(session, searchAttributes, func(object pkcs11.ObjectHandle) (err error) {
		// Get the key attributes and translate them into Notary's preferred form
		var attributes []*pkcs11.Attribute
		if attributes, err = p.ctx.GetAttributeValue(session, object, keyAttributes); err != nil {
			logrus.Errorf("C_GetAttributeValue: %s", err)
			return
		}
		if bytes.Compare(attributes[0].Value, privateAttribute.Value) == 0 {
			if privateObject != 0 {
				err = fmt.Errorf("multiple PKSC#11 private key objects for %s", keyID)
				return
			}
			privateObject = object
		} else {
			if publicObject != 0 {
				err = fmt.Errorf("multiple PKSC#11 public key objects for %s", keyID)
				return
			}
			publicObject = object
		}
		return
	}); err != nil {
		return
	}
	if privateObject == 0 || publicObject == 0 {
		logrus.Errorf("missing PKCS#11 key object for %s", keyID)
		err = trustmanager.ErrKeyNotFound{keyID}
		return
	}
	return
}
