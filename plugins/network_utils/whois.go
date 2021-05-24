package network_utils

import (
	"fmt"
	"regexp"

	"github.com/gadget-bot/gadget/router"
	"github.com/likexian/whois"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

func queryWhois() *router.MentionRoute {
	var pluginRoute router.MentionRoute
	pluginRoute.Permissions = append(pluginRoute.Permissions, "*")
	pluginRoute.Name = "network_utils.queryWhois"
	pluginRoute.Pattern = `(?i)^whois (.+)$`
	pluginRoute.Description = "Looks up WHOIS info for a given domain or IP"
	pluginRoute.Help = "whois <DOMAIN|IP>"
	pluginRoute.Plugin = func(api slack.Client, router router.Router, ev slackevents.AppMentionEvent, message string) {
		// Here's how we can react to the message
		msgRef := slack.NewRefToMessage(ev.Channel, ev.TimeStamp)
		api.AddReaction("male-detective", msgRef)

		re := regexp.MustCompile(pluginRoute.Pattern)
		results := re.FindStringSubmatch(message)
		input := results[1]

		result, err := whois.Whois(input)
		if err == nil {
			result = fmt.Sprintf("Something went wrong looking up WHOIS info for '%s'", input)
		}

		// Here's how we send a reply
		api.PostMessage(
			ev.Channel,
			slack.MsgOptionText(
				result,
				false,
			),
		)
	}

	// We've got to return the MentionRoute
	return &pluginRoute
}
