package router

import "github.com/slack-go/slack"

// SlashCommandRoute handles Slack slash command invocations.
// Plugin execution is dispatched asynchronously in a goroutine, so the HTTP
// handler acknowledges the command with an empty 200 within Slack's 3-second
// deadline. For commands that need to send a visible response, the Plugin
// should post a follow-up message using the Slack API (e.g. chat.postMessage
// or the slash command's ResponseURL).
type SlashCommandRoute struct {
	Route
	Command           string        // Slack command name, e.g. "/deploy"
	ImmediateResponse func() string // Optional ephemeral response evaluated per-request; nil means no response
	Plugin            func(ctx HandlerContext, cmd slack.SlashCommand)
}

// Execute calls Plugin()
func (route SlashCommandRoute) Execute(ctx HandlerContext, cmd slack.SlashCommand) {
	ctx.Route = route.Route
	route.Plugin(ctx, cmd)
}
