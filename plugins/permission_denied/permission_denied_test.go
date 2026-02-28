package permission_denied

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMentionRoute_Metadata(t *testing.T) {
	route := GetMentionRoute()

	assert.NotNil(t, route)
	assert.Equal(t, "permission_denied", route.Name)
	assert.Empty(t, route.Pattern, "permission_denied mention route should have no pattern")
	assert.Empty(t, route.Description, "permission_denied mention route has no description")
	assert.Empty(t, route.Help, "permission_denied mention route has no help text")
	assert.Equal(t, []string{"*"}, route.Permissions)
	assert.NotNil(t, route.Plugin)
}

func TestGetChannelMessageRoute_Metadata(t *testing.T) {
	route := GetChannelMessageRoute()

	assert.NotNil(t, route)
	assert.Equal(t, "permission_denied", route.Name)
	assert.Empty(t, route.Pattern, "permission_denied channel message route should have no pattern")
	assert.Empty(t, route.Description, "permission_denied channel message route has no description")
	assert.Empty(t, route.Help, "permission_denied channel message route has no help text")
	assert.Equal(t, []string{"*"}, route.Permissions)
	assert.NotNil(t, route.Plugin)
}

func TestGetSlashCommandRoute_Metadata(t *testing.T) {
	route := GetSlashCommandRoute()

	assert.NotNil(t, route)
	assert.Equal(t, "permission_denied", route.Name)
	assert.Empty(t, route.Pattern, "permission_denied slash command route should have no pattern")
	assert.Empty(t, route.Description, "permission_denied slash command route has no description")
	assert.Empty(t, route.Help, "permission_denied slash command route has no help text")
	assert.Equal(t, []string{"*"}, route.Permissions)
	assert.NotNil(t, route.Plugin)
}
