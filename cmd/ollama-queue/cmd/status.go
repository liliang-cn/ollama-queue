package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/liliang-cn/ollama-queue/pkg/config"
	"github.com/liliang-cn/ollama-queue/pkg/queue"
)

var (
	statusOutput string
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status [task-id]",
	Short: "Get the status of a task or queue statistics",
	Long: `Get the status of a specific task by ID, or show overall queue statistics if no ID is provided.

Examples:
  # Show queue statistics
  ollama-queue status

  # Show specific task status
  ollama-queue status abc123def456`,
	Args: cobra.MaximumNArgs(1),
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
	
	statusCmd.Flags().StringVar(&statusOutput, "output", "table", "Output format (table, json)")
}

func runStatus(cmd *cobra.Command, args []string) error {
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

	if len(args) == 0 {
		// Show queue statistics
		return showQueueStats(qm)
	} else {
		// Show specific task status
		return showTaskStatus(qm, args[0])
	}
}

func showQueueStats(qm *queue.QueueManager) error {
	stats, err := qm.GetQueueStats()
	if err != nil {
		return fmt.Errorf("failed to get queue stats: %w", err)
	}

	if statusOutput == "json" {
		output, err := json.MarshalIndent(stats, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal stats: %w", err)
		}
		fmt.Println(string(output))
	} else {
		fmt.Println("Queue Statistics:")
		fmt.Println("================")
		fmt.Printf("Pending Tasks:   %d\n", stats.PendingTasks)
		fmt.Printf("Running Tasks:   %d\n", stats.RunningTasks)
		fmt.Printf("Completed Tasks: %d\n", stats.CompletedTasks)
		fmt.Printf("Failed Tasks:    %d\n", stats.FailedTasks)
		fmt.Printf("Cancelled Tasks: %d\n", stats.CancelledTasks)
		fmt.Printf("Total Tasks:     %d\n", stats.TotalTasks)
		
		fmt.Println("\nWorker Statistics:")
		fmt.Println("==================")
		fmt.Printf("Active Workers:  %d\n", stats.WorkersActive)
		fmt.Printf("Idle Workers:    %d\n", stats.WorkersIdle)
		
		fmt.Println("\nPriority Queues:")
		fmt.Println("================")
		for priority, count := range stats.QueuesByPriority {
			priorityName := fmt.Sprintf("%d", priority)
			switch priority {
			case 1:
				priorityName = "Low"
			case 5:
				priorityName = "Normal"
			case 10:
				priorityName = "High"
			case 15:
				priorityName = "Critical"
			}
			fmt.Printf("%-10s: %d tasks\n", priorityName, count)
		}
	}

	return nil
}

func showTaskStatus(qm *queue.QueueManager, taskID string) error {
	task, err := qm.GetTask(taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	if statusOutput == "json" {
		output, err := json.MarshalIndent(task, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal task: %w", err)
		}
		fmt.Println(string(output))
	} else {
		fmt.Printf("Task Status: %s\n", taskID)
		fmt.Println("===================")
		fmt.Printf("Type:        %s\n", task.Type)
		fmt.Printf("Model:       %s\n", task.Model)
		fmt.Printf("Status:      %s\n", task.Status)
		fmt.Printf("Priority:    %d\n", task.Priority)
		fmt.Printf("Created:     %s\n", task.CreatedAt.Format("2006-01-02 15:04:05"))
		
		if task.StartedAt != nil {
			fmt.Printf("Started:     %s\n", task.StartedAt.Format("2006-01-02 15:04:05"))
		}
		
		if task.CompletedAt != nil {
			fmt.Printf("Completed:   %s\n", task.CompletedAt.Format("2006-01-02 15:04:05"))
		}
		
		if task.RetryCount > 0 {
			fmt.Printf("Retry Count: %d/%d\n", task.RetryCount, task.MaxRetries)
		}
		
		if task.Error != "" {
			fmt.Printf("Error:       %s\n", task.Error)
		}
		
		// Show payload summary
		fmt.Println("\nPayload:")
		fmt.Println("========")
		switch task.Type {
		case "chat":
			if payload, ok := task.Payload.(map[string]interface{}); ok {
				if messages, ok := payload["messages"].([]interface{}); ok {
					fmt.Printf("Messages:    %d\n", len(messages))
				}
				if system, ok := payload["system"].(string); ok && system != "" {
					fmt.Printf("System:      %s\n", system)
				}
			}
		case "generate":
			if payload, ok := task.Payload.(map[string]interface{}); ok {
				if prompt, ok := payload["prompt"].(string); ok {
					promptPreview := prompt
					if len(promptPreview) > 50 {
						promptPreview = promptPreview[:50] + "..."
					}
					fmt.Printf("Prompt:      %s\n", promptPreview)
				}
			}
		case "embed":
			if payload, ok := task.Payload.(map[string]interface{}); ok {
				if input, ok := payload["input"].(string); ok {
					inputPreview := input
					if len(inputPreview) > 50 {
						inputPreview = inputPreview[:50] + "..."
					}
					fmt.Printf("Input:       %s\n", inputPreview)
				}
			}
		}
	}

	return nil
}