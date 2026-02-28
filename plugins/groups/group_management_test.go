package groups

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMentionRoutes_ReturnsAllRoutes(t *testing.T) {
	routes := GetMentionRoutes()

	assert.Len(t, routes, 4)

	expectedNames := []string{
		"groups.getMyGroups",
		"groups.getAllGroups",
		"groups.addUserToGroup",
		"groups.removeUserFromGroup",
	}

	for i, name := range expectedNames {
		assert.Equal(t, name, routes[i].Name)
	}
}

func TestGetMyGroups_Metadata(t *testing.T) {
	route := getMyGroups()

	assert.Equal(t, "groups.getMyGroups", route.Name)
	assert.Equal(t, `(?i)^((list )?my groups|which groups am I (in|a member of))[.?]?$`, route.Pattern)
	assert.Equal(t, []string{"*"}, route.Permissions)
	assert.NotNil(t, route.Plugin)
}

func TestGetAllGroups_Metadata(t *testing.T) {
	route := getAllGroups()

	assert.Equal(t, "groups.getAllGroups", route.Name)
	assert.Equal(t, `(?i)^(list|list all|all) groups\.?$`, route.Pattern)
	assert.Equal(t, []string{"admins"}, route.Permissions)
	assert.NotNil(t, route.Plugin)
}

func TestAddUserToGroup_Metadata(t *testing.T) {
	route := addUserToGroup()

	assert.Equal(t, "groups.addUserToGroup", route.Name)
	assert.Equal(t, `(?i)^add <@([a-z0-9]+)> to( group)? ([a-z0-9]+)\.?$`, route.Pattern)
	assert.Equal(t, []string{"admins"}, route.Permissions)
	assert.NotNil(t, route.Plugin)
}

func TestRemoveUserFromGroup_Metadata(t *testing.T) {
	route := removeUserFromGroup()

	assert.Equal(t, "groups.removeUserFromGroup", route.Name)
	assert.Equal(t, `(?i)^remove <@([a-z0-9]+)> from( group)? ([a-z0-9]+)\.?$`, route.Pattern)
	assert.Equal(t, []string{"admins"}, route.Permissions)
	assert.NotNil(t, route.Plugin)
}
