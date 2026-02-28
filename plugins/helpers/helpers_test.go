package helpers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
)

func TestThreadReplyOption_NonEmpty(t *testing.T) {
	opt := ThreadReplyOption("1234567890.123456")
	assert.NotNil(t, opt)
}

func TestThreadReplyOption_Empty(t *testing.T) {
	opt := ThreadReplyOption("")
	assert.NotNil(t, opt)
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
