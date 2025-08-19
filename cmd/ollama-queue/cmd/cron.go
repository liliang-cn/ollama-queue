package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/liliang-cn/ollama-queue/internal/models"
	"github.com/liliang-cn/ollama-queue/pkg/client"
	"github.com/liliang-cn/ollama-queue/pkg/config"
)

var cronCmd = &cobra.Command{
	Use:   "cron",
	Short: "Manage cron tasks",
	Long:  `Manage recurring tasks using cron expressions`,
}

var cronAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new cron task",
	Long: `Add a new recurring task with cron expression.
	
Examples:
  # Run every hour
  ollama-queue cron add --name "hourly-chat" --cron "0 * * * *" --type chat --model qwen3 --prompt "What time is it?"
  
  # Run every weekday at 9 AM
  ollama-queue cron add --name "morning-summary" --cron "0 9 * * 1-5" --type chat --model qwen3 --prompt "Give me a morning briefing"
  
  # Run every 15 minutes
  ollama-queue cron add --name "quick-check" --cron "*/15 * * * *" --type generate --model qwen3 --prompt "Status check"
`,
	RunE: runCronAdd,
}

var cronListCmd = &cobra.Command{
	Use:   "list",
	Short: "List cron tasks",
	Long:  `List all cron tasks with their status and next run times`,
	RunE:  runCronList,
}

var cronRemoveCmd = &cobra.Command{
	Use:   "remove [cron-id]",
	Short: "Remove a cron task",
	Long:  `Remove a cron task by ID`,
	Args:  cobra.ExactArgs(1),
	RunE:  runCronRemove,
}

var cronEnableCmd = &cobra.Command{
	Use:   "enable [cron-id]",
	Short: "Enable a cron task",
	Long:  `Enable a disabled cron task`,
	Args:  cobra.ExactArgs(1),
	RunE:  runCronEnable,
}

var cronDisableCmd = &cobra.Command{
	Use:   "disable [cron-id]",
	Short: "Disable a cron task",
	Long:  `Disable an active cron task`,
	Args:  cobra.ExactArgs(1),
	RunE:  runCronDisable,
}

var cronShowCmd = &cobra.Command{
	Use:   "show [cron-id]",
	Short: "Show details of a cron task",
	Long:  `Show detailed information about a specific cron task`,
	Args:  cobra.ExactArgs(1),
	RunE:  runCronShow,
}

var cronUpdateCmd = &cobra.Command{
	Use:   "update [cron-id]",
	Short: "Update a cron task",
	Long:  `Update the configuration of an existing cron task`,
	Args:  cobra.ExactArgs(1),
	RunE:  runCronUpdate,
}

var cronTestCmd = &cobra.Command{
	Use:   "test [cron-expression]",
	Short: "Test a cron expression",
	Long:  `Test and validate a cron expression, showing next execution times`,
	Args:  cobra.ExactArgs(1),
	RunE:  runCronTest,
}

