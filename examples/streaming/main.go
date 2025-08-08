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
	// Create queue manager
	qm, err := queue.NewQueueManagerWithOptions(
		queue.WithOllamaHost("http://localhost:11434"),
		queue.WithMaxWorkers(2),
		queue.WithStoragePath("./streaming_data"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer qm.Close()

	ctx := context.Background()
	if err := qm.Start(ctx); err != nil {
		log.Fatal(err)
	}

	// Example 1: Streaming Chat
	fmt.Println("=== Streaming Chat Example ===")
	chatTask := queue.NewChatTask("llama2", []models.ChatMessage{
		{Role: "user", Content: "Tell me a short story about a robot"},
	},
		queue.WithChatStreaming(true),
		queue.WithTaskPriority(models.PriorityHigh),
	)

	fmt.Println("Starting streaming chat...")
	streamChan, err := qm.SubmitStreamingTask(chatTask)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print("Response: ")
	for chunk := range streamChan {
		if chunk.Error != nil {
			fmt.Printf("\nError: %v\n", chunk.Error)
			break
		}

		if chunk.Done {
			fmt.Println("\n[Chat stream completed]")
			break
		}

		if chunk.Data != "" {
			fmt.Print(chunk.Data)
		}
	}

	// Example 2: Streaming Generation
	fmt.Println("\n=== Streaming Generation Example ===")
	genTask := queue.NewGenerateTask(
		"codellama",
		"Write a Python function to implement a binary search algorithm with detailed comments",
		queue.WithGenerateStreaming(true),
		queue.WithGenerateSystem("You are a computer science expert"),
		queue.WithTaskPriority(models.PriorityNormal),
	)

	fmt.Println("Starting streaming generation...")
	genStreamChan, err := qm.SubmitStreamingTask(genTask)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print("Code:\n")
	for chunk := range genStreamChan {
		if chunk.Error != nil {
			fmt.Printf("\nError: %v\n", chunk.Error)
			break
		}

		if chunk.Done {
			fmt.Println("\n[Generation stream completed]")
			break
		}

		if chunk.Data != "" {
			fmt.Print(chunk.Data)
		}
	}

	// Example 3: Multiple Concurrent Streams
	fmt.Println("\n=== Multiple Concurrent Streams ===")

	// Create multiple streaming tasks
	tasks := []*models.Task{
		queue.NewChatTask("llama2", []models.ChatMessage{
			{Role: "user", Content: "Explain machine learning in simple terms"},
		}, queue.WithChatStreaming(true)),

		queue.NewGenerateTask(
			"codellama",
			"Create a simple REST API in Go",
			queue.WithGenerateStreaming(true),
		),
	}

	// Start multiple streams concurrently
	streams := make([]<-chan *models.StreamChunk, len(tasks))
	for i, task := range tasks {
		streamChan, err := qm.SubmitStreamingTask(task)
		if err != nil {
			log.Printf("Failed to submit streaming task %d: %v", i, err)
			continue
		}
		streams[i] = streamChan
		fmt.Printf("Started stream %d for task %s\n", i+1, task.ID)
	}

	// Monitor all streams
	for i, stream := range streams {
		if stream == nil {
			continue
		}

		fmt.Printf("\n--- Stream %d Output ---\n", i+1)
		go func(streamIndex int, streamChan <-chan *models.StreamChunk) {
			for chunk := range streamChan {
				if chunk.Error != nil {
					fmt.Printf("[Stream %d] Error: %v\n", streamIndex+1, chunk.Error)
					return
				}

				if chunk.Done {
					fmt.Printf("[Stream %d] Completed\n", streamIndex+1)
					return
				}

				if chunk.Data != "" {
					fmt.Printf("[Stream %d] %s", streamIndex+1, chunk.Data)
				}
			}
		}(i, stream)
	}

	// Wait a bit for streams to complete
	time.Sleep(30 * time.Second)

	fmt.Println("\n=== Streaming examples completed ===")
}