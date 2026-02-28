package permission_denied

import (
	"github.com/gadget-bot/gadget/router"
	"github.com/rs/zerolog/log"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

func GetMentionRoute() *router.MentionRoute {
	var pluginRoute router.MentionRoute
	pluginRoute.Permissions = append(pluginRoute.Permissions, "*")
	pluginRoute.Name = "permission_denied"
	pluginRoute.Plugin = func(router router.Router, route router.Route, api slack.Client, ev slackevents.AppMentionEvent, message string) {
		msgRef := slack.NewRefToMessage(ev.Channel, ev.TimeStamp)
		api.AddReaction("astonished", msgRef)
		api.PostMessage(
			ev.Channel,
			slack.MsgOptionText("I'm sorry, <@"+ev.User+">, but you're not allowed to do that.", false),
		)
	}
	return &pluginRoute
}

func GetChannelMessageRoute() *router.ChannelMessageRoute {
	var pluginRoute router.ChannelMessageRoute
	pluginRoute.Permissions = append(pluginRoute.Permissions, "*")
	pluginRoute.Name = "permission_denied"
	pluginRoute.Plugin = func(r router.Router, route router.Route, api slack.Client, ev slackevents.MessageEvent, message string) {
		log.Warn().Str("user", ev.User).Str("channel", ev.Channel).Msg("Channel message permission denied")
	}
	return &pluginRoute
}

func GetSlashCommandRoute() *router.SlashCommandRoute {
	var pluginRoute router.SlashCommandRoute
	pluginRoute.Permissions = append(pluginRoute.Permissions, "*")
	pluginRoute.Name = "permission_denied"
	pluginRoute.Plugin = func(r router.Router, route router.Route, api slack.Client, cmd slack.SlashCommand) {
		log.Warn().Str("user", cmd.UserID).Str("command", cmd.Command).Msg("Slash command permission denied")
	}
	return &pluginRoute
}
