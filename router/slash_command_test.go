package router

import (
	"testing"

	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
)

func TestSlashCommandRoute_Execute(t *testing.T) {
	pluginCalled := false

	route := SlashCommandRoute{
		Route: Route{
			Name:        "test-slash",
			Description: "A test slash command",
		},
		Command: "/test",
		Plugin: func(ctx HandlerContext, cmd slack.SlashCommand) {
			pluginCalled = true
		},
	}

	ctx := HandlerContext{BotClient: &slack.Client{}}
	route.Execute(ctx, slack.SlashCommand{})
	assert.True(t, pluginCalled, "expected Plugin function to be called")
}

func TestSlashCommandRoute_ImmediateResponse(t *testing.T) {
	route := SlashCommandRoute{
		Route: Route{
			Name:        "deploy",
			Description: "Deploy the app",
		},
		Command:           "/deploy",
		ImmediateResponse: func() string { return "Deploying..." },
		Plugin: func(ctx HandlerContext, cmd slack.SlashCommand) {
		},
	}

	assert.Equal(t, "Deploying...", route.ImmediateResponse())
}
