package groups

import (
	"net/http"
	"net/http/httptest"
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

func setupGroupTestDB(t *testing.T) *gorm.DB {
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

func TestGetMentionRoutes_ReturnsAllRoutes(t *testing.T) {
	routes := GetMentionRoutes()

	assert.Len(t, routes, 4)

	expectedNames := []string{
		"groups.getMyGroups",
		"groups.getAllGroups",
		"groups.addUserToGroup",
		"groups.removeUserFromGroup",
	}

	actualNames := make([]string, len(routes))
	for i, route := range routes {
		actualNames[i] = route.Name
	}
	assert.ElementsMatch(t, expectedNames, actualNames, "route names should match regardless of order")
}

func TestGetMyGroups_Metadata(t *testing.T) {
	route := getMyGroups()

	assert.Equal(t, "groups.getMyGroups", route.Name)
	assert.Equal(t, `(?i)^((list )?my groups|which groups am I (in|a member of))[.?]?$`, route.Pattern)
	assert.Equal(t, []string{"*"}, route.Permissions)
	assert.NotNil(t, route.Plugin)
}

func TestGetAllGroups_Metadata(t *testing.T) {
	route := getAllGroups()

	assert.Equal(t, "groups.getAllGroups", route.Name)
	assert.Equal(t, `(?i)^(list|list all|all) groups\.?$`, route.Pattern)
	assert.Equal(t, []string{"admins"}, route.Permissions)
	assert.NotNil(t, route.Plugin)
}

func TestAddUserToGroup_Metadata(t *testing.T) {
	route := addUserToGroup()

	assert.Equal(t, "groups.addUserToGroup", route.Name)
	assert.Equal(t, `(?i)^add <@([a-z0-9]+)> to( group)? ([a-z0-9]+)\.?$`, route.Pattern)
	assert.Equal(t, []string{"admins"}, route.Permissions)
	assert.NotNil(t, route.Plugin)
}

func TestRemoveUserFromGroup_Metadata(t *testing.T) {
	route := removeUserFromGroup()

	assert.Equal(t, "groups.removeUserFromGroup", route.Name)
	assert.Equal(t, `(?i)^remove <@([a-z0-9]+)> from( group)? ([a-z0-9]+)\.?$`, route.Pattern)
	assert.Equal(t, []string{"admins"}, route.Permissions)
	assert.NotNil(t, route.Plugin)
}

func TestGetMyGroups_PostsGroupList(t *testing.T) {
	db := setupGroupTestDB(t)

	// Create user with groups
	user := models.User{Uuid: "U_USER"}
	db.Create(&user)
	group := models.Group{Name: "deployers"}
	db.Create(&group)
	db.Model(&group).Association("Members").Append(&user)

	var messages []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/chat.postMessage" {
			r.ParseForm()
			messages = append(messages, r.FormValue("text"))
		}
		w.Write([]byte(`{"ok":true,"channel":"C123","ts":"1234567890.123456"}`))
	}))
	defer server.Close()

	api := slack.New("xoxb-fake", slack.OptionAPIURL(server.URL+"/"))

	route := getMyGroups()
	r := router.Router{DbConnection: db}
	ev := slackevents.AppMentionEvent{
		User:    "U_USER",
		Channel: "C123",
	}

	route.Plugin(r, route.Route, *api, ev, "my groups")

	// Should post two messages: header + group list
	assert.GreaterOrEqual(t, len(messages), 2)
	assert.Contains(t, messages[1], "deployers")
}

func TestGetMyGroups_NoGroups(t *testing.T) {
	db := setupGroupTestDB(t)

	user := models.User{Uuid: "U_LONELY"}
	db.Create(&user)

	var messages []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/chat.postMessage" {
			r.ParseForm()
			messages = append(messages, r.FormValue("text"))
		}
		w.Write([]byte(`{"ok":true,"channel":"C123","ts":"1234567890.123456"}`))
	}))
	defer server.Close()

	api := slack.New("xoxb-fake", slack.OptionAPIURL(server.URL+"/"))

	route := getMyGroups()
	r := router.Router{DbConnection: db}
	ev := slackevents.AppMentionEvent{
		User:    "U_LONELY",
		Channel: "C123",
	}

	route.Plugin(r, route.Route, *api, ev, "my groups")

	assert.GreaterOrEqual(t, len(messages), 2)
	assert.Contains(t, messages[1], "don't seem to be a member")
}

