package fallback

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMentionRoute_Metadata(t *testing.T) {
	route := GetMentionRoute()

	assert.NotNil(t, route)
	assert.Equal(t, "fallback", route.Name)
	assert.Equal(t, []string{"*"}, route.Permissions)
	assert.NotNil(t, route.Plugin)
}
