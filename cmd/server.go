package cmd

import (
	gadget "github.com/gadget-bot/gadget/core"
	"github.com/spf13/cobra"
)

func newServerCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "server",
		Aliases: []string{"serve"},
		Short:   "Run the bot",
		Long:    `Run the bot`,
		RunE:    server,
	}
}

func server(cmd *cobra.Command, args []string) error {
	myBot, err := gadget.Setup()
	if err != nil {
		return err
	}

	// Plugin handlers go here
	return myBot.Run()
}
