package eightball

import (
	"math/rand"

	"github.com/gadget-bot/gadget/router"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

func getEightballAnswer() string {
	answers := []string{
		"It is certain",
		"It is decidedly so",
		"Without a doubt",
		"Yes - definitely",
		"You may rely on it",
		"As I see it, yes",
		"Most likely",
		"Outlook good",
		"Signs point to yes",
		"Yes",
		"Reply hazy, try again",
		"Ask again later",
		"Better not tell you now",
		"Cannot predict now",
		"Concentrate and ask again",
		"Don't count on it",
		"My reply is no",
		"My sources say no",
		"Outlook not so good",
		"Very doubtful",
	}

	return answers[rand.Intn(len(answers))]
}

func askEightball() *router.MentionRoute {
	var pluginRoute router.MentionRoute
	pluginRoute.Permissions = append(pluginRoute.Permissions, "*")
	pluginRoute.Name = "eightball.askEightball"
	pluginRoute.Description = "Asks a magic 8-ball a question"
	pluginRoute.Help = "Will|Can|Am I ... ?"
	pluginRoute.Priority = -10
	pluginRoute.Pattern = `(?i)^(will|can|am I) .+[?]?$`
	pluginRoute.Plugin = func(api slack.Client, router router.Router, ev slackevents.AppMentionEvent, message string) {
		// Here's how we can react to the message
		msgRef := slack.NewRefToMessage(ev.Channel, ev.TimeStamp)
		api.AddReaction("8ball", msgRef)

		// Here's how we send a reply
		api.PostMessage(
			ev.Channel,
			slack.MsgOptionText(
				getEightballAnswer(),
				false,
			),
		)
	}

	// We've got to return the MentionRoute
	return &pluginRoute
}

// This function is used to retrieve all Mention Routes from this plugin
func GetMentionRoutes() []router.MentionRoute {
	return []router.MentionRoute{
		*askEightball(),
	}
}
