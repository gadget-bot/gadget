// Package gadgettest provides testing utilities for Gadget route handlers.
// It allows dispatching synthetic events synchronously without requiring
// a database, HTTP server, or Slack signature verification.
package gadgettest

import (
	"errors"

	"github.com/gadget-bot/gadget/router"
	"github.com/rs/zerolog"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"gorm.io/gorm"
)

// Dispatcher dispatches synthetic events to registered routes synchronously.
type Dispatcher struct {
	router     router.Router
	botClient  *slack.Client
	userClient *slack.Client
	logger     zerolog.Logger
}

// Option configures a Dispatcher.
type Option func(*Dispatcher)

// WithBotClient sets the bot Slack client available as ctx.BotClient.
func WithBotClient(c *slack.Client) Option {
	return func(d *Dispatcher) { d.botClient = c }
}

// WithUserClient sets the user Slack client available as ctx.UserClient.
func WithUserClient(c *slack.Client) Option {
	return func(d *Dispatcher) { d.userClient = c }
}

// WithDB sets the database connection on the router.
func WithDB(db *gorm.DB) Option {
	return func(d *Dispatcher) { d.router.DbConnection = db }
}

// WithLogger sets the logger available as ctx.Logger.
func WithLogger(l zerolog.Logger) Option {
	return func(d *Dispatcher) { d.logger = l }
}

// WithMentionRoutes registers mention routes on the dispatcher.
func WithMentionRoutes(routes ...router.MentionRoute) Option {
	return func(d *Dispatcher) {
		d.router.AddMentionRoutes(routes)
	}
}

// WithChannelMessageRoutes registers channel message routes on the dispatcher.
func WithChannelMessageRoutes(routes ...router.ChannelMessageRoute) Option {
	return func(d *Dispatcher) {
		d.router.AddChannelMessageRoutes(routes)
	}
}

// WithSlashCommandRoutes registers slash command routes on the dispatcher.
func WithSlashCommandRoutes(routes ...router.SlashCommandRoute) Option {
	return func(d *Dispatcher) {
		d.router.AddSlashCommandRoutes(routes)
	}
}

// NewDispatcher creates a test Dispatcher with the given options.
func NewDispatcher(opts ...Option) *Dispatcher {
	d := &Dispatcher{
		router:    *router.NewRouter(),
		botClient: &slack.Client{},
		logger:    zerolog.Nop(),
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

func (d *Dispatcher) ctx() router.HandlerContext {
	return router.HandlerContext{
		Router:     d.router,
		BotClient:  d.botClient,
		UserClient: d.userClient,
		Logger:     d.logger,
	}
}

// DispatchMention finds the matching mention route for message and executes it
// synchronously. Returns an error if no route matches.
func (d *Dispatcher) DispatchMention(ev slackevents.AppMentionEvent, message string) error {
	route, found := d.router.FindMentionRouteByMessage(message)
	if !found {
		return errors.New("no matching mention route for: " + message)
	}
	route.Execute(d.ctx(), ev, message)
	return nil
}

// DispatchChannelMessage finds the matching channel message route for message
// and executes it synchronously. Returns an error if no route matches.
func (d *Dispatcher) DispatchChannelMessage(ev slackevents.MessageEvent, message string) error {
	route, found := d.router.FindChannelMessageRouteByMessage(message)
	if !found {
		return errors.New("no matching channel message route for: " + message)
	}
	route.Execute(d.ctx(), ev, message)
	return nil
}

// DispatchSlashCommand finds the matching slash command route and executes it
// synchronously. Returns an error if no route matches.
func (d *Dispatcher) DispatchSlashCommand(cmd slack.SlashCommand) error {
	route, found := d.router.FindSlashCommandRouteByCommand(cmd.Command)
	if !found {
		return errors.New("no matching slash command route for: " + cmd.Command)
	}
	route.Execute(d.ctx(), cmd)
	return nil
}
