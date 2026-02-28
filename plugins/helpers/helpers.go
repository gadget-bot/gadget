package helpers

import (
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
func PostMessage(api slack.Client, channel, plugin string, options ...slack.MsgOption) {
	_, _, err := api.PostMessage(channel, options...)
	if err != nil {
		log.Error().Err(err).Str("channel", channel).Str("plugin", plugin).Msg("Failed to post message")
	}
}

// AddReaction adds a reaction to a message and logs any error using zerolog
// with consistent structured fields.
func AddReaction(api slack.Client, channel, plugin, reaction, timestamp string) {
	msgRef := slack.NewRefToMessage(channel, timestamp)
	if err := api.AddReaction(reaction, msgRef); err != nil {
		log.Error().Err(err).Str("channel", channel).Str("plugin", plugin).Msg("Failed to add reaction")
	}
}
