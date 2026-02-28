package user_info

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/gadget-bot/gadget/models"
	"github.com/gadget-bot/gadget/router"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func setupUserInfoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to open in-memory SQLite: %v", err)
	}
	if err := db.AutoMigrate(&models.Group{}, &models.User{}); err != nil {
		t.Fatalf("Failed to auto-migrate: %v", err)
	}
	return db
}

func compileMentionRouteForTest(t *testing.T, route *router.MentionRoute) {
	t.Helper()
	route.CompiledPattern = regexp.MustCompile(route.Pattern)
}

func TestGetMentionRoutes_ReturnsOneRoute(t *testing.T) {
	routes := GetMentionRoutes()

	assert.Len(t, routes, 1)

	route := routes[0]
	assert.Equal(t, "user_info.userInfo", route.Name)
	assert.Equal(t, []string{"admins"}, route.Permissions)
	assert.Equal(t, `(?i)^(tell me about|who is) <@([a-z0-9]+)>[.?]?$`, route.Pattern)
	assert.Equal(t, "Responds with information about a Slack user", route.Description)
	assert.Equal(t, "who is USER", route.Help)
	assert.NotNil(t, route.Plugin)
}

func TestUserInfoPlugin_PostsUserDetails(t *testing.T) {
	db := setupUserInfoTestDB(t)

	// Create user in DB
	user := models.User{Uuid: "u456"}
	db.Create(&user)

	var postedMessage string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/users.info":
			w.Write([]byte(`{
				"ok": true,
				"user": {
					"id": "u456",
					"name": "testuser",
					"real_name": "Test User",
					"tz": "America/Chicago",
					"locale": "en-US",
					"profile": {"email": "test@example.com"}
				}
			}`))
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

	route := userInfo()
	compileMentionRouteForTest(t, route)
	r := router.Router{DbConnection: db}
	ev := slackevents.AppMentionEvent{
		User:    "U_ADMIN",
		Channel: "C123",
	}

	route.Plugin(r, route.Route, *api, ev, "who is <@u456>")

	assert.Contains(t, postedMessage, "Test User")
	assert.Contains(t, postedMessage, "America/Chicago")
	assert.Contains(t, postedMessage, "test@example.com")
}

func TestUserInfoPlugin_UserNotFound(t *testing.T) {
	db := setupUserInfoTestDB(t)

	var postedMessage string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/users.info":
			w.Write([]byte(`{"ok":false,"error":"user_not_found"}`))
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

	route := userInfo()
	compileMentionRouteForTest(t, route)
	r := router.Router{DbConnection: db}
	ev := slackevents.AppMentionEvent{
		User:    "U_ADMIN",
		Channel: "C123",
	}

	route.Plugin(r, route.Route, *api, ev, "who is <@u999>")

	assert.Contains(t, postedMessage, "couldn't look up")
}
