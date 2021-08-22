package main

import (
	"os"

	gadget "github.com/gadget-bot/gadget/core"
)

func main() {
	myBot, err := gadget.Setup()
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}

	// Add your custom plugins here

	// This launches your bot

	myBot.Run()
}
