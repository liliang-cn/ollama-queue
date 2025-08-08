package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/liliang-cn/ollama-queue/internal/models"
	"github.com/liliang-cn/ollama-queue/pkg/config"
	"github.com/liliang-cn/ollama-queue/pkg/queue"
)

var (
	listStatus   []string
	listType     []string
	listPriority []string
	listLimit    int
	listOffset   int
	listOutput   string
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks in the queue",
	Long: `List tasks in the queue with optional filtering.

Examples:
  # List all tasks
  ollama-queue list

  # List only running tasks
  ollama-queue list --status running

  # List failed and cancelled tasks
  ollama-queue list --status failed,cancelled

  # List high priority tasks
  ollama-queue list --priority high

  # List first 10 tasks
  ollama-queue list --limit 10`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringSliceVar(&listStatus, "status", nil, "Filter by status (pending, running, completed, failed, cancelled)")
	listCmd.Flags().StringSliceVar(&listType, "type", nil, "Filter by task type (chat, generate, embed)")
	listCmd.Flags().StringSliceVar(&listPriority, "priority", nil, "Filter by priority (low, normal, high, critical)")
	listCmd.Flags().IntVar(&listLimit, "limit", 0, "Maximum number of tasks to return")
	listCmd.Flags().IntVar(&listOffset, "offset", 0, "Number of tasks to skip")
	listCmd.Flags().StringVar(&listOutput, "output", "table", "Output format (table, json)")
}

func runList(cmd *cobra.Command, args []string) error {
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

	// Build filter
	filter := models.TaskFilter{
		Limit:  listLimit,
		Offset: listOffset,
	}

	// Parse status filter
	if len(listStatus) > 0 {
		for _, status := range listStatus {
			switch strings.ToLower(status) {
			case "pending":
				filter.Status = append(filter.Status, models.StatusPending)
			case "running":
				filter.Status = append(filter.Status, models.StatusRunning)
			case "completed":
				filter.Status = append(filter.Status, models.StatusCompleted)
			case "failed":
				filter.Status = append(filter.Status, models.StatusFailed)
			case "cancelled":
				filter.Status = append(filter.Status, models.StatusCancelled)
			default:
				return fmt.Errorf("invalid status: %s", status)
			}
		}
	}

	// Parse type filter
	if len(listType) > 0 {
		for _, taskType := range listType {
			switch strings.ToLower(taskType) {
			case "chat":
				filter.Type = append(filter.Type, models.TaskTypeChat)
			case "generate":
				filter.Type = append(filter.Type, models.TaskTypeGenerate)
			case "embed":
				filter.Type = append(filter.Type, models.TaskTypeEmbed)
			default:
				return fmt.Errorf("invalid task type: %s", taskType)
			}
		}
	}

	// Parse priority filter
	if len(listPriority) > 0 {
		for _, priority := range listPriority {
			p, err := parsePriority(priority)
			if err != nil {
				return fmt.Errorf("invalid priority: %w", err)
			}
			filter.Priority = append(filter.Priority, p)
		}
	}

	// List tasks
	tasks, err := qm.ListTasks(filter)
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	// Output results
	if listOutput == "json" {
		output, err := json.MarshalIndent(tasks, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal tasks: %w", err)
		}
		fmt.Println(string(output))
	} else {
		printTasksTable(tasks)
	}

	return nil
}

func printTasksTable(tasks []*models.Task) {
	if len(tasks) == 0 {
		fmt.Println("No tasks found")
		return
	}

	// Print header
	fmt.Printf("%-36s %-10s %-8s %-10s %-15s %-20s %-20s\n", 
		"ID", "TYPE", "PRIORITY", "STATUS", "MODEL", "CREATED", "COMPLETED")
	fmt.Println(strings.Repeat("-", 120))

	// Print tasks
	for _, task := range tasks {
		id := task.ID
		if len(id) > 8 {
			id = id[:8] + "..."
		}

		priority := fmt.Sprintf("%d", task.Priority)
		switch task.Priority {
		case models.PriorityLow:
			priority = "low"
		case models.PriorityNormal:
			priority = "normal"
		case models.PriorityHigh:
			priority = "high"
		case models.PriorityCritical:
			priority = "critical"
		}

		created := task.CreatedAt.Format("2006-01-02 15:04:05")
		completed := "-"
		if task.CompletedAt != nil {
			completed = task.CompletedAt.Format("2006-01-02 15:04:05")
		}

		model := task.Model
		if len(model) > 12 {
			model = model[:12] + "..."
		}

		fmt.Printf("%-36s %-10s %-8s %-10s %-15s %-20s %-20s\n",
			id, task.Type, priority, task.Status, model, created, completed)
	}

	fmt.Printf("\nTotal: %d tasks\n", len(tasks))
}