package router

import (
	"regexp"
	"sort"

	"github.com/gadget-bot/gadget/models"

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
	MentionRoutes        map[string]MentionRoute
	ChannelMessageRoutes map[string]ChannelMessageRoute
	DefaultMentionRoute  MentionRoute
	DeniedMentionRoute   MentionRoute
	DbConnection         *gorm.DB
}

// NewRouter returns a new Router
func NewRouter() *Router {
	var newRouter Router
	newRouter.MentionRoutes = make(map[string]MentionRoute)
	newRouter.ChannelMessageRoutes = make(map[string]ChannelMessageRoute)
	return &newRouter
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
func (router Router) AddMentionRoute(route MentionRoute) {
	router.MentionRoutes[route.Name] = route
}

// AddMentionRoutes calls `AddMentionRoute()` for each element in `routes`
func (router Router) AddMentionRoutes(routes []MentionRoute) {
	for _, route := range routes {
		router.AddMentionRoute(route)
	}
}

// AddChannelMessageRoute sets the key for ChannelMessages key to route.Name and it's value to route
func (router Router) AddChannelMessageRoute(route ChannelMessageRoute) {
	router.ChannelMessageRoutes[route.Name] = route
}

// AddChannelMessageRoutes same as AddChannelMessageRoute but plural
func (router Router) AddChannelMessageRoutes(routes []ChannelMessageRoute) {
	for _, route := range routes {
		router.AddChannelMessageRoute(route)
	}
}
