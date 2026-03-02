package core

import (
	"testing"

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
