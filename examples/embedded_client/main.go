package main

import (
	"fmt"
	"log"
	"time"

	"github.com/liliang-cn/ollama-queue/pkg/models"
	"github.com/liliang-cn/ollama-queue/pkg/client"
	"github.com/liliang-cn/ollama-queue/pkg/config"
)

type MyService struct {
	queueClient *client.Client
}

func NewMyService() (*MyService, error) {
	// 加载配置（或直接指定服务器地址）
	configLoader := config.NewConfigLoader()
	cfg, err := configLoader.LoadConfig()
	if err != nil {
		// 如果没有配置文件，使用默认地址
		cfg = &models.Config{
			ListenAddr: "localhost:8080", // 队列服务器地址
		}
	}

	// 创建队列客户端
	queueClient := client.New(cfg.ListenAddr)

	return &MyService{
		queueClient: queueClient,
	}, nil
}

func (s *MyService) ProcessDocument(content string) error {
	log.Printf("Processing document: %s", content[:50]+"...")

	// 创建嵌入任务
	task := &models.Task{
		Type:     models.TaskTypeEmbed,
		Model:    "qwen3",
		Priority: models.PriorityHigh,
		Payload: map[string]interface{}{
			"input": content,
		},
	}

	// 提交任务
	taskID, err := s.queueClient.SubmitTask(task)
	if err != nil {
		return err
	}

	log.Printf("Document processing task submitted: %s", taskID)

	// 可选：等待任务完成
	return s.waitForCompletion(taskID)
}

func (s *MyService) GenerateReport() error {
	// 创建生成任务
	task := &models.Task{
		Type:     models.TaskTypeGenerate,
		Model:    "qwen3",
		Priority: models.PriorityNormal,
		Payload: map[string]interface{}{
			"prompt": "Generate a daily report based on recent activities",
			"system": "You are a professional report generator",
		},
	}

	taskID, err := s.queueClient.SubmitTask(task)
	if err != nil {
		return err
	}

	log.Printf("Report generation task submitted: %s", taskID)
	return nil
}

func (s *MyService) SetupCronJobs() error {
	// 设置每日报告定时任务
	cronTask := &models.CronTask{
		Name:     "Daily Report",
		CronExpr: "0 9 * * 1-5", // 工作日上午9点
		Enabled:  true,
		TaskTemplate: models.TaskTemplate{
			Type:     models.TaskTypeGenerate,
			Model:    "qwen3",
			Priority: models.PriorityNormal,
			Payload: map[string]interface{}{
				"prompt": "Generate today's business summary",
			},
		},
		Metadata: map[string]interface{}{
			"service": "MyService",
			"type":    "daily_report",
		},
	}

	cronID, err := s.queueClient.AddCronTask(cronTask)
	if err != nil {
		return err
	}

	log.Printf("Daily report cron task created: %s", cronID)
	return nil
}

func (s *MyService) waitForCompletion(taskID string) error {
	// 轮询等待任务完成
	for {
		task, err := s.queueClient.GetTask(taskID)
		if err != nil {
			return err
		}

		switch task.Status {
		case models.StatusCompleted:
			log.Printf("Task %s completed successfully", taskID)
			return nil
		case models.StatusFailed:
			log.Printf("Task %s failed: %s", taskID, task.Error)
			return fmt.Errorf("task failed: %s", task.Error)
		case models.StatusCancelled:
			log.Printf("Task %s was cancelled", taskID)
			return fmt.Errorf("task was cancelled")
		default:
			// 仍在处理中，继续等待
			time.Sleep(1 * time.Second)
		}
	}
}

func (s *MyService) GetQueueStatus() (*models.QueueStats, error) {
	return s.queueClient.GetQueueStats()
}

func main() {
	service, err := NewMyService()
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	// 设置定时任务
	if err := service.SetupCronJobs(); err != nil {
		log.Printf("Failed to setup cron jobs: %v", err)
	}

	// 你的服务逻辑
	for {
		// 模拟业务逻辑
		if err := service.ProcessDocument("Sample document content..."); err != nil {
			log.Printf("Failed to process document: %v", err)
		}

		// 检查队列状态
		if stats, err := service.GetQueueStatus(); err == nil {
			log.Printf("Queue status - Pending: %d, Running: %d", 
				stats.PendingTasks, stats.RunningTasks)
		}

		time.Sleep(30 * time.Second)
	}
}