package helpers

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestThreadReplyOption_NonEmpty(t *testing.T) {
	// ThreadReplyOption with a non-empty timestamp should return a non-nil
	// MsgOption that threads the reply under the given timestamp.
	// Deeper assertion is not practical because slack.MsgOption is an opaque
	// function type with no exported fields to inspect.
	opt := ThreadReplyOption("1234567890.123456")
	assert.NotNil(t, opt, "expected a non-nil MsgOption for a non-empty thread timestamp")
}

func TestThreadReplyOption_Empty(t *testing.T) {
	// ThreadReplyOption with an empty timestamp should return a no-op
	// MsgOption (via MsgOptionCompose with zero arguments) so callers can
	// include the option unconditionally without checking for empty strings.
	// Deeper assertion is not practical because slack.MsgOption is an opaque
	// function type with no exported fields to inspect.
	opt := ThreadReplyOption("")
	assert.NotNil(t, opt, "expected a non-nil no-op MsgOption for an empty thread timestamp")
}

// newErrorAPI returns a slack.Client backed by a test server that always
// returns an HTTP error response, causing Slack API calls to fail gracefully.
func newErrorAPI(t *testing.T) slack.Client {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":false,"error":"invalid_auth"}`))
	}))
	t.Cleanup(ts.Close)

	api := slack.New("xoxb-fake-token", slack.OptionAPIURL(ts.URL+"/"))
	return *api
}

// Note: This test cannot use t.Parallel() because it mutates the global zerolog logger.
func TestPostMessage_LogsError(t *testing.T) {
	var buf bytes.Buffer
	origLogger := log.Logger
	log.Logger = zerolog.New(&buf)
	defer func() { log.Logger = origLogger }()

	api := newErrorAPI(t)
	PostMessage(api, "C123", "test_plugin", slack.MsgOptionText("hello", false))

	logOutput := buf.String()
	assert.Contains(t, logOutput, "Failed to post message")
	assert.Contains(t, logOutput, "C123")
	assert.Contains(t, logOutput, "test_plugin")
}

func TestPostMessage_ReturnsValues(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true,"channel":"C123","ts":"1234567890.123456"}`))
	}))
	t.Cleanup(ts.Close)

	api := slack.New("xoxb-fake-token", slack.OptionAPIURL(ts.URL+"/"))
	ch, ts2 := PostMessage(*api, "C123", "test_plugin", slack.MsgOptionText("hello", false))
	assert.Equal(t, "C123", ch)
	assert.Equal(t, "1234567890.123456", ts2)
}

// Note: This test cannot use t.Parallel() because it mutates the global zerolog logger.
func TestAddReaction_LogsError(t *testing.T) {
	var buf bytes.Buffer
	origLogger := log.Logger
	log.Logger = zerolog.New(&buf)
	defer func() { log.Logger = origLogger }()

	api := newErrorAPI(t)
	AddReaction(api, "C123", "test_plugin", "thumbsup", "1234567890.123456")

	logOutput := buf.String()
	assert.Contains(t, logOutput, "Failed to add reaction")
	assert.Contains(t, logOutput, "C123")
	assert.Contains(t, logOutput, "test_plugin")
	assert.Contains(t, logOutput, "thumbsup")
}

func TestAddReaction_NoErrorOnSuccess(t *testing.T) {
	var buf bytes.Buffer
	origLogger := log.Logger
	log.Logger = zerolog.New(&buf)
	defer func() { log.Logger = origLogger }()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(ts.Close)

	api := slack.New("xoxb-fake-token", slack.OptionAPIURL(ts.URL+"/"))
	AddReaction(*api, "C123", "test_plugin", "thumbsup", "1234567890.123456")

	assert.Empty(t, buf.String())
}

// newConversationListServer returns a test server that serves paginated
// conversations.list responses. Each call to pages consumes the next page.
// It also asserts that every request includes the expected channel types.
func newConversationListServer(t *testing.T, pages []string) slack.Client {
	t.Helper()
	call := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertConversationListTypes(t, r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if call < len(pages) {
			_, _ = w.Write([]byte(pages[call]))
		} else {
			_, _ = w.Write([]byte(`{"ok":true,"channels":[],"response_metadata":{"next_cursor":""}}`))
		}
		call++
	}))
	t.Cleanup(ts.Close)
	api := slack.New("xoxb-fake-token", slack.OptionAPIURL(ts.URL+"/"))
	return *api
}

// assertConversationListTypes verifies that a conversations.list request
// includes both public_channel and private_channel in the types parameter.
func assertConversationListTypes(t *testing.T, r *http.Request) {
	t.Helper()
	if err := r.ParseForm(); err != nil {
		t.Fatalf("failed to parse form: %v", err)
	}
	types := r.FormValue("types")
	assert.Contains(t, types, "public_channel", "conversations.list must request public_channel")
	assert.Contains(t, types, "private_channel", "conversations.list must request private_channel")
}

// newMultiHandlerServer returns a test server that routes requests by path.
// Requests to /conversations.list are also checked for correct types parameter.
func newMultiHandlerServer(t *testing.T, handlers map[string]http.HandlerFunc) slack.Client {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/conversations.list" {
			assertConversationListTypes(t, r)
		}
		if h, ok := handlers[r.URL.Path]; ok {
			h(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":false,"error":"unexpected_path"}`))
	}))
	t.Cleanup(ts.Close)
	api := slack.New("xoxb-fake-token", slack.OptionAPIURL(ts.URL+"/"))
	return *api
}

