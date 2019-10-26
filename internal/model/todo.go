package model

import (
	"github.com/jinzhu/gorm"
)

type Todo struct {
	gorm.Model

	Name string `json:"name"`
	Done bool   `json:"done"`

	User   User `json:"-"`
	UserID uint `json:"-"`
}