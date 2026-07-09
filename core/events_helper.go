package core

import (
	"github.com/slack-go/slack/slackevents"
)

func userFromInnerEvent(event *slackevents.EventsAPIInnerEvent) string {
	switch ev := event.Data.(type) {
	case *slackevents.AppMentionEvent:
		return ev.User
	case *slackevents.MessageEvent:
		return ev.User
	default:
		return ""
	}
}
