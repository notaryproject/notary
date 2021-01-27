// +build pkcs11

package externalstore

import (
	"fmt"
	"net/rpc"

	"github.com/miekg/pkcs11"
	"github.com/sirupsen/logrus"
	"github.com/theupdateframework/notary/trustmanager/pkcs11/common"
	"github.com/theupdateframework/notary/tuf/data"
)

const (
	ExternalStoreSocketPath = "/var/run/notary/hardwarestore.sock"
)

type ExternalStoreClient struct {
	client *rpc.Client
}

func NewClient() (*ExternalStoreClient, error) {
	client, err := rpc.Dial("unix", ExternalStoreSocketPath)
	return &ExternalStoreClient{client: client}, err
}

func (c *ExternalStoreClient) Name() (string, error) {
	client := c.client
	req := ESNameReq{}
	res := new(ESNameRes)
	err := client.Call("ESServer.Name", req, res)
	if err != nil {
		return "", err
	}
	return res.Name, nil
}

func (c *ExternalStoreClient) AddECDSAKey(session pkcs11.SessionHandle, privKey data.PrivateKey, hwslot common.HardwareSlot, passwd string, role data.RoleName) error {
	client := c.client
	req := ESAddECDSAKeyReq{
		Session:    uint(session),
		PrivateKey: NewESPrivateKey(privKey),
		Slot:       hwslot,
		Pass:       passwd,
		Role:       role,
	}
	res := new(ESAddECDSAKeyRes)
	return client.Call("ESServer.AddECDSAKey", req, res)
}

func (c *ExternalStoreClient) GetECDSAKey(session pkcs11.SessionHandle, hwslot common.HardwareSlot, passwd string) (*data.ECDSAPublicKey, data.RoleName, error) {
	client := c.client
	req := ESGetECDSAKeyReq{
		Session: uint(session),
		Slot:    hwslot,
		Pass:    passwd,
	}
	res := new(ESGetECDSAKeyRes)
	err := client.Call("ESServer.GetECDSAKey", req, res)
	if err != nil {
		return nil, "", err
	}
	pubKey, ok := ESPublicKeyToPublicKey(res.PublicKey).(*data.ECDSAPublicKey)
	if !ok {
		return nil, "", fmt.Errorf("Got wrong type of Public Key, need data.ECDSAPublicKey")
	}
	return pubKey, res.Role, nil
}

func (c *ExternalStoreClient) Sign(session pkcs11.SessionHandle, hwslot common.HardwareSlot, passwd string, payload []byte) ([]byte, error) {
	client := c.client
	req := ESSignReq{
		Session: uint(session),
		Slot:    hwslot,
		Pass:    passwd,
		Payload: payload,
	}
	res := new(ESSignRes)
	err := client.Call("ESServer.Sign", req, res)
	if err != nil {
		return nil, err
	}
	return res.Result, nil
}

func (c *ExternalStoreClient) HardwareRemoveKey(session pkcs11.SessionHandle, hwslot common.HardwareSlot, passwd string, keyID string) error {
	client := c.client
	req := ESHardwareRemoveKeyReq{
		Session: uint(session),
		Slot:    hwslot,
		Pass:    passwd,
		KeyID:   keyID,
	}
	res := new(ESHardwareRemoveKeyRes)
	return client.Call("ESServer.HardwareRemoveKey", req, res)
}

func (c *ExternalStoreClient) HardwareListKeys(session pkcs11.SessionHandle) (map[string]common.HardwareSlot, error) {
	client := c.client
	req := ESHardwareListKeysReq{Session: uint(session)}
	res := new(ESHardwareListKeysRes)
	err := client.Call("ESServer.HardwareListKeys", req, res)
	if err != nil {
		return nil, err
	}
	return res.Keys, nil
}

func (c *ExternalStoreClient) GetNextEmptySlot(session pkcs11.SessionHandle) ([]byte, error) {
	client := c.client
	req := ESGetNextEmptySlotReq{Session: uint(session)}
	res := new(ESGetNextEmptySlotRes)
	err := client.Call("ESServer.GetNextEmptySlot", req, res)
	if err != nil {
		return nil, err
	}
	return res.Slot, nil
}

func (c *ExternalStoreClient) SetupHSMEnv() (pkcs11.SessionHandle, error) {
	client := c.client
	req := ESSetupHSMEnvReq{}
	res := new(ESSetupHSMEnvRes)
	err := client.Call("ESServer.SetupHSMEnv", req, res)
	if err != nil {
		return 0, err
	}
	session := pkcs11.SessionHandle(res.Session)
	return session, nil
}

func (c *ExternalStoreClient) Cleanup(session pkcs11.SessionHandle) {
	client := c.client
	req := ESCleanupReq{Session: uint(session)}
	res := new(ESCleanupRes)
	err := client.Call("ESServer.Cleanup", req, res)
	if err != nil {
		logrus.Debugf("Could not cleanup context: %v", err)
	}
}

func (c *ExternalStoreClient) NeedLogin(function_id uint) (bool, uint, error) {
	client := c.client
	req := ESNeedLoginReq{Function_ID: function_id}
	res := new(ESNeedLoginRes)
	err := client.Call("ESServer.NeedLogin", req, res)
	if err != nil {
		return true, pkcs11.CKU_CONTEXT_SPECIFIC, err
	}
	return res.NeedLogin, res.UserFlag, nil
}

func (c *ExternalStoreClient) Close() {
	c.client.Close()
}
