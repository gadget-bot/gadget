package permission_denied

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMentionRoute_Metadata(t *testing.T) {
	route := GetMentionRoute()

	assert.NotNil(t, route)
	assert.Equal(t, "permission_denied", route.Name)
	assert.Equal(t, []string{"*"}, route.Permissions)
	assert.NotNil(t, route.Plugin)
}

func TestGetChannelMessageRoute_Metadata(t *testing.T) {
	route := GetChannelMessageRoute()

	assert.NotNil(t, route)
	assert.Equal(t, "permission_denied", route.Name)
	assert.Equal(t, []string{"*"}, route.Permissions)
	assert.NotNil(t, route.Plugin)
}

func TestGetSlashCommandRoute_Metadata(t *testing.T) {
	route := GetSlashCommandRoute()

	assert.NotNil(t, route)
	assert.Equal(t, "permission_denied", route.Name)
	assert.Equal(t, []string{"*"}, route.Permissions)
	assert.NotNil(t, route.Plugin)
}
