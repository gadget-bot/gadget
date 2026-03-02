package core

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
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
	gormlogger "gorm.io/gorm/logger"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

// Config holds all configuration needed to initialize a Gadget instance.
type Config struct {
	SlackOAuthToken string
	SigningSecret   string
	DBUser          string
	DBPass          string
	DBHost          string
	DBName          string
	ListenPort      string
	GlobalAdmins    []string
}

// ConfigFromEnv returns a Config populated from environment variables.
func ConfigFromEnv() Config {
	return Config{
		SlackOAuthToken: os.Getenv("SLACK_OAUTH_TOKEN"),
		SigningSecret:   os.Getenv("SLACK_SIGNING_SECRET"),
		DBUser:          os.Getenv("GADGET_DB_USER"),
		DBPass:          os.Getenv("GADGET_DB_PASS"),
		DBHost:          os.Getenv("GADGET_DB_HOST"),
		DBName:          os.Getenv("GADGET_DB_NAME"),
		ListenPort:      os.Getenv("GADGET_LISTEN_PORT"),
		GlobalAdmins:    globalAdminsFromString(os.Getenv("GADGET_GLOBAL_ADMINS")),
	}
}

type Gadget struct {
	Router        router.Router
	Client        *slack.Client
	signingSecret string
	listenPort    string
}

func requestLog(code int, r http.Request, denied bool, start time.Time, logger zerolog.Logger) {
	event := logger.Info().
		Str("method", r.Method).
		Str("code", strconv.Itoa(code)).
		Str("uri", r.URL.String()).
		Dur("duration", time.Since(start))
	if denied {
		event = event.Str("access", "denied")
	}
	event.Msg("Request handled")
}

func generateRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		log.Error().Err(err).Msg("Failed to generate random request ID")
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// verifySlackRequest reads the request body, verifies the Slack signing secret,
// and returns the body bytes. On failure it writes the appropriate HTTP status
// and returns a non-nil error.
func verifySlackRequest(w http.ResponseWriter, r *http.Request, secret string, logger zerolog.Logger) ([]byte, int, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to read request body")
		w.WriteHeader(http.StatusBadRequest)
		return nil, http.StatusBadRequest, err
	}

	sv, err := slack.NewSecretsVerifier(r.Header, secret)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create secrets verifier")
		w.WriteHeader(http.StatusUnauthorized)
		return nil, http.StatusUnauthorized, err
	}
	if _, err := sv.Write(body); err != nil {
		logger.Error().Err(err).Msg("Failed to write body to verifier")
		w.WriteHeader(http.StatusInternalServerError)
		return nil, http.StatusInternalServerError, err
	}
	if err := sv.Ensure(); err != nil {
		logger.Warn().Err(err).Msg("Request signature verification failed")
		w.WriteHeader(http.StatusUnauthorized)
		return nil, http.StatusUnauthorized, err
	}

	return body, http.StatusOK, nil
}

func (g Gadget) getListenPort() string {
	if g.listenPort != "" {
		return g.listenPort
	}
	return "3000"
}

func globalAdminsFromString(admins string) []string {
	if admins == "" {
		return []string{}
	}
	uuids := strings.Split(admins, ",")
	trimmedUuids := []string{}
	for _, uuid := range uuids {
		trimmed := strings.TrimSpace(uuid)
		if trimmed != "" {
			trimmedUuids = append(trimmedUuids, trimmed)
		}
	}
	return trimmedUuids
}

func stripBotMention(body string, botUuid string) string {
	return strings.TrimSpace(strings.ReplaceAll(body, "<@"+botUuid+">", ""))
}

func safeGo(routeName string, logger zerolog.Logger, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error().
					Interface("panic", r).
					Str("route", routeName).
					Bytes("stack", debug.Stack()).
					Msg("Plugin panicked")
			}
		}()
		fn()
	}()
}

// Setup creates a new Gadget instance using configuration from environment variables.
func Setup() (*Gadget, error) {
	return SetupWithConfig(ConfigFromEnv())
}

