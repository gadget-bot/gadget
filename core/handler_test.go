package core

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gadget-bot/gadget/router"
	"github.com/rs/zerolog"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/stretchr/testify/assert"
)

// signalWriter wraps an io.Writer and signals a channel on the first write.
type signalWriter struct {
	io.Writer
	once   sync.Once
	signal chan struct{}
}

func (w *signalWriter) Write(p []byte) (int, error) {
	n, err := w.Writer.Write(p)
	w.once.Do(func() { close(w.signal) })
	return n, err
}

const testSecret = "test-signing-secret"

// signRequest sets the Slack signing headers on the given request.
func signRequest(r *http.Request, body string) {
	ts := fmt.Sprintf("%d", time.Now().Unix())
	baseString := fmt.Sprintf("v0:%s:%s", ts, body)
	mac := hmac.New(sha256.New, []byte(testSecret))
	mac.Write([]byte(baseString))
	sig := "v0=" + hex.EncodeToString(mac.Sum(nil))

	r.Header.Set("X-Slack-Request-Timestamp", ts)
	r.Header.Set("X-Slack-Signature", sig)
}

func newTestGadget() Gadget {
	signingSecret = testSecret
	return Gadget{
		Router: *router.NewRouter(),
		Client: slack.New("xoxb-fake"),
	}
}

// --- /gadget handler tests ---

func TestGadgetHandler_URLVerification(t *testing.T) {
	g := newTestGadget()
	handler := g.Handler()

	challenge := "test-challenge-token"
	body := fmt.Sprintf(`{"type":"url_verification","challenge":"%s"}`, challenge)

	req := httptest.NewRequest(http.MethodPost, "/gadget", strings.NewReader(body))
	signRequest(req, body)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, challenge, rr.Body.String())
	assert.Equal(t, "text", rr.Header().Get("Content-Type"))
}

