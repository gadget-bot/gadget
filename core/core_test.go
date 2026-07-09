package core

import (
	"testing"

	"github.com/slack-go/slack/slackevents"
	"github.com/stretchr/testify/assert"
)

func TestSetupWithConfig_PopulatesClientField(t *testing.T) {
	gadget, _ := SetupWithConfig(Config{ //nolint:gosec // test credentials
		SlackOAuthToken: "xoxb-fake-token",
		SigningSecret:   "fake-secret",
		ListenPort:      "3000",
	})

	assert.NotNil(t, gadget.Client, "Expected gadget.Client to be populated after SetupWithConfig")
}

func TestSetupWithConfig_UserClientNilWhenNoUserToken(t *testing.T) {
	gadget, _ := SetupWithConfig(Config{ //nolint:gosec // test credentials
		SlackOAuthToken: "xoxb-fake-token",
		SigningSecret:   "fake-secret",
		ListenPort:      "3000",
	})

	assert.Nil(t, gadget.UserClient, "Expected gadget.UserClient to be nil without SlackUserToken")
}

func TestSetupWithConfig_UserClientPopulatedWhenUserTokenProvided(t *testing.T) {
	gadget, _ := SetupWithConfig(Config{ //nolint:gosec // test credentials
		SlackOAuthToken: "xoxb-fake-token",
		SlackUserToken:  "xoxp-fake-token",
		SigningSecret:   "fake-secret",
		ListenPort:      "3000",
	})

	assert.NotNil(t, gadget.UserClient, "Expected gadget.UserClient to be populated with SlackUserToken")
}

func TestConfigFromEnv_ReadsSlackUserToken(t *testing.T) {
	t.Setenv("SLACK_USER_OAUTH_TOKEN", "xoxp-from-env")

	cfg := ConfigFromEnv()

	assert.Equal(t, "xoxp-from-env", cfg.SlackUserToken)
}

func TestConfigFromEnv_ReadsDBPort(t *testing.T) {
	t.Setenv("GADGET_DB_PORT", "3307")

	cfg := ConfigFromEnv()

	assert.Equal(t, "3307", cfg.DBPort)
}

func TestGlobalAdminsFromString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"empty string", "", []string{}},
		{"single value", "U123", []string{"U123"}},
		{"multiple values", "U123,U456,U789", []string{"U123", "U456", "U789"}},
		{"values with whitespace", " U123 , U456 , U789 ", []string{"U123", "U456", "U789"}},
		{"trailing comma", "U123,U456,", []string{"U123", "U456"}},
		{"only commas", ",,", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := globalAdminsFromString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUnsupportedEventType(t *testing.T) {
	tests := []struct {
		name     string
		event    *slackevents.EventsAPIInnerEvent
		expected string
	}{
		{
			name: "AppMentionEvent",
			event: &slackevents.EventsAPIInnerEvent{
				Data: &slackevents.AppMentionEvent{
					User: "U123",
				},
			},
			expected: "U123",
		},
		{
			name: "MessageEvent",
			event: &slackevents.EventsAPIInnerEvent{
				Data: &slackevents.MessageEvent{
					User: "U456",
				},
			},
			expected: "U456",
		},
		{
			name: "ChannelCreatedEvent",
			event: &slackevents.EventsAPIInnerEvent{
				Data: &slackevents.ChannelCreatedEvent{
					Channel: slackevents.ChannelCreatedInfo{
						Name: "U789",
					},
				},
			},
			expected: "",
		},
		{
			name: "FileDeletedEvent",
			event: &slackevents.EventsAPIInnerEvent{
				Data: slackevents.FileDeletedEvent{
					FileID: "F123",
				},
			},
			expected: "",
		},
		{
			name: "UnsupportedEvent",
			event: &slackevents.EventsAPIInnerEvent{
				Data: struct{}{},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := userFromInnerEvent(tt.event)
			if got != tt.expected {
				t.Errorf("got: %q, want: %q", got, tt.expected)
			}
		})
	}
}
