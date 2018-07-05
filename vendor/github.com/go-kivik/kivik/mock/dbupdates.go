package mock

import "github.com/go-kivik/kivik/driver"

// DBUpdates mocks driver.DBUpdates
type DBUpdates struct {
	// ID identifies a specific DBUpdates instance.
	ID        string
	NextFunc  func(*driver.DBUpdate) error
	CloseFunc func() error
}

var _ driver.DBUpdates = &DBUpdates{}

// Next calls u.NextFunc
func (u *DBUpdates) Next(dbupdate *driver.DBUpdate) error {
	return u.NextFunc(dbupdate)
}

// Close calls u.CloseFunc
func (u *DBUpdates) Close() error {
	return u.CloseFunc()
}
