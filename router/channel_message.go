package router

import (
	"github.com/slack-go/slack/slackevents"
)

// ChannelMessageRoute handles the `message.channels` Event
type ChannelMessageRoute struct {
	Route
	Plugin func(ctx HandlerContext, ev slackevents.MessageEvent, message string)
}

// channelMessageRoutesSortedByPriority implements Sort such that those with higher priority are first
type channelMessageRoutesSortedByPriority []ChannelMessageRoute

// Execute calls Plugin()
func (route ChannelMessageRoute) Execute(ctx HandlerContext, ev slackevents.MessageEvent, message string) {
	ctx.Route = route.Route
	route.Plugin(ctx, ev, message)
}

func (a channelMessageRoutesSortedByPriority) Len() int { return len(a) }

func (a channelMessageRoutesSortedByPriority) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a channelMessageRoutesSortedByPriority) Less(i, j int) bool {
	return a[i].Priority > a[j].Priority
}
