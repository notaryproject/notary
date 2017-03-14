package api

import (
	"path/filepath"

	"github.com/Sirupsen/logrus"
	google_protobuf "github.com/golang/protobuf/ptypes/empty"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/docker/notary"
	"github.com/docker/notary/client"
	"github.com/docker/notary/client/changelist"
	"github.com/docker/notary/cryptoservice"
	"github.com/docker/notary/storage"
	"github.com/docker/notary/trustmanager"
	"github.com/docker/notary/trustpinning"
	"github.com/docker/notary/tuf/data"
	"github.com/docker/notary/utils"
)

// NewServer creates a new instance of a Client API server with a configured
// upstream Notary Server.
func NewServer(upstream string, upstreamCAPath string, grpcSrv *grpc.Server) (*grpc.Server, error) {
	srv := &Server{
		upstream:       upstream,
		upstreamCAPath: upstreamCAPath,
	}
	RegisterNotaryServer(grpcSrv, srv)
	return grpcSrv, nil
}

type Server struct {
	upstream       string
	upstreamCAPath string
}

func (srv *Server) Initialize(ctx context.Context, initMessage *InitMessage) (*BasicResponse, error) {
	r, err := srv.initRepo(data.GUN(initMessage.Gun))
	if err != nil {
		return nil, err
	}

	roles := make([]data.RoleName, len(initMessage.ServerManagedRoles.Roles))
	for index, role := range initMessage.ServerManagedRoles.Roles {
		roles[index] = data.RoleName(role)
	}

	err = r.Initialize(initMessage.RootKeyIDs, roles...)
	if err != nil {
		return nil, err
	}

	return &BasicResponse{
		Success: true,
	}, nil
}

func (srv *Server) Publish(ctx context.Context, gun *GunMessage) (*BasicResponse, error) {
	r, err := srv.initRepo(data.GUN(gun.Gun))
	if err != nil {
		return nil, err
	}

	err = r.Publish()
	if err != nil {
		return nil, err
	}

	return &BasicResponse{
		Success: true,
	}, nil
}

func (srv *Server) AddTarget(ctx context.Context, t *Target) (*BasicResponse, error) {
	r, err := srv.initRepo(data.GUN(t.GetGun()))
	if err != nil {
		return nil, err
	}
	if err := r.AddTarget(
		&client.Target{
			Name:   t.GetName(),
			Hashes: data.Hashes(t.Hashes),
			Length: t.Length,
		},
	); err != nil {
		return nil, err
	}
	if err := publishRepo(r); err != nil {
		return nil, err
	}

	return &BasicResponse{
		Success: true,
	}, nil
}

func (srv *Server) RemoveTarget(ctx context.Context, t *Target) (*BasicResponse, error) {
	r, err := srv.initRepo(data.GUN(t.GetGun()))
	if err != nil {
		return nil, err
	}
	if err := r.RemoveTarget(
		t.GetName(), "targets",
	); err != nil {
		return nil, err
	}
	if err := publishRepo(r); err != nil {
		return nil, err
	}

	return &BasicResponse{
		Success: true,
	}, nil
}

func (srv *Server) ListTargets(ctx context.Context, message *RoleNameListMessage) (*TargetWithRoleNameListResponse, error) {
	r, err := srv.initRepo(data.GUN(message.Gun))
	if err != nil {
		return nil, err
	}

	roles := make([]data.RoleName, len(message.Roles))
	for index, role := range message.Roles {
		roles[index] = data.RoleName(role)
	}

	targets, err := r.ListTargets(roles...)
	if err != nil {
		return nil, err
	}

	res := make([]*TargetWithRole, len(targets))
	for index, target := range targets {
		res[index] = &TargetWithRole{
			Target: &Target{
				Gun: message.Gun,
				Name: target.Name,
				Length: target.Length,
				Hashes: target.Hashes,

			},
			Role: target.Role.String()
		}
	}

	return &TargetWithRoleNameListResponse{
		TargetWithRoleNameList: &TargetWithRoleNameList{
			Targets: targets,
		},
		Success: true,
	}, nil



	return nil, ErrNotImplemented
}

// GetTargetByName returns a target by the given name.
func (srv *Server) GetTargetByName(context.Context, *TargetByNameAction) (*TargetWithRoleResponse, error) {
	return nil, ErrNotImplemented
}

// GetAllTargetMetadataByName
func (srv *Server) GetAllTargetMetadataByName(context.Context, *TargetNameMessage) (*TargetSignedListResponse, error) {
	return nil, ErrNotImplemented
}

// GetChangelist returns the list of the repository's unpublished changes
func (srv *Server) GetChangelist(context.Context, *google_protobuf.Empty) (*ChangeListResponse, error) {
	return nil, ErrNotImplemented
}

func (srv *Server) ListRoles(context.Context, *google_protobuf.Empty) (*RoleWithSignaturesListResponse, error) {
	return nil, ErrNotImplemented
}

func (srv *Server) GetDelegationRoles(context.Context, *google_protobuf.Empty) (*RoleListResponse, error) {
	return nil, ErrNotImplemented
}

func (srv *Server) AddDelegation(context.Context, *AddDelegationMessage) (*BasicResponse, error) {
	return nil, ErrNotImplemented
}

func (srv *Server) AddDelegationRoleAndKeys(context.Context, *AddDelegationRoleAndKeysMessage) (*BasicResponse, error) {
	return nil, ErrNotImplemented
}

