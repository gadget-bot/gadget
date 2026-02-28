package groups

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/gadget-bot/gadget/models"
	"github.com/gadget-bot/gadget/router"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"gorm.io/gorm"
)

func getMyGroups() *router.MentionRoute {
	var pluginRoute router.MentionRoute
	pluginRoute.Permissions = append(pluginRoute.Permissions, "*")
	pluginRoute.Name = "groups.getMyGroups"
	pluginRoute.Pattern = `(?i)^((list )?my groups|which groups am I (in|a member of))[.?]?$`
	pluginRoute.Plugin = func(router router.Router, route router.Route, api slack.Client, ev slackevents.AppMentionEvent, message string) {
		api.PostMessage(
			ev.Channel,
			slack.MsgOptionText("Here are your groups, <@"+ev.User+">:", false),
		)

		var currentUser models.User
		router.DbConnection.Preload("Groups").FirstOrCreate(&currentUser, models.User{Uuid: ev.User})

		var response string
		groupList := currentUser.Groups

		if len(groupList) > 0 {
			for _, group := range groupList {
				response += fmt.Sprintf("*-* %s\n", group.Name)
			}
		} else {
			response = "You don't seem to be a member of _any_ groups. Bummer."
		}

		api.PostMessage(
			ev.Channel,
			slack.MsgOptionText(response, false),
		)
	}
	return &pluginRoute
}

func getAllGroups() *router.MentionRoute {
	var pluginRoute router.MentionRoute
	pluginRoute.Permissions = append(pluginRoute.Permissions, "admins")
	pluginRoute.Name = "groups.getAllGroups"
	pluginRoute.Pattern = `(?i)^(list|list all|all) groups\.?$`
	pluginRoute.Plugin = func(router router.Router, route router.Route, api slack.Client, ev slackevents.AppMentionEvent, message string) {
		var groups []models.Group

		api.PostMessage(
			ev.Channel,
			slack.MsgOptionText("Here are *all* the groups I know about:", false),
		)

		router.DbConnection.Find(&groups)

		var response string

		for _, group := range groups {
			response += fmt.Sprintf("*-* %s\n", group.Name)
		}

		api.PostMessage(
			ev.Channel,
			slack.MsgOptionText(response, false),
		)
	}
	return &pluginRoute
}

func addUserToGroup() *router.MentionRoute {
	var pluginRoute router.MentionRoute
	pluginRoute.Permissions = append(pluginRoute.Permissions, "admins")
	pluginRoute.Name = "groups.addUserToGroup"
	pluginRoute.Pattern = `(?i)^add <@([a-z0-9]+)> to( group)? ([a-z0-9]+)\.?$`
	pluginRoute.Plugin = func(router router.Router, route router.Route, api slack.Client, ev slackevents.AppMentionEvent, message string) {
		msgRef := slack.NewRefToMessage(ev.Channel, ev.TimeStamp)
		api.AddReaction("tada", msgRef)

		re := regexp.MustCompile(route.Pattern)
		results := re.FindStringSubmatch(message)
		userName := results[1]
		groupName := results[3]
		var foundGroup models.Group
		var foundUser models.User

		router.DbConnection.Where(models.Group{Name: groupName}).FirstOrCreate(&foundGroup)
		router.DbConnection.Where(models.User{Uuid: userName}).FirstOrCreate(&foundUser)
		router.DbConnection.Model(&foundGroup).Association("Members").Append(&foundUser)

		api.PostMessage(
			ev.Channel,
			slack.MsgOptionText(fmt.Sprintf("I successfully added <@%s> to %s!", userName, groupName), false),
		)
	}
	return &pluginRoute
}

func removeUserFromGroup() *router.MentionRoute {
	var pluginRoute router.MentionRoute
	pluginRoute.Permissions = append(pluginRoute.Permissions, "admins")
	pluginRoute.Name = "groups.removeUserFromGroup"
	pluginRoute.Pattern = `(?i)^remove <@([a-z0-9]+)> from( group)? ([a-z0-9]+)\.?$`
	pluginRoute.Plugin = func(router router.Router, route router.Route, api slack.Client, ev slackevents.AppMentionEvent, message string) {
		msgRef := slack.NewRefToMessage(ev.Channel, ev.TimeStamp)
		api.AddReaction("slightly_frowning_face", msgRef)

		re := regexp.MustCompile(route.Pattern)
		results := re.FindStringSubmatch(message)
		userName := results[1]
		groupName := results[3]
		var foundGroup models.Group
		var foundUser models.User
		var response string
		var wasMember bool

		router.DbConnection.Where(models.User{Uuid: userName}).FirstOrCreate(&foundUser)
		groupQueryResult := router.DbConnection.Preload("Members").Where(models.Group{Name: groupName}).First(&foundGroup)

		if errors.Is(groupQueryResult.Error, gorm.ErrRecordNotFound) {
			response = fmt.Sprintf("I couldn't find a group named '%s'.", groupName)
		} else {
			var newMembersList []models.User

			// Create a new list of members to replace the old (removing the specified user)
			for _, member := range foundGroup.Members {
				if member.Uuid == foundUser.Uuid {
					wasMember = true
				} else {
					newMembersList = append(newMembersList, member)
				}
			}

			if wasMember {
				router.DbConnection.Model(&foundGroup).Association("Members").Replace(newMembersList)
				response = fmt.Sprintf("<@%s> is no longer a member of %s!", userName, groupName)
			} else {
				response = fmt.Sprintf("It doesn't look like <@%s> is a member of %s.", userName, groupName)
			}
		}

		api.PostMessage(
			ev.Channel,
			slack.MsgOptionText(response, false),
		)
	}
	return &pluginRoute
}

// GetMentionRoutes Slice of all MentionRoutes
func GetMentionRoutes() []router.MentionRoute {
	return []router.MentionRoute{
		*getMyGroups(),
		*getAllGroups(),
		*addUserToGroup(),
		*removeUserFromGroup(),
	}
}
