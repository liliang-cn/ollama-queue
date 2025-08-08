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
	Short: "A task queue management system for Ollama models",
	Long: `Ollama Queue is a Go-based task queue system that provides efficient task management,
priority scheduling, and task cancellation for Ollama models. It can be used as both a
library and a command-line tool for managing AI model tasks.`,
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