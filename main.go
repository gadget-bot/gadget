package main

import (
	gadget "github.com/gadget-bot/gadget/core"
	"github.com/gadget-bot/gadget/plugins/eightball"
)

func main() {
	myBot := gadget.Setup()

	// Add your custom plugins here
	myBot.Router.AddMentionRoutes(eightball.GetMentionRoutes())

	// This launches your bot
	myBot.Run()
}
