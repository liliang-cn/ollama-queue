package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/liliang-cn/ollama-queue/pkg/models"
	"github.com/liliang-cn/ollama-queue/pkg/queue"
)

func main() {
	// Create queue manager
	qm, err := queue.NewQueueManagerWithOptions(
		queue.WithOllamaHost("http://localhost:11434"),
		queue.WithMaxWorkers(4),
		queue.WithStoragePath("./batch_data"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer qm.Close()

	ctx := context.Background()
	if err := qm.Start(ctx); err != nil {
		log.Fatal(err)
	}

	// Example 1: Synchronous Batch Processing
	fmt.Println("=== Synchronous Batch Processing ===")

	// Create a batch of different tasks
	batchTasks := []*models.Task{
		queue.NewChatTask("qwen3", []models.ChatMessage{
			{Role: "user", Content: "What is artificial intelligence?"},
		}, queue.WithTaskPriority(models.PriorityHigh)),

		queue.NewChatTask("qwen3", []models.ChatMessage{
			{Role: "user", Content: "Explain quantum computing"},
		}, queue.WithTaskPriority(models.PriorityNormal)),

		queue.NewGenerateTask("qwen3", "Write a function to reverse a string",
			queue.WithTaskPriority(models.PriorityNormal)),

		queue.NewEmbedTask("nomic-embed-text", "Sample text for embedding",
			queue.WithTaskPriority(models.PriorityLow)),
	}

	fmt.Printf("Submitting batch of %d tasks...\n", len(batchTasks))
	startTime := time.Now()

	// Submit batch and wait for all to complete
	results, err := qm.SubmitBatchTasks(batchTasks)
	if err != nil {
		log.Fatal(err)
	}

	duration := time.Since(startTime)
	fmt.Printf("Batch completed in %v\n\n", duration)

	// Display results
	for i, result := range results {
		fmt.Printf("Task %d (%s): Success=%v\n", i+1, batchTasks[i].Type, result.Success)
		if !result.Success {
			fmt.Printf("  Error: %s\n", result.Error)
		}
	}

	// Example 2: Asynchronous Batch Processing with Callback
	fmt.Println("\n=== Asynchronous Batch Processing ===")

	// Create another batch of tasks
	asyncBatchTasks := []*models.Task{
		queue.NewChatTask("qwen3", []models.ChatMessage{
			{Role: "user", Content: "Tell me about machine learning"},
		}),

		queue.NewChatTask("qwen3", []models.ChatMessage{
			{Role: "user", Content: "What is deep learning?"},
		}),

		queue.NewGenerateTask("qwen3", "Create a simple sorting algorithm"),
	}

	var completedCount int
	var mu sync.Mutex

	// Submit async batch
	taskIDs, err := qm.SubmitBatchTasksAsync(asyncBatchTasks, func(results []*models.TaskResult) {
		fmt.Printf("Async batch of %d tasks completed!\n", len(results))

		successCount := 0
		for _, result := range results {
			if result.Success {
				successCount++
			}
		}

		fmt.Printf("Success rate: %d/%d (%.1f%%)\n",
			successCount, len(results),
			float64(successCount)/float64(len(results))*100)
	})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Submitted async batch with task IDs: %v\n", taskIDs)

	// Example 3: Mixed Priority Batch
	fmt.Println("\n=== Mixed Priority Batch Processing ===")

	// Create tasks with different priorities
	mixedBatch := []*models.Task{
		// Critical priority - should be processed first
		queue.NewChatTask("qwen3", []models.ChatMessage{
			{Role: "user", Content: "URGENT: System status check"},
		}, queue.WithTaskPriority(models.PriorityCritical)),

		// Low priority tasks
		queue.NewGenerateTask("qwen3", "Write documentation",
			queue.WithTaskPriority(models.PriorityLow)),

		queue.NewGenerateTask("qwen3", "Refactor legacy code",
			queue.WithTaskPriority(models.PriorityLow)),

		// High priority task
		queue.NewChatTask("qwen3", []models.ChatMessage{
			{Role: "user", Content: "Quick question about API"},
		}, queue.WithTaskPriority(models.PriorityHigh)),

		// Normal priority tasks
		queue.NewEmbedTask("nomic-embed-text", "Regular embedding task",
			queue.WithTaskPriority(models.PriorityNormal)),
	}

	fmt.Printf("Submitting mixed priority batch...\n")

	// Submit individually to observe execution order
	for i, task := range mixedBatch {
		taskID, err := qm.SubmitTaskWithCallback(task, func(result *models.TaskResult) {
			mu.Lock()
			completedCount++
			current := completedCount
			mu.Unlock()

			fmt.Printf("[%d] Task %s completed (Priority: %d) - Success: %v\n",
				current, result.TaskID[:8], task.Priority, result.Success)
		})

		if err != nil {
			log.Printf("Failed to submit task %d: %v", i, err)
			continue
		}

		fmt.Printf("  Submitted task %d (%s priority) - ID: %s\n",
			i+1, getPriorityName(task.Priority), taskID[:8])
	}

	// Wait for all tasks to complete
	for {
		mu.Lock()
		completed := completedCount
		mu.Unlock()

		if completed >= len(mixedBatch) {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	// Example 4: Monitor Queue Status During Batch Processing
	fmt.Println("\n=== Queue Monitoring ===")

	// Subscribe to task events
	eventChan, err := qm.Subscribe([]string{"task_completed", "task_failed"})
	if err != nil {
		log.Fatal(err)
	}

	// Monitor events in background
	go func() {
		eventCount := 0
		for event := range eventChan {
			eventCount++
			fmt.Printf("Event #%d: %s - Task %s (%s)\n",
				eventCount, event.Type, event.TaskID[:8], event.Status)

			if eventCount >= 5 { // Stop after 5 events for demo
				break
			}
		}
	}()

	// Create final batch
	finalBatch := []*models.Task{
		queue.NewChatTask("qwen3", []models.ChatMessage{
			{Role: "user", Content: "Final test message 1"},
		}),
		queue.NewChatTask("qwen3", []models.ChatMessage{
			{Role: "user", Content: "Final test message 2"},
		}),
		queue.NewGenerateTask("qwen3", "Final code generation test"),
	}

	fmt.Println("Submitting final monitoring batch...")
	_, err = qm.SubmitBatchTasksAsync(finalBatch, func(results []*models.TaskResult) {
		fmt.Println("Final batch completed!")
	})
	if err != nil {
		log.Fatal(err)
	}

	// Wait a bit and show final stats
	time.Sleep(5 * time.Second)

	stats, err := qm.GetQueueStats()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\nFinal Queue Statistics:\n")
	fmt.Printf("  Total processed: %d\n", stats.TotalTasks)
	fmt.Printf("  Completed: %d\n", stats.CompletedTasks)
	fmt.Printf("  Failed: %d\n", stats.FailedTasks)
	fmt.Printf("  Currently pending: %d\n", stats.PendingTasks)
	fmt.Printf("  Currently running: %d\n", stats.RunningTasks)

	fmt.Println("\n=== Batch processing examples completed ===")
}

func getPriorityName(priority models.Priority) string {
	switch priority {
	case models.PriorityCritical:
		return "critical"
	case models.PriorityHigh:
		return "high"
	case models.PriorityNormal:
		return "normal"
	case models.PriorityLow:
		return "low"
	default:
		return fmt.Sprintf("custom(%d)", priority)
	}
}
