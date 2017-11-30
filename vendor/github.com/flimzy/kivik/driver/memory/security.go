package memory

import (
	"context"

	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/driver"
	"github.com/flimzy/kivik/errors"
)

func cloneSecurity(in *driver.Security) *driver.Security {
	return &driver.Security{
		Admins: driver.Members{
			Names: in.Admins.Names,
			Roles: in.Admins.Roles,
		},
		Members: driver.Members{
			Names: in.Members.Names,
			Roles: in.Members.Roles,
		},
	}
}

func (d *db) Security(_ context.Context) (*driver.Security, error) {
	d.db.mu.RLock()
	defer d.db.mu.RUnlock()
	if d.db.deleted {
		return nil, errors.Status(kivik.StatusNotFound, "missing")
	}
	return cloneSecurity(d.db.security), nil
}

func (d *db) SetSecurity(_ context.Context, sec *driver.Security) error {
	d.db.mu.Lock()
	defer d.db.mu.Unlock()
	if d.db.deleted {
		return errors.Status(kivik.StatusNotFound, "missing")
	}
	d.db.security = cloneSecurity(sec)
	return nil
}
