package fallback

import (
	"github.com/gadget-bot/gadget/router"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

func GetMentionRoute() *router.MentionRoute {
	var pluginRoute router.MentionRoute
	pluginRoute.Permissions = append(pluginRoute.Permissions, "*")
	pluginRoute.Name = "fallback"
	pluginRoute.Plugin = func(router router.Router, route router.Route, api slack.Client, ev slackevents.AppMentionEvent, message string) {
		api.PostMessage(
			ev.Channel,
			slack.MsgOptionText("Hi there! I see you sent me a message, <@"+ev.User+">, but I'm not sure what to do with that.", false),
		)
	}
	return &pluginRoute
}
