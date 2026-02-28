package user_info

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMentionRoutes_ReturnsOneRoute(t *testing.T) {
	routes := GetMentionRoutes()

	assert.Len(t, routes, 1)

	route := routes[0]
	assert.Equal(t, "user_info.userInfo", route.Name)
	assert.Equal(t, []string{"admins"}, route.Permissions)
	assert.Equal(t, `(?i)^(tell me about|who is) <@([a-z0-9]+)>[.?]?$`, route.Pattern)
	assert.Equal(t, "Responds with information about a Slack user", route.Description)
	assert.Equal(t, "who is USER", route.Help)
	assert.NotNil(t, route.Plugin)
}
