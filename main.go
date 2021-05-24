package main

import (
	gadget "github.com/gadget-bot/gadget/core"
	"github.com/gadget-bot/gadget/plugins/dice"
	"github.com/gadget-bot/gadget/plugins/how"
	"github.com/gadget-bot/gadget/plugins/network_utils"
)

func main() {
	myBot := gadget.Setup()

	// Add your custom plugins here
	myBot.Router.AddMentionRoutes(dice.GetMentionRoutes())
	myBot.Router.AddMentionRoutes(how.GetMentionRoutes())
	myBot.Router.AddMentionRoutes(network_utils.GetMentionRoutes())

	// This launches your bot
	myBot.Run()
}
