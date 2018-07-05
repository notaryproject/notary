package mock

import "github.com/go-kivik/kivik/driver"

// Attachments mocks driver.Attachments
type Attachments struct {
	// ID identifies a specific Attachments instance
	ID        string
	NextFunc  func(*driver.Attachment) error
	CloseFunc func() error
}

var _ driver.Attachments = &Attachments{}

// Next calls a.NextFunc
func (a *Attachments) Next(att *driver.Attachment) error {
	return a.NextFunc(att)
}

// Close calls a.CloseFunc
func (a *Attachments) Close() error {
	return a.CloseFunc()
}
