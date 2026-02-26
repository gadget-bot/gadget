package router

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