func init() {
	// Add cron flags for add command
	cronAddCmd.Flags().String("name", "", "Name for the cron task (required)")
	cronAddCmd.Flags().String("cron", "", "Cron expression (required)")
	cronAddCmd.Flags().String("type", "chat", "Task type (chat, generate, embed)")
	cronAddCmd.Flags().String("model", "qwen3", "Model to use")
	cronAddCmd.Flags().String("priority", "normal", "Task priority (low, normal, high, critical)")
	cronAddCmd.Flags().String("prompt", "", "Prompt for generate/chat tasks")
	cronAddCmd.Flags().String("system", "", "System message for chat tasks")
	cronAddCmd.Flags().String("input", "", "Input for embed tasks")
	cronAddCmd.Flags().Bool("stream", false, "Enable streaming for chat/generate tasks")
	cronAddCmd.Flags().Bool("enabled", true, "Enable the cron task immediately")
	cronAddCmd.Flags().String("metadata", "", "JSON metadata for the cron task")

	cronAddCmd.MarkFlagRequired("name")
	cronAddCmd.MarkFlagRequired("cron")

	// Add cron flags for update command
	cronUpdateCmd.Flags().String("name", "", "New name for the cron task")
	cronUpdateCmd.Flags().String("cron", "", "New cron expression")
	cronUpdateCmd.Flags().String("type", "", "New task type (chat, generate, embed)")
	cronUpdateCmd.Flags().String("model", "", "New model to use")
	cronUpdateCmd.Flags().String("priority", "", "New task priority (low, normal, high, critical)")
	cronUpdateCmd.Flags().String("prompt", "", "New prompt for generate/chat tasks")
	cronUpdateCmd.Flags().String("system", "", "New system message for chat tasks")
	cronUpdateCmd.Flags().String("input", "", "New input for embed tasks")
	cronUpdateCmd.Flags().Bool("stream", false, "Enable/disable streaming")
	cronUpdateCmd.Flags().String("metadata", "", "New JSON metadata")

	// Add cron flags for list command
	cronListCmd.Flags().Bool("enabled", false, "Show only enabled cron tasks")
	cronListCmd.Flags().Bool("disabled", false, "Show only disabled cron tasks")
	cronListCmd.Flags().Int("limit", 0, "Limit number of results")
	cronListCmd.Flags().Int("offset", 0, "Offset for pagination")

	// Add cron flags for test command
	cronTestCmd.Flags().Int("count", 5, "Number of next execution times to show")

	// Add subcommands
	cronCmd.AddCommand(cronAddCmd)
	cronCmd.AddCommand(cronListCmd)
	cronCmd.AddCommand(cronRemoveCmd)
	cronCmd.AddCommand(cronEnableCmd)
	cronCmd.AddCommand(cronDisableCmd)
	cronCmd.AddCommand(cronShowCmd)
	cronCmd.AddCommand(cronUpdateCmd)
	cronCmd.AddCommand(cronTestCmd)

	rootCmd.AddCommand(cronCmd)
}

// Helper function to create client
func createClient() (*client.Client, error) {
	configLoader := config.NewConfigLoader()
	cfg, err := configLoader.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return client.New(cfg.ListenAddr), nil
}

func runCronAdd(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	cronExpr, _ := cmd.Flags().GetString("cron")
	taskType, _ := cmd.Flags().GetString("type")
	model, _ := cmd.Flags().GetString("model")
	priorityStr, _ := cmd.Flags().GetString("priority")
	prompt, _ := cmd.Flags().GetString("prompt")
	system, _ := cmd.Flags().GetString("system")
	input, _ := cmd.Flags().GetString("input")
	stream, _ := cmd.Flags().GetBool("stream")
	enabled, _ := cmd.Flags().GetBool("enabled")
	metadataStr, _ := cmd.Flags().GetString("metadata")

	// Validate cron expression
	cronParsed, err := models.ParseCronExpression(cronExpr)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	// Parse priority
	priority, err := parsePriority(priorityStr)
	if err != nil {
		return err
	}

	// Parse metadata if provided
	var metadata map[string]interface{}
	if metadataStr != "" {
		if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
			return fmt.Errorf("invalid metadata JSON: %w", err)
		}
	}

	// Build task template based on type
	var payload interface{}
	options := make(map[string]interface{})

	switch models.TaskType(taskType) {
	case models.TaskTypeChat:
		if prompt == "" {
			return fmt.Errorf("prompt is required for chat tasks")
		}
		messages := []models.ChatMessage{
			{Role: "user", Content: prompt},
		}
		if system != "" {
			messages = append([]models.ChatMessage{{Role: "system", Content: system}}, messages...)
		}
		payload = messages
		options["stream"] = stream
	case models.TaskTypeGenerate:
		if prompt == "" {
			return fmt.Errorf("prompt is required for generate tasks")
		}
		payload = map[string]interface{}{
			"prompt": prompt,
		}
		if system != "" {
			payload.(map[string]interface{})["system"] = system
		}
		options["stream"] = stream
	case models.TaskTypeEmbed:
		if input == "" {
			return fmt.Errorf("input is required for embed tasks")
		}
		payload = map[string]interface{}{
			"input": input,
		}
	default:
		return fmt.Errorf("unsupported task type: %s", taskType)
	}

	// Create cron task
	cronTask := &models.CronTask{
		Name:     name,
		CronExpr: cronExpr,
		Enabled:  enabled,
		TaskTemplate: models.TaskTemplate{
			Type:     models.TaskType(taskType),
			Model:    model,
			Priority: priority,
			Payload:  payload,
			Options:  options,
		},
		Metadata: metadata,
	}

	// Create client and add cron task
	c, err := createClient()
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	cronID, err := c.AddCronTask(cronTask)
	if err != nil {
		return fmt.Errorf("failed to add cron task: %w", err)
	}

	// Show next execution times
	nextRun := cronParsed.NextTime(time.Now())
	fmt.Printf("Cron task added successfully!\n")
	fmt.Printf("ID: %s\n", cronID)
	fmt.Printf("Name: %s\n", name)
	fmt.Printf("Expression: %s\n", cronExpr)
	fmt.Printf("Enabled: %t\n", enabled)
	if enabled {
		fmt.Printf("Next run: %s\n", nextRun.Format(time.RFC3339))
	}

	return nil
}

