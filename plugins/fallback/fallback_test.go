package fallback

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
	assert.Equal(t, "fallback", route.Name)
	assert.Empty(t, route.Pattern, "fallback route should have no pattern (matches nothing explicitly)")
	assert.Empty(t, route.Description, "fallback route has no description")
	assert.Empty(t, route.Help, "fallback route has no help text")
	assert.Equal(t, []string{"*"}, route.Permissions)
	assert.NotNil(t, route.Plugin)
}

func TestFallbackPlugin_PostsMessage(t *testing.T) {
	var postedMessage string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chat.postMessage" {
			if err := r.ParseForm(); err != nil {
				t.Fatalf("ParseForm failed: %v", err)
			}
			postedMessage = r.FormValue("text")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true,"channel":"C123","ts":"1234567890.123456"}`)) //nolint:errcheck // test HTTP response on loopback
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`)) //nolint:errcheck // test HTTP response on loopback
	}))
	defer server.Close()

	api := slack.New("xoxb-fake", slack.OptionAPIURL(server.URL+"/"))

	route := GetMentionRoute()
	ev := slackevents.AppMentionEvent{
		User:    "U_USER",
		Channel: "C123",
		Text:    "something unrecognized",
	}

	route.Plugin(router.Router{}, route.Route, *api, ev, "something unrecognized")

	assert.Contains(t, postedMessage, "U_USER")
	assert.Contains(t, postedMessage, "not sure what to do")
}
