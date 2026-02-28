package models

import (
	"github.com/rs/zerolog/log"
	"github.com/slack-go/slack"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Uuid   string  `gorm:"index:,unique"`
	Groups []Group `gorm:"many2many:user_groups;"`
}

func (u User) Info(api slack.Client) *slack.User {
	info, err := api.GetUserInfo(u.Uuid)
	if err != nil {
		log.Warn().Err(err).Str("uuid", u.Uuid).Msg("Failed to get user info")
		return nil
	}

	return info
}
