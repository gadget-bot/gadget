# Gadget

Gadget is a [Golang](https://golang.org/) bot for Slack's [Events API](https://api.slack.com/events). The concepts in Gadget are heavily inspired by [Lita](https://www.lita.io/) in that it contains support for Regular Expression-based _routing_ of requests to plugins, it supports permissions based on groups managed via the bot, and it uses a DB for persisting these details.

Gadget makes use of [Goroutines](https://golangbot.com/goroutines/) to support many users and to respond quickly, but it can also be scaled out as needed. For persistence, Gadget uses [GORM](https://gorm.io/) but it only supports [MySQL](https://www.mysql.com/) (and MySQL-compatible RDBMS's like [MariaDB](https://mariadb.org/)).

Note that Gadget is still very much a **work in progress**, so please don't use it in production yet (or if you do, don't complain).

## Why Gadget?

Why not just use [slack-go/slack](https://github.com/slack-go/slack) or some other Golang Slack client? There don't seem to be many (maybe even _any_) good, full-featured frameworks for building bots in Golang. It's true, [slack-go/slack](https://github.com/slack-go/slack) is great. So great, in fact, that Gadget _uses_ it for talking to Slack. What [slack-go/slack](https://github.com/slack-go/slack) (and other projects, it seems) are lacking are the built-in features like permissions, a simple plugin-based approach to adding capabilities, and intuitive support for persisting data.

Having a ChatBot is fine if everyone can do everything, but this isn't always the case. Being able to restrict certain capabilities in a straightforward way is a pretty important feature and seemed worth developing. It is also really handy to simplify making a bot, cutting out all the boiler-plate. Gadget is meant to solve these problems; making a bot that you can talk to is made very simple so you can get to work writing features.

## Building your own Bot

Here is what your bot's `main.go` should look like:

```golang
package main

import (
	gadget "github.com/gadget-bot/gadget/core"
	"github.com/gadget-bot/gadget/plugins/how"
)

func main() {
	myBot := gadget.Setup()

	// Add your custom mention plugins here
  
	// Add a single Route
	myBot.Router.AddMentionRoute(*myPlugin.SomeFunction())
	// Add a slice of Routes
	myBot.Router.AddMentionRoutes(myPlugin.GetMentionRoutes())

	// This launches your bot
	myBot.Run()
}
```

## Writing a Plugin

Gadget is built around specialized plugins called `Routes`. A Route **must** provide:

* a `Pattern` (a `string` that can be parsed as a `regexp.Regexp`) that defines when it should be called
* a `Name` (a unique `string`) that is used for logging
* a `Plugin`, which is the meat of what the Route should do when called. It needs to return a function, but this depends on which type of `Route` is being written. For normal `MentionRoute`s, the returned function must look something like:
```golang
func(api slack.Client, router router.Router, ev slackevents.AppMentionEvent, message string) {
  // ... do something awesome here ...
}
```

A `Route` can optionally provide:

* a `Permissions` list (of type `[]string`) that provides a list of `Group`s that can use the `Route`. If that list is empty, not provided, or includes `"*"`, it will allow all users.
* a `Help` (of type `string`) that explains how to access the `Route`
* a `Description` (of type `string`) to describe what the `Route` does
* a `Priority` (of type `int`) to inform Gadget's `Router` which `Route` to choose when more than one match (higher `Priority` wins)

Let's work on a simple example. This is taken straight from our [dice-rolling example](plugins/dice/dice.go):

```golang
package dice

import (
	"fmt"
	"math/rand"

	"github.com/gadget-bot/gadget/router"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

func rollD6() *router.MentionRoute {
	var pluginRoute router.MentionRoute
	pluginRoute.Permissions = append(pluginRoute.Permissions, "*")
	pluginRoute.Name = "dice.rollD6"
	pluginRoute.Help = "roll some dice"
	pluginRoute.Description = "Rolls two d6 dice"
	pluginRoute.Pattern = `(?i)^roll some dice[!.]?$`

	// Here is where we define what we want this plugin to do
	pluginRoute.Plugin = func(api slack.Client, router router.Router, ev slackevents.AppMentionEvent, message string) {
		// Here's how we can react to the message
		msgRef := slack.NewRefToMessage(ev.Channel, ev.TimeStamp)
		api.AddReaction("game_die", msgRef)

		// Roll a virtual dice
		dice := []int{1, 2, 3, 4, 5, 6}
		rollIndex1 := rand.Intn(len(dice))
		rollIndex2 := rand.Intn(len(dice))
		roll1 := dice[rollIndex1]
		roll2 := dice[rollIndex2]

		// Here's how we send a reply
		api.PostMessage(
			ev.Channel,
			slack.MsgOptionText(
				fmt.Sprintf("<@%s> rolled a %d and a %d", ev.User, roll1, roll2),
				false,
			),
		)
	}

	// We've got to return the MentionRoute
	return &pluginRoute
}

// This function is used to retrieve all Mention Routes from this plugin
func GetMentionRoutes() []router.MentionRoute {
	return []router.MentionRoute{
		*rollD6(),
	}
}
```

From there, we can just extend the `main.go` above, changing the `func main()` to look like this:

```golang
func main() {
	myBot := gadget.Setup()

	// Add your custom mention plugins here
  
	// Add a slice of Routes
	myBot.Router.AddMentionRoutes(dice.GetMentionRoutes())

	// This launches your bot
	myBot.Run()
}
```

That's it! The above actually _is_ a real plugin and lives in its [own repo](https://github.com/gadget-bot/gadget-plugin-dice). PRs welcome!

## Starting a Demo

If you just want to try Gadget out, you can use the `main.go` in this repo like this:

```sh
#!/bin/sh

# These users are global admins for Gadget
export GADGET_GLOBAL_ADMINS="U0.....,U1....."
# These two variables are for connecting to Slack
export SLACK_OAUTH_TOKEN="xoxb-...."
export SLACK_SIGNING_SECRET="a...a"
# DB Connection details
export GADGET_DB_USER="gadgetuser"
export GADGET_DB_PASS="secretpassword"
export GADGET_DB_HOST="127.0.0.1:3306"
export GADGET_DB_NAME="gadget_dev"
# The port Gadget's webhook server listens on
export GADGET_LISTEN_PORT="3000"

go run .
```

Gadget will be listening on port 3000. You can use something like [`ngrok`](https://ngrok.com/) to expose Gadget in a way that you can configure Slack to talk to it.
