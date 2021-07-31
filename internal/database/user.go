package database

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Name         string
	PasswordHash string
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

func (d Database) CreateUser(username, passwordHash string) *User {
	user := User{Name: username, PasswordHash: passwordHash}
	db := d.db.Create(&user)
	if db.Error != nil {
		return nil
	}
	return &user
}
