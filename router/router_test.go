package router

import (
	"testing"

	"github.com/gadget-bot/gadget/models"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to open in-memory SQLite: %v", err)
	}
	if err := db.AutoMigrate(&models.Group{}, &models.User{}); err != nil {
		t.Fatalf("Failed to auto-migrate: %v", err)
	}
	return db
}

func TestUpdateBotUID_ValidBody(t *testing.T) {
	r := NewRouter()
	body := []byte(`{"authorizations":[{"user_id":"U_BOT","team_id":"T123"}]}`)
	err := r.UpdateBotUID(body)
	assert.NoError(t, err)
	assert.Equal(t, "U_BOT", r.BotUID)
}

func TestUpdateBotUID_InvalidJSON(t *testing.T) {
	r := NewRouter()
	err := r.UpdateBotUID([]byte(`not json`))
	assert.Error(t, err)
	assert.Empty(t, r.BotUID)
}

func TestUpdateBotUID_MissingAuthorizations(t *testing.T) {
	r := NewRouter()
	err := r.UpdateBotUID([]byte(`{"authorizations":[]}`))
	assert.Error(t, err)
	assert.Empty(t, r.BotUID)
}

func TestUpdateBotUID_AlreadySet(t *testing.T) {
	r := NewRouter()
	r.BotUID = "U_EXISTING"
	body := []byte(`{"authorizations":[{"user_id":"U_NEW","team_id":"T123"}]}`)
	err := r.UpdateBotUID(body)
	assert.NoError(t, err)
	assert.Equal(t, "U_EXISTING", r.BotUID)
}

func TestRegisteredRoutes_Empty(t *testing.T) {
	r := NewRouter()
	routes := r.RegisteredRoutes()
	assert.Empty(t, routes)
}

func TestRegisteredRoutes_IncludesBothTypes(t *testing.T) {
	r := NewRouter()
	r.AddMentionRoute(MentionRoute{
		Route: Route{Name: "mention1", Description: "A mention route", Priority: 1},
	})
	r.AddChannelMessageRoute(ChannelMessageRoute{
		Route: Route{Name: "channel1", Description: "A channel route", Priority: 1},
	})

	routes := r.RegisteredRoutes()
	assert.Len(t, routes, 2)

	types := map[string]bool{}
	for _, route := range routes {
		types[route.Type] = true
	}
	assert.True(t, types[RouteTypeMention])
	assert.True(t, types[RouteTypeChannelMessage])
}

func TestRegisteredRoutes_SortedByPriorityDescending(t *testing.T) {
	r := NewRouter()
	r.AddMentionRoute(MentionRoute{
		Route: Route{Name: "low", Priority: 1},
	})
	r.AddMentionRoute(MentionRoute{
		Route: Route{Name: "high", Priority: 10},
	})
	r.AddMentionRoute(MentionRoute{
		Route: Route{Name: "mid", Priority: 5},
	})

	routes := r.RegisteredRoutes()
	assert.Equal(t, "high", routes[0].Name)
	assert.Equal(t, "mid", routes[1].Name)
	assert.Equal(t, "low", routes[2].Name)
}

func TestRegisteredRoutes_SortedByNameWhenPriorityEqual(t *testing.T) {
	r := NewRouter()
	r.AddMentionRoute(MentionRoute{
		Route: Route{Name: "beta", Priority: 5},
	})
	r.AddMentionRoute(MentionRoute{
		Route: Route{Name: "alpha", Priority: 5},
	})

	routes := r.RegisteredRoutes()
	assert.Equal(t, "alpha", routes[0].Name)
	assert.Equal(t, "beta", routes[1].Name)
}

func TestRegisteredRoutes_ExcludesDefaultAndDenied(t *testing.T) {
	r := NewRouter()
	r.DefaultMentionRoute = MentionRoute{
		Route: Route{Name: "fallback"},
	}
	r.DeniedMentionRoute = MentionRoute{
		Route: Route{Name: "permission_denied"},
	}
	r.AddMentionRoute(MentionRoute{
		Route: Route{Name: "real_route", Priority: 1},
	})

	routes := r.RegisteredRoutes()
	assert.Len(t, routes, 1)
	assert.Equal(t, "real_route", routes[0].Name)
}

func TestAddSlashCommandRoute(t *testing.T) {
	r := NewRouter()
	route := SlashCommandRoute{
		Route:   Route{Name: "deploy", Description: "Deploy the app"},
		Command: "/deploy",
	}
	r.AddSlashCommandRoute(route)

	assert.Len(t, r.SlashCommandRoutes, 1)
	assert.Equal(t, "deploy", r.SlashCommandRoutes["/deploy"].Name)
}

