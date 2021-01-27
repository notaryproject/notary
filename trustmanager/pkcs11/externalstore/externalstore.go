// +build pkcs11

package externalstore

import (
	"fmt"
	"strings"

	"github.com/miekg/pkcs11"
	"github.com/sirupsen/logrus"
	"github.com/theupdateframework/notary"
	"github.com/theupdateframework/notary/trustmanager"
	"github.com/theupdateframework/notary/trustmanager/pkcs11/common"
	"github.com/theupdateframework/notary/tuf/data"
)

// KeyStore is the hardwarespecific keystore implementing all functions
type KeyStore struct {
	Client *ExternalStoreClient
}

// NewKeyStore initializes the Client
func NewKeyStore() *KeyStore {
	client, err := NewClient()
	if err != nil {
		logrus.Debugf("Failed to connect to ExternalStore: %v", err)
		return &KeyStore{Client: nil}
	}
	return &KeyStore{Client: client}
}

func (ks *KeyStore) Name() string {
	const defaultName = "ExternalStore"
	if ks.Client != nil {
		name, err := ks.Client.Name()
		if err != nil {
			logrus.Debugf("Could not get Name of ExternalStore: %v", err)
			return defaultName
		}
		return name
	}
	return defaultName
}

// AddECDSAKey adds a key to the external store
func (ks *KeyStore) AddECDSAKey(session pkcs11.SessionHandle, privKey data.PrivateKey, hwslot common.HardwareSlot, passRetriever notary.PassRetriever, role data.RoleName) error {
	if ks.Client != nil {
		workload := func(passwd string) (LoginResult, error) {
			err := ks.Client.AddECDSAKey(session, privKey, hwslot, passwd, role)
			return LoginResult{}, err
		}
		_, err := ks.attemptWithlogin(passRetriever, FUNCTION_ADDECDSAKEY, workload)
		return err
	}
	return fmt.Errorf("No HardwareStore available")
}

//GetECDSAKey gets a key by id from the external store
func (ks *KeyStore) GetECDSAKey(session pkcs11.SessionHandle, hwslot common.HardwareSlot, passRetriever notary.PassRetriever) (*data.ECDSAPublicKey, data.RoleName, error) {
	if ks.Client != nil {
		workload := func(passwd string) (LoginResult, error) {
			pubKey, role, err := ks.Client.GetECDSAKey(session, hwslot, passwd)
			if err != nil {
				return LoginResult{}, err
			}
			result := LoginResult{
				PubKey: pubKey,
				Role:   role,
			}
			return result, nil
		}
		result, err := ks.attemptWithlogin(passRetriever, FUNCTION_GETECDSAKEY, workload)
		if err != nil {
			return nil, "", err
		}
		return result.PubKey, result.Role, nil
	}
	return nil, "", fmt.Errorf("No HardwareStore available")
}

// Sign signs the payload with the key of the given ID
func (ks *KeyStore) Sign(session pkcs11.SessionHandle, hwslot common.HardwareSlot, passRetriever notary.PassRetriever, payload []byte) ([]byte, error) {
	if ks.Client != nil {
		workload := func(passwd string) (LoginResult, error) {
			data, err := ks.Client.Sign(session, hwslot, passwd, payload)
			if err != nil {
				return LoginResult{}, err
			} else {
				result := LoginResult{Data: data}
				return result, nil
			}
		}
		result, err := ks.attemptWithlogin(passRetriever, FUNCTION_SIGN, workload)
		if err != nil {
			return nil, err
		}
		return result.Data, nil
	}
	return nil, fmt.Errorf("No HardwareStore available")
}

// HardwareRemoveKey removes the Key with a specified ID from the external store
func (ks *KeyStore) HardwareRemoveKey(session pkcs11.SessionHandle, hwslot common.HardwareSlot, passRetriever notary.PassRetriever, keyID string) error {
	if ks.Client != nil {
		workload := func(passwd string) (LoginResult, error) {
			err := ks.Client.HardwareRemoveKey(session, hwslot, passwd, keyID)
			return LoginResult{}, err
		}
		_, err := ks.attemptWithlogin(passRetriever, FUNCTION_HARDWAREREMOVEKEY, workload)
		return err
	}
	return fmt.Errorf("No HardwareStore available")
}

//HardwareListKeys lists all available Keys stored by the external store
func (ks *KeyStore) HardwareListKeys(session pkcs11.SessionHandle) (map[string]common.HardwareSlot, error) {
	if ks.Client != nil {
		return ks.Client.HardwareListKeys(session)
	}
	return nil, fmt.Errorf("No HardwareStore available")
}

//GetNextEmptySlot returns the first empty slot found by the external store to store a key
func (ks *KeyStore) GetNextEmptySlot(session pkcs11.SessionHandle) ([]byte, error) {
	if ks.Client != nil {
		return ks.Client.GetNextEmptySlot(session)
	}
	return nil, fmt.Errorf("No HardwareStore available")
}

//SetupHSMEnv is responsible for opening the HSM session and performing some checks before (lib available, right version, mechanism available, etc)
func (ks *KeyStore) SetupHSMEnv() (pkcs11.SessionHandle, error) {
	if ks.Client != nil {
		return ks.Client.SetupHSMEnv()
	}
	return 0, fmt.Errorf("No HardwareStore available")
}

// Ends pkcs11 session
func (ks *KeyStore) Cleanup(session pkcs11.SessionHandle) {
	if ks.Client != nil {
		ks.Client.Cleanup(session)
	}
}

// Closes connection to Client
func (ks *KeyStore) Close() {
	if ks.Client != nil {
		ks.Client.Close()
	}
}

func falsePinError(err error) bool {
	return strings.Contains(err.Error(), "CKR_PIN_INCORRECT") || strings.Contains(err.Error(), "CKR_ARGUMENTS_BAD")
}

type LoginResult struct {
	PubKey *data.ECDSAPublicKey
	Role   data.RoleName
	Data   []byte
}

// LoginWorkload represents a workload receiving a password as input
type LoginWorkload func(string) (LoginResult, error)

func (ks *KeyStore) attemptWithlogin(passRetriever notary.PassRetriever, function_id uint, workload LoginWorkload) (LoginResult, error) {
	needsLogin, userFlag, err := ks.Client.NeedLogin(function_id)
	if err != nil {
		return LoginResult{}, err
	}
	if !needsLogin {
		return workload("")
	}

	name := ks.Name()

	for attempts := 0; attempts < 2; attempts++ {
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
		passwd, giveup, err := passRetriever(user, name, false, attempts)
		if giveup || err != nil {
			return LoginResult{}, trustmanager.ErrPasswordInvalid{}
		}

		result, err := workload(passwd)
		if err == nil {
			return result, nil
		} else if !falsePinError(err) {
			return LoginResult{}, err
		}
	}
	return LoginResult{}, trustmanager.ErrAttemptsExceeded{}
}
