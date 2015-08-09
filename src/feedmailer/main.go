package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

var CfgFile string

var RootCmd = &cobra.Command{
	Use:   "feedmailer",
	Short: "feedmailer is an RSS to Email aggregator",
	Long: `feedmailer aggregates a list of RSS feeds into
	a daily digest email with all the new stuff.`,
	Run: rootRun,
}

func rootRun(cmd *cobra.Command, args []string) {
	Fetcher()

	// Provides a way to cancel the feed fetching
	// when it is setup to run forever.
	//
	// sigChan := make(chan os.Signal, 1)
	// signal.Notify(sigChan, os.Interrupt)
	// <-sigChan
}

const defaultDataDir = "$HOME/.feedmailer"

func init() {
	cobra.OnInitialize(initConfig)

	viper.SetDefault("subject", "Feeds powered by feedmailer")
	viper.SetDefault("feeds", []string{"http://spf13.com/index.xml"})
	viper.SetDefault("data_dir", defaultDataDir)

	RootCmd.PersistentFlags().StringVar(&CfgFile, "config", "", "config file (default is "+defaultDataDir+"/config.yml)")
}

func initConfig() {
	if CfgFile != "" {
		viper.SetConfigFile(CfgFile)
	}
	viper.SetConfigName("config")             // name of config file (without extension)
	viper.AddConfigPath(defaultDataDir + "/") // path to look for the config file in
	viper.ReadInConfig()
}

func main() {
	addCommands()

	dataDir := os.ExpandEnv(viper.GetString("data_dir"))
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func addCommands() {
	RootCmd.AddCommand(fetchCmd)
}
