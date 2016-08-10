package storage

import "github.com/jinzhu/gorm"

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

// CreateTUFTable creates the DB table for TUFFile
func CreateTUFTable(db gorm.DB) error {
	// TODO: gorm
	query := db.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8").CreateTable(&TUFFile{})
	if query.Error != nil {
		return query.Error
	}
	query = db.Model(&TUFFile{}).AddUniqueIndex(
		"idx_gun", "gun", "role", "version")
	if query.Error != nil {
		return query.Error
	}
	return nil
}
