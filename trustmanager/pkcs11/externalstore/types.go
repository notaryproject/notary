// +build pkcs11

package externalstore

import (
	"github.com/theupdateframework/notary/trustmanager/pkcs11/common"
	"github.com/theupdateframework/notary/tuf/data"
)

// Used to Identify Functions with Login
const (
	FUNCTION_ADDECDSAKEY       = 1
	FUNCTION_GETECDSAKEY       = 2
	FUNCTION_SIGN              = 3
	FUNCTION_HARDWAREREMOVEKEY = 4
)

// Interface defining the rpc communication
// A struct implementing this interface has to be named ESServer
type ESServer interface {
	Name(ESNameReq, *ESNameRes) error
	AddECDSAKey(ESAddECDSAKeyReq, *ESAddECDSAKeyRes) error
	GetECDSAKey(ESGetECDSAKeyReq, *ESGetECDSAKeyRes) error
	Sign(ESSignReq, *ESSignRes) error
	HardwareRemoveKey(ESHardwareRemoveKeyReq, *ESHardwareRemoveKeyRes) error
	HardwareListKeys(ESHardwareListKeysReq, *ESHardwareListKeysRes) error
	GetNextEmptySlot(ESGetNextEmptySlotReq, *ESGetNextEmptySlotRes) error
	SetupHSMEnv(ESSetupHSMEnvReq, *ESSetupHSMEnvRes) error
	Cleanup(ESCleanupReq, *ESCleanupReq) error
	NeedLogin(ESNeedLoginReq, *ESNeedLoginRes) error
}

type ESNameReq struct {
}

type ESNameRes struct {
	Name string
}

type ESAddECDSAKeyReq struct {
	Session    uint
	PrivateKey ESPrivateKey
	Slot       common.HardwareSlot
	Pass       string
	Role       data.RoleName
}

type ESAddECDSAKeyRes struct {
}

type ESGetECDSAKeyReq struct {
	Session uint
	Slot    common.HardwareSlot
	Pass    string
}

type ESGetECDSAKeyRes struct {
	PublicKey ESPublicKey
	Role      data.RoleName
}

type ESSignReq struct {
	Session uint
	Slot    common.HardwareSlot
	Pass    string
	Payload []byte
}

type ESSignRes struct {
	Result []byte
}

type ESHardwareRemoveKeyReq struct {
	Session uint
	Slot    common.HardwareSlot
	Pass    string
	KeyID   string
}

type ESHardwareRemoveKeyRes struct {
}

type ESHardwareListKeysReq struct {
	Session uint
}

type ESHardwareListKeysRes struct {
	Keys map[string]common.HardwareSlot
}

type ESGetNextEmptySlotReq struct {
	Session uint
}

type ESGetNextEmptySlotRes struct {
	Slot []byte
}

type ESSetupHSMEnvReq struct {
}

type ESSetupHSMEnvRes struct {
	Session uint
}

type ESCleanupReq struct {
	Session uint
}

type ESCleanupRes struct {
}

type ESNeedLoginReq struct {
	Function_ID uint
}

type ESNeedLoginRes struct {
	NeedLogin bool
	UserFlag  uint
}

type ESPublicKey struct {
	Public    []byte
	Algorithm string
}

type ESPrivateKey struct {
	Public    []byte
	Algorithm string
	Private   []byte
}

func NewESPublicKey(pubKey data.PublicKey) ESPublicKey {
	return ESPublicKey{
		Public:    pubKey.Public(),
		Algorithm: pubKey.Algorithm(),
	}
}

func NewESPrivateKey(privKey data.PrivateKey) ESPrivateKey {
	return ESPrivateKey{
		Public:    privKey.Public(),
		Algorithm: privKey.Algorithm(),
		Private:   privKey.Private(),
	}
}

func ESPublicKeyToPublicKey(esPubKey ESPublicKey) data.PublicKey {
	return data.NewPublicKey(esPubKey.Algorithm, esPubKey.Public)
}

func ESPrivateKeyToPrivateKey(esPrivKey ESPrivateKey) (data.PrivateKey, error) {
	pubKey := data.NewPublicKey(esPrivKey.Algorithm, esPrivKey.Public)
	return data.NewPrivateKey(pubKey, esPrivKey.Private)
}
