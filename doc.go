/*
Package ollama-queue provides a high-performance task queue management system for Ollama AI models.

This library enables efficient queuing, scheduling, and execution of AI model tasks with support for
different task types including chat, text generation, and embeddings. It features priority-based
scheduling, retry mechanisms, persistence, streaming support, and comprehensive monitoring capabilities.

# Quick Start

Basic usage example:

	package main

	import (
		"context"
		"log"
		
		"github.com/liliang-cn/ollama-queue/internal/models"
		"github.com/liliang-cn/ollama-queue/pkg/queue"
	)

	func main() {
		// Create and configure queue manager
		qm, err := queue.NewQueueManagerWithOptions(
			queue.WithOllamaHost("http://localhost:11434"),
			queue.WithMaxWorkers(4),
		)
		if err != nil {
			log.Fatal(err)
		}
		defer qm.Close()

		// Start the queue manager
		ctx := context.Background()
		if err := qm.Start(ctx); err != nil {
			log.Fatal(err)
		}

		// Create and submit a chat task
		task := queue.NewChatTask("llama2", []models.ChatMessage{
			{Role: "user", Content: "Hello, world!"},
		})

		taskID, err := qm.SubmitTask(task)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Task submitted: %s", taskID)
	}

# Features

• Priority-based task scheduling (Critical, High, Normal, Low)
• Support for multiple task types: Chat, Generate, Embed
• Persistent task storage using BadgerDB
• Retry mechanism with configurable policies
• Streaming support for real-time responses
• Comprehensive monitoring and statistics
• Event subscription system
• Batch task processing
• Graceful shutdown and resource cleanup

# Task Types

The library supports three main task types:

1. Chat Tasks - Interactive conversations with AI models
2. Generation Tasks - Text generation from prompts
3. Embedding Tasks - Vector embeddings for text input

Each task type has specific configuration options and supports various execution modes.

# Configuration

The queue manager can be configured using functional options:

	qm, err := queue.NewQueueManagerWithOptions(
		queue.WithOllamaHost("http://localhost:11434"),
		queue.WithMaxWorkers(8),
		queue.WithStoragePath("./data"),
		queue.WithOllamaTimeout(30 * time.Second),
		queue.WithCleanupInterval(1 * time.Hour),
		queue.WithMaxCompletedTasks(1000),
		queue.WithLogLevel("info"),
	)

# Examples

The library includes several comprehensive examples:

• Basic usage with different task types
• Batch processing of multiple tasks
• Streaming responses for real-time applications
• Library integration patterns

See the examples/ directory for detailed implementation examples.

# Requirements

• Go 1.21 or later
• Ollama server running locally or remotely
• BadgerDB for task persistence

# Architecture

The library follows a modular architecture with separate components for:

• Queue management and task orchestration
• Task scheduling with priority support
• Ollama API integration and execution
• Persistent storage and state management
• Event handling and monitoring

This design ensures scalability, reliability, and ease of maintenance.
*/
package ollama_queue