func TestAddSlashCommandRoutes(t *testing.T) {
	r := NewRouter()
	routes := []SlashCommandRoute{
		{Route: Route{Name: "deploy", Description: "Deploy"}, Command: "/deploy"},
		{Route: Route{Name: "rollback", Description: "Rollback"}, Command: "/rollback"},
	}
	r.AddSlashCommandRoutes(routes)

	assert.Len(t, r.SlashCommandRoutes, 2)
	assert.Equal(t, "deploy", r.SlashCommandRoutes["/deploy"].Name)
	assert.Equal(t, "rollback", r.SlashCommandRoutes["/rollback"].Name)
}

func TestFindSlashCommandRouteByCommand_Found(t *testing.T) {
	r := NewRouter()
	r.AddSlashCommandRoute(SlashCommandRoute{
		Route:   Route{Name: "deploy", Description: "Deploy"},
		Command: "/deploy",
	})

	route, exists := r.FindSlashCommandRouteByCommand("/deploy")
	assert.True(t, exists)
	assert.Equal(t, "deploy", route.Name)
	assert.Equal(t, "/deploy", route.Command)
}

func TestFindSlashCommandRouteByCommand_NotFound(t *testing.T) {
	r := NewRouter()
	r.AddSlashCommandRoute(SlashCommandRoute{
		Route:   Route{Name: "deploy", Description: "Deploy"},
		Command: "/deploy",
	})

	_, exists := r.FindSlashCommandRouteByCommand("/rollback")
	assert.False(t, exists)
}

func TestRegisteredRoutes_IncludesSlashCommands(t *testing.T) {
	r := NewRouter()
	r.AddMentionRoute(MentionRoute{
		Route: Route{Name: "mention1", Priority: 1},
	})
	r.AddSlashCommandRoute(SlashCommandRoute{
		Route:   Route{Name: "deploy", Priority: 1},
		Command: "/deploy",
	})

	routes := r.RegisteredRoutes()
	assert.Len(t, routes, 2)

	types := map[string]bool{}
	for _, route := range routes {
		types[route.Type] = true
	}
	assert.True(t, types[RouteTypeMention])
	assert.True(t, types[RouteTypeSlashCommand])
}

func TestRegisteredRoutes_PreservesMetadata(t *testing.T) {
	r := NewRouter()
	r.AddMentionRoute(MentionRoute{
		Route: Route{
			Name:        "test",
			Pattern:     `(?i)^test$`,
			Description: "A test route",
			Help:        "Say 'test' to test",
			Permissions: []string{"admins"},
			Priority:    5,
		},
	})

	routes := r.RegisteredRoutes()
	assert.Len(t, routes, 1)
	assert.Equal(t, "test", routes[0].Name)
	assert.Equal(t, `(?i)^test$`, routes[0].Pattern)
	assert.Equal(t, "A test route", routes[0].Description)
	assert.Equal(t, "Say 'test' to test", routes[0].Help)
	assert.Equal(t, []string{"admins"}, routes[0].Permissions)
	assert.Equal(t, 5, routes[0].Priority)
	assert.Equal(t, RouteTypeMention, routes[0].Type)
}

func TestAddMentionRoute_CompilesPattern(t *testing.T) {
	r := NewRouter()
	r.AddMentionRoute(MentionRoute{
		Route: Route{Name: "test", Pattern: `(?i)^hello`},
	})
	route := r.MentionRoutes["test"]
	assert.NotNil(t, route.CompiledPattern)
	assert.True(t, route.CompiledPattern.MatchString("Hello world"))
	assert.False(t, route.CompiledPattern.MatchString("goodbye"))
}

func TestAddMentionRoute_EmptyPatternNilCompiled(t *testing.T) {
	r := NewRouter()
	r.AddMentionRoute(MentionRoute{
		Route: Route{Name: "no-pattern"},
	})
	route := r.MentionRoutes["no-pattern"]
	assert.Nil(t, route.CompiledPattern)
}

func TestAddMentionRoute_InvalidPatternPanics(t *testing.T) {
	r := NewRouter()
	assert.Panics(t, func() {
		r.AddMentionRoute(MentionRoute{
			Route: Route{Name: "bad", Pattern: `(?P<invalid`},
		})
	})
}

func TestAddChannelMessageRoute_CompilesPattern(t *testing.T) {
	r := NewRouter()
	r.AddChannelMessageRoute(ChannelMessageRoute{
		Route: Route{Name: "test", Pattern: `(?i)^deploy`},
	})
	route := r.ChannelMessageRoutes["test"]
	assert.NotNil(t, route.CompiledPattern)
	assert.True(t, route.CompiledPattern.MatchString("deploy production"))
}

func TestAddChannelMessageRoute_InvalidPatternPanics(t *testing.T) {
	r := NewRouter()
	assert.Panics(t, func() {
		r.AddChannelMessageRoute(ChannelMessageRoute{
			Route: Route{Name: "bad", Pattern: `[invalid`},
		})
	})
}

func TestAddSlashCommandRoute_CompilesPattern(t *testing.T) {
	r := NewRouter()
	r.AddSlashCommandRoute(SlashCommandRoute{
		Route:   Route{Name: "deploy", Pattern: `(?i)^production`},
		Command: "/deploy",
	})
	route := r.SlashCommandRoutes["/deploy"]
	assert.NotNil(t, route.CompiledPattern)
}

