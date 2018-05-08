// +build pkcs11

package common

import (
	"fmt"

	"github.com/miekg/pkcs11"
	"github.com/sirupsen/logrus"
	"github.com/theupdateframework/notary"
	"github.com/theupdateframework/notary/trustmanager"
	"github.com/theupdateframework/notary/tuf/data"
)

// an interface around the pkcs11 library, so that things can be mocked out
// for testing
const ecdsaPrivateKeySize = 32

var (
	hardwareName     string
	hardwareKeyStore HardwareSpecificStore
)

func SetKeyStore(ks HardwareSpecificStore) {
	hardwareKeyStore = ks
	hardwareName = hardwareKeyStore.Name()
}

// IPKCS11 is an interface for wrapping github.com/miekg/pkcs11
type Pkcs11LibLoader func(module string) IPKCS11Ctx

func DefaultLoader(module string) IPKCS11Ctx {
	return pkcs11.New(module)
}

// IPKCS11Ctx is an interface for wrapping the parts of
// github.com/miekg/pkcs11.Ctx that hardwarekeystore requires
type IPKCS11Ctx interface {
	Destroy()
	Initialize() error
	Finalize() error
	GetSlotList(tokenPresent bool) ([]uint, error)
	GetInfo() (pkcs11.Info, error)
	OpenSession(slotID uint, flags uint) (pkcs11.SessionHandle, error)
	CloseSession(sh pkcs11.SessionHandle) error
	Login(sh pkcs11.SessionHandle, userType uint, pin string) error
	Logout(sh pkcs11.SessionHandle) error
	CreateObject(sh pkcs11.SessionHandle, temp []*pkcs11.Attribute) (pkcs11.ObjectHandle, error)
	DestroyObject(sh pkcs11.SessionHandle, oh pkcs11.ObjectHandle) error
	GetAttributeValue(sh pkcs11.SessionHandle, o pkcs11.ObjectHandle, a []*pkcs11.Attribute) ([]*pkcs11.Attribute, error)
	FindObjectsInit(sh pkcs11.SessionHandle, temp []*pkcs11.Attribute) error
	FindObjects(sh pkcs11.SessionHandle, max int) ([]pkcs11.ObjectHandle, bool, error)
	FindObjectsFinal(sh pkcs11.SessionHandle) error
	SignInit(sh pkcs11.SessionHandle, m []*pkcs11.Mechanism, o pkcs11.ObjectHandle) error
	Sign(sh pkcs11.SessionHandle, message []byte) ([]byte, error)
	GetMechanismList(slotID uint) ([]*pkcs11.Mechanism, error)
	GetTokenInfo(slotID uint) (pkcs11.TokenInfo, error)
}

//Common Functions and Structs that may be used by different PKCS11 Implementations

type HardwareSlot struct {
	Role   data.RoleName
	SlotID []byte
	KeyID  string
}

// An error indicating that the HSM is not present (as opposed to failing),
// i.e. that we can confidently claim that the key is not stored in the HSM
// without notifying the user about a missing or failing HSM.
type ErrHSMNotPresent struct {
	Err string
}

func (err ErrHSMNotPresent) Error() string {
	return err.Err
}

type HardwareSpecificStore interface {
	Name() string
	AddECDSAKey(IPKCS11Ctx, pkcs11.SessionHandle, data.PrivateKey, HardwareSlot, notary.PassRetriever, data.RoleName) error
	GetECDSAKey(IPKCS11Ctx, pkcs11.SessionHandle, HardwareSlot, notary.PassRetriever) (*data.ECDSAPublicKey, data.RoleName, error)
	Sign(IPKCS11Ctx, pkcs11.SessionHandle, HardwareSlot, notary.PassRetriever, []byte) ([]byte, error)
	HardwareRemoveKey(IPKCS11Ctx, pkcs11.SessionHandle, HardwareSlot, notary.PassRetriever, string) error
	HardwareListKeys(IPKCS11Ctx, pkcs11.SessionHandle) (map[string]HardwareSlot, error)
	GetNextEmptySlot(IPKCS11Ctx, pkcs11.SessionHandle) ([]byte, error)
	SetupHSMEnv(Pkcs11LibLoader) (IPKCS11Ctx, pkcs11.SessionHandle, error)
}

// ErrBackupFailed is returned when a YubiStore fails to back up a key that
// is added
type ErrBackupFailed struct {
	err string
}

func (err ErrBackupFailed) Error() string {
	return fmt.Sprintf("Failed to backup private key to: %s", err.err)
}

// IsAccessible returns true if a Hardwarestore can be accessed
func IsAccessible() bool {
	ctx, session, err := hardwareKeyStore.SetupHSMEnv(DefaultLoader)
	if err != nil {
		return false
	}
	defer Cleanup(ctx, session)
	return true
}

// Login is responsible for getting a working login on attached hardwarekeystore
func Login(ctx IPKCS11Ctx, session pkcs11.SessionHandle, passRetriever notary.PassRetriever, userFlag uint, defaultPassw string, hardwareName string) error {
	err := ctx.Login(session, userFlag, defaultPassw)
	if err == nil {
		return nil
	}

	for attempts := 0; ; attempts++ {
		var (
			giveup bool
			err    error
			user   string
		)
		if userFlag == pkcs11.CKU_SO {
			user = "SO Pin"
		} else {
			user = "User Pin"
		}
		passwd, giveup, err := passRetriever(user, hardwareName, false, attempts)
		if giveup || err != nil {
			return trustmanager.ErrPasswordInvalid{}
		}
		if attempts > 2 {
			return trustmanager.ErrAttemptsExceeded{}
		}

		err = ctx.Login(session, userFlag, passwd)
		if err == nil {
			return nil
		}
	}
}

//Cleanup is responsible for cleaning up the pkcs11 session on the hardware
func Cleanup(ctx IPKCS11Ctx, session pkcs11.SessionHandle) {
	err := ctx.CloseSession(session)
	if err != nil {
		logrus.Debugf("Error closing session: %s", err.Error())
	}
	FinalizeAndDestroy(ctx)
}

//FinalizeAndDestroy is responsible for finalizing the session on the hardware
func FinalizeAndDestroy(ctx IPKCS11Ctx) {
	err := ctx.Finalize()
	if err != nil {
		logrus.Debugf("Error finalizing: %s", err.Error())
	}
	ctx.Destroy()
}

func BuildKeyMap(keys map[string]HardwareSlot) map[string]trustmanager.KeyInfo {
	res := make(map[string]trustmanager.KeyInfo)
	for k, v := range keys {
		res[k] = trustmanager.KeyInfo{Role: v.Role, Gun: ""}
	}
	return res
}

// If a byte array is less than the number of bytes specified by
// ecdsaPrivateKeySize, left-zero-pad the byte array until
// it is the required size.
func EnsurePrivateKeySize(payload []byte) []byte {
	final := payload
	if len(payload) < ecdsaPrivateKeySize {
		final = make([]byte, ecdsaPrivateKeySize)
		copy(final[ecdsaPrivateKeySize-len(payload):], payload)
	}
	return final
}
