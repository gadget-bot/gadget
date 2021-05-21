package models

import (
	"time"

	"gorm.io/gorm"
)

type Group struct {
	gorm.Model
	ID        uint
	Name      string `gorm:"index:,unique"`
	Members   []User `gorm:"many2many:user_groups;"`
	CreatedAt time.Time
	UpdatedAt time.Time
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