func channelListJSONWithMembership(id, name string, isMember bool, cursor string) string {
	member := "false"
	if isMember {
		member = "true"
	}
	return fmt.Sprintf(
		`{"ok":true,"channels":[{"id":%q,"name":%q,"name_normalized":%q,"is_member":%s}],"response_metadata":{"next_cursor":%q}}`,
		id, name, name, member, cursor,
	)
}

func channelListJSON(id, name, cursor string) string {
	nextCursor := ""
	if cursor != "" {
		nextCursor = cursor
	}
	return fmt.Sprintf(
		`{"ok":true,"channels":[{"id":%q,"name":%q,"name_normalized":%q}],"response_metadata":{"next_cursor":%q}}`,
		id, name, name, nextCursor,
	)
}

func emptyChannelListJSON() string {
	return `{"ok":true,"channels":[],"response_metadata":{"next_cursor":""}}`
}

func TestGetJoinedChannels_ReturnsMemberChannels(t *testing.T) {
	// Two pages: first has one member and one non-member, second has one member.
	page1 := `{"ok":true,"channels":[{"id":"C001","name":"general","name_normalized":"general","is_member":true},{"id":"C002","name":"random","name_normalized":"random","is_member":false}],"response_metadata":{"next_cursor":"cursor-page2"}}`
	page2 := channelListJSONWithMembership("C003", "spam-feed", true, "")
	api := newConversationListServer(t, []string{page1, page2})
	channels, err := GetJoinedChannels(api)
	require.NoError(t, err)
	require.Len(t, channels, 2)
	assert.Equal(t, "C001", channels[0].ID)
	assert.Equal(t, "C003", channels[1].ID)
}

func TestGetJoinedChannels_EmptyWhenNoneJoined(t *testing.T) {
	api := newConversationListServer(t, []string{emptyChannelListJSON()})
	channels, err := GetJoinedChannels(api)
	require.NoError(t, err)
	assert.Empty(t, channels)
}

func TestGetJoinedChannels_APIError(t *testing.T) {
	api := newErrorAPI(t)
	_, err := GetJoinedChannels(api)
	assert.ErrorContains(t, err, "listing conversations")
}

func TestFindChannelByName_Found(t *testing.T) {
	api := newConversationListServer(t, []string{
		channelListJSON("C001", "spam-feed", ""),
	})
	ch, err := FindChannelByName(api, "spam-feed")
	require.NoError(t, err)
	assert.Equal(t, "C001", ch.ID)
	assert.Equal(t, "spam-feed", ch.NameNormalized)
}

func TestFindChannelByName_FoundAfterPagination(t *testing.T) {
	api := newConversationListServer(t, []string{
		channelListJSON("C001", "general", "cursor-page2"),
		channelListJSON("C002", "spam-feed", ""),
	})
	ch, err := FindChannelByName(api, "spam-feed")
	require.NoError(t, err)
	assert.Equal(t, "C002", ch.ID)
}

func TestFindChannelByName_NotFound(t *testing.T) {
	api := newConversationListServer(t, []string{
		channelListJSON("C001", "general", ""),
	})
	_, err := FindChannelByName(api, "spam-feed")
	assert.ErrorContains(t, err, "channel not found: spam-feed")
}

func TestFindChannelByName_APIError(t *testing.T) {
	api := newErrorAPI(t)
	_, err := FindChannelByName(api, "spam-feed")
	assert.ErrorContains(t, err, "listing conversations")
}

func TestJoinChannelByName_Success(t *testing.T) {
	api := newMultiHandlerServer(t, map[string]http.HandlerFunc{
		"/conversations.list": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(channelListJSON("C001", "spam-feed", "")))
		},
		"/conversations.join": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true,"channel":{"id":"C001","name":"spam-feed"}}`))
		},
	})
	err := JoinChannelByName(api, "spam-feed")
	assert.NoError(t, err)
}

func TestJoinChannelByName_ChannelNotFound(t *testing.T) {
	api := newConversationListServer(t, []string{emptyChannelListJSON()})
	err := JoinChannelByName(api, "spam-feed")
	assert.ErrorContains(t, err, "channel not found: spam-feed")
}

func TestJoinChannelByName_JoinError(t *testing.T) {
	api := newMultiHandlerServer(t, map[string]http.HandlerFunc{
		"/conversations.list": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(channelListJSON("C001", "spam-feed", "")))
		},
		"/conversations.join": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":false,"error":"cant_join_channel"}`))
		},
	})
	err := JoinChannelByName(api, "spam-feed")
	assert.ErrorContains(t, err, "joining channel spam-feed")
}

func TestFindChannelByName_FindsPrivateChannel(t *testing.T) {
	page := `{"ok":true,"channels":[{"id":"C_PRIV","name":"secret-ops","name_normalized":"secret-ops","is_private":true}],"response_metadata":{"next_cursor":""}}`
	api := newConversationListServer(t, []string{page})
	ch, err := FindChannelByName(api, "secret-ops")
	require.NoError(t, err)
	assert.Equal(t, "C_PRIV", ch.ID)
}

func TestGetJoinedChannels_IncludesPrivateChannels(t *testing.T) {
	page := `{"ok":true,"channels":[{"id":"C_PUB","name":"general","name_normalized":"general","is_member":true,"is_private":false},{"id":"C_PRIV","name":"secret-ops","name_normalized":"secret-ops","is_member":true,"is_private":true}],"response_metadata":{"next_cursor":""}}`
	api := newConversationListServer(t, []string{page})
	channels, err := GetJoinedChannels(api)
	require.NoError(t, err)
	require.Len(t, channels, 2)
	assert.Equal(t, "C_PUB", channels[0].ID)
	assert.Equal(t, "C_PRIV", channels[1].ID)
}
