package router

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"

	"github.com/gadget-bot/gadget/models"

	"gorm.io/gorm"
)

// Route The primary type used by event specific routes
type Route struct {
	Name        string
	Pattern     string
	Description string
	Help        string
	Permissions []string
	Priority    int
}

const (
	RouteTypeMention        = "mention"
	RouteTypeChannelMessage = "channel_message"
	RouteTypeSlashCommand   = "slash_command"
)

// RegisteredRoute wraps a Route with its type for introspection
type RegisteredRoute struct {
	Route
	Type string // RouteTypeMention, RouteTypeChannelMessage, or RouteTypeSlashCommand
}

// Router the HTTP router which handles Events from Slack
type Router struct {
	MentionRoutes           map[string]MentionRoute
	ChannelMessageRoutes    map[string]ChannelMessageRoute
	SlashCommandRoutes      map[string]SlashCommandRoute
	DefaultMentionRoute     MentionRoute
	DeniedMentionRoute      MentionRoute
	DeniedSlashCommandRoute SlashCommandRoute
	DbConnection            *gorm.DB
	BotUID                  string
}

// this is required because slack-go doesn't seem to provide a way to get the bot's own ID
type EventsAPICallbackEvent struct {
	Type           string                      `json:"type"`
	Token          string                      `json:"token"`
	TeamID         string                      `json:"team_id"`
	APIAppID       string                      `json:"api_app_id"`
	Authorizations []EventMessageAuthorization `json:"authorizations"`
	EventID        string                      `json:"event_id"`
	EventTime      int                         `json:"event_time"`
	EventContext   string                      `json:"event_context"`
}
type EventMessageAuthorization struct {
	UserId string `json:"user_id"`
	TeamId string `json:"team_id"`
}

// NewRouter returns a new Router
func NewRouter() *Router {
	var newRouter Router
	newRouter.MentionRoutes = make(map[string]MentionRoute)
	newRouter.ChannelMessageRoutes = make(map[string]ChannelMessageRoute)
	newRouter.SlashCommandRoutes = make(map[string]SlashCommandRoute)
	return &newRouter
}

// UpdateUID sets the UID field from an event body. Only updates if currently empty
func (r *Router) UpdateBotUID(body []byte) error {
	if r.BotUID != "" {
		return nil
	}
	uid, err := getBotUidFromBody(body)
	r.BotUID = uid
	return err
}

func getBotUidFromBody(body []byte) (string, error) {
	var authorizedUsers EventsAPICallbackEvent
	if err := json.Unmarshal(body, &authorizedUsers); err != nil {
		return "", fmt.Errorf("unmarshal event body: %w", err)
	}

	if len(authorizedUsers.Authorizations) > 0 {
		return authorizedUsers.Authorizations[0].UserId, nil
	} else {
		return "", errors.New("no authorized users in event body")
	}
}

// SetupDb migrates the shcemas
func (router Router) SetupDb() {
	// Migrate the schema
	router.DbConnection.AutoMigrate(&models.Group{})
	router.DbConnection.AutoMigrate(&models.User{})
}

// FindChannelMessageRouteByName looks up and return the ChannelMessageRoute by the provided Name field value
func (router Router) FindChannelMessageRouteByName(name string) (ChannelMessageRoute, bool) {
	route, exists := router.ChannelMessageRoutes[name]
	return route, exists
}

// FindMentionRouteByName Returns the named mention route
func (router Router) FindMentionRouteByName(name string) (MentionRoute, bool) {
	route, exists := router.MentionRoutes[name]
	return route, exists
}

// FindChannelMessageRouteByMessage Returns the ChannelMessageRoute that matches the provided message
func (router Router) FindChannelMessageRouteByMessage(message string) (ChannelMessageRoute, bool) {
	var matchingRoute ChannelMessageRoute
	foundRoute := false
	sortedRoutes := make([]ChannelMessageRoute, 0, len(router.ChannelMessageRoutes))

	// Just need the Routes themselves for sorting
	for _, value := range router.ChannelMessageRoutes {
		sortedRoutes = append(sortedRoutes, value)
	}

	sort.Sort(channelMessageRoutesSortedByPriority(sortedRoutes))

	for _, route := range sortedRoutes {
		re := regexp.MustCompile(route.Pattern)
		if re.MatchString(message) {
			matchingRoute = route
			foundRoute = true
			break
		}
	}

	return matchingRoute, foundRoute
}

