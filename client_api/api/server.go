package api

import (
	"github.com/docker/notary/client"
	"github.com/docker/notary/storage"
	"github.com/docker/notary/trustpinning"
	"github.com/docker/notary/tuf/data"
	"github.com/docker/notary/utils"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// NewServer creates a new instance of a Client API server with a configured
// upstream Notary Server.
func NewServer(upstream string, serverOpts []grpc.ServerOption) (*grpc.Server, error) {
	grpcSrv := grpc.NewServer(serverOpts...)
	srv := &Server{
		upstream: upstream,
	}
	RegisterNotaryServer(grpcSrv, srv)
	return grpcSrv, nil
}

type Server struct {
	upstream string
}

func (srv *Server) AddTarget(ctx context.Context, t *TargetAction) (*BasicResponse, error) {
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

func (srv *Server) RemoveTarget(ctx context.Context, t *TargetAction) (*BasicResponse, error) {
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
	rootKeyList := r.CryptoService.ListKeys(data.CanonicalRootRole)
	var rootKeyID string
	if len(rootKeyList) < 1 {
		rootPublicKey, err := r.CryptoService.Create(data.CanonicalRootRole, "", data.ECDSAKey)
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
	rt, err := utils.GetReadOnlyAuthTransport(
		srv.upstream,
		[]string{gun.String()},
		"",
		"",
		"/fixtures/root-ca.crt",
	)
	if err != nil {
		return nil, err
	}
	return client.NewNotaryRepository(
		"/var/lib/clientapi",
		gun,
		srv.upstream,
		rt,
		storage.NewMemoryStore(nil),
		retriever,
		trustpinning.TrustPinConfig{},
	)
}

func retriever(keyName, alias string, createNew bool, attempts int) (string, bool, error) {
	return "password", false, nil
}
