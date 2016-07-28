package client

import (
	"github.com/docker/notary/client/changelist"
	"github.com/docker/notary/tuf"
	"github.com/docker/notary/tuf/data"
	"path/filepath"
)

// Witness creates change objects to witness (i.e. re-sign) the given
// roles on the next publish. One change is created per role
func (r *NotaryRepository) Witness(roles ...string) ([]string, error) {
	cl, err := changelist.NewFileChangelist(filepath.Join(r.tufRepoPath, "changelist"))
	if err != nil {
		return nil, err
	}
	defer cl.Close()

	successful := make([]string, 0, len(roles))
	for _, role := range roles {
		// scope is role
		c := changelist.NewTUFChange(
			changelist.ActionUpdate,
			role,
			changelist.TypeWitness,
			"",
			nil,
		)
		err = cl.Add(c)
		if err != nil {
			break
		}
		successful = append(successful, role)
	}
	return successful, err
}

func witnessTargets(repo *tuf.Repo, invalid *tuf.Repo, role string) error {
	if r, ok := repo.Targets[role]; ok {
		// role is already valid, mark for re-signing/updating
		r.Dirty = true
		return nil
	}
	if invalid != nil {
		if r, ok := invalid.Targets[role]; ok {
			// role is recognized but invalid, move to valid data and mark for re-signing
			repo.Targets[role] = r
			r.Dirty = true
			return nil
		}
	}
	// role isn't recognized, even as invalid
	return data.ErrInvalidRole{
		Role:   role,
		Reason: "this role is not known",
	}
}
