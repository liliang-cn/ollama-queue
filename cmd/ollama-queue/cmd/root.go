package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile    string
	configDir  string
	dataDir    string
	ollamaHost string
	verbose    bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ollama-queue",
	Short: "Ollama Queue Client - Command-line client for Ollama Queue Server",
	Long: `Ollama Queue Client is a command-line interface for interacting with Ollama Queue Server.
It provides commands for task submission, monitoring, and management.

Before using the client, make sure the Ollama Queue Server is running:
  ollama-queue-server

Client commands:
- Task management: submit, cancel, status, list, priority
- Cron scheduling: cron add, cron list, cron enable/disable
- Queue monitoring: status, list with filters

All commands connect to the server via HTTP API.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute(ctx context.Context) error {
	rootCmd.SetContext(ctx)
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.ollama-queue/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&configDir, "config-dir", "", "config directory (default is $HOME/.ollama-queue)")
	rootCmd.PersistentFlags().StringVar(&dataDir, "data-dir", "", "data directory (default is ./data)")
	rootCmd.PersistentFlags().StringVar(&ollamaHost, "ollama-host", "http://localhost:11434", "Ollama host URL")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Bind flags to viper
	viper.BindPFlag("ollama.host", rootCmd.PersistentFlags().Lookup("ollama-host"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

// initConfig reads in config file and ENV variables.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".ollama-queue" (without extension).
		configPath := home + "/.ollama-queue"
		if configDir != "" {
			configPath = configDir
		}

		viper.AddConfigPath(configPath)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	// Environment variable settings
	viper.AutomaticEnv() // read in environment variables that match
	viper.SetEnvPrefix("OLLAMA_QUEUE")

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}
}