package database

import (
	"strings"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Name         string `gorm:"uniqueIndex"`
	PasswordHash string
	Roles        string
}

func (u *User) HasRole(role string) bool {
	for _, r := range strings.Split(u.Roles, ",") {
		if r == role {
			return true
		}
	}
	return false
}

func (d Database) HasUsers() bool {
	user := User{}
	tx := d.db.First(&user)
	return tx.Error == nil
}

func (d Database) GetUserByName(username string) *User {
	ret := new(User)
	tx := d.db.Where(&User{Name: username}).First(ret)
	if tx.Error != nil {
		return nil
	}
	return ret
}

func (d Database) CreateUser(username, passwordHash, roles string) *User {
	user := User{Name: username, PasswordHash: passwordHash, Roles: roles}
	db := d.db.Create(&user)
	if db.Error != nil {
		return nil
	}
	return &user
}