func (srv *Server) AddDelegationPaths(context.Context, *AddDelegationPathsMessage) (*BasicResponse, error) {
	return nil, ErrNotImplemented
}

func (srv *Server) RemoveDelegationKeysAndPaths(context.Context, *RemoveDelegationKeysAndPathsMessage) (*BasicResponse, error) {
	return nil, ErrNotImplemented
}

func (srv *Server) RemoveDelegationRole(context.Context, *RemoveDelegationRoleMessage) (*BasicResponse, error) {
	return nil, ErrNotImplemented
}

func (srv *Server) RemoveDelegationPaths(context.Context, *RemoveDelegationPathsMessage) (*BasicResponse, error) {
	return nil, ErrNotImplemented
}

func (srv *Server) RemoveDelegationKeys(context.Context, *RemoveDelegationKeysMessage) (*BasicResponse, error) {
	return nil, ErrNotImplemented
}

func (srv *Server) ClearDelegationPaths(context.Context, *RoleNameMessage) (*BasicResponse, error) {
	return nil, ErrNotImplemented
}

func (srv *Server) Witness(context.Context, *RoleNameList) (*RoleNameListResponse, error) {
	return nil, ErrNotImplemented
}

func (srv *Server) RotateKey(context.Context, *RotateKeyMessage) (*BasicResponse, error) {
	return nil, ErrNotImplemented
}

// CryptoService implementation
func (srv *Server) CryptoService(context.Context, *google_protobuf.Empty) (*CryptoServiceMessage, error) {
	return nil, ErrNotImplemented
}

func (srv *Server) CryptoServiceCreate(context.Context, *CryptoServiceCreateMessage) (*PublicKeyResponse, error) {
	return nil, ErrNotImplemented
}

func (srv *Server) CryptoServiceAddKey(context.Context, *CryptoServiceAddKeyMessage) (*BasicResponse, error) {
	return nil, ErrNotImplemented
}

func (srv *Server) CryptoServiceGetKey(context.Context, *KeyIDMessage) (*PublicKeyResponse, error) {
	return nil, ErrNotImplemented
}

func (srv *Server) CryptoServiceGetPrivateKey(context.Context, *KeyIDMessage) (*PrivateKeyResponse, error) {
	return nil, ErrNotImplemented
}

func (srv *Server) CryptoServiceRemoveKey(context.Context, *KeyIDMessage) (*BasicResponse, error) {
	return nil, ErrNotImplemented
}

func (srv *Server) CryptoServiceListKeys(context.Context, *RoleNameMessage) (*KeyIDsListResponse, error) {
	return nil, ErrNotImplemented
}

func (srv *Server) CryptoServiceListAllKeys(context.Context, *google_protobuf.Empty) (*SigningKeyIDsToRolesResponse, error) {
	return nil, ErrNotImplemented
}

func (srv *Server) SetLegacyVersions(context.Context, *VersionMessage) (*google_protobuf.Empty, error) {
	return nil, ErrNotImplemented
}

func publishRepo(r *client.NotaryRepository) error {
	if err := r.Publish(); err != nil {
		if _, ok := err.(client.ErrRepoNotInitialized); !ok {
			return err
		}
		if err := initializeRepo(r); err != nil {
			return err
		}
		return r.Publish()
	}
	return nil
}

func initializeRepo(r *client.NotaryRepository) error {
	rootKeyList := r.CryptoService().ListKeys(data.CanonicalRootRole)
	var rootKeyID string
	if len(rootKeyList) < 1 {
		rootPublicKey, err := r.CryptoService().Create(data.CanonicalRootRole, "", data.ECDSAKey)
		if err != nil {
			return err
		}
		rootKeyID = rootPublicKey.ID()
	} else {
		// Chooses the first root key available, which is initialization specific
		// but should return the HW one first.
		rootKeyID = rootKeyList[0]
	}
	return r.Initialize([]string{rootKeyID})
}

func (srv *Server) initRepo(gun data.GUN) (*client.NotaryRepository, error) {
	logrus.Errorf("initializing with upstream ca file %s", srv.upstreamCAPath)
	baseDir := "var/lib/clientapi"
	rt, err := utils.GetReadOnlyAuthTransport(
		srv.upstream,
		[]string{gun.String()},
		"",
		"",
		srv.upstreamCAPath,
	)
	if err != nil {
		return nil, err
	}

	keyStore, err := trustmanager.NewKeyFileStore(filepath.Join(baseDir, notary.PrivDir), retriever)
	if err != nil {
		return nil, err
	}

	cryptoService := cryptoservice.NewCryptoService(keyStore)

	remoteStore, err := storage.NewHTTPStore(
		srv.upstream+"/v2/"+gun.String()+"/_trust/tuf/",
		"",
		"json",
		"key",
		rt,
	)

	return client.NewNotaryRepository(
		baseDir,
		gun,
		srv.upstream,
		remoteStore, // remote store
		storage.NewMemoryStore(nil),
		trustpinning.TrustPinConfig{},
		cryptoService,
		changelist.NewMemChangelist(),
	)
}

func retriever(keyName, alias string, createNew bool, attempts int) (string, bool, error) {
	return "password", false, nil
}

func DefaultPermissions() map[string][]string {
	return map[string][]string{
		"/api.Notary/AddTarget":    {"push", "pull"},
		"/api.Notary/RemoveTarget": {"push", "pull"},
	}
}
