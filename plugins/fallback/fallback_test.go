package fallback

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMentionRoute_Metadata(t *testing.T) {
	route := GetMentionRoute()

	assert.NotNil(t, route)
	assert.Equal(t, "fallback", route.Name)
	assert.Empty(t, route.Pattern, "fallback route should have no pattern (matches nothing explicitly)")
	assert.Empty(t, route.Description, "fallback route has no description")
	assert.Empty(t, route.Help, "fallback route has no help text")
	assert.Equal(t, []string{"*"}, route.Permissions)
	assert.NotNil(t, route.Plugin)
}
