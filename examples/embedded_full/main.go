package main

import (
	"context"
	"log"
	"time"

	"github.com/liliang-cn/ollama-queue/pkg/models"
	"github.com/liliang-cn/ollama-queue/pkg/queue"
)

func main() {
	// 创建配置
	config := models.DefaultConfig()
	config.StoragePath = "./myapp-queue-data"
	config.OllamaHost = "http://localhost:11434"
	config.MaxWorkers = 5

	// 创建队列管理器
	manager, err := queue.NewQueueManager(config)
	if err != nil {
		log.Fatalf("Failed to create queue manager: %v", err)
	}
	defer manager.Close()

	// 启动队列管理器
	ctx := context.Background()
	if err := manager.Start(ctx); err != nil {
		log.Fatalf("Failed to start queue manager: %v", err)
	}

	// 在你的应用中使用队列
	go processUserRequests(manager)
	go monitorQueue(manager)

	// 你的应用主逻辑
	log.Println("Application running with embedded ollama-queue...")
	select {} // 保持运行
}

func processUserRequests(manager *queue.QueueManager) {
	// 示例：处理用户请求并提交任务
	for {
		// 模拟接收用户请求
		time.Sleep(5 * time.Second)

		// 创建聊天任务
		task := &models.Task{
			Type:     models.TaskTypeChat,
			Model:    "qwen3",
			Priority: models.PriorityNormal,
			Payload: []models.ChatMessage{
				{Role: "user", Content: "Hello from embedded queue!"},
			},
		}

		// 提交任务
		taskID, err := manager.SubmitTask(task)
		if err != nil {
			log.Printf("Failed to submit task: %v", err)
			continue
		}

		log.Printf("Submitted task: %s", taskID)
	}
}

func monitorQueue(manager *queue.QueueManager) {
	// 监控队列状态
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats, err := manager.GetQueueStats()
		if err != nil {
			log.Printf("Failed to get stats: %v", err)
			continue
		}

		log.Printf("Queue stats - Pending: %d, Running: %d, Completed: %d",
			stats.PendingTasks, stats.RunningTasks, stats.CompletedTasks)
	}
}