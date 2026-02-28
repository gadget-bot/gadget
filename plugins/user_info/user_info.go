package user_info

import (
	"fmt"
	"math/rand"
	"regexp"

	"github.com/gadget-bot/gadget/models"
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
	pluginRoute.Plugin = func(router router.Router, route router.Route, api slack.Client, ev slackevents.AppMentionEvent, message string) {
		re := regexp.MustCompile(route.Pattern)
		results := re.FindStringSubmatch(message)
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

		randomIndex := rand.Intn(len(animals))
		randomAnimal := animals[randomIndex]

		router.DbConnection.Where(models.User{Uuid: userName}).FirstOrCreate(&foundUser)

		slackInfo := foundUser.Info(api)
		if slackInfo == nil {
			api.PostMessage(
				ev.Channel,
				slack.MsgOptionText(fmt.Sprintf("Sorry, I couldn't look up info for <@%s>.", userName), false),
			)
			return
		}
		response += fmt.Sprintf("- *Real Name:* %s\n", slackInfo.RealName)
		response += fmt.Sprintf("- *Time Zone:* %s\n", slackInfo.TZ)
		response += fmt.Sprintf("- *Email:* %s\n", slackInfo.Profile.Email)
		response += fmt.Sprintf("- *Locale:* %s\n", slackInfo.Locale)
		response += fmt.Sprintf("- *Spirit Animal:* %s\n", randomAnimal)

		api.PostMessage(
			ev.Channel,
			slack.MsgOptionText(response, false),
		)
	}
	return &pluginRoute
}

func GetMentionRoutes() []router.MentionRoute {
	return []router.MentionRoute{
		*userInfo(),
	}
}
