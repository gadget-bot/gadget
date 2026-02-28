package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/gadget-bot/gadget/models"
	"github.com/gadget-bot/gadget/plugins/fallback"
	"github.com/gadget-bot/gadget/plugins/groups"
	"github.com/gadget-bot/gadget/plugins/permission_denied"
	"github.com/gadget-bot/gadget/plugins/user_info"
	"github.com/gadget-bot/gadget/router"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

var (
	// Slack Bot User OAuth Access Token which starts with "xoxb-"
	slackOauthToken = os.Getenv("SLACK_OAUTH_TOKEN")

	// Slack signing secret
	signingSecret = os.Getenv("SLACK_SIGNING_SECRET")

	dbUser     = os.Getenv("GADGET_DB_USER")
	dbPass     = os.Getenv("GADGET_DB_PASS")
	dbHost     = os.Getenv("GADGET_DB_HOST")
	dbName     = os.Getenv("GADGET_DB_NAME")
	listenPort = os.Getenv("GADGET_LISTEN_PORT")
	admins     = globalAdminsFromString(os.Getenv("GADGET_GLOBAL_ADMINS"))

	api *slack.Client
)

type Gadget struct {
	Router router.Router
	Client *slack.Client
}

func requestLog(code int, r http.Request, denied bool) {
	string_code := strconv.Itoa(code)
	event := log.Info().Str("method", r.Method).Str("code", string_code).Str("uri", r.URL.String())
	if denied {
		event = event.Str("access", "denied")
	}
	event.Msg("")
}

// verifySlackRequest reads the request body, verifies the Slack signing secret,
// and returns the body bytes. On failure it writes the appropriate HTTP status
// and returns a non-nil error.
func verifySlackRequest(w http.ResponseWriter, r *http.Request) ([]byte, int, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, http.StatusBadRequest, err
	}

	sv, err := slack.NewSecretsVerifier(r.Header, signingSecret)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return nil, http.StatusUnauthorized, err
	}
	if _, err := sv.Write(body); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return nil, http.StatusInternalServerError, err
	}
	if err := sv.Ensure(); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return nil, http.StatusUnauthorized, err
	}

	return body, http.StatusOK, nil
}

func getListenPort() string {
	if listenPort != "" {
		return listenPort
	} else {
		return "3000"
	}
}

func globalAdminsFromString(admins string) []string {
	uuids := strings.Split(admins, ",")
	var trimmedUuids []string
	for _, uuid := range uuids {
		trimmedUuids = append(trimmedUuids, strings.TrimSpace(uuid))
	}

	return trimmedUuids
}

func stripBotMention(body string, botUuid string) string {
	return strings.TrimSpace(strings.ReplaceAll(body, "<@"+botUuid+">", ""))
}

func safeGo(routeName string, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error().
					Interface("panic", r).
					Str("route", routeName).
					Str("stack", string(debug.Stack())).
					Msg("Plugin panicked")
			}
		}()
		fn()
	}()
}

func Setup() (*Gadget, error) {
	var gadget Gadget
	api = slack.New(slackOauthToken)
	gadget.Client = api

	log.Debug().Str("globalAdmins", strings.Join(admins, ", ")).Msg("Pulled globalAdmins")

	gadget.Router = *router.NewRouter()

	gadget.Router.DefaultMentionRoute = *fallback.GetMentionRoute()
	gadget.Router.DeniedMentionRoute = *permission_denied.GetMentionRoute()
	gadget.Router.DeniedSlashCommandRoute = *permission_denied.GetSlashCommandRoute()
	gadget.Router.AddMentionRoutes(groups.GetMentionRoutes())
	gadget.Router.AddMentionRoutes(user_info.GetMentionRoutes())
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	log.Debug().Msg("Connecting to DB...")
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True", dbUser, dbPass, dbHost, dbName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return &gadget, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return &gadget, err
	}
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(3 * time.Minute)

	var version string
	db.Raw("SELECT VERSION() as version").Scan(&version)
	log.Debug().Msg(fmt.Sprintf("Connected to DB: %s", version))

	gadget.Router.DbConnection = db
	gadget.Router.SetupDb()

	var globalAdmins models.Group
	var globalAdminUsers []models.User

	for _, userName := range admins {
		var user models.User
		db.FirstOrCreate(&user, models.User{Uuid: userName})
		globalAdminUsers = append(globalAdminUsers, user)
	}

	db.Where(models.Group{Name: "globalAdmins"}).FirstOrCreate(&globalAdmins)
	db.Model(&globalAdmins).Association("Members").Replace(globalAdminUsers)

	return &gadget, nil
}

func SetupWithConfig(token, secret, databaseUser, databasePass, databaseHost, databaseName, port string, globalAdmins []string) (*Gadget, error) {
	// quick and dirty, just override the global values which were set from ENV vars
	slackOauthToken = token
	admins = globalAdmins
	signingSecret = secret
	dbUser = databaseUser
	dbPass = databasePass
	dbHost = databaseHost
	dbName = databaseName
	listenPort = port

	return Setup()
}

