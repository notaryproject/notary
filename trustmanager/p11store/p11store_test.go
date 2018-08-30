package p11store

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"github.com/miekg/pkcs11"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theupdateframework/notary/trustmanager"
	"github.com/theupdateframework/notary/tuf/data"
	"math/big"
	"testing"
)

// mockProvider behaves somewhat like an nShield HSM
type mockProvider struct {
}

// Data and state for the mock HSM

// tokenInfos contains example C_GetTokenInfo output
var tokenInfos = map[uint]pkcs11.TokenInfo{
	492971157: pkcs11.TokenInfo{Label: "accelerator", ManufacturerID: "nCipher Corp. Ltd", Model: "", SerialNumber: "6262-799D-D906", Flags: 0x201, MaxSessionCount: 0x0, SessionCount: 0x1, MaxRwSessionCount: 0x0, RwSessionCount: 0x0, MaxPinLen: 0x100, MinPinLen: 0x0, TotalPublicMemory: 0xffffffffffffffff, FreePublicMemory: 0xffffffffffffffff, TotalPrivateMemory: 0xffffffffffffffff, FreePrivateMemory: 0xffffffffffffffff, HardwareVersion: pkcs11.Version{Major: 0x9, Minor: 0x63}, FirmwareVersion: pkcs11.Version{Major: 0xc, Minor: 0x32}, UTCTime: ""},
	492971158: pkcs11.TokenInfo{Label: "ocs2", ManufacturerID: "nCipher Corp. Ltd", Model: "", SerialNumber: "3d39ca03182939d8", Flags: 0x20d, MaxSessionCount: 0x0, SessionCount: 0x1, MaxRwSessionCount: 0x0, RwSessionCount: 0x0, MaxPinLen: 0x100, MinPinLen: 0x0, TotalPublicMemory: 0xffffffffffffffff, FreePublicMemory: 0xffffffffffffffff, TotalPrivateMemory: 0xffffffffffffffff, FreePrivateMemory: 0xffffffffffffffff, HardwareVersion: pkcs11.Version{Major: 0x9, Minor: 0x63}, FirmwareVersion: pkcs11.Version{Major: 0xc, Minor: 0x32}, UTCTime: ""},
	492971159: pkcs11.TokenInfo{Label: "accelerator", ManufacturerID: "nCipher Corp. Ltd", Model: "", SerialNumber: "5E69-A92C-7097", Flags: 0x201, MaxSessionCount: 0x0, SessionCount: 0x1, MaxRwSessionCount: 0x0, RwSessionCount: 0x0, MaxPinLen: 0x100, MinPinLen: 0x0, TotalPublicMemory: 0xffffffffffffffff, FreePublicMemory: 0xffffffffffffffff, TotalPrivateMemory: 0xffffffffffffffff, FreePrivateMemory: 0xffffffffffffffff, HardwareVersion: pkcs11.Version{Major: 0x9, Minor: 0x63}, FirmwareVersion: pkcs11.Version{Major: 0xc, Minor: 0x32}, UTCTime: ""},
	492971160: pkcs11.TokenInfo{Label: "ocs1", ManufacturerID: "nCipher Corp. Ltd", Model: "", SerialNumber: "7e585d361027d0e6", Flags: 0x20d, MaxSessionCount: 0x0, SessionCount: 0x1, MaxRwSessionCount: 0x0, RwSessionCount: 0x0, MaxPinLen: 0x100, MinPinLen: 0x0, TotalPublicMemory: 0xffffffffffffffff, FreePublicMemory: 0xffffffffffffffff, TotalPrivateMemory: 0xffffffffffffffff, FreePrivateMemory: 0xffffffffffffffff, HardwareVersion: pkcs11.Version{Major: 0x9, Minor: 0x63}, FirmwareVersion: pkcs11.Version{Major: 0xc, Minor: 0x32}, UTCTime: ""},
}

// mockClassPublicKey is the attribute value for CKO_PUBLIC_KEY.
// (The format is platform-dependent so it's convenient to compare
// byte strings rather than turn back into an integer value.)
var mockClassPublicKey = pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PUBLIC_KEY).Value

// mockClassPublicKey is the attribute value for CKO_PRIVATE_KEY.
var mockClassPrivateKey = pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PRIVATE_KEY).Value