func runCronList(cmd *cobra.Command, args []string) error {
	enabled, _ := cmd.Flags().GetBool("enabled")
	disabled, _ := cmd.Flags().GetBool("disabled")
	limit, _ := cmd.Flags().GetInt("limit")
	offset, _ := cmd.Flags().GetInt("offset")

	// Create filter
	filter := models.CronFilter{
		Limit:  limit,
		Offset: offset,
	}

	if enabled && !disabled {
		enabledValue := true
		filter.Enabled = &enabledValue
	} else if disabled && !enabled {
		enabledValue := false
		filter.Enabled = &enabledValue
	}

	// Create client and list cron tasks
	c, err := createClient()
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	cronTasks, err := c.ListCronTasks(filter)
	if err != nil {
		return fmt.Errorf("failed to list cron tasks: %w", err)
	}

	if len(cronTasks) == 0 {
		fmt.Println("No cron tasks found")
		return nil
	}

	// Display results in table format
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID\tNAME\tEXPRESSION\tENABLED\tRUN COUNT\tNEXT RUN\tLAST RUN\n")

	for _, cronTask := range cronTasks {
		nextRunStr := "N/A"
		if cronTask.NextRun != nil {
			nextRunStr = cronTask.NextRun.Format("2006-01-02 15:04")
		}

		lastRunStr := "Never"
		if cronTask.LastRun != nil {
			lastRunStr = cronTask.LastRun.Format("2006-01-02 15:04")
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%t\t%d\t%s\t%s\n",
			safeTaskIDShort(cronTask.ID),
			cronTask.Name,
			cronTask.CronExpr,
			cronTask.Enabled,
			cronTask.RunCount,
			nextRunStr,
			lastRunStr)
	}

	return w.Flush()
}

func runCronRemove(cmd *cobra.Command, args []string) error {
	cronID := args[0]

	// Create client and remove cron task
	c, err := createClient()
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	err = c.RemoveCronTask(cronID)
	if err != nil {
		return fmt.Errorf("failed to remove cron task: %w", err)
	}

	fmt.Printf("Cron task %s removed successfully\n", cronID)
	return nil
}

func runCronEnable(cmd *cobra.Command, args []string) error {
	cronID := args[0]

	// Create client and enable cron task
	c, err := createClient()
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	err = c.EnableCronTask(cronID)
	if err != nil {
		return fmt.Errorf("failed to enable cron task: %w", err)
	}

	fmt.Printf("Cron task %s enabled successfully\n", cronID)
	return nil
}

func runCronDisable(cmd *cobra.Command, args []string) error {
	cronID := args[0]

	// Create client and disable cron task
	c, err := createClient()
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	err = c.DisableCronTask(cronID)
	if err != nil {
		return fmt.Errorf("failed to disable cron task: %w", err)
	}

	fmt.Printf("Cron task %s disabled successfully\n", cronID)
	return nil
}

