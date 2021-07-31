package database

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Database struct {
	db *gorm.DB
}

func New(filename string) (Database, error) {
	db, err := gorm.Open(sqlite.Open(filename), &gorm.Config{})
	if err != nil {
		return Database{}, err
	}

	db.AutoMigrate(
		&User{},
		&setting{},
	)

	return Database{db: db}, nil
}
