package mock

import (
	"context"

	"github.com/go-kivik/kivik/driver"
)

// Sessioner mocks driver.Client and driver.Sessioner
type Sessioner struct {
	*Client
	SessionFunc func(context.Context) (*driver.Session, error)
}

var _ driver.Sessioner = &Sessioner{}

// Session calls s.SessionFunc
func (s *Sessioner) Session(ctx context.Context) (*driver.Session, error) {
	return s.SessionFunc(ctx)
}