func TestFindMentionRouteByMessage_UsesCompiledPattern(t *testing.T) {
	r := NewRouter()
	r.AddMentionRoute(MentionRoute{
		Route:  Route{Name: "greet", Pattern: `(?i)^hello`, Priority: 1},
		Plugin: func(router Router, route Route, api slack.Client, ev slackevents.AppMentionEvent, message string) {},
	})
	r.AddMentionRoute(MentionRoute{
		Route:  Route{Name: "farewell", Pattern: `(?i)^goodbye`, Priority: 1},
		Plugin: func(router Router, route Route, api slack.Client, ev slackevents.AppMentionEvent, message string) {},
	})

	route, found := r.FindMentionRouteByMessage("Hello there")
	assert.True(t, found)
	assert.Equal(t, "greet", route.Name)

	route, found = r.FindMentionRouteByMessage("goodbye friend")
	assert.True(t, found)
	assert.Equal(t, "farewell", route.Name)

	_, found = r.FindMentionRouteByMessage("something else")
	assert.False(t, found)
}

func TestFindChannelMessageRouteByMessage_UsesCompiledPattern(t *testing.T) {
	r := NewRouter()
	r.AddChannelMessageRoute(ChannelMessageRoute{
		Route:  Route{Name: "deploy", Pattern: `(?i)^deploy`, Priority: 1},
		Plugin: func(router Router, route Route, api slack.Client, ev slackevents.MessageEvent, message string) {},
	})

	route, found := r.FindChannelMessageRouteByMessage("deploy production")
	assert.True(t, found)
	assert.Equal(t, "deploy", route.Name)

	_, found = r.FindChannelMessageRouteByMessage("rollback")
	assert.False(t, found)
}

func TestCan_GlobalAdminAllowed(t *testing.T) {
	db := setupTestDB(t)
	r := NewRouter()
	r.DbConnection = db

	user := models.User{Uuid: "U_ADMIN"}
	db.Create(&user)
	group := models.Group{Name: "globalAdmins"}
	db.Create(&group)
	db.Model(&group).Association("Members").Append(&user)

	assert.True(t, r.Can(user, []string{"some_permission"}))
}

func TestCan_EmptyPermissionsAllowAll(t *testing.T) {
	db := setupTestDB(t)
	r := NewRouter()
	r.DbConnection = db

	user := models.User{Uuid: "U_REGULAR"}
	db.Create(&user)

	assert.True(t, r.Can(user, []string{}))
}

func TestCan_WildcardPermissionAllowsAll(t *testing.T) {
	db := setupTestDB(t)
	r := NewRouter()
	r.DbConnection = db

	user := models.User{Uuid: "U_REGULAR"}
	db.Create(&user)

	assert.True(t, r.Can(user, []string{"*"}))
}

func TestCan_UserInMatchingGroup(t *testing.T) {
	db := setupTestDB(t)
	r := NewRouter()
	r.DbConnection = db

	user := models.User{Uuid: "U_DEPLOYER"}
	db.Create(&user)
	group := models.Group{Name: "deployers"}
	db.Create(&group)
	db.Model(&group).Association("Members").Append(&user)

	assert.True(t, r.Can(user, []string{"deployers"}))
}

func TestCan_UserInNonMatchingGroup(t *testing.T) {
	db := setupTestDB(t)
	r := NewRouter()
	r.DbConnection = db

	user := models.User{Uuid: "U_VIEWER"}
	db.Create(&user)
	group := models.Group{Name: "viewers"}
	db.Create(&group)
	db.Model(&group).Association("Members").Append(&user)

	assert.False(t, r.Can(user, []string{"deployers"}))
}

func TestCan_UserInMultipleGroupsOneMatching(t *testing.T) {
	db := setupTestDB(t)
	r := NewRouter()
	r.DbConnection = db

	user := models.User{Uuid: "U_MULTI"}
	db.Create(&user)
	viewers := models.Group{Name: "viewers"}
	deployers := models.Group{Name: "deployers"}
	db.Create(&viewers)
	db.Create(&deployers)
	db.Model(&viewers).Association("Members").Append(&user)
	db.Model(&deployers).Association("Members").Append(&user)

	assert.True(t, r.Can(user, []string{"deployers"}))
}

func TestCan_UserWithNoGroupsDenied(t *testing.T) {
	db := setupTestDB(t)
	r := NewRouter()
	r.DbConnection = db

	user := models.User{Uuid: "U_LONELY"}
	db.Create(&user)

	assert.False(t, r.Can(user, []string{"deployers"}))
}

func TestCan_NilPermissionsAllowAll(t *testing.T) {
	db := setupTestDB(t)
	r := NewRouter()
	r.DbConnection = db

	user := models.User{Uuid: "U_NIL"}
	db.Create(&user)

	assert.True(t, r.Can(user, nil))
}
