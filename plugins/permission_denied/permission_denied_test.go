package permission_denied

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gadget-bot/gadget/router"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/stretchr/testify/assert"
)

func TestGetMentionRoute_Metadata(t *testing.T) {
	route := GetMentionRoute()

	assert.NotNil(t, route)
	assert.Equal(t, "permission_denied", route.Name)
	assert.Empty(t, route.Pattern, "permission_denied mention route should have no pattern")
	assert.Empty(t, route.Description, "permission_denied mention route has no description")
	assert.Empty(t, route.Help, "permission_denied mention route has no help text")
	assert.Equal(t, []string{"*"}, route.Permissions)
	assert.NotNil(t, route.Plugin)
}

func TestGetChannelMessageRoute_Metadata(t *testing.T) {
	route := GetChannelMessageRoute()

	assert.NotNil(t, route)
	assert.Equal(t, "permission_denied", route.Name)
	assert.Empty(t, route.Pattern, "permission_denied channel message route should have no pattern")
	assert.Empty(t, route.Description, "permission_denied channel message route has no description")
	assert.Empty(t, route.Help, "permission_denied channel message route has no help text")
	assert.Equal(t, []string{"*"}, route.Permissions)
	assert.NotNil(t, route.Plugin)
}

func TestGetSlashCommandRoute_Metadata(t *testing.T) {
	route := GetSlashCommandRoute()

	assert.NotNil(t, route)
	assert.Equal(t, "permission_denied", route.Name)
	assert.Empty(t, route.Pattern, "permission_denied slash command route should have no pattern")
	assert.Empty(t, route.Description, "permission_denied slash command route has no description")
	assert.Empty(t, route.Help, "permission_denied slash command route has no help text")
	assert.Equal(t, []string{"*"}, route.Permissions)
	assert.NotNil(t, route.Plugin)
}

func TestMentionPlugin_AddsReactionAndPostsMessage(t *testing.T) {
	var postedMessage string
	var addedReaction string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/reactions.add":
			r.ParseForm()
			addedReaction = r.FormValue("name")
			w.Write([]byte(`{"ok":true}`))
		case "/chat.postMessage":
			r.ParseForm()
			postedMessage = r.FormValue("text")
			w.Write([]byte(`{"ok":true,"channel":"C123","ts":"1234567890.123456"}`))
		default:
			w.Write([]byte(`{"ok":true}`))
		}
	}))
	defer server.Close()

	api := slack.New("xoxb-fake", slack.OptionAPIURL(server.URL+"/"))

	route := GetMentionRoute()
	ev := slackevents.AppMentionEvent{
		User:      "U_USER",
		Channel:   "C123",
		TimeStamp: "1234567890.123456",
	}

	route.Plugin(router.Router{}, route.Route, *api, ev, "restricted command")

	assert.Equal(t, "astonished", addedReaction)
	assert.Contains(t, postedMessage, "U_USER")
	assert.Contains(t, postedMessage, "not allowed")
}

func TestChannelMessagePlugin_AddsReactionAndPostsMessage(t *testing.T) {
	var postedMessage string
	var addedReaction string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/reactions.add":
			r.ParseForm()
			addedReaction = r.FormValue("name")
			w.Write([]byte(`{"ok":true}`))
		case "/chat.postMessage":
			r.ParseForm()
			postedMessage = r.FormValue("text")
			w.Write([]byte(`{"ok":true,"channel":"C123","ts":"1234567890.123456"}`))
		default:
			w.Write([]byte(`{"ok":true}`))
		}
	}))
	defer server.Close()

	api := slack.New("xoxb-fake", slack.OptionAPIURL(server.URL+"/"))

	route := GetChannelMessageRoute()
	ev := slackevents.MessageEvent{
		User:      "U_USER",
		Channel:   "C123",
		TimeStamp: "1234567890.123456",
	}

	route.Plugin(router.Router{}, route.Route, *api, ev, "restricted command")

	assert.Equal(t, "astonished", addedReaction)
	assert.Contains(t, postedMessage, "U_USER")
	assert.Contains(t, postedMessage, "not allowed")
}
