package gadgettest

import (
	"testing"

	"github.com/gadget-bot/gadget/router"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/stretchr/testify/assert"
)

func TestDispatchMention_MatchingRoute(t *testing.T) {
	var called bool
	d := NewDispatcher(
		WithMentionRoutes(router.MentionRoute{
			Route: router.Route{
				Name:    "greet",
				Pattern: `(?i)^hello`,
			},
			Plugin: func(ctx router.HandlerContext, ev slackevents.AppMentionEvent, message string) {
				called = true
				assert.Equal(t, "U_USER", ev.User)
				assert.Equal(t, "hello world", message)
			},
		}),
	)

	err := d.DispatchMention(slackevents.AppMentionEvent{User: "U_USER", Channel: "C123"}, "hello world")
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestDispatchMention_NoMatch(t *testing.T) {
	d := NewDispatcher()
	err := d.DispatchMention(slackevents.AppMentionEvent{}, "unknown")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no matching mention route")
}

func TestDispatchChannelMessage_MatchingRoute(t *testing.T) {
	var called bool
	d := NewDispatcher(
		WithChannelMessageRoutes(router.ChannelMessageRoute{
			Route: router.Route{
				Name:    "deploy",
				Pattern: `(?i)^deploy`,
			},
			Plugin: func(ctx router.HandlerContext, ev slackevents.MessageEvent, message string) {
				called = true
				assert.Equal(t, "deploy prod", message)
			},
		}),
	)

	err := d.DispatchChannelMessage(slackevents.MessageEvent{Channel: "C123"}, "deploy prod")
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestDispatchChannelMessage_NoMatch(t *testing.T) {
	d := NewDispatcher()
	err := d.DispatchChannelMessage(slackevents.MessageEvent{}, "unknown")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no matching channel message route")
}

func TestDispatchSlashCommand_MatchingRoute(t *testing.T) {
	var called bool
	d := NewDispatcher(
		WithSlashCommandRoutes(router.SlashCommandRoute{
			Route:   router.Route{Name: "deploy"},
			Command: "/deploy",
			Plugin: func(ctx router.HandlerContext, cmd slack.SlashCommand) {
				called = true
				assert.Equal(t, "/deploy", cmd.Command)
			},
		}),
	)

	err := d.DispatchSlashCommand(slack.SlashCommand{Command: "/deploy"})
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestDispatchSlashCommand_NoMatch(t *testing.T) {
	d := NewDispatcher()
	err := d.DispatchSlashCommand(slack.SlashCommand{Command: "/unknown"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no matching slash command route")
}

func TestWithBotClient_Available(t *testing.T) {
	client := slack.New("xoxb-test")
	var receivedClient *slack.Client

	d := NewDispatcher(
		WithBotClient(client),
		WithMentionRoutes(router.MentionRoute{
			Route: router.Route{
				Name:    "check-client",
				Pattern: `(?i)^test`,
			},
			Plugin: func(ctx router.HandlerContext, ev slackevents.AppMentionEvent, message string) {
				receivedClient = ctx.BotClient
			},
		}),
	)

	_ = d.DispatchMention(slackevents.AppMentionEvent{}, "test")
	assert.Equal(t, client, receivedClient)
}

func TestWithUserClient_Available(t *testing.T) {
	userClient := slack.New("xoxp-test")
	var receivedClient *slack.Client

	d := NewDispatcher(
		WithUserClient(userClient),
		WithMentionRoutes(router.MentionRoute{
			Route: router.Route{
				Name:    "check-user-client",
				Pattern: `(?i)^test`,
			},
			Plugin: func(ctx router.HandlerContext, ev slackevents.AppMentionEvent, message string) {
				receivedClient = ctx.UserClient
			},
		}),
	)

	_ = d.DispatchMention(slackevents.AppMentionEvent{}, "test")
	assert.Equal(t, userClient, receivedClient)
}
