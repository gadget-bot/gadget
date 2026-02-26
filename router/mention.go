package router

import (
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

type MentionRoute struct {
	Route
	Plugin func(router Router, route Route, api slack.Client, ev slackevents.AppMentionEvent, message string)
}

// mentionRoutesSortedByPriority implements Sort such that those with higher priority are first
type mentionRoutesSortedByPriority []MentionRoute

// Execute calls Plugin()
func (route MentionRoute) Execute(router Router, api slack.Client, ev slackevents.AppMentionEvent, message string) {
	route.Plugin(router, route.Route, api, ev, message)
}

func (a mentionRoutesSortedByPriority) Len() int { return len(a) }

func (a mentionRoutesSortedByPriority) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a mentionRoutesSortedByPriority) Less(i, j int) bool {
	return a[i].Priority > a[j].Priority
}
