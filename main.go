package main

import (
	gadget "github.com/gadget-bot/gadget/core"

	"github.com/rs/zerolog/log"
)

func main() {
	myBot, err := gadget.Setup()
	if err != nil {
		log.Fatal().Err(err).Msg("Setup failed")
	}

	// Add your custom plugins here

	// This launches your bot

	myBot.Run()
}
