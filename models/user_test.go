package models

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
)

func TestUserInfo_ReturnsNilOnAPIError(t *testing.T) {
	// Create a fake Slack API that always returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":false,"error":"user_not_found"}`))
	}))
	defer server.Close()

	api := slack.New("xoxb-fake", slack.OptionAPIURL(server.URL+"/"))

	user := User{Uuid: "U_NONEXISTENT"}
	info := user.Info(*api)

	assert.Nil(t, info)
}

func TestUserInfo_ReturnsUserOnSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"ok": true,
			"user": {
				"id": "U123",
				"name": "testuser",
				"real_name": "Test User",
				"tz": "America/New_York",
				"locale": "en-US",
				"profile": {
					"email": "test@example.com"
				}
			}
		}`))
	}))
	defer server.Close()

	api := slack.New("xoxb-fake", slack.OptionAPIURL(server.URL+"/"))

	user := User{Uuid: "U123"}
	info := user.Info(*api)

	assert.NotNil(t, info)
	assert.Equal(t, "Test User", info.RealName)
	assert.Equal(t, "America/New_York", info.TZ)
	assert.Equal(t, "test@example.com", info.Profile.Email)
}
