package models

import (
	"github.com/slack-go/slack"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Uuid   string  `gorm:"index:,unique"`
	Groups []Group `gorm:"many2many:user_groups;"`
}

func (u User) Info(api slack.Client) *slack.User {
	info, _ := api.GetUserInfo(u.Uuid)

	return info
}
