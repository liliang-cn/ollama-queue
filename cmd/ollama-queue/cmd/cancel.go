package cmd

import (
	"fmt"

	"github.com/liliang-cn/ollama-queue/pkg/client"
	"github.com/liliang-cn/ollama-queue/pkg/config"
	"github.com/spf13/cobra"
)

// cancelCmd represents the cancel command
var cancelCmd = &cobra.Command{
	Use:   "cancel <task-id>",
	Short: "Cancel a task",
	Long: `Cancel a pending or running task by its ID.

Examples:
  # Cancel a specific task
  ollama-queue cancel abc123def456

  # Cancel multiple tasks
  ollama-queue cancel abc123def456 def456ghi789`,
	Args: cobra.MinimumNArgs(1),
	RunE: runCancel,
}

func init() {
	rootCmd.AddCommand(cancelCmd)
}

func runCancel(cmd *cobra.Command, args []string) error {
	// Create a new client
	configLoader := config.NewConfigLoader()
	cfg, err := configLoader.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	cli := client.New(cfg.ListenAddr)

	// Cancel each task
	for _, taskID := range args {
		err := cli.CancelTask(taskID)
		if err != nil {
			fmt.Printf("Failed to cancel task %s: %v\n", taskID, err)
			continue
		}
		
		fmt.Printf("Task %s cancelled successfully\n", taskID)
	}

	return nil
}