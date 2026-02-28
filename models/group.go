package models

import (
	"gorm.io/gorm"
)

type Group struct {
	gorm.Model
	Name    string `gorm:"index:,unique"`
	Members []User `gorm:"many2many:user_groups;"`
}

func (g Group) HasMember(user User) bool {
	hasMember := false
	for _, member := range g.Members {
		if member.Uuid == user.Uuid {
			hasMember = true
			break
		}
	}
	return hasMember
}