// sessionInfo holds information about a synthetic PKCS#11 session handle.
type sessionInfo struct {
	// Slot that session was opened to
	slot uint

	// Flags used when opening session
	flags uint

	// Session state
	state uint

	// Objects to be returned by C_FindObjects
	objects []pkcs11.ObjectHandle

	// Key to be used by C_Sign
	signer *ecdsa.PrivateKey
}

// sessionCounter is the last session handle allocated.
var sessionCounter = pkcs11.SessionHandle(0)

// sessions maps session handles to session information.
var sessions = map[pkcs11.SessionHandle]*sessionInfo{}

// objectInfo holds information about a synthetic PKCS#11 object.
type objectInfo struct {
	// CKA_LABEL value
	label []byte

	// CKA_ID value
	id []byte

	// CKA_CLASS value
	class []byte

	// Encoded public key (if CKA_CLASS=CKO_PUBLIC_KEY)
	pubder []byte

	// Signing key (if CKA_CLASS=CKO_PRIVATE_KEY)
	privkey *ecdsa.PrivateKey
}

// objectCounter is the last object handle allocated
var objectCounter = pkcs11.ObjectHandle(0)

// ovjects maps object handles to object information.
var objects = map[pkcs11.ObjectHandle]*objectInfo{}

// Mock methods

func (m *mockProvider) Initialize() (err error) {
	return
}

func (m *mockProvider) GetSlotList(tokenPresent bool) (slots []uint, err error) {
	slots = make([]uint, 0, len(tokenInfos))
	for slot, _ := range tokenInfos {
		slots = append(slots, slot)
	}
	return
}

func (m *mockProvider) GetTokenInfo(slotID uint) (ti pkcs11.TokenInfo, err error) {
	ti = tokenInfos[slotID]
	return
}

func (m *mockProvider) OpenSession(slotID uint, flags uint) (sh pkcs11.SessionHandle, err error) {
	sessionCounter += 1
	sessions[sessionCounter] = &sessionInfo{slotID, flags, 0, nil, nil}
	sh = sessionCounter
	return
}

func (m *mockProvider) CloseSession(sh pkcs11.SessionHandle) (err error) {
	if _, ok := sessions[sh]; !ok {
		err = pkcs11.Error(pkcs11.CKR_SESSION_HANDLE_INVALID)
		return
	}
	delete(sessions, sh)
	return
}

func (m *mockProvider) GetSessionInfo(sh pkcs11.SessionHandle) (si pkcs11.SessionInfo, err error) {
	var s *sessionInfo
	var ok bool
	if s, ok = sessions[sh]; !ok {
		err = pkcs11.Error(pkcs11.CKR_SESSION_HANDLE_INVALID)
		return
	}
	si.SlotID = s.slot
	si.State = s.state
	si.Flags = s.flags
	si.DeviceError = 0
	return
}

func (m *mockProvider) Login(sh pkcs11.SessionHandle, userType uint, pin string) (err error) {
	var expectedPassword string
	switch sessions[sh].slot {
	case 492971158:
		expectedPassword = "test"
	case 492971160:
		expectedPassword = "test2"
	}
	if pin != expectedPassword {
		err = pkcs11.Error(pkcs11.CKR_PIN_INCORRECT)
		return
	}
	sessions[sh].state = 3
	return
}

func (m *mockProvider) GenerateKeyPair(sh pkcs11.SessionHandle, mech []*pkcs11.Mechanism, public, private []*pkcs11.Attribute) (pubObj pkcs11.ObjectHandle, privObj pkcs11.ObjectHandle, err error) {
	var label []byte
	var id []byte
	for _, a := range public {
		switch a.Type {
		case pkcs11.CKA_LABEL:
			label = a.Value
		case pkcs11.CKA_ID:
			id = a.Value
		}
	}
	var k *ecdsa.PrivateKey
	if k, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader); err != nil {
		return
	}
	pubDer := elliptic.Marshal(elliptic.P256(), k.PublicKey.X, k.PublicKey.Y)
	if pubDer, err = asn1.Marshal(pubDer); err != nil {
		return
	}
	objectCounter += 1
	pubObj = objectCounter
	objects[pubObj] = &objectInfo{label, id, mockClassPublicKey, pubDer, nil}
	objectCounter += 1
	privObj = objectCounter
	objects[privObj] = &objectInfo{label, id, mockClassPrivateKey, nil, k}
	return
}

