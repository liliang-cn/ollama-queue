package main

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/liliang-cn/ollama-queue/pkg/models"
	"github.com/liliang-cn/ollama-queue/pkg/queue"
)

type MyApp struct {
	queueManager *queue.QueueManager
	server       *http.Server
}

func NewMyApp() (*MyApp, error) {
	// 配置队列
	config := models.DefaultConfig()
	config.StoragePath = "./myapp-data"
	config.ListenAddr = "localhost:8081" // 不与主应用端口冲突

	// 创建队列管理器
	manager, err := queue.NewQueueManager(config)
	if err != nil {
		return nil, err
	}

	app := &MyApp{
		queueManager: manager,
	}

	return app, nil
}

func (app *MyApp) Start() error {
	// 启动队列管理器
	ctx := context.Background()
	if err := app.queueManager.Start(ctx); err != nil {
		return err
	}

	// 设置路由 - 既有你的应用API，也有队列API
	router := gin.Default()

	// 你的应用API
	api := router.Group("/api/v1")
	{
		api.GET("/health", app.healthCheck)
		api.POST("/process", app.processRequest)
	}

	// 队列管理API (嵌入到你的应用中)
	queueAPI := router.Group("/api/queue")
	{
		queueAPI.POST("/tasks", app.submitTask)
		queueAPI.GET("/tasks/:id", app.getTask)
		queueAPI.GET("/stats", app.getStats)
		queueAPI.POST("/tasks/:id/cancel", app.cancelTask)
	}

	// 启动HTTP服务器
	app.server = &http.Server{
		Addr:    ":8080", // 你的主应用端口
		Handler: router,
	}

	log.Println("MyApp starting with embedded queue on :8080")
	log.Println("Queue API available at /api/queue/*")
	
	return app.server.ListenAndServe()
}

func (app *MyApp) Stop() error {
	if app.server != nil {
		app.server.Shutdown(context.Background())
	}
	return app.queueManager.Close()
}

// 你的应用逻辑
func (app *MyApp) healthCheck(c *gin.Context) {
	stats, _ := app.queueManager.GetQueueStats()
	c.JSON(http.StatusOK, gin.H{
		"status":      "healthy",
		"queue_stats": stats,
	})
}

func (app *MyApp) processRequest(c *gin.Context) {
	var request struct {
		Message string `json:"message"`
		Model   string `json:"model"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 创建队列任务
	task := &models.Task{
		Type:     models.TaskTypeChat,
		Model:    request.Model,
		Priority: models.PriorityNormal,
		Payload: []models.ChatMessage{
			{Role: "user", Content: request.Message},
		},
	}

	// 异步提交任务
	taskID, err := app.queueManager.SubmitTask(task)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"task_id": taskID,
		"message": "Task submitted successfully",
	})
}

// 队列API方法
func (app *MyApp) submitTask(c *gin.Context) {
	var task models.Task
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	taskID, err := app.queueManager.SubmitTask(&task)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"task_id": taskID})
}

func (app *MyApp) getTask(c *gin.Context) {
	taskID := c.Param("id")
	task, err := app.queueManager.GetTask(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, task)
}

func (app *MyApp) getStats(c *gin.Context) {
	stats, err := app.queueManager.GetQueueStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (app *MyApp) cancelTask(c *gin.Context) {
	taskID := c.Param("id")
	if err := app.queueManager.CancelTask(taskID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "cancelled"})
}

func main() {
	app, err := NewMyApp()
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}
	defer app.Stop()

	if err := app.Start(); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}
}