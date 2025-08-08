package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/liliang-cn/ollama-queue/internal/models"
	"github.com/liliang-cn/ollama-queue/pkg/config"
	"github.com/liliang-cn/ollama-queue/pkg/queue"
)

// priorityCmd represents the priority command
var priorityCmd = &cobra.Command{
	Use:   "priority <task-id> <priority>",
	Short: "Update task priority",
	Long: `Update the priority of a pending task.

Priority values:
  - low (1)
  - normal (5) 
  - high (10)
  - critical (15)
  - Or any integer value

Examples:
  # Set task priority to high
  ollama-queue priority abc123def456 high

  # Set task priority to a custom value
  ollama-queue priority abc123def456 20`,
	Args: cobra.ExactArgs(2),
	RunE: runPriority,
}

func init() {
	rootCmd.AddCommand(priorityCmd)
}

func runPriority(cmd *cobra.Command, args []string) error {
	taskID := args[0]
	priorityStr := args[1]

	// Parse priority
	priority, err := parsePriority(priorityStr)
	if err != nil {
		return fmt.Errorf("invalid priority: %w", err)
	}

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

	// Update task priority
	err = qm.UpdateTaskPriority(taskID, priority)
	if err != nil {
		return fmt.Errorf("failed to update task priority: %w", err)
	}

	priorityName := priorityStr
	switch priority {
	case models.PriorityLow:
		priorityName = "low"
	case models.PriorityNormal:
		priorityName = "normal"
	case models.PriorityHigh:
		priorityName = "high"
	case models.PriorityCritical:
		priorityName = "critical"
	}

	fmt.Printf("Task %s priority updated to %s (%d)\n", taskID, priorityName, priority)
	return nil
}