func (m *mockProvider) DestroyObject(sh pkcs11.SessionHandle, oh pkcs11.ObjectHandle) (err error) {
	if _, ok := objects[oh]; !ok {
		err = pkcs11.Error(pkcs11.CKR_OBJECT_HANDLE_INVALID)
		return
	}
	delete(objects, oh)
	return
}

func (m *mockProvider) GetAttributeValue(sh pkcs11.SessionHandle, oh pkcs11.ObjectHandle, attrs []*pkcs11.Attribute) (result []*pkcs11.Attribute, err error) {
	oinfo := objects[oh]
	for _, a := range attrs {
		var value []byte
		switch a.Type {
		case pkcs11.CKA_LABEL:
			value = oinfo.label
		case pkcs11.CKA_ID:
			value = oinfo.id
		case pkcs11.CKA_EC_POINT:
			value = oinfo.pubder
		case pkcs11.CKA_EC_PARAMS:
			value = secp256r1
		case pkcs11.CKA_CLASS:
			value = oinfo.class
		default:
			panic("not implemented")
		}
		result = append(result, pkcs11.NewAttribute(a.Type, value))
	}
	return
}

func (m *mockProvider) SetAttributeValue(sh pkcs11.SessionHandle, oh pkcs11.ObjectHandle, attrs []*pkcs11.Attribute) (err error) {
	oinfo := objects[oh]
	for _, a := range attrs {
		switch a.Type {
		case pkcs11.CKA_LABEL:
			oinfo.label = a.Value
		case pkcs11.CKA_ID:
			oinfo.id = a.Value
		case pkcs11.CKA_EC_POINT:
			oinfo.pubder = a.Value
		case pkcs11.CKA_CLASS:
			oinfo.class = a.Value
		default:
			panic("not implemented")
		}
	}
	return
}

func (m *mockProvider) FindObjectsInit(sh pkcs11.SessionHandle, temp []*pkcs11.Attribute) (err error) {
	results := make([]pkcs11.ObjectHandle, 0)
	for oh, oinfo := range objects {
		objectMatches := true
		for _, a := range temp {
			var attributeMatches bool
			switch a.Type {
			case pkcs11.CKA_ID:
				attributeMatches = (bytes.Compare(a.Value, oinfo.id) == 0)
			case pkcs11.CKA_LABEL:
				attributeMatches = (bytes.Compare(a.Value, oinfo.label) == 0)
			case pkcs11.CKA_TOKEN:
				attributeMatches = true
			case pkcs11.CKA_CLASS:
				attributeMatches = (bytes.Compare(a.Value, oinfo.class) == 0)
			case pkcs11.CKA_KEY_TYPE:
				attributeMatches = true
			default:
				panic("not implemented")
			}
			if !attributeMatches {
				objectMatches = false
				break
			}
		}
		if objectMatches {
			results = append(results, oh)
		}
	}
	sessions[sh].objects = results
	return
}

func (m *mockProvider) FindObjects(sh pkcs11.SessionHandle, max int) (objects []pkcs11.ObjectHandle, more bool, err error) {
	objects = sessions[sh].objects
	more = false
	return
}

func (m *mockProvider) FindObjectsFinal(sh pkcs11.SessionHandle) (err error) {
	sessions[sh].objects = nil
	return
}

func (m *mockProvider) SignInit(sh pkcs11.SessionHandle, mech []*pkcs11.Mechanism, oh pkcs11.ObjectHandle) (err error) {
	if mech[0].Mechanism != pkcs11.CKM_ECDSA {
		err = pkcs11.Error(pkcs11.CKR_MECHANISM_INVALID)
		return
	}
	sessions[sh].signer = objects[oh].privkey
	return
}

func (m *mockProvider) Sign(sh pkcs11.SessionHandle, message []byte) (sig []byte, err error) {
	var r, s *big.Int
	if r, s, err = ecdsa.Sign(rand.Reader, sessions[sh].signer, message); err != nil {
		return
	}
	rbytes := r.Bytes()
	rbytes = append(make([]byte, 32-len(rbytes)), rbytes...)
	sbytes := s.Bytes()
	sbytes = append(make([]byte, 32-len(sbytes)), sbytes...)
	sig = append(rbytes, sbytes...)
	return
}

