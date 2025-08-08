package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/liliang-cn/ollama-queue/internal/models"
	"github.com/liliang-cn/ollama-queue/pkg/client"
	"github.com/liliang-cn/ollama-queue/pkg/config"
	"github.com/spf13/cobra"
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
	// Create a new client
	configLoader := config.NewConfigLoader()
	cfg, err := configLoader.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	cli := client.New(cfg.ListenAddr)

	// Build filter
	filter := models.TaskFilter{
		Limit:  listLimit,
		Offset: listOffset,
	}

	// Parse status filter
	if len(listStatus) > 0 {
		for _, status := range listStatus {
			filter.Status = append(filter.Status, models.TaskStatus(status))
		}
	}

	// Parse type filter
	if len(listType) > 0 {
		for _, taskType := range listType {
			filter.Type = append(filter.Type, models.TaskType(taskType))
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
	tasks, err := cli.ListTasks(filter)
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	// Filter tasks
	filteredTasks := filterTasks(tasks, listStatus, listType, listPriority)

	// Output results
	if listOutput == "json" {
		output, err := json.MarshalIndent(filteredTasks, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal tasks: %w", err)
		}
		fmt.Println(string(output))
	} else {
		printTasksTable(filteredTasks)
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
	fmt.Println(strings.Repeat("-", 136))

	// Print tasks
	for _, task := range tasks {
		id := task.ID

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
		default:
			// 处理未知优先级值，可能是旧数据或直接数字输入
			if task.Priority == 0 {
				priority = "low"
			} else if task.Priority == 1 {
				priority = "normal"
			} else if task.Priority == 2 {
				priority = "high"
			} else if task.Priority == 3 {
				priority = "critical"
			}
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

func filterTasks(tasks []*models.Task, statuses []string, types []string, priorities []string) []*models.Task {
	var filtered []*models.Task

	for _, task := range tasks {
		// Status filter
		if len(statuses) > 0 {
			found := false
			for _, status := range statuses {
				if strings.ToLower(string(task.Status)) == strings.ToLower(status) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Type filter
		if len(types) > 0 {
			found := false
			for _, taskType := range types {
				if strings.ToLower(string(task.Type)) == strings.ToLower(taskType) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Priority filter
		if len(priorities) > 0 {
			found := false
			for _, priority := range priorities {
				p, err := parsePriority(priority)
				if err == nil && task.Priority == p {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		filtered = append(filtered, task)
	}

	return filtered
}