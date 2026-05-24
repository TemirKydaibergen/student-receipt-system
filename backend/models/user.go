package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Username string `gorm:"unique;not null" json:"Username"`
	Password string `gorm:"not null" json:"-"` // не возвращаем пароль в JSON
	Role     string `gorm:"default:'user'" json:"Role"` // "user" or "admin"
}

