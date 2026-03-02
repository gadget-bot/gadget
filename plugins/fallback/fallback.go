package fallback

import (
	"github.com/gadget-bot/gadget/plugins/helpers"
	"github.com/gadget-bot/gadget/router"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

func GetMentionRoute() *router.MentionRoute {
	var pluginRoute router.MentionRoute
	pluginRoute.Permissions = append(pluginRoute.Permissions, "*")
	pluginRoute.Name = "fallback"
	pluginRoute.Plugin = func(ctx router.HandlerContext, ev slackevents.AppMentionEvent, message string) {
		helpers.PostMessage(*ctx.BotClient, ev.Channel, "fallback",
			slack.MsgOptionText("Hi there! I see you sent me a message, <@"+ev.User+">, but I'm not sure what to do with that.", false),
			helpers.ThreadReplyOption(ev.ThreadTimeStamp),
		)
	}
	return &pluginRoute
}
