package handlers

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution/registry/auth"
	"github.com/docker/notary/errors"
	pb "github.com/docker/notary/proto"
	"github.com/docker/notary/server/snapshot"
	"github.com/docker/notary/server/storage"
	"github.com/docker/notary/server/timestamp"
	"github.com/docker/notary/tuf/data"
	"github.com/docker/notary/tuf/signed"
	"golang.org/x/net/context"
)

type NotaryServer struct {
	CryptoService signed.CryptoService
	Storage       storage.MetaStore
	Access        auth.AccessController
	KeyAlgorithm  string
}

var notImplemented = fmt.Errorf("Not implemented")

func (s *NotaryServer) Auth(ctx context.Context, tok *pb.Token) (*pb.AuthResponse, error) {
	if err := authed(tok.Jwt, "", "push", "pull"); err != nil {
		return &pb.AuthResponse{
			Authed: false,
		}, err
	}
	return &pb.AuthResponse{
		Authed: true,
	}, nil
}

func (s *NotaryServer) Update(ctx context.Context, up *pb.Updates) (*pb.UpdateResponse, error) {
	if err := authed(up.Jwt, up.Gun, "push", "pull"); err != nil {
		return nil, err
	}
	return nil, notImplemented
}

func (s *NotaryServer) Get(ctx context.Context, rr *pb.RepoRole) (*pb.Repository, error) {
	if err := authed(rr.Jwt, rr.Gun, "pull"); err != nil {
		return nil, err
	}
	out, err := s.Storage.GetCurrent(rr.Gun, rr.Role)
	if err != nil {
		return nil, err
	}
	repo := &pb.Repository{}
	switch rr.Role {
	case data.CanonicalRootRole:
		repo.Root = out
	case data.CanonicalSnapshotRole:
		repo.Snapshot = out
	case data.CanonicalTimestampRole:
		repo.Timestamp = out
	default:
		repo.Targets = make(map[string][]byte)
		repo.Targets[rr.Role] = out
	}
	return repo, nil
}

func (s *NotaryServer) GetKey(ctx context.Context, rr *pb.RepoRole) (*pb.KeyResponse, error) {
	if err := authed(rr.Jwt, rr.Gun, "push", "pull"); err != nil {
		return nil, err
	}
	var (
		key data.PublicKey
		err error
	)
	switch rr.Role {
	case data.CanonicalTimestampRole:
		key, err = timestamp.GetOrCreateTimestampKey(rr.Gun, s.Storage, s.CryptoService, s.KeyAlgorithm)
	case data.CanonicalSnapshotRole:
		key, err = snapshot.GetOrCreateSnapshotKey(rr.Gun, s.Storage, s.CryptoService, s.KeyAlgorithm)
	default:
		logrus.Errorf("400 GET %s key: %v", rr.Role, err)
		return nil, errors.ErrInvalidRole.WithDetail(rr.Role)
	}
	if err != nil {
		logrus.Errorf("500 GET %s key: %v", rr.Role, err)
		return nil, errors.ErrUnknown.WithDetail(err)
	}
	return &pb.KeyResponse{Public: key.Public()}, nil
}

// TODO: updates are required in docker/distribution/registry/auth
//       to provide an http independent way to authorize. This will
//       then need to be updated to make use of that.
func authed(tok []byte, gun string, actions ...string) error {
	return nil
}
