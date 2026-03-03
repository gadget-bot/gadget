package manifest

import (
	"encoding/json"
	"testing"

	"github.com/gadget-bot/gadget/router"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate_MentionRoutes(t *testing.T) {
	r := *router.NewRouter()
	r.AddMentionRoute(router.MentionRoute{
		Route: router.Route{Name: "greet", Pattern: `(?i)^hello`},
	})

	m := Generate(r, "TestBot", "A test bot", "https://example.com")

	assert.Equal(t, "TestBot", m.DisplayInfo.Name)
	assert.Equal(t, "A test bot", m.DisplayInfo.Description)
	assert.Contains(t, m.Settings.EventSubscriptions.BotEvents, "app_mention")
	assert.NotContains(t, m.Settings.EventSubscriptions.BotEvents, "message.channels")
	assert.Contains(t, m.OAuthConfig.Scopes.Bot, "app_mentions:read")
	assert.Contains(t, m.OAuthConfig.Scopes.Bot, "chat:write")
	assert.Equal(t, "https://example.com/gadget", m.Settings.EventSubscriptions.RequestURL)
}

func TestGenerate_ChannelMessageRoutes(t *testing.T) {
	r := *router.NewRouter()
	r.AddChannelMessageRoute(router.ChannelMessageRoute{
		Route: router.Route{Name: "deploy", Pattern: `(?i)^deploy`},
	})

	m := Generate(r, "DeployBot", "", "https://example.com")

	assert.Contains(t, m.Settings.EventSubscriptions.BotEvents, "message.channels")
	assert.NotContains(t, m.Settings.EventSubscriptions.BotEvents, "app_mention")
	assert.Contains(t, m.OAuthConfig.Scopes.Bot, "channels:history")
}

func TestGenerate_SlashCommandRoutes(t *testing.T) {
	r := *router.NewRouter()
	r.AddSlashCommandRoute(router.SlashCommandRoute{
		Route:   router.Route{Name: "deploy-cmd", Description: "Deploy the app"},
		Command: "/deploy",
	})

	m := Generate(r, "DeployBot", "", "https://example.com")

	require.Len(t, m.Features.Slash, 1)
	assert.Equal(t, "/deploy", m.Features.Slash[0].Command)
	assert.Equal(t, "Deploy the app", m.Features.Slash[0].Description)
	assert.Contains(t, m.OAuthConfig.Scopes.Bot, "commands")
	require.NotNil(t, m.Settings.Interactivity)
	assert.True(t, m.Settings.Interactivity.Enabled)
	assert.Equal(t, "https://example.com/gadget/command", m.Settings.Interactivity.RequestURL)
}

func TestGenerate_SlashCommandFallsBackToName(t *testing.T) {
	r := *router.NewRouter()
	r.AddSlashCommandRoute(router.SlashCommandRoute{
		Route:   router.Route{Name: "help"},
		Command: "/help",
	})

	m := Generate(r, "Bot", "", "")

	require.Len(t, m.Features.Slash, 1)
	assert.Equal(t, "help", m.Features.Slash[0].Description)
}

func TestGenerate_ExtraScopes(t *testing.T) {
	r := *router.NewRouter()
	r.AddMentionRoute(router.MentionRoute{
		Route: router.Route{Name: "test", Pattern: `test`},
	})

	m := Generate(r, "Bot", "", "", "users:read", "reactions:write")

	assert.Contains(t, m.OAuthConfig.Scopes.Bot, "users:read")
	assert.Contains(t, m.OAuthConfig.Scopes.Bot, "reactions:write")
}

func TestGenerate_ExtraScopesDedup(t *testing.T) {
	r := *router.NewRouter()
	r.AddMentionRoute(router.MentionRoute{
		Route: router.Route{Name: "test", Pattern: `test`},
	})

	m := Generate(r, "Bot", "", "", "chat:write")

	count := 0
	for _, s := range m.OAuthConfig.Scopes.Bot {
		if s == "chat:write" {
			count++
		}
	}
	assert.Equal(t, 1, count, "chat:write should appear exactly once")
}

func TestGenerate_EmptyRouter(t *testing.T) {
	r := *router.NewRouter()

	m := Generate(r, "EmptyBot", "", "")

	assert.Empty(t, m.Settings.EventSubscriptions.BotEvents)
	assert.Empty(t, m.OAuthConfig.Scopes.Bot)
	assert.Empty(t, m.Features.Slash)
	assert.Nil(t, m.Settings.Interactivity)
}

func TestManifest_JSON(t *testing.T) {
	r := *router.NewRouter()
	r.AddMentionRoute(router.MentionRoute{
		Route: router.Route{Name: "greet", Pattern: `(?i)^hello`},
	})

	m := Generate(r, "TestBot", "A test bot", "https://example.com")
	jsonStr, err := m.JSON()
	require.NoError(t, err)

	// Verify it's valid JSON by round-tripping
	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(jsonStr), &parsed)
	require.NoError(t, err)
	assert.Equal(t, "TestBot", parsed["display_information"].(map[string]interface{})["name"])
}

func TestGenerate_BotUser(t *testing.T) {
	r := *router.NewRouter()
	m := Generate(r, "MyBot", "", "")

	require.NotNil(t, m.Features.BotUser)
	assert.Equal(t, "MyBot", m.Features.BotUser.DisplayName)
	assert.True(t, m.Features.BotUser.AlwaysOnline)
}
