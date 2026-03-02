package user_info

import (
	"fmt"
	"math/rand/v2"

	"github.com/gadget-bot/gadget/models"
	"github.com/gadget-bot/gadget/plugins/helpers"
	"github.com/gadget-bot/gadget/router"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

func userInfo() *router.MentionRoute {
	var pluginRoute router.MentionRoute
	pluginRoute.Permissions = append(pluginRoute.Permissions, "admins")
	pluginRoute.Name = "user_info.userInfo"
	pluginRoute.Description = "Responds with information about a Slack user"
	pluginRoute.Help = "who is USER"
	pluginRoute.Pattern = `(?i)^(tell me about|who is) <@([a-z0-9]+)>[.?]?$`
	pluginRoute.Plugin = func(ctx router.HandlerContext, ev slackevents.AppMentionEvent, message string) {
		results := ctx.Route.CompiledPattern.FindStringSubmatch(message)
		userName := results[2]
		var foundUser models.User
		var response string

		animals := []string{
			"Giant Panda",
			"Blue Whale",
			"Bengal Tiger",
			"Asian Elephant",
			"Gorilla",
			"Snow Leopard",
			"Orangutan",
			"Sea Turtle",
			"Black Rhino",
			"African Penguin",
			"Red Panda",
			"Polar Bear",
		}

		randomIndex := rand.IntN(len(animals)) //nolint:gosec // G404: random animal selection has no security requirement
		randomAnimal := animals[randomIndex]

		ctx.Router.DbConnection.Where(models.User{Uuid: userName}).FirstOrCreate(&foundUser)

		threadOpt := helpers.ThreadReplyOption(ev.ThreadTimeStamp)

		slackInfo := foundUser.Info(*ctx.BotClient)
		if slackInfo == nil {
			helpers.PostMessage(*ctx.BotClient, ev.Channel, "user_info",
				slack.MsgOptionText(fmt.Sprintf("Sorry, I couldn't look up info for <@%s>.", userName), false),
				threadOpt,
			)
			return
		}
		response += fmt.Sprintf("- *Real Name:* %s\n", slackInfo.RealName)
		response += fmt.Sprintf("- *Time Zone:* %s\n", slackInfo.TZ)
		response += fmt.Sprintf("- *Email:* %s\n", slackInfo.Profile.Email)
		response += fmt.Sprintf("- *Locale:* %s\n", slackInfo.Locale)
		response += fmt.Sprintf("- *Spirit Animal:* %s\n", randomAnimal)

		helpers.PostMessage(*ctx.BotClient, ev.Channel, "user_info",
			slack.MsgOptionText(response, false),
			threadOpt,
		)
	}
	return &pluginRoute
}

func GetMentionRoutes() []router.MentionRoute {
	return []router.MentionRoute{
		*userInfo(),
	}
}
