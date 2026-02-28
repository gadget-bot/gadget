package permission_denied

import (
	"github.com/gadget-bot/gadget/plugins/helpers"
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
		log.Warn().Str("user", ev.User).Str("channel", ev.Channel).Msg("Mention permission denied")
		helpers.AddReaction(api, ev.Channel, "permission_denied", "astonished", ev.TimeStamp)
		helpers.PostMessage(api, ev.Channel, "permission_denied",
			slack.MsgOptionText("I'm sorry, <@"+ev.User+">, but you're not allowed to do that.", false),
			helpers.ThreadReplyOption(ev.ThreadTimeStamp),
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
		helpers.AddReaction(api, ev.Channel, "permission_denied", "astonished", ev.TimeStamp)
		helpers.PostMessage(api, ev.Channel, "permission_denied",
			slack.MsgOptionText("I'm sorry, <@"+ev.User+">, but you're not allowed to do that.", false),
			helpers.ThreadReplyOption(ev.ThreadTimeStamp),
		)
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
