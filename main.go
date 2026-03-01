package main

import (
	gadget "github.com/gadget-bot/gadget/core"
	"github.com/gadget-bot/gadget/conf"

	"github.com/rs/zerolog/log"
)

func main() {
	log.Info().
		Str("executable", conf.Executable).
		Str("version", conf.GitVersion).
		Msg("Starting")

	myBot, err := gadget.Setup()
	if err != nil {
		log.Fatal().Err(err).Msg("Setup failed")
	}

	// Add your custom plugins here

	// This launches your bot

	myBot.Run()
}
