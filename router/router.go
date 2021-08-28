package router

import (
	"regexp"
	"sort"

	"github.com/gadget-bot/gadget/models"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"gorm.io/gorm"
)

//Route The primary type used by event specific routes
type Route struct {
	Name        string
	Pattern     string
	Description string
	Help        string
	Permissions []string
	Priority    int
}

//Router the HTTP router which handles Events from Slack
type Router struct {
	MentionRoutes       map[string]MentionRoute
	DefaultMentionRoute MentionRoute
	DeniedMentionRoute  MentionRoute
	DbConnection        *gorm.DB
}

// NewRouter returns a new Router
func NewRouter() *Router {
	var newRouter Router
	newRouter.MentionRoutes = make(map[string]MentionRoute)
	return &newRouter
}

// SetupDb migrates the shcemas
func (router Router) SetupDb() {
	// Migrate the schema
	router.DbConnection.AutoMigrate(&models.Group{})
	router.DbConnection.AutoMigrate(&models.User{})
}

// Find MentionRouteByName  Returns the named mention route
func (router Router) FindMentionRouteByName(name string) (MentionRoute, bool) {
	route, exists := router.MentionRoutes[name]
	return route, exists
}

// Find MentionRouteByMessage Returns the route to execute based on the first matched Route.Pattern.
func (router Router) FindMentionRouteByMessage(message string) (MentionRoute, bool) {
	var matchingRoute MentionRoute
	foundRoute := false
	sortedRoutes := make([]MentionRoute, 0, len(router.MentionRoutes))

	// Just need the Routes themselves for sorting
	for _, value := range router.MentionRoutes {
		sortedRoutes = append(sortedRoutes, value)
	}

	// Sort routes with highest priority first
	// TODO move this such that types Route implements Sort
	sort.Slice(sortedRoutes, func(i, j int) bool {
		return sortedRoutes[i].Priority > sortedRoutes[j].Priority
	})

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

// Execute calls the Plugin function provided
func (route MentionRoute) Execute(api slack.Client, router Router, ev slackevents.AppMentionEvent, message string) {
	route.Plugin(router, route.Route, api, ev, message)
}