// Handler returns an http.Handler with all Gadget routes registered.
func (gadget Gadget) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/gadget", func(w http.ResponseWriter, r *http.Request) {
		statusCode := http.StatusOK
		accessDenied := false
		defer func() { requestLog(statusCode, *r, accessDenied) }()

		body, code, err := verifySlackRequest(w, r)
		if err != nil {
			statusCode = code
			return
		}

		eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
		if err != nil {
			statusCode = http.StatusInternalServerError
			w.WriteHeader(statusCode)
			return
		}

		if eventsAPIEvent.Type == slackevents.URLVerification {
			var res *slackevents.ChallengeResponse

			err := json.Unmarshal([]byte(body), &res)
			if err != nil {
				statusCode = http.StatusInternalServerError
				w.WriteHeader(statusCode)
				return
			}
			w.Header().Set("Content-Type", "text")
			w.Write([]byte(res.Challenge))
		}

		if eventsAPIEvent.Type == slackevents.CallbackEvent {
			innerEvent := eventsAPIEvent.InnerEvent
			err := gadget.Router.UpdateBotUID(body)
			if err != nil {
				statusCode = http.StatusInternalServerError
				w.WriteHeader(statusCode)
				return
			}

			eventUser := userFromInnerEvent(&innerEvent)
			// Ignore all events that Gadget produces to avoid infinite loops
			if gadget.Router.BotUID == eventUser {
				w.WriteHeader(http.StatusOK)
				return
			}

			var currentUser models.User
			gadget.Router.DbConnection.FirstOrCreate(&currentUser, models.User{Uuid: eventUser})

			switch ev := innerEvent.Data.(type) {
			case *slackevents.AppMentionEvent:
				trimmedMessage := stripBotMention(ev.Text, gadget.Router.BotUID)
				route, exists := gadget.Router.FindMentionRouteByMessage(trimmedMessage)
				if !exists {
					route = gadget.Router.DefaultMentionRoute
				}

				if !gadget.Router.Can(currentUser, route.Permissions) {
					log.Warn().Str("user", currentUser.Uuid).Str("route", route.Name).Msg("Permission failure")
					accessDenied = true
					route = gadget.Router.DeniedMentionRoute
				}

				log.Debug().Str("user", currentUser.Uuid).Str("route", route.Name).Msg(trimmedMessage)

				safeGo(route.Name, func() { route.Execute(gadget.Router, *gadget.Client, *ev, trimmedMessage) })
			case *slackevents.MessageEvent:
				trimmedMessage := stripBotMention(ev.Text, gadget.Router.BotUID)
				route, exists := gadget.Router.FindChannelMessageRouteByMessage(trimmedMessage)
				if !exists {
					statusCode = http.StatusNotFound
					w.WriteHeader(statusCode)
					return
				}

				safeGo(route.Name, func() { route.Execute(gadget.Router, *gadget.Client, *ev, trimmedMessage) })
			}
		}
	})
	mux.HandleFunc("/gadget/command", func(w http.ResponseWriter, r *http.Request) {
		statusCode := http.StatusOK
		accessDenied := false
		defer func() { requestLog(statusCode, *r, accessDenied) }()

		body, code, err := verifySlackRequest(w, r)
		if err != nil {
			statusCode = code
			return
		}

		// Restore body so SlashCommandParse can read it via ParseForm
		r.Body = io.NopCloser(bytes.NewBuffer(body))
		cmd, err := slack.SlashCommandParse(r)
		if err != nil {
			statusCode = http.StatusBadRequest
			w.WriteHeader(statusCode)
			return
		}

		route, exists := gadget.Router.FindSlashCommandRouteByCommand(cmd.Command)
		if !exists {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"response_type":"ephemeral","text":"Unknown command."}`))
			return
		}

		var currentUser models.User
		gadget.Router.DbConnection.FirstOrCreate(&currentUser, models.User{Uuid: cmd.UserID})

		if !gadget.Router.Can(currentUser, route.Permissions) {
			log.Warn().Str("user", currentUser.Uuid).Str("route", route.Name).Msg("Permission failure")
			accessDenied = true
			w.Header().Set("Content-Type", "application/json")
			if _, err := w.Write([]byte(`{"response_type":"ephemeral","text":"Permission denied."}`)); err != nil {
				log.Error().Err(err).Msg("Failed to write permission denied response")
			}
			safeGo(gadget.Router.DeniedSlashCommandRoute.Name, func() {
				gadget.Router.DeniedSlashCommandRoute.Execute(gadget.Router, *gadget.Client, cmd)
			})
			return
		}

		log.Debug().Str("user", currentUser.Uuid).Str("route", route.Name).Str("command", cmd.Command).Msg("Slash command")
		if route.ImmediateResponse != "" {
			resp, _ := json.Marshal(map[string]string{
				"response_type": "ephemeral",
				"text":          route.ImmediateResponse,
			})
			w.Header().Set("Content-Type", "application/json")
			w.Write(resp)
		}
		safeGo(route.Name, func() { route.Execute(gadget.Router, *gadget.Client, cmd) })
		if route.ImmediateResponse == "" {
			w.WriteHeader(http.StatusOK)
		}
	})

	return mux
}

func (gadget Gadget) Run() error {
	handler := gadget.Handler()
	log.Print(fmt.Sprintf("Server listening on port %s", getListenPort()))
	return http.ListenAndServe(fmt.Sprintf(":%s", getListenPort()), handler)
}
