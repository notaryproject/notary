package storage

import (
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
)

// TUFFile represents a TUF file in the database
type TUFFile struct {
	gorm.Model
	Gun     string `sql:"type:varchar(255);not null"`
	Role    string `sql:"type:varchar(255);not null"`
	Version int    `sql:"not null"`
	Sha256  string `sql:"type:varchar(64);"`
	Data    []byte `sql:"type:longblob;not null"`
}

// TableName sets a specific table name for TUFFile
func (g TUFFile) TableName() string {
	return "tuf_files"
}

// Changefeed defines the Change object for SQL databases
type Changefeed struct {
	ID        uint `gorm:"primary_key" sql:"not null"`
	CreatedAt time.Time
	Gun       string `sql:"type:varchar(255);not null"`
	Ver       int    `gorm:"column:version" sql:"not null"`
	Sha256    string `sql:"type:varchar(64);"`
}

// TableName sets a specific table name for Changefeed
func (c Changefeed) TableName() string {
	return "changefeed"
}

// ChangeID returns the unique ID for this update
func (c Changefeed) ChangeID() string { return fmt.Sprintf("%d", c.ID) }

// GUN returns the GUN for this update
func (c Changefeed) GUN() string { return c.Gun }

// Version returns the timestamp version for the published update
func (c Changefeed) Version() int { return c.Ver }

// Checksum returns the timestamp.json checksum for the published update
func (c Changefeed) Checksum() string { return c.Sha256 }

// RecordedAt returns the time at which the update was recorded
func (c Changefeed) RecordedAt() time.Time { return c.CreatedAt }

// CreateTUFTable creates the DB table for TUFFile
func CreateTUFTable(db gorm.DB) error {
	// TODO: gorm
	query := db.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8").CreateTable(&TUFFile{})
	if query.Error != nil {
		return query.Error
	}
	query = db.Model(&TUFFile{}).AddUniqueIndex(
		"idx_gun", "gun", "role", "version")
	return query.Error
}

// CreateChangefeedTable creates the DB table for Changefeed
func CreateChangefeedTable(db gorm.DB) error {
	query := db.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8").CreateTable(&Changefeed{})
	return query.Error
}