// SetupWithConfig creates a new Gadget instance using the provided Config.
func SetupWithConfig(cfg Config) (*Gadget, error) {
	var gadget Gadget

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logLevel := os.Getenv("GADGET_LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	level, parseErr := zerolog.ParseLevel(logLevel)
	if parseErr != nil || level == zerolog.NoLevel {
		log.Warn().Str("GADGET_LOG_LEVEL", logLevel).Msg("Invalid log level, defaulting to info")
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)
	log.Info().Str("level", level.String()).Msg("Log level configured")

	gadget.Client = slack.New(cfg.SlackOAuthToken)
	gadget.signingSecret = cfg.SigningSecret
	gadget.listenPort = cfg.ListenPort

	log.Debug().Str("globalAdmins", strings.Join(cfg.GlobalAdmins, ", ")).Msg("Pulled globalAdmins")

	gadget.Router = *router.NewRouter()

	gadget.Router.DefaultMentionRoute = *fallback.GetMentionRoute()
	gadget.Router.DeniedMentionRoute = *permission_denied.GetMentionRoute()
	gadget.Router.DeniedChannelMessageRoute = *permission_denied.GetChannelMessageRoute()
	gadget.Router.DeniedSlashCommandRoute = *permission_denied.GetSlashCommandRoute()
	gadget.Router.AddMentionRoutes(groups.GetMentionRoutes())
	gadget.Router.AddMentionRoutes(user_info.GetMentionRoutes())

	log.Debug().Msg("Connecting to DB...")
	var gormLogLevel gormlogger.LogLevel
	switch {
	case level <= zerolog.DebugLevel:
		gormLogLevel = gormlogger.Info
	case level <= zerolog.WarnLevel:
		gormLogLevel = gormlogger.Warn
	default:
		gormLogLevel = gormlogger.Silent
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True", cfg.DBUser, cfg.DBPass, cfg.DBHost, cfg.DBName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormLogLevel),
	})
	if err != nil {
		return &gadget, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return &gadget, err
	}
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(3 * time.Minute)
	log.Debug().Int("maxOpenConns", 10).Int("maxIdleConns", 5).Msg("DB connection pool configured")

	var version string
	db.Raw("SELECT VERSION() as version").Scan(&version)
	log.Debug().Str("version", version).Msg("Connected to DB")

	gadget.Router.DbConnection = db
	if err := gadget.Router.SetupDb(); err != nil {
		return &gadget, fmt.Errorf("setup database: %w", err)
	}

	var globalAdmins models.Group
	var globalAdminUsers []models.User

	for _, userName := range cfg.GlobalAdmins {
		var user models.User
		db.FirstOrCreate(&user, models.User{Uuid: userName})
		globalAdminUsers = append(globalAdminUsers, user)
	}

	db.Where(models.Group{Name: "globalAdmins"}).FirstOrCreate(&globalAdmins)
	if err := db.Model(&globalAdmins).Association("Members").Replace(globalAdminUsers); err != nil {
		return &gadget, fmt.Errorf("replace global admin members: %w", err)
	}

	return &gadget, nil
}

