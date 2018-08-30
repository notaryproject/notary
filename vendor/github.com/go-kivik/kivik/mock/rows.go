package mock

import "github.com/go-kivik/kivik/driver"

// Rows mocks driver.Rows
type Rows struct {
	// ID identifies a specific Rows instance.
	ID            string
	CloseFunc     func() error
	NextFunc      func(*driver.Row) error
	OffsetFunc    func() int64
	TotalRowsFunc func() int64
	UpdateSeqFunc func() string
}

var _ driver.Rows = &Rows{}

// Close calls r.CloseFunc
func (r *Rows) Close() error {
	return r.CloseFunc()
}

// Next calls r.NextFunc
func (r *Rows) Next(row *driver.Row) error {
	return r.NextFunc(row)
}

// Offset calls r.OffsetFunc
func (r *Rows) Offset() int64 {
	return r.OffsetFunc()
}

// TotalRows calls r.TotalRowsFunc
func (r *Rows) TotalRows() int64 {
	return r.TotalRowsFunc()
}

// UpdateSeq calls r.UpdateSeqFunc
func (r *Rows) UpdateSeq() string {
	return r.UpdateSeqFunc()
}

// RowsWarner wraps driver.RowsWarner
type RowsWarner struct {
	*Rows
	WarningFunc func() string
}

var _ driver.RowsWarner = &RowsWarner{}

// Warning calls r.WarningFunc
func (r *RowsWarner) Warning() string {
	return r.WarningFunc()
}

// Bookmarker wraps driver.Bookmarker
type Bookmarker struct {
	*Rows
	BookmarkFunc func() string
}

var _ driver.Bookmarker = &Bookmarker{}

// Bookmark calls r.BookmarkFunc
func (r *Bookmarker) Bookmark() string {
	return r.BookmarkFunc()
}
