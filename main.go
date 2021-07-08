package main

import (
	gadget "github.com/gadget-bot/gadget/core"
)

func main() {
	myBot := gadget.Setup()

	// Add your custom plugins here

	// This launches your bot
	myBot.Run()
}
