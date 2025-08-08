package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/liliang-cn/ollama-queue/pkg/config"
	"github.com/liliang-cn/ollama-queue/pkg/queue"
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
	// Load configuration
	configLoader := config.NewConfigLoader()
	cfg, err := configLoader.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Override with CLI flags
	if ollamaHost != "" {
		cfg.OllamaHost = ollamaHost
	}
	if dataDir != "" {
		cfg.StoragePath = dataDir
	}

	// Create queue manager
	qm, err := queue.NewQueueManager(cfg)
	if err != nil {
		return fmt.Errorf("failed to create queue manager: %w", err)
	}
	defer qm.Close()

	// Cancel each task
	for _, taskID := range args {
		err := qm.CancelTask(taskID)
		if err != nil {
			fmt.Printf("Failed to cancel task %s: %v\n", taskID, err)
			continue
		}
		
		fmt.Printf("Task %s cancelled successfully\n", taskID)
	}

	return nil
}