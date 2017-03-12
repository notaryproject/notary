package api

import (
	"google.golang.org/grpc"

	"github.com/docker/notary/client"
	"github.com/docker/notary/client/changelist"
	"github.com/docker/notary/tuf/data"
	"github.com/docker/notary/tuf/signed"
	"golang.org/x/net/context"
)

type Client struct {
	client NotaryClient
	cs     signed.CryptoService
	gun    data.GUN
}

func NewClient(conn *grpc.ClientConn, gun data.GUN) *Client {
	return &Client{
		client: NewNotaryClient(conn),
		gun:    gun,
	}
}

func (c *Client) Initialize(rootKeyIDs []string, serverManagedRoles ...data.RoleName) error {
	return ErrNotImplemented
}

func (c *Client) Publish() error {
	return ErrNotImplemented
}

func (c *Client) DeleteTrustData(deleteRemote bool) error {
	return ErrNotImplemented
}

func (c *Client) AddTarget(target *client.Target, roles ...data.RoleName) error {
	t := &Target{
		Gun:    c.gun.String(),
		Name:   target.Name,
		Length: target.Length,
		Hashes: target.Hashes,
	}
	_, err := c.client.AddTarget(context.Background(), t)
	return err
}

func (c *Client) RemoveTarget(targetName string, roles ...data.RoleName) error {
	t := &Target{
		Gun:  c.gun.String(),
		Name: targetName,
	}
	_, err := c.client.RemoveTarget(context.Background(), t)
	return err
}

func (c *Client) ListTargets(roles ...data.RoleName) ([]*client.TargetWithRole, error) {
	rolesList := make([]string, len(roles))
	for index, value := range roles {
		rolesList[index] = value.String()
	}

	targetWithRoleList, err := c.client.ListTargets(context.Background(), &RoleNameList{Roles:rolesList})
	if err != nil {
		return []*client.TargetWithRole{}, err
	}

	targets := targetWithRoleList.TargetWithRoleNameList.Targets
	res := make([]*client.TargetWithRole, len(targets))

	for index, target := range targets {
		t := target.Target
		r := target.Role

		currTarget := client.Target{
			Name: t.GetName(),
			Hashes: data.Hashes(t.Hashes),
			Length: t.GetLength(),
		}

		currRole := data.RoleName(r)

		targetWithRole := &client.TargetWithRole{
			Target: currTarget,
			Role: currRole,
		}

		res[index] = targetWithRole
	}

	return res, nil
}

func (c *Client) GetTargetByName(name string, roles ...data.RoleName) (*client.TargetWithRole, error) {
	rolesList := make([]string, len(roles))
	for index, value := range roles {
		rolesList[index] = value.String()
	}

	targetByNameAction := &TargetByNameAction{
		Name: name,
		Roles: &RoleNameList{Roles:rolesList},
	}

	targetWithRole, err := c.client.GetTargetByName(context.Background(), targetByNameAction)
	if err != nil {
		return nil, err
	}

	target := targetWithRole.TargetWithRole.Target
	role := targetWithRole.TargetWithRole.Role

	res := &client.TargetWithRole{
		Target: client.Target{
			Name: target.GetName(),
			Hashes: data.Hashes(target.Hashes),
			Length: target.GetLength(),
		},
		Role: data.RoleName(role),
	}

	return res, nil
}

func (c *Client) GetAllTargetMetadataByName(name string) ([]client.TargetSignedStruct, error) {
	targetName := &TargetName{
		Name: name,
	}

	targetSignedListResponse, err := c.client.GetAllTargetMetadataByName(context.Background(), targetName)
	if err != nil {
		return nil, err
	}

	targetsSigned := targetSignedListResponse.TargetSignedList.Targets

	res := make([]*client.TargetWithRole, len(targetsSigned))
	for index, value := range targetsSigned {
		
	}

	return nil, ErrNotImplemented
}

func (c *Client) GetChangelist() (changelist.Changelist, error) {
	return nil, ErrNotImplemented
}

func (c *Client) ListRoles() ([]client.RoleWithSignatures, error) {
	return nil, ErrNotImplemented
}

func (c *Client) GetDelegationRoles() ([]data.Role, error) {
	return nil, ErrNotImplemented
}

func (c *Client) AddDelegation(name data.RoleName, delegationKeys []data.PublicKey, paths []string) error {
	return ErrNotImplemented
}

func (c *Client) AddDelegationRoleAndKeys(name data.RoleName, delegationKeys []data.PublicKey) error {
	return ErrNotImplemented
}

func (c *Client) AddDelegationPaths(name data.RoleName, paths []string) error {
	return ErrNotImplemented
}

func (c *Client) RemoveDelegationKeysAndPaths(name data.RoleName, keyIDs, paths []string) error {
	return ErrNotImplemented
}

func (c *Client) RemoveDelegationRole(name data.RoleName) error {
	return ErrNotImplemented
}

func (c *Client) RemoveDelegationPaths(name data.RoleName, paths []string) error {
	return ErrNotImplemented
}

func (c *Client) RemoveDelegationKeys(name data.RoleName, keyIDs []string) error {
	return ErrNotImplemented
}

func (c *Client) ClearDelegationPaths(name data.RoleName) error {
	return ErrNotImplemented
}

func (c *Client) Witness(roles ...data.RoleName) ([]data.RoleName, error) {
	return nil, ErrNotImplemented
}

func (c *Client) RotateKey(role data.RoleName, serverManagesKey bool, keyList []string) error {
	return ErrNotImplemented
}

func (c *Client) SetLegacyVersions(n int) {
	// do nothing. New client API based repos only support new format root key rotation
}

func (c *Client) CryptoService() signed.CryptoService {
	return c.cs
}

func (c *Client) GetGUN() data.GUN {
	return c.gun
}

type CryptoService struct {
	client NotaryClient
}

// Create issues a new key pair and is responsible for loading
// the private key into the appropriate signing service.
func (cs *CryptoService) Create(role data.RoleName, gun data.GUN, algorithm string) (data.PublicKey, error) {
	return nil, ErrNotImplemented
}

// AddKey adds a private key to the specified role and gun
func (cs *CryptoService) AddKey(role data.RoleName, gun data.GUN, key data.PrivateKey) error {
	return ErrNotImplemented
}

// GetKey retrieves the public key if present, otherwise it returns nil
func (cs *CryptoService) GetKey(keyID string) data.PublicKey {
	return nil
}

// GetPrivateKey retrieves the private key and role if present and retrievable,
// otherwise it returns nil and an error
func (cs *CryptoService) GetPrivateKey(keyID string) (data.PrivateKey, data.RoleName, error) {
	return nil, "", ErrNotImplemented
}

// RemoveKey deletes the specified key, and returns an error only if the key
// removal fails. If the key doesn't exist, no error should be returned.
func (cs *CryptoService) RemoveKey(keyID string) error {
	return ErrNotImplemented
}

// ListKeys returns a list of key IDs for the role, or an empty list or
// nil if there are no keys.
func (cs *CryptoService) ListKeys(role data.RoleName) []string {
	return nil
}

// ListAllKeys returns a map of all available signing key IDs to role, or
// an empty map or nil if there are no keys.
func (cs *CryptoService) ListAllKeys() map[string]data.RoleName {
	return nil
}
