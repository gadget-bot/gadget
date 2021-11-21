package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

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

// Slack Bot User OAuth Access Token" which starts with "xoxb-"
var slackOauthToken = os.Getenv("SLACK_OAUTH_TOKEN")

// Slack signing secret
var signingSecret = os.Getenv("SLACK_SIGNING_SECRET")

var dbUser = os.Getenv("GADGET_DB_USER")
var dbPass = os.Getenv("GADGET_DB_PASS")
var dbHost = os.Getenv("GADGET_DB_HOST")
var dbName = os.Getenv("GADGET_DB_NAME")
var listenPort = os.Getenv("GADGET_LISTEN_PORT")

var api = slack.New(slackOauthToken)

type EventMessageAuthorization struct {
	UserId string `json:"user_id"`
	TeamId string `json:"team_id"`
}

// this is required because slack-go doesn't seem to provide a way to get the bot's own ID
type EventsAPICallbackEvent struct {
	Type            string                      `json:"type"`
	Token           string                      `json:"token"`
	TeamID          string                      `json:"team_id"`
	APIAppID        string                      `json:"api_app_id"`
	Authoritzations []EventMessageAuthorization `json:"authorizations"`
	EventID         string                      `json:"event_id"`
	EventTime       int                         `json:"event_time"`
	EventContext    string                      `json:"event_context"`
}

type Gadget struct {
	Router router.Router
	UID    string
}

func requestLog(code int, r http.Request) {
	string_code := strconv.Itoa(code)
	log.Info().Str("method", r.Method).Str("code", string_code).Str("uri", r.URL.String()).Msg("")
}

// updateUID sets the UID field if it is empty
func (g *Gadget) updateUID(body []byte) error {
	if g.UID != "" {
		return nil
	}
	uid, err := getBotUuidFromBody(body)
	g.UID = uid
	return err
}

func getBotUuidFromBody(body []byte) (string, error) {
	var authorizedUsers EventsAPICallbackEvent
	json.Unmarshal([]byte(body), &authorizedUsers)

	if len(authorizedUsers.Authoritzations) > 0 {
		return authorizedUsers.Authoritzations[0].UserId, nil
	} else {
		return "", errors.New("Weird")
	}
}

func getListenPort() string {
	if listenPort != "" {
		return listenPort
	} else {
		return "3000"
	}
}

func globalAdminsFromEnv() []string {
	adminFromEnv := os.Getenv("GADGET_GLOBAL_ADMINS")

	uuids := strings.Split(adminFromEnv, ",")
	var trimmedUuids []string
	for _, uuid := range uuids {
		trimmedUuids = append(trimmedUuids, strings.TrimSpace(uuid))
	}

	return trimmedUuids
}

func stripBotMention(body string, botUuid string) string {
	return strings.TrimSpace(strings.ReplaceAll(body, "<@"+botUuid+">", ""))
}

func Setup() (*Gadget, error) {
	var gadget Gadget

	log.Debug().Str("globalAdmins", strings.Join(globalAdminsFromEnv(), ", ")).Msg("Pulled globalAdmins from Env")

	gadget.Router = *router.NewRouter()

	gadget.Router.DefaultMentionRoute = *fallback.GetMentionRoute()
	gadget.Router.DeniedMentionRoute = *permission_denied.GetMentionRoute()
	gadget.Router.AddMentionRoutes(groups.GetMentionRoutes())
	gadget.Router.AddMentionRoutes(user_info.GetMentionRoutes())
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	log.Debug().Msg("Connecting to DB...")
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True", dbUser, dbPass, dbHost, dbName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return &gadget, err
	}

	var version string
	db.Raw("SELECT VERSION() as version").Scan(&version)
	log.Debug().Msg(fmt.Sprintf("Connected to DB: %s", version))

	gadget.Router.DbConnection = db
	gadget.Router.SetupDb()

	var globalAdmins models.Group
	var globalAdminUsers []models.User

	for _, userName := range globalAdminsFromEnv() {
		var user models.User
		db.FirstOrCreate(&user, models.User{Uuid: userName})
		globalAdminUsers = append(globalAdminUsers, user)
	}

	db.Where(models.Group{Name: "globalAdmins"}).FirstOrCreate(&globalAdmins)
	db.Model(&globalAdmins).Association("Members").Replace(globalAdminUsers)

	return &gadget, nil
}

func (gadget Gadget) Run() error {
	http.HandleFunc("/gadget", func(w http.ResponseWriter, r *http.Request) {
		defer requestLog(200, *r)
		body, err := ioutil.ReadAll(r.Body)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			requestLog(http.StatusBadRequest, *r)
			return
		}
		// verify the signature
		sv, err := slack.NewSecretsVerifier(r.Header, signingSecret)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			requestLog(http.StatusUnauthorized, *r)
			return
		}
		if _, err := sv.Write(body); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			requestLog(http.StatusInternalServerError, *r)
			return
		}
		if err := sv.Ensure(); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			requestLog(http.StatusUnauthorized, *r)
			return
		}
		eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			requestLog(http.StatusInternalServerError, *r)
			return
		}

		if eventsAPIEvent.Type == slackevents.URLVerification {
			var res *slackevents.ChallengeResponse

			err := json.Unmarshal([]byte(body), &res)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text")
			w.Write([]byte(res.Challenge))
		}

		if eventsAPIEvent.Type == slackevents.CallbackEvent {
			innerEvent := eventsAPIEvent.InnerEvent
			err := gadget.updateUID(body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			eventUser := userFromInnerEvent(&innerEvent)
			// Ignore all events that Gadget produces to avoid infinite loops
			if gadget.UID == eventUser {
				w.WriteHeader(http.StatusOK)
				return
			}

			var currentUser models.User
			gadget.Router.DbConnection.FirstOrCreate(&currentUser, models.User{Uuid: eventUser})

			switch ev := innerEvent.Data.(type) {
			case *slackevents.AppMentionEvent:
				trimmedMessage := stripBotMention(ev.Text, gadget.UID)
				route, exists := gadget.Router.FindMentionRouteByMessage(trimmedMessage)
				if !exists {
					route = gadget.Router.DefaultMentionRoute
				}

				if !gadget.Router.Can(currentUser, route.Permissions) {
					log.Warn().Str("user", currentUser.Uuid).Str("route", route.Name).Msg("Permission failure")
					route = gadget.Router.DeniedMentionRoute
				}

				log.Debug().Str("user", currentUser.Uuid).Str("route", route.Name).Msg(trimmedMessage)

				go route.Execute(*api, gadget.Router, *ev, trimmedMessage)
			case *slackevents.MessageEvent:
				trimmedMessage := stripBotMention(ev.Text, gadget.UID)
				route, exists := gadget.Router.FindChannelMessageRouteByMessage(trimmedMessage)
				if !exists {
					w.WriteHeader(http.StatusNotFound)
					return
				}

				go route.Execute(*api, gadget.Router, *ev, trimmedMessage)
			}
		}
	})
	log.Print(fmt.Sprintf("Server listening on port %s", getListenPort()))
	return http.ListenAndServe(fmt.Sprintf(":%s", getListenPort()), nil)
}
