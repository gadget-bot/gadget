package permission_denied

import (
	"github.com/gadget-bot/gadget/router"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

func GetMentionRoute() *router.MentionRoute {
	var pluginRoute router.MentionRoute
	pluginRoute.Permissions = append(pluginRoute.Permissions, "*")
	pluginRoute.Name = "permission_denied"
	pluginRoute.Plugin = func(api slack.Client, router router.Router, ev slackevents.AppMentionEvent, message string) {
		msgRef := slack.NewRefToMessage(ev.Channel, ev.TimeStamp)
		api.AddReaction("astonished", msgRef)
		api.PostMessage(
			ev.Channel,
			slack.MsgOptionText("I'm sorry, <@"+ev.User+">, but you're not allowed to do that.", false),
		)
	}
	return &pluginRoute
}