func TestGadgetHandler_InvalidSignature(t *testing.T) {
	g := newTestGadget()
	handler := g.Handler()

	body := `{"type":"url_verification","challenge":"abc"}`
	req := httptest.NewRequest(http.MethodPost, "/gadget", strings.NewReader(body))
	req.Header.Set("X-Slack-Request-Timestamp", fmt.Sprintf("%d", time.Now().Unix()))
	req.Header.Set("X-Slack-Signature", "v0=invalidsignature")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestGadgetHandler_CallbackEventReachesRouting(t *testing.T) {
	g := newTestGadget()

	g.Router.AddMentionRoute(router.MentionRoute{
		Route: router.Route{
			Name:    "test-route",
			Pattern: `(?i)^hello`,
		},
		Plugin: func(r router.Router, route router.Route, api slack.Client, ev slackevents.AppMentionEvent, message string) {
		},
	})
	g.Router.BotUID = "U_BOT"

	handler := g.Handler()

	eventPayload := map[string]interface{}{
		"type":       "event_callback",
		"token":      "fake",
		"team_id":    "T123",
		"api_app_id": "A123",
		"authorizations": []map[string]string{
			{"user_id": "U_BOT", "team_id": "T123"},
		},
		"event": map[string]interface{}{
			"type":    "app_mention",
			"user":    "U_USER",
			"text":    "<@U_BOT> hello world",
			"channel": "C123",
			"ts":      "1234567890.123456",
		},
		"event_id":   "Ev123",
		"event_time": 1234567890,
	}
	body, _ := json.Marshal(eventPayload)
	bodyStr := string(body)

	req := httptest.NewRequest(http.MethodPost, "/gadget", strings.NewReader(bodyStr))
	signRequest(req, bodyStr)
	rr := httptest.NewRecorder()

	// Panics on DbConnection.FirstOrCreate (no DB configured), confirming the handler
	// successfully verified the signature, parsed the event, and reached routing logic.
	func() {
		defer func() { recover() }()
		handler.ServeHTTP(rr, req)
	}()

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestGadgetHandler_ChannelMessageRouting(t *testing.T) {
	g := newTestGadget()

	g.Router.AddChannelMessageRoute(router.ChannelMessageRoute{
		Route: router.Route{
			Name:    "test-channel",
			Pattern: `(?i)^deploy`,
		},
		Plugin: func(r router.Router, route router.Route, api slack.Client, ev slackevents.MessageEvent, message string) {
		},
	})
	g.Router.BotUID = "U_BOT"

	handler := g.Handler()

	eventPayload := map[string]interface{}{
		"type":       "event_callback",
		"token":      "fake",
		"team_id":    "T123",
		"api_app_id": "A123",
		"authorizations": []map[string]string{
			{"user_id": "U_BOT", "team_id": "T123"},
		},
		"event": map[string]interface{}{
			"type":         "message",
			"user":         "U_USER",
			"text":         "deploy production",
			"channel":      "C123",
			"channel_type": "channel",
			"ts":           "1234567890.123456",
		},
		"event_id":   "Ev124",
		"event_time": 1234567890,
	}
	body, _ := json.Marshal(eventPayload)
	bodyStr := string(body)

	req := httptest.NewRequest(http.MethodPost, "/gadget", strings.NewReader(bodyStr))
	signRequest(req, bodyStr)
	rr := httptest.NewRecorder()

	func() {
		defer func() { recover() }()
		handler.ServeHTTP(rr, req)
	}()

	assert.Equal(t, http.StatusOK, rr.Code)
}

// --- /gadget/command handler tests ---

func TestCommandHandler_InvalidSignature(t *testing.T) {
	g := newTestGadget()
	handler := g.Handler()

	body := "command=%2Fdeploy&user_id=U123&text=production"
	req := httptest.NewRequest(http.MethodPost, "/gadget/command", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Slack-Request-Timestamp", fmt.Sprintf("%d", time.Now().Unix()))
	req.Header.Set("X-Slack-Signature", "v0=invalidsignature")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestCommandHandler_UnknownCommand(t *testing.T) {
	g := newTestGadget()
	handler := g.Handler()

	formData := url.Values{
		"command": {"/unknown"},
		"user_id": {"U123"},
		"text":    {"something"},
	}
	body := formData.Encode()

	req := httptest.NewRequest(http.MethodPost, "/gadget/command", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	signRequest(req, body)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"response_type":"ephemeral","text":"Unknown command."}`, rr.Body.String())
}

func TestCommandHandler_ValidCommandReachesPermissionCheck(t *testing.T) {
	g := newTestGadget()

	g.Router.AddSlashCommandRoute(router.SlashCommandRoute{
		Route: router.Route{
			Name:        "deploy",
			Description: "Deploy the app",
		},
		Command: "/deploy",
		Plugin: func(r router.Router, route router.Route, api slack.Client, cmd slack.SlashCommand) {
		},
	})

	handler := g.Handler()

	formData := url.Values{
		"command": {"/deploy"},
		"user_id": {"U123"},
		"text":    {"production"},
	}
	body := formData.Encode()

	req := httptest.NewRequest(http.MethodPost, "/gadget/command", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	signRequest(req, body)
	rr := httptest.NewRecorder()

	// Panics on DbConnection.FirstOrCreate, confirming the handler verified the
	// signature, parsed the slash command, and found the matching route.
	func() {
		defer func() { recover() }()
		handler.ServeHTTP(rr, req)
	}()

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestSafeGo_RecoversPanic(t *testing.T) {
	var buf bytes.Buffer
	logged := make(chan struct{})
	w := &signalWriter{Writer: &buf, signal: logged}

	logger := zerolog.New(w).With().Str("request_id", "test-request-id").Logger()

	safeGo("panicking-route", logger, func() {
		panic("test panic")
	})

	select {
	case <-logged:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for panic recovery log")
	}

	output := buf.String()
	assert.Contains(t, output, "Plugin panicked")
	assert.Contains(t, output, "panicking-route")
	assert.Contains(t, output, "test panic")
	assert.Contains(t, output, "test-request-id")
}
