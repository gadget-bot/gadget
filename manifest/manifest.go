// Package manifest generates Slack app manifests from registered Gadget routes.
package manifest

import (
	"encoding/json"
	"sort"

	"github.com/gadget-bot/gadget/router"
)

// EventSubscriptions represents the event subscriptions section of the manifest.
type EventSubscriptions struct {
	RequestURL string   `json:"request_url,omitempty"`
	BotEvents  []string `json:"bot_events"`
}

// OAuthConfig represents the OAuth section of the manifest.
type OAuthConfig struct {
	Scopes BotScopes `json:"scopes"`
}

// BotScopes holds the bot token scopes.
type BotScopes struct {
	Bot []string `json:"bot"`
}

// Features represents the features section of the manifest.
type Features struct {
	BotUser *BotUser    `json:"bot_user,omitempty"`
	Slash   []SlashInfo `json:"slash_commands,omitempty"`
}

// BotUser represents the bot user configuration.
type BotUser struct {
	DisplayName  string `json:"display_name"`
	AlwaysOnline bool   `json:"always_online"`
}

// SlashInfo is the slash command entry within the features section.
type SlashInfo struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

// Settings represents the settings section of the manifest.
type Settings struct {
	EventSubscriptions EventSubscriptions `json:"event_subscriptions"`
	Interactivity      *Interactivity     `json:"interactivity,omitempty"`
	OrgDeploy          bool               `json:"org_deploy_enabled"`
	SocketMode         bool               `json:"socket_mode_enabled"`
}

// Interactivity represents the interactivity section of the manifest.
type Interactivity struct {
	Enabled    bool   `json:"is_enabled"`
	RequestURL string `json:"request_url,omitempty"`
}

// Manifest represents a Slack app manifest.
type Manifest struct {
	DisplayInfo DisplayInfo `json:"display_information"`
	Features    Features    `json:"features"`
	OAuthConfig OAuthConfig `json:"oauth_config"`
	Settings    Settings    `json:"settings"`
}

// DisplayInfo holds the app display information.
type DisplayInfo struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Generate creates a Manifest from the given router and app metadata.
// The requestURL is the base URL for event subscriptions (e.g. "https://example.com").
// Additional bot scopes can be provided via extraScopes.
func Generate(r router.Router, name, description, requestURL string, extraScopes ...string) Manifest {
	botEvents := []string{}
	scopes := map[string]bool{}
	var slashCommands []SlashInfo

	hasMentions := len(r.MentionRoutes) > 0
	hasChannelMessages := len(r.ChannelMessageRoutes) > 0
	hasSlashCommands := len(r.SlashCommandRoutes) > 0

	if hasMentions {
		botEvents = append(botEvents, "app_mention")
		scopes["app_mentions:read"] = true
	}
	if hasChannelMessages {
		botEvents = append(botEvents, "message.channels")
		scopes["channels:history"] = true
	}

	// chat:write is needed for nearly every bot
	if hasMentions || hasChannelMessages || hasSlashCommands {
		scopes["chat:write"] = true
	}

	for cmd, route := range r.SlashCommandRoutes {
		desc := route.Description
		if desc == "" {
			desc = route.Name
		}
		slashCommands = append(slashCommands, SlashInfo{
			Command:     cmd,
			Description: desc,
		})
		scopes["commands"] = true
	}

	sort.Slice(slashCommands, func(i, j int) bool {
		return slashCommands[i].Command < slashCommands[j].Command
	})

	scopeList := make([]string, 0, len(scopes))
	for s := range scopes {
		scopeList = append(scopeList, s)
	}
	for _, s := range extraScopes {
		if !scopes[s] {
			scopeList = append(scopeList, s)
		}
	}
	sort.Strings(scopeList)

	eventURL := ""
	if requestURL != "" {
		eventURL = requestURL + "/gadget"
	}

	m := Manifest{
		DisplayInfo: DisplayInfo{
			Name:        name,
			Description: description,
		},
		Features: Features{
			BotUser: &BotUser{
				DisplayName:  name,
				AlwaysOnline: true,
			},
			Slash: slashCommands,
		},
		OAuthConfig: OAuthConfig{
			Scopes: BotScopes{Bot: scopeList},
		},
		Settings: Settings{
			EventSubscriptions: EventSubscriptions{
				RequestURL: eventURL,
				BotEvents:  botEvents,
			},
			OrgDeploy:  false,
			SocketMode: false,
		},
	}

	if hasSlashCommands && requestURL != "" {
		m.Settings.Interactivity = &Interactivity{
			Enabled:    true,
			RequestURL: requestURL + "/gadget/command",
		}
	}

	return m
}

// JSON returns the manifest as a pretty-printed JSON string.
func (m Manifest) JSON() (string, error) {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
