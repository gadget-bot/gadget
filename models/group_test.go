package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGroup_HasMember_ReturnsTrueWhenUserIsMember(t *testing.T) {
	group := Group{
		Name: "admins",
		Members: []User{
			{Uuid: "U111"},
			{Uuid: "U222"},
		},
	}
	user := User{Uuid: "U222"}

	assert.True(t, group.HasMember(user))
}

func TestGroup_HasMember_ReturnsFalseWhenUserIsNotMember(t *testing.T) {
	group := Group{
		Name: "admins",
		Members: []User{
			{Uuid: "U111"},
		},
	}
	user := User{Uuid: "U999"}

	assert.False(t, group.HasMember(user))
}

func TestGroup_HasMember_ReturnsFalseForEmptyGroup(t *testing.T) {
	group := Group{Name: "empty"}
	user := User{Uuid: "U111"}

	assert.False(t, group.HasMember(user))
}

func TestGroup_HasMember_ReturnsFalseForEmptyUserUuid(t *testing.T) {
	group := Group{
		Name:    "admins",
		Members: []User{{Uuid: "U111"}},
	}
	user := User{Uuid: ""}

	assert.False(t, group.HasMember(user))
}

func TestGroup_HasMember_ReturnsFalseWhenAllMembersHaveEmptyUuid(t *testing.T) {
	group := Group{
		Name:    "broken",
		Members: []User{{Uuid: ""}, {Uuid: ""}},
	}
	user := User{Uuid: "U111"}

	assert.False(t, group.HasMember(user))
}