// Handler returns an http.Handler with all Gadget routes registered.
func (gadget Gadget) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/gadget", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := generateRequestID()
		logger := log.With().Str("request_id", requestID).Logger()
		statusCode := http.StatusOK
		accessDenied := false
		defer func() { requestLog(statusCode, *r, accessDenied, start, logger) }()

		body, code, err := verifySlackRequest(w, r, gadget.signingSecret, logger)
		if err != nil {
			statusCode = code
			return
		}

		eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
		if err != nil {
			logger.Error().Err(err).Msg("Failed to parse Slack event")
			statusCode = http.StatusInternalServerError
			w.WriteHeader(statusCode)
			return
		}

		if eventsAPIEvent.Type == slackevents.URLVerification {
			var res *slackevents.ChallengeResponse

			err := json.Unmarshal([]byte(body), &res)
			if err != nil {
				logger.Error().Err(err).Msg("Failed to unmarshal URL verification challenge")
				statusCode = http.StatusInternalServerError
				w.WriteHeader(statusCode)
				return
			}
			w.Header().Set("Content-Type", "text")
			if _, err := w.Write([]byte(res.Challenge)); err != nil {
				logger.Error().Err(err).Msg("Failed to write URL verification challenge response")
			}
		}

		if eventsAPIEvent.Type == slackevents.CallbackEvent {
			innerEvent := eventsAPIEvent.InnerEvent
			err := gadget.Router.UpdateBotUID(body)
			if err != nil {
				logger.Error().Err(err).Msg("Failed to update bot UID")
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

			ctx := router.HandlerContext{
				Router:    gadget.Router,
				BotClient: gadget.Client,
				Logger:    logger,
			}

			switch ev := innerEvent.Data.(type) {
			case *slackevents.AppMentionEvent:
				trimmedMessage := stripBotMention(ev.Text, gadget.Router.BotUID)
				route, exists := gadget.Router.FindMentionRouteByMessage(trimmedMessage)
				if !exists {
					route = gadget.Router.DefaultMentionRoute
				}

				if !gadget.Router.Can(currentUser, route.Permissions) {
					logger.Warn().Str("user", currentUser.Uuid).Str("route", route.Name).Msg("Permission failure")
					accessDenied = true
					route = gadget.Router.DeniedMentionRoute
				}

				logger.Debug().Str("user", currentUser.Uuid).Str("route", route.Name).Msg(trimmedMessage)

				safeGo(route.Name, logger, func() { route.Execute(ctx, *ev, trimmedMessage) })
			case *slackevents.MessageEvent:
				trimmedMessage := stripBotMention(ev.Text, gadget.Router.BotUID)
				route, exists := gadget.Router.FindChannelMessageRouteByMessage(trimmedMessage)
				if !exists {
					statusCode = http.StatusOK
					w.WriteHeader(statusCode)
					return
				}

				if !gadget.Router.Can(currentUser, route.Permissions) {
					logger.Warn().Str("user", currentUser.Uuid).Str("route", route.Name).Msg("Permission failure")
					accessDenied = true
					route = gadget.Router.DeniedChannelMessageRoute
				}

				logger.Debug().Str("user", currentUser.Uuid).Str("route", route.Name).Msg(trimmedMessage)
				safeGo(route.Name, logger, func() { route.Execute(ctx, *ev, trimmedMessage) })
			}
		}
	})
	mux.HandleFunc("/gadget/command", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := generateRequestID()
		logger := log.With().Str("request_id", requestID).Logger()
		statusCode := http.StatusOK
		accessDenied := false
		defer func() { requestLog(statusCode, *r, accessDenied, start, logger) }()

		body, code, err := verifySlackRequest(w, r, gadget.signingSecret, logger)
		if err != nil {
			statusCode = code
			return
		}

		// Restore body so SlashCommandParse can read it via ParseForm
		r.Body = io.NopCloser(bytes.NewBuffer(body))
		cmd, err := slack.SlashCommandParse(r)
		if err != nil {
			logger.Warn().Err(err).Msg("Failed to parse slash command")
			statusCode = http.StatusBadRequest
			w.WriteHeader(statusCode)
			return
		}

		route, exists := gadget.Router.FindSlashCommandRouteByCommand(cmd.Command)
		if !exists {
			w.Header().Set("Content-Type", "application/json")
			if _, err := w.Write([]byte(`{"response_type":"ephemeral","text":"Unknown command."}`)); err != nil {
				logger.Error().Err(err).Msg("Failed to write unknown command response")
			}
			return
		}

		var currentUser models.User
		gadget.Router.DbConnection.FirstOrCreate(&currentUser, models.User{Uuid: cmd.UserID})

		ctx := router.HandlerContext{
			Router:    gadget.Router,
			BotClient: gadget.Client,
			Logger:    logger,
		}

		if !gadget.Router.Can(currentUser, route.Permissions) {
			logger.Warn().Str("user", currentUser.Uuid).Str("route", route.Name).Msg("Permission failure")
			accessDenied = true
			w.Header().Set("Content-Type", "application/json")
			if _, err := w.Write([]byte(`{"response_type":"ephemeral","text":"Permission denied."}`)); err != nil {
				logger.Error().Err(err).Msg("Failed to write permission denied response")
			}
			safeGo(gadget.Router.DeniedSlashCommandRoute.Name, logger, func() {
				gadget.Router.DeniedSlashCommandRoute.Execute(ctx, cmd)
			})
			return
		}

		logger.Debug().Str("user", currentUser.Uuid).Str("route", route.Name).Str("command", cmd.Command).Msg("Slash command")
		if route.ImmediateResponse != "" {
			resp, err := json.Marshal(map[string]string{
				"response_type": "ephemeral",
				"text":          route.ImmediateResponse,
			})
			if err != nil {
				logger.Error().Err(err).Msg("Failed to marshal immediate response")
				statusCode = http.StatusInternalServerError
				w.WriteHeader(statusCode)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			if _, err := w.Write(resp); err != nil {
				logger.Error().Err(err).Msg("Failed to write immediate response")
			}
		}
		safeGo(route.Name, logger, func() { route.Execute(ctx, cmd) })
		if route.ImmediateResponse == "" {
			w.WriteHeader(http.StatusOK)
		}
	})

	return mux
}

func (gadget Gadget) Run() error {
	handler := gadget.Handler()
	port := gadget.getListenPort()
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	log.Info().Str("port", port).Msg("Server listening")
	return srv.ListenAndServe()
}