// getTestStore returns a Pkcs11Store using a mock PKCS#11 provider.
func getTestStore(t *testing.T, pin string) (p *Pkcs11Store) {
	p = &Pkcs11Store{
		ctx:      &mockProvider{},
		sessions: map[uint]pkcs11.SessionHandle{},
		keyInfos: map[string]pkcs11KeyInfo{},
		passRetriever: func(keyName, alias string, createNew bool, attempts int) (passphrase string, giveup bool, err error) {
			passphrase = pin
			return
		},
	}
	return
}

func TestPkcs11Store(t *testing.T) {
	store := getTestStore(t, "test2")
	// Store expected to be empty at startup
	keyInfos := store.ListKeys()
	require.Empty(t, keyInfos)
	// Key generation must work
	keyID, pubKeyGen, err := store.Generate(trustmanager.KeyInfo{"anything", "root"}, "label:ocs1", "ecdsa")
	require.NoError(t, err)
	// Key identifiers must not be trivial
	assert.True(t, len(keyID) > 0)
	// Key must show up in key list
	keyInfos = store.ListKeys()
	keyInfo, ok := keyInfos[keyID]
	require.True(t, ok)
	assert.Equal(t, data.RoleName("root"), keyInfo.Role)
	assert.Equal(t, data.GUN("anything"), keyInfo.Gun)
	// Must be able to retrieve key
	var privKey data.PrivateKey
	var role data.RoleName
	privKey, role, err = store.GetKey(keyID)
	require.NoError(t, err)
	assert.Equal(t, data.RoleName("root"), role)
	pubKey := privKey.CryptoSigner().Public().(*ecdsa.PublicKey)
	// Must be able to sign. data.PrivateKey.Sign is message -> r||s.
	message := []byte("test message")
	digest := sha256.Sum256(message)
	var sig []byte
	sig, err = privKey.Sign(rand.Reader, message, nil)
	require.NoError(t, err)
	// Signature must verify
	var rs dsaSignature
	rs.R = big.NewInt(0)
	rs.S = big.NewInt(0)
	rs.R.SetBytes(sig[:len(sig)/2])
	rs.S.SetBytes(sig[len(sig)/2:])
	assert.True(t, ecdsa.Verify(pubKey, digest[:], rs.R, rs.S))
	// Must be able to sign via crypto.Signer, which is digest -> DER.
	sig, err = privKey.CryptoSigner().Sign(rand.Reader, digest[:], nil)
	require.NoError(t, err)
	// Signature must verify
	rs.R = nil
	rs.S = nil
	_, err = asn1.Unmarshal(sig, &rs)
	require.NoError(t, err)
	assert.True(t, ecdsa.Verify(pubKey, digest[:], rs.R, rs.S))
	// Remove the key
	assert.NoError(t, store.RemoveKey(keyID))
	// It must be gone from the key list
	keyInfos = store.ListKeys()
	_, ok = keyInfos[keyID]
	require.False(t, ok)
	// It must not be findable
	_, _, err = store.GetKey(keyID)
	require.NotNil(t, err)
	// Consistency check between public key from generation and signer
	var pubKeyDecoded crypto.PublicKey
	pubKeyDecoded, err = x509.ParsePKIXPublicKey(pubKeyGen.Public())
	require.NoError(t, err)
	pubKeyEcdsa := pubKeyDecoded.(*ecdsa.PublicKey)
	assert.EqualValues(t, pubKeyEcdsa, pubKey)
}

func TestPkcs11Store_Negative(t *testing.T) {
	var err error
	store := getTestStore(t, "test2")
	// Nonexistent keys mustn't be findable
	_, err = store.GetKeyInfo("9b5d4d02a77f2b4d645bff290d753f641571c0dfa09c7581d6019b67dec2ad45")
	require.NotNil(t, err)
	_, _, err = store.GetKey("9b5d4d02a77f2b4d645bff290d753f641571c0dfa09c7581d6019b67dec2ad45")
	require.NotNil(t, err)
	// Nonexistent keys mustn't be removable
	assert.NotNil(t, store.RemoveKey("9b5d4d02a77f2b4d645bff290d753f641571c0dfa09c7581d6019b67dec2ad45"))
	// Import isn't supported
	err = store.AddKey(trustmanager.KeyInfo{}, nil)
	assert.NotNil(t, err)
	// Wrong passphrase
	storeWrongPP := getTestStore(t, "open sesame")
	_, _, err = storeWrongPP.Generate(trustmanager.KeyInfo{"anything", "root"}, "label:ocs1", "ecdsa")
	assert.NotNil(t, err)
}