func TestAddUserToGroup_AddsSuccessfully(t *testing.T) {
	db := setupGroupTestDB(t)

	var postedMessage string
	var addedReaction string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/reactions.add":
			r.ParseForm()
			addedReaction = r.FormValue("name")
		case "/chat.postMessage":
			r.ParseForm()
			postedMessage = r.FormValue("text")
		}
		w.Write([]byte(`{"ok":true,"channel":"C123","ts":"1234567890.123456"}`))
	}))
	defer server.Close()

	api := slack.New("xoxb-fake", slack.OptionAPIURL(server.URL+"/"))

	route := addUserToGroup()
	r := router.Router{DbConnection: db}
	ev := slackevents.AppMentionEvent{
		User:      "U_ADMIN",
		Channel:   "C123",
		TimeStamp: "1234567890.123456",
	}

	route.Plugin(r, route.Route, *api, ev, "add <@u123> to deployers")

	assert.Equal(t, "tada", addedReaction)
	assert.Contains(t, postedMessage, "successfully added")
	assert.Contains(t, postedMessage, "u123")
	assert.Contains(t, postedMessage, "deployers")

	// Verify DB state
	var dbGroup models.Group
	db.Preload("Members").Where("name = ?", "deployers").First(&dbGroup)
	assert.Equal(t, 1, len(dbGroup.Members))
	assert.Equal(t, "u123", dbGroup.Members[0].Uuid)
}

func TestRemoveUserFromGroup_RemovesSuccessfully(t *testing.T) {
	db := setupGroupTestDB(t)

	// Set up user in group
	user := models.User{Uuid: "u123"}
	db.Create(&user)
	group := models.Group{Name: "deployers"}
	db.Create(&group)
	db.Model(&group).Association("Members").Append(&user)

	var postedMessage string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/chat.postMessage" {
			r.ParseForm()
			postedMessage = r.FormValue("text")
		}
		w.Write([]byte(`{"ok":true,"channel":"C123","ts":"1234567890.123456"}`))
	}))
	defer server.Close()

	api := slack.New("xoxb-fake", slack.OptionAPIURL(server.URL+"/"))

	route := removeUserFromGroup()
	r := router.Router{DbConnection: db}
	ev := slackevents.AppMentionEvent{
		User:      "U_ADMIN",
		Channel:   "C123",
		TimeStamp: "1234567890.123456",
	}

	route.Plugin(r, route.Route, *api, ev, "remove <@u123> from deployers")

	assert.Contains(t, postedMessage, "no longer a member")

	// Verify DB state
	db.Preload("Members").First(&group, group.ID)
	assert.Equal(t, 0, len(group.Members))
}

func TestRemoveUserFromGroup_NonexistentGroup(t *testing.T) {
	db := setupGroupTestDB(t)

	var postedMessage string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/chat.postMessage" {
			r.ParseForm()
			postedMessage = r.FormValue("text")
		}
		w.Write([]byte(`{"ok":true,"channel":"C123","ts":"1234567890.123456"}`))
	}))
	defer server.Close()

	api := slack.New("xoxb-fake", slack.OptionAPIURL(server.URL+"/"))

	route := removeUserFromGroup()
	r := router.Router{DbConnection: db}
	ev := slackevents.AppMentionEvent{
		User:      "U_ADMIN",
		Channel:   "C123",
		TimeStamp: "1234567890.123456",
	}

	route.Plugin(r, route.Route, *api, ev, "remove <@u123> from nonexistent")

	assert.Contains(t, postedMessage, "couldn't find a group")
}