func runCronShow(cmd *cobra.Command, args []string) error {
	cronID := args[0]

	// Create client and get cron task
	c, err := createClient()
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	cronTask, err := c.GetCronTask(cronID)
	if err != nil {
		return fmt.Errorf("failed to get cron task: %w", err)
	}

	// Display detailed information
	fmt.Printf("Cron Task Details:\n")
	fmt.Printf("ID: %s\n", cronTask.ID)
	fmt.Printf("Name: %s\n", cronTask.Name)
	fmt.Printf("Expression: %s\n", cronTask.CronExpr)
	fmt.Printf("Enabled: %t\n", cronTask.Enabled)
	fmt.Printf("Run Count: %d\n", cronTask.RunCount)
	fmt.Printf("Created: %s\n", cronTask.CreatedAt.Format(time.RFC3339))
	fmt.Printf("Updated: %s\n", cronTask.UpdatedAt.Format(time.RFC3339))

	if cronTask.NextRun != nil {
		fmt.Printf("Next Run: %s\n", cronTask.NextRun.Format(time.RFC3339))
	} else {
		fmt.Printf("Next Run: N/A (disabled)\n")
	}

	if cronTask.LastRun != nil {
		fmt.Printf("Last Run: %s\n", cronTask.LastRun.Format(time.RFC3339))
	} else {
		fmt.Printf("Last Run: Never\n")
	}

	fmt.Printf("\nTask Template:\n")
	fmt.Printf("  Type: %s\n", cronTask.TaskTemplate.Type)
	fmt.Printf("  Model: %s\n", cronTask.TaskTemplate.Model)
	fmt.Printf("  Priority: %s (%d)\n", priorityToString(cronTask.TaskTemplate.Priority), cronTask.TaskTemplate.Priority)

	// Display payload based on task type
	switch cronTask.TaskTemplate.Type {
	case models.TaskTypeChat:
		if messages, ok := cronTask.TaskTemplate.Payload.([]interface{}); ok {
			fmt.Printf("  Messages:\n")
			for i, msg := range messages {
				if msgMap, ok := msg.(map[string]interface{}); ok {
					role := msgMap["role"]
					content := msgMap["content"]
					fmt.Printf("    %d. %s: %s\n", i+1, role, content)
				}
			}
		}
	case models.TaskTypeGenerate:
		if payloadMap, ok := cronTask.TaskTemplate.Payload.(map[string]interface{}); ok {
			if prompt, ok := payloadMap["prompt"]; ok {
				fmt.Printf("  Prompt: %s\n", prompt)
			}
			if system, ok := payloadMap["system"]; ok {
				fmt.Printf("  System: %s\n", system)
			}
		}
	case models.TaskTypeEmbed:
		if payloadMap, ok := cronTask.TaskTemplate.Payload.(map[string]interface{}); ok {
			if input, ok := payloadMap["input"]; ok {
				fmt.Printf("  Input: %s\n", input)
			}
		}
	}

	// Display options
	if len(cronTask.TaskTemplate.Options) > 0 {
		fmt.Printf("  Options:\n")
		for key, value := range cronTask.TaskTemplate.Options {
			fmt.Printf("    %s: %v\n", key, value)
		}
	}

	// Display metadata
	if len(cronTask.Metadata) > 0 {
		fmt.Printf("\nMetadata:\n")
		for key, value := range cronTask.Metadata {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}

	return nil
}

func runCronUpdate(cmd *cobra.Command, args []string) error {
	cronID := args[0]

	// Create client and get existing cron task
	c, err := createClient()
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	cronTask, err := c.GetCronTask(cronID)
	if err != nil {
		return fmt.Errorf("failed to get cron task: %w", err)
	}

	// Update fields from flags
	if cmd.Flags().Changed("name") {
		name, _ := cmd.Flags().GetString("name")
		cronTask.Name = name
	}

	if cmd.Flags().Changed("cron") {
		cronExpr, _ := cmd.Flags().GetString("cron")
		// Validate new expression
		if _, err := models.ParseCronExpression(cronExpr); err != nil {
			return fmt.Errorf("invalid cron expression: %w", err)
		}
		cronTask.CronExpr = cronExpr
	}

	if cmd.Flags().Changed("type") {
		taskType, _ := cmd.Flags().GetString("type")
		cronTask.TaskTemplate.Type = models.TaskType(taskType)
	}

	if cmd.Flags().Changed("model") {
		model, _ := cmd.Flags().GetString("model")
		cronTask.TaskTemplate.Model = model
	}

	if cmd.Flags().Changed("priority") {
		priorityStr, _ := cmd.Flags().GetString("priority")
		priority, err := parsePriority(priorityStr)
		if err != nil {
			return err
		}
		cronTask.TaskTemplate.Priority = priority
	}

	// Update payload fields based on task type
	if cmd.Flags().Changed("prompt") || cmd.Flags().Changed("system") || cmd.Flags().Changed("input") {
		prompt, _ := cmd.Flags().GetString("prompt")
		system, _ := cmd.Flags().GetString("system")
		input, _ := cmd.Flags().GetString("input")

		switch cronTask.TaskTemplate.Type {
		case models.TaskTypeChat:
			messages := []models.ChatMessage{
				{Role: "user", Content: prompt},
			}
			if system != "" {
				messages = append([]models.ChatMessage{{Role: "system", Content: system}}, messages...)
			}
			cronTask.TaskTemplate.Payload = messages
		case models.TaskTypeGenerate:
			payload := map[string]interface{}{
				"prompt": prompt,
			}
			if system != "" {
				payload["system"] = system
			}
			cronTask.TaskTemplate.Payload = payload
		case models.TaskTypeEmbed:
			cronTask.TaskTemplate.Payload = map[string]interface{}{
				"input": input,
			}
		}
	}

	if cmd.Flags().Changed("stream") {
		stream, _ := cmd.Flags().GetBool("stream")
		if cronTask.TaskTemplate.Options == nil {
			cronTask.TaskTemplate.Options = make(map[string]interface{})
		}
		cronTask.TaskTemplate.Options["stream"] = stream
	}

	if cmd.Flags().Changed("metadata") {
		metadataStr, _ := cmd.Flags().GetString("metadata")
		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
			return fmt.Errorf("invalid metadata JSON: %w", err)
		}
		cronTask.Metadata = metadata
	}

	// Update cron task
	err = c.UpdateCronTask(cronTask)
	if err != nil {
		return fmt.Errorf("failed to update cron task: %w", err)
	}

	fmt.Printf("Cron task %s updated successfully\n", cronID)
	return nil
}

func runCronTest(cmd *cobra.Command, args []string) error {
	cronExpr := args[0]
	count, _ := cmd.Flags().GetInt("count")

	// Parse and validate cron expression
	cronParsed, err := models.ParseCronExpression(cronExpr)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	fmt.Printf("Cron expression: %s\n", cronExpr)
	fmt.Printf("Next %d execution times:\n\n", count)

	current := time.Now()
	for i := 0; i < count; i++ {
		next := cronParsed.NextTime(current)
		fmt.Printf("%d. %s (%s)\n", i+1, next.Format(time.RFC3339), humanizeDuration(next.Sub(time.Now())))
		current = next
	}

	return nil
}

func priorityToString(priority models.Priority) string {
	switch priority {
	case models.PriorityLow:
		return "low"
	case models.PriorityNormal:
		return "normal"
	case models.PriorityHigh:
		return "high"
	case models.PriorityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

func humanizeDuration(duration time.Duration) string {
	if duration < time.Minute {
		return fmt.Sprintf("%.0fs", duration.Seconds())
	} else if duration < time.Hour {
		return fmt.Sprintf("%.0fm", duration.Minutes())
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%.1fh", duration.Hours())
	} else {
		return fmt.Sprintf("%.1fd", duration.Hours()/24)
	}
}