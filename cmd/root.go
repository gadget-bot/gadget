package cmd

import (
	"os"

	"github.com/gadget-bot/gadget/conf"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

func newRootCmd() *cobra.Command {
	return &cobra.Command{
		Version: conf.GitVersion,
		Use:     conf.Executable,
		Short:   "Gadget is a bot for Slack's Events API",
		Long: `Gadget is a bot for Slack's Events API. The concepts in Gadget are
heavily inspired by Lita in that it contains support for Regular
Expression-based _routing_ of requests to plugins, it supports permissions
based on groups managed via the bot, and it uses a DB for persisting these
details.`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	cobra.OnInitialize(initConfig)
	rootCmd := newRootCmd()
	setupFlags(rootCmd)
	addSubcommands(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		println(err)
		os.Exit(1)
	}
}

func setupFlags(c *cobra.Command) {
	c.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gadget.yaml)")
	c.MarkPersistentFlagFilename("config")
}

func addSubcommands(c *cobra.Command) {
	c.AddCommand(newVersionCmd())
	c.AddCommand(newServerCmd())
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := homedir.Dir()
		if err != nil {
			println(err)
			os.Exit(1)
		}

		viper.AddConfigPath(home)
		viper.SetConfigName(".gadget")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		println("Using config file:", viper.ConfigFileUsed())
	}
}
