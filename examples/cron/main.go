package main

import (
	"fmt"
	"log"
	"time"

	"github.com/liliang-cn/ollama-queue/internal/models"
	"github.com/liliang-cn/ollama-queue/pkg/client"
	"github.com/liliang-cn/ollama-queue/pkg/config"
)

func main() {
	// Create client
	configLoader := config.NewConfigLoader()
	cfg, err := configLoader.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	
	cli := client.New(cfg.ListenAddr)
	
	// Example 1: Create a cron task that runs every hour
	fmt.Println("Creating hourly status check cron task...")
	
	hourlyCron := &models.CronTask{
		Name:     "Hourly Status Check",
		CronExpr: "0 * * * *", // Every hour at minute 0
		Enabled:  true,
		TaskTemplate: models.TaskTemplate{
			Type:     models.TaskTypeGenerate,
			Model:    "qwen3",
			Priority: models.PriorityNormal,
			Payload: map[string]interface{}{
				"prompt": "Generate a brief status report with current timestamp: " + time.Now().Format(time.RFC3339),
			},
			Options: map[string]interface{}{
				"stream": false,
			},
		},
		Metadata: map[string]interface{}{
			"purpose": "monitoring",
			"owner":   "system",
		},
	}
	
	cronID1, err := cli.AddCronTask(hourlyCron)
	if err != nil {
		log.Fatalf("Failed to add hourly cron task: %v", err)
	}
	fmt.Printf("Created hourly cron task: %s\n", cronID1)
	
	// Example 2: Create a daily morning briefing (weekdays only)
	fmt.Println("\nCreating daily morning briefing cron task...")
	
	dailyBriefing := &models.CronTask{
		Name:     "Daily Morning Briefing",
		CronExpr: "0 9 * * 1-5", // 9 AM Monday-Friday
		Enabled:  true,
		TaskTemplate: models.TaskTemplate{
			Type:     models.TaskTypeChat,
			Model:    "qwen3",
			Priority: models.PriorityHigh,
			Payload: []models.ChatMessage{
				{
					Role:    "system",
					Content: "You are a professional assistant that provides daily briefings.",
				},
				{
					Role:    "user",
					Content: "Provide a morning briefing with weather outlook, key reminders, and motivational message for the day.",
				},
			},
			Options: map[string]interface{}{
				"stream": false,
			},
		},
		Metadata: map[string]interface{}{
			"department": "operations",
			"priority":   "high",
			"recipients": []string{"team@company.com"},
		},
	}
	
	cronID2, err := cli.AddCronTask(dailyBriefing)
	if err != nil {
		log.Fatalf("Failed to add daily briefing cron task: %v", err)
	}
	fmt.Printf("Created daily briefing cron task: %s\n", cronID2)
	
	// Example 3: Create a weekly report (every Friday at 5 PM)
	fmt.Println("\nCreating weekly report cron task...")
	
	weeklyReport := &models.CronTask{
		Name:     "Weekly Report",
		CronExpr: "0 17 * * 5", // 5 PM every Friday
		Enabled:  true,
		TaskTemplate: models.TaskTemplate{
			Type:     models.TaskTypeChat,
			Model:    "qwen3",
			Priority: models.PriorityCritical,
			Payload: []models.ChatMessage{
				{
					Role:    "system",
					Content: "You are an expert analyst that creates comprehensive weekly reports.",
				},
				{
					Role:    "user",
					Content: "Generate a detailed weekly summary report including accomplishments, challenges, and next week's priorities.",
				},
			},
			Options: map[string]interface{}{
				"stream": false,
			},
		},
		Metadata: map[string]interface{}{
			"report_type": "weekly",
			"format":      "markdown",
			"distribution": "management",
		},
	}
	
	cronID3, err := cli.AddCronTask(weeklyReport)
	if err != nil {
		log.Fatalf("Failed to add weekly report cron task: %v", err)
	}
	fmt.Printf("Created weekly report cron task: %s\n", cronID3)
	
	// Example 4: Create a frequent monitoring task (every 15 minutes)
	fmt.Println("\nCreating frequent monitoring cron task...")
	
	frequentMonitor := &models.CronTask{
		Name:     "System Health Check",
		CronExpr: "*/15 * * * *", // Every 15 minutes
		Enabled:  false, // Start disabled for demonstration
		TaskTemplate: models.TaskTemplate{
			Type:     models.TaskTypeGenerate,
			Model:    "qwen3",
			Priority: models.PriorityLow,
			Payload: map[string]interface{}{
				"prompt": "Perform a quick system health check and report any anomalies.",
			},
			Options: map[string]interface{}{
				"stream": false,
			},
		},
		Metadata: map[string]interface{}{
			"monitoring": true,
			"frequency":  "high",
			"alerting":   true,
		},
	}
	
	cronID4, err := cli.AddCronTask(frequentMonitor)
	if err != nil {
		log.Fatalf("Failed to add monitoring cron task: %v", err)
	}
	fmt.Printf("Created monitoring cron task (disabled): %s\n", cronID4)
	
	// List all cron tasks
	fmt.Println("\nListing all cron tasks:")
	cronTasks, err := cli.ListCronTasks(models.CronFilter{})
	if err != nil {
		log.Fatalf("Failed to list cron tasks: %v", err)
	}
	
	for _, cronTask := range cronTasks {
		status := "disabled"
		if cronTask.Enabled {
			status = "enabled"
		}
		
		nextRun := "N/A"
		if cronTask.NextRun != nil {
			nextRun = cronTask.NextRun.Format("2006-01-02 15:04:05")
		}
		
		fmt.Printf("- %s (%s): %s [%s] - Next: %s - Runs: %d\n", 
			cronTask.Name, 
			cronTask.ID[:8], 
			cronTask.CronExpr, 
			status,
			nextRun,
			cronTask.RunCount)
	}
	
	// Enable the monitoring task
	fmt.Println("\nEnabling the monitoring task...")
	err = cli.EnableCronTask(cronID4)
	if err != nil {
		log.Fatalf("Failed to enable monitoring task: %v", err)
	}
	fmt.Printf("Enabled monitoring task: %s\n", cronID4)
	
	// Show detailed information for one task
	fmt.Println("\nShowing detailed info for daily briefing task:")
	detailTask, err := cli.GetCronTask(cronID2)
	if err != nil {
		log.Fatalf("Failed to get cron task details: %v", err)
	}
	
	fmt.Printf("Name: %s\n", detailTask.Name)
	fmt.Printf("Expression: %s\n", detailTask.CronExpr)
	fmt.Printf("Enabled: %t\n", detailTask.Enabled)
	fmt.Printf("Run Count: %d\n", detailTask.RunCount)
	fmt.Printf("Created: %s\n", detailTask.CreatedAt.Format(time.RFC3339))
	if detailTask.NextRun != nil {
		fmt.Printf("Next Run: %s\n", detailTask.NextRun.Format(time.RFC3339))
	}
	if detailTask.LastRun != nil {
		fmt.Printf("Last Run: %s\n", detailTask.LastRun.Format(time.RFC3339))
	}
	
	// Update a cron task (change expression)
	fmt.Println("\nUpdating weekly report to run at 4 PM instead of 5 PM...")
	detailTask.CronExpr = "0 16 * * 5" // 4 PM every Friday
	err = cli.UpdateCronTask(detailTask)
	if err != nil {
		log.Fatalf("Failed to update cron task: %v", err)
	}
	fmt.Printf("Updated cron task: %s\n", detailTask.ID)
	
	// Demonstrate cron expression parsing
	fmt.Println("\nDemonstrating cron expression parsing:")
	testExpressions := []string{
		"0 9 * * 1-5",    // 9 AM weekdays
		"*/15 * * * *",   // Every 15 minutes
		"0 0 1 * *",      // First day of every month
		"0 12 * * 0",     // Noon every Sunday
		"30 14 * * 6",    // 2:30 PM every Saturday
	}
	
	for _, expr := range testExpressions {
		cronParsed, err := models.ParseCronExpression(expr)
		if err != nil {
			fmt.Printf("Invalid expression '%s': %v\n", expr, err)
			continue
		}
		
		fmt.Printf("Expression: %-12s | Next 3 runs:\n", expr)
		current := time.Now()
		for i := 0; i < 3; i++ {
			next := cronParsed.NextTime(current)
			fmt.Printf("  %d. %s\n", i+1, next.Format("2006-01-02 15:04:05 Mon"))
			current = next
		}
		fmt.Println()
	}
	
	fmt.Println("Cron examples completed successfully!")
	fmt.Println("\nCommon cron expressions:")
	fmt.Println("  */5 * * * *    - Every 5 minutes")
	fmt.Println("  0 */2 * * *    - Every 2 hours")
	fmt.Println("  0 9 * * 1-5    - 9 AM Monday-Friday")
	fmt.Println("  0 0 * * 0      - Midnight every Sunday")
	fmt.Println("  0 12 1 * *     - Noon on 1st of each month")
	fmt.Println("  0 0 1 1 *      - Midnight on New Year's Day")
}