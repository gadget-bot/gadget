package helpers

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/slack-go/slack"
)

// ThreadReplyOption returns a slack.MsgOptionTS for threading replies when
// the given threadTS is non-empty. When threadTS is empty (i.e. the
// triggering message was not in a thread), a no-op MsgOption is returned
// so callers can include it unconditionally.
func ThreadReplyOption(threadTS string) slack.MsgOption {
	if threadTS != "" {
		return slack.MsgOptionTS(threadTS)
	}
	// Return a no-op option by composing zero options.
	return slack.MsgOptionCompose()
}

// PostMessage sends a Slack message to the given channel and logs any error
// using zerolog with consistent structured fields.
func PostMessage(api slack.Client, channel, plugin string, options ...slack.MsgOption) (string, string) {
	ch, ts, err := api.PostMessage(channel, options...)
	if err != nil {
		log.Error().Err(err).Str("channel", channel).Str("plugin", plugin).Msg("Failed to post message")
	}
	return ch, ts
}

// AddReaction adds a reaction to a message and logs any error using zerolog
// with consistent structured fields.
func AddReaction(api slack.Client, channel, plugin, reaction, timestamp string) {
	msgRef := slack.NewRefToMessage(channel, timestamp)
	if err := api.AddReaction(reaction, msgRef); err != nil {
		log.Error().Err(err).Str("channel", channel).Str("plugin", plugin).Str("reaction", reaction).Msg("Failed to add reaction")
	}
}

// FindChannelByName searches all conversations for a channel whose
// NameNormalized matches name, handling pagination internally.
// Returns the matching channel or an error if not found or if any API call fails.
func FindChannelByName(api slack.Client, name string) (slack.Channel, error) {
	params := &slack.GetConversationsParameters{}
	for {
		channels, cursor, err := api.GetConversations(params)
		if err != nil {
			return slack.Channel{}, fmt.Errorf("listing conversations: %w", err)
		}
		for _, ch := range channels {
			if ch.NameNormalized == name {
				return ch, nil
			}
		}
		if cursor == "" {
			break
		}
		params.Cursor = cursor
	}
	return slack.Channel{}, fmt.Errorf("channel not found: %s", name)
}

// GetJoinedChannels returns all conversations the bot is currently a member of.
// Pagination is handled internally.
func GetJoinedChannels(api slack.Client) ([]slack.Channel, error) {
	var joined []slack.Channel
	params := &slack.GetConversationsParameters{}
	for {
		channels, cursor, err := api.GetConversations(params)
		if err != nil {
			return nil, fmt.Errorf("listing conversations: %w", err)
		}
		for _, ch := range channels {
			if ch.IsMember {
				joined = append(joined, ch)
			}
		}
		if cursor == "" {
			break
		}
		params.Cursor = cursor
	}
	return joined, nil
}

// JoinChannelByName finds the channel with the given name and joins it.
// Pagination is handled internally. Returns an error if the channel is not
// found or if any API call fails.
func JoinChannelByName(api slack.Client, name string) error {
	ch, err := FindChannelByName(api, name)
	if err != nil {
		return err
	}
	if _, _, _, err = api.JoinConversation(ch.ID); err != nil {
		return fmt.Errorf("joining channel %s: %w", name, err)
	}
	return nil
}
