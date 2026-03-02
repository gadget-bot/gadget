package router

import (
	"github.com/rs/zerolog"
	"github.com/slack-go/slack"
)

// HandlerContext provides dependencies to plugin handlers.
// New fields can be added here without changing plugin signatures.
type HandlerContext struct {
	Router    Router
	Route     Route
	BotClient *slack.Client
	Logger    zerolog.Logger
}
