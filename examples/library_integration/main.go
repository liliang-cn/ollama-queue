package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/liliang-cn/ollama-queue/internal/models"
	"github.com/liliang-cn/ollama-queue/pkg/queue"
)

func main() {
	// Initialize queue manager
	qm, err := queue.NewQueueManagerWithOptions(
		queue.WithOllamaHost("http://localhost:11434"),
		queue.WithMaxWorkers(4),
		queue.WithStoragePath("./library_example_data"),
		queue.WithLogLevel("info"),
	)
	if err != nil {
		log.Fatal("Failed to create queue manager:", err)
	}
	defer qm.Close()

	// Start the queue manager
	ctx := context.Background()
	if err := qm.Start(ctx); err != nil {
		log.Fatal("Failed to start queue manager:", err)
	}

	fmt.Println("🚀 Ollama Queue started successfully!")

	// Example 1: Simple chat task
	fmt.Println("\n📝 Example 1: Simple Chat Task")
	chatTask := queue.NewChatTask("qwen3", []models.ChatMessage{
		{Role: "user", Content: "What is the capital of France?"},
	}, queue.WithTaskPriority(models.PriorityHigh))

	taskID, err := qm.SubmitTask(chatTask)
	if err != nil {
		log.Fatal("Failed to submit task:", err)
	}
	fmt.Printf("✅ Chat task submitted with ID: %s\n", taskID)

	// Example 2: Async processing with callback
	fmt.Println("\n🔄 Example 2: Async Processing")
	asyncTask := queue.NewGenerateTask(
		"qwen3",
		"Write a simple 'Hello, World!' program in Go",
		queue.WithTaskPriority(models.PriorityNormal),
	)

	_, err = qm.SubmitTaskWithCallback(asyncTask, func(result *models.TaskResult) {
		if result.Success {
			fmt.Printf("✅ Code generation completed successfully!\n")
			fmt.Printf("📄 Result preview: %v\n", fmt.Sprintf("%.100s...", fmt.Sprintf("%v", result.Data)))
		} else {
			fmt.Printf("❌ Code generation failed: %s\n", result.Error)
		}
	})
	if err != nil {
		log.Fatal("Failed to submit async task:", err)
	}

	// Example 3: Streaming task
	fmt.Println("\n🌊 Example 3: Streaming Task")
	streamTask := queue.NewChatTask("qwen3", []models.ChatMessage{
		{Role: "user", Content: "Tell me a short joke"},
	},
		queue.WithChatStreaming(true),
		queue.WithTaskPriority(models.PriorityHigh),
	)

	streamChan, err := qm.SubmitStreamingTask(streamTask)
	if err != nil {
		log.Fatal("Failed to submit streaming task:", err)
	}

	fmt.Print("🤖 Response: ")
	for chunk := range streamChan {
		if chunk.Error != nil {
			fmt.Printf("\n❌ Stream error: %v\n", chunk.Error)
			break
		}
		if chunk.Done {
			fmt.Println("\n✅ Stream completed!")
			break
		}
		if chunk.Data != "" {
			fmt.Print(chunk.Data)
		}
	}

	// Example 4: Batch processing
	fmt.Println("\n📦 Example 4: Batch Processing")
	batchTasks := []*models.Task{
		queue.NewChatTask("qwen3", []models.ChatMessage{
			{Role: "user", Content: "What is Go language?"},
		}),
		queue.NewChatTask("qwen3", []models.ChatMessage{
			{Role: "user", Content: "What is Python?"},
		}),
		queue.NewEmbedTask("nomic-embed-text", "Sample text for embedding"),
	}

	fmt.Printf("📤 Submitting batch of %d tasks...\n", len(batchTasks))
	results, err := qm.SubmitBatchTasks(batchTasks)
	if err != nil {
		log.Fatal("Failed to submit batch tasks:", err)
	}

	successCount := 0
	for i, result := range results {
		if result.Success {
			successCount++
			fmt.Printf("✅ Task %d completed successfully\n", i+1)
		} else {
			fmt.Printf("❌ Task %d failed: %s\n", i+1, result.Error)
		}
	}
	fmt.Printf("📊 Batch results: %d/%d tasks successful\n", successCount, len(results))

	// Example 5: Queue monitoring
	fmt.Println("\n📊 Example 5: Queue Statistics")
	stats, err := qm.GetQueueStats()
	if err != nil {
		log.Fatal("Failed to get queue stats:", err)
	}

	fmt.Printf("📈 Queue Statistics:\n")
	fmt.Printf("   • Pending: %d\n", stats.PendingTasks)
	fmt.Printf("   • Running: %d\n", stats.RunningTasks)
	fmt.Printf("   • Completed: %d\n", stats.CompletedTasks)
	fmt.Printf("   • Failed: %d\n", stats.FailedTasks)
	fmt.Printf("   • Total: %d\n", stats.TotalTasks)
	fmt.Printf("   • Active Workers: %d\n", stats.WorkersActive)
	fmt.Printf("   • Idle Workers: %d\n", stats.WorkersIdle)

	// Wait a bit for async tasks to complete
	time.Sleep(5 * time.Second)

	fmt.Println("\n🎉 Library integration example completed!")
}
