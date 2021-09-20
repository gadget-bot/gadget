package core

import (
	"reflect"

	"github.com/slack-go/slack/slackevents"
)

func userFromInnerEvent(event *slackevents.EventsAPIInnerEvent) string {
	return reflect.ValueOf(event.Data).Elem().FieldByName("User").String()
}