// FindMentionRouteByMessage Returns the route to execute based on the first matched Route.Pattern.
func (router Router) FindMentionRouteByMessage(message string) (MentionRoute, bool) {
	var matchingRoute MentionRoute
	foundRoute := false
	sortedRoutes := make([]MentionRoute, 0, len(router.MentionRoutes))

	// Just need the Routes themselves for sorting
	for _, value := range router.MentionRoutes {
		sortedRoutes = append(sortedRoutes, value)
	}
	sort.Sort(mentionRoutesSortedByPriority(sortedRoutes))

	for _, route := range sortedRoutes {
		re := regexp.MustCompile(route.Pattern)
		if re.MatchString(message) {
			matchingRoute = route
			foundRoute = true
			break
		}
	}
	return matchingRoute, foundRoute
}

// Can Returns true if `u` possesses the provided permissions
func (router Router) Can(u models.User, permissions []string) bool {
	isAllowed := false
	var userGroupNames []string
	var userGroups []models.Group

	router.DbConnection.Model(&u).Association("Groups").Find(&userGroups)

	for _, userGroup := range userGroups {
		groupName := userGroup.Name
		// If the user is a global admin, let them through
		if groupName == "globalAdmins" {
			isAllowed = true
			break
		}
		userGroupNames = append(userGroupNames, userGroup.Name)
	}

	if isAllowed {
		return isAllowed
	} else if len(permissions) == 0 {
		// if no permissions are defined, assume it is open/allow all
		return true
	} else {
		for _, groupName := range permissions {
			// If everyone is allowed, stop checking
			if groupName == "*" {
				isAllowed = true
				break
			}

			// user groups _must_ be smaller than all groups
			for _, userGroup := range userGroupNames {
				if groupName == userGroup {
					isAllowed = true
					break
				}
			}
			if isAllowed {
				break
			}
		}
	}
	return isAllowed
}

// AddMentionRoute sets upserts and element into `MentionRoutes` whose key is the provided `Name` field
func (router *Router) AddMentionRoute(route MentionRoute) {
	router.MentionRoutes[route.Name] = route
}

// AddMentionRoutes calls `AddMentionRoute()` for each element in `routes`
func (router *Router) AddMentionRoutes(routes []MentionRoute) {
	for _, route := range routes {
		router.AddMentionRoute(route)
	}
}

// AddChannelMessageRoute sets the key for ChannelMessages key to route.Name and it's value to route
func (router *Router) AddChannelMessageRoute(route ChannelMessageRoute) {
	router.ChannelMessageRoutes[route.Name] = route
}

// AddChannelMessageRoutes same as AddChannelMessageRoute but plural
func (router *Router) AddChannelMessageRoutes(routes []ChannelMessageRoute) {
	for _, route := range routes {
		router.AddChannelMessageRoute(route)
	}
}

// AddSlashCommandRoute adds a slash command route keyed by its Name
func (router *Router) AddSlashCommandRoute(route SlashCommandRoute) {
	router.SlashCommandRoutes[route.Command] = route
}

// AddSlashCommandRoutes calls AddSlashCommandRoute for each element in routes
func (router *Router) AddSlashCommandRoutes(routes []SlashCommandRoute) {
	for _, route := range routes {
		router.AddSlashCommandRoute(route)
	}
}

// FindSlashCommandRouteByCommand looks up a SlashCommandRoute by command name
func (router Router) FindSlashCommandRouteByCommand(command string) (SlashCommandRoute, bool) {
	route, exists := router.SlashCommandRoutes[command]
	return route, exists
}

// RegisteredRoutes returns all registered routes sorted by priority (descending),
// then by name (alphabetical). DefaultMentionRoute and DeniedMentionRoute are excluded
// because they are stored as separate struct fields, not entries in the route maps.
func (router Router) RegisteredRoutes() []RegisteredRoute {
	routes := make([]RegisteredRoute, 0, len(router.MentionRoutes)+len(router.ChannelMessageRoutes)+len(router.SlashCommandRoutes))

	for _, r := range router.MentionRoutes {
		routes = append(routes, RegisteredRoute{Route: r.Route, Type: RouteTypeMention})
	}
	for _, r := range router.ChannelMessageRoutes {
		routes = append(routes, RegisteredRoute{Route: r.Route, Type: RouteTypeChannelMessage})
	}
	for _, r := range router.SlashCommandRoutes {
		routes = append(routes, RegisteredRoute{Route: r.Route, Type: RouteTypeSlashCommand})
	}

	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Priority != routes[j].Priority {
			return routes[i].Priority > routes[j].Priority
		}
		return routes[i].Name < routes[j].Name
	})

	return routes
}
