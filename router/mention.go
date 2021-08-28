package router

import (
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

type MentionRoute struct {
	Route
	// Plugin func(api slack.Client, router *Router, ev slackevents.AppMentionEvent, message string)
	Plugin func(router Router, route Route, api slack.Client, ev slackevents.AppMentionEvent, message string)
}

func (router Router) AddMentionRoute(route MentionRoute) {
	router.MentionRoutes[route.Name] = route
}

func (router Router) AddMentionRoutes(routes []MentionRoute) {
	for _, route := range routes {
		router.MentionRoutes[route.Name] = route
	}
}
