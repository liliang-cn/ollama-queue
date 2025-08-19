package queue

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/liliang-cn/ollama-queue/pkg/models"
	"github.com/liliang-cn/ollama-queue/pkg/executor"
	"github.com/liliang-cn/ollama-queue/pkg/scheduler"
	"github.com/liliang-cn/ollama-queue/pkg/storage"
)

// QueueManager represents the main queue management system
type QueueManager struct {
	config        *models.Config
	storage       storage.Storage
	scheduler     scheduler.Scheduler
	retryScheduler *scheduler.RetryScheduler
	executor      executor.Executor
	
	// Worker management
	workerPool    chan struct{}
	activeWorkers map[string]context.CancelFunc
	workersMu     sync.RWMutex
	
	// Task management
	runningTasks  map[string]*models.Task
	tasksMu       sync.RWMutex
	
	// Event system
	eventSubscribers map[chan *models.TaskEvent]bool
	eventsMu         sync.RWMutex
	
	// Control
	ctx       context.Context
	cancel    context.CancelFunc
	stopped   chan struct{}
	startOnce sync.Once
	stopOnce  sync.Once
}

// NewQueueManager creates a new queue manager with the given configuration
func NewQueueManager(config *models.Config) (*QueueManager, error) {
	if config == nil {
		config = models.DefaultConfig()
	}
	
	// Initialize storage
	badgerStorage := storage.NewBadgerStorage(config.StoragePath)
	if err := badgerStorage.Open(); err != nil {
		return nil, fmt.Errorf("failed to open storage: %w", err)
	}
	
	// Initialize executor
	ollamaExecutor, err := executor.NewOllamaExecutor(config)
	if err != nil {
		badgerStorage.Close()
		return nil, fmt.Errorf("failed to create executor: %w", err)
	}
	
	// Initialize scheduler (with remote scheduling if enabled)
	var taskScheduler scheduler.Scheduler
	localScheduler := scheduler.NewPriorityScheduler()
	
	if config.RemoteScheduling.Enabled {
		// Convert config endpoints to scheduler endpoints
		var endpoints []scheduler.RemoteEndpoint
		for _, ep := range config.RemoteScheduling.Endpoints {
			if ep.Enabled {
				endpoints = append(endpoints, scheduler.RemoteEndpoint{
					Name:      ep.Name,
					BaseURL:   ep.BaseURL,
					APIKey:    ep.APIKey,
					Priority:  ep.Priority,
					Available: true,
				})
			}
		}
		
		// Create remote scheduler with local fallback
		remoteConfig := scheduler.RemoteSchedulerConfig{
			Endpoints:           endpoints,
			HealthCheckInterval: config.RemoteScheduling.HealthCheckInterval,
			FallbackScheduler:   localScheduler,
			MaxLocalQueueSize:   config.RemoteScheduling.MaxLocalQueueSize,
			LocalFirstPolicy:    config.RemoteScheduling.LocalFirstPolicy,
			Storage:             badgerStorage,
		}
		
		taskScheduler = scheduler.NewRemoteScheduler(remoteConfig)
	} else {
		taskScheduler = localScheduler
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	qm := &QueueManager{
		config:           config,
		storage:          badgerStorage,
		scheduler:        taskScheduler,
		retryScheduler:   scheduler.NewRetryScheduler(config.RetryConfig),
		executor:         ollamaExecutor,
		workerPool:       make(chan struct{}, config.MaxWorkers),
		activeWorkers:    make(map[string]context.CancelFunc),
		runningTasks:     make(map[string]*models.Task),
		eventSubscribers: make(map[chan *models.TaskEvent]bool),
		ctx:              ctx,
		cancel:           cancel,
		stopped:          make(chan struct{}),
	}
	
	// Initialize worker pool
	for i := 0; i < config.MaxWorkers; i++ {
		qm.workerPool <- struct{}{}
	}
	
	return qm, nil
}

// Start starts the queue manager and its background processes
func (qm *QueueManager) Start(ctx context.Context) error {
	var startErr error
	
	qm.startOnce.Do(func() {
		log.Println("Starting queue manager...")
		
		// Load pending tasks from storage
		if err := qm.loadPendingTasks(); err != nil {
			startErr = fmt.Errorf("failed to load pending tasks: %w", err)
			return
		}
		
		// Start background processes
		go qm.schedulerLoop()
		go qm.retryLoop()
		go qm.cleanupLoop()
		
		log.Printf("Queue manager started with %d workers", qm.config.MaxWorkers)
	})
	
	return startErr
}

// Stop stops the queue manager and all workers
func (qm *QueueManager) Stop() error {
	var stopErr error
	
	qm.stopOnce.Do(func() {
		log.Println("Stopping queue manager...")
		
		qm.cancel()
		
		// Cancel all active workers
		qm.workersMu.Lock()
		for workerID, cancelFunc := range qm.activeWorkers {
			log.Printf("Cancelling worker %s", workerID)
			cancelFunc()
		}
		qm.workersMu.Unlock()
		
		// Wait for all workers to finish with timeout
		select {
		case <-qm.stopped:
			log.Println("All workers stopped")
		case <-time.After(30 * time.Second):
			log.Println("Timeout waiting for workers to stop")
		}
		
		close(qm.stopped)
		log.Println("Queue manager stopped")
	})
	
	return stopErr
}

// Close closes the queue manager and releases resources
func (qm *QueueManager) Close() error {
	if err := qm.Stop(); err != nil {
		return err
	}
	
	return qm.storage.Close()
}

// SubmitTask submits a new task to the queue
func (qm *QueueManager) SubmitTask(task *models.Task) (string, error) {
	if task.ID == "" {
		task.ID = uuid.New().String()
	}
	
	// Set default values
	if task.Status == "" {
		task.Status = models.StatusPending
	}
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}
	if task.MaxRetries == 0 {
		task.MaxRetries = qm.config.RetryConfig.MaxRetries
	}
	
	// Validate task
	if err := qm.validateTask(task); err != nil {
		return "", fmt.Errorf("task validation failed: %w", err)
	}
	
	// Save to storage
	if err := qm.storage.SaveTask(task); err != nil {
		return "", fmt.Errorf("failed to save task: %w", err)
	}
	
	// Add to scheduler
	if err := qm.scheduler.AddTask(task); err != nil {
		return "", fmt.Errorf("failed to add task to scheduler: %w", err)
	}
	
	// Publish event
	qm.publishEvent(&models.TaskEvent{
		TaskID:    task.ID,
		Type:      "task_submitted",
		Status:    task.Status,
		Timestamp: time.Now(),
	})
	
	log.Printf("Task %s submitted successfully", task.ID)
	return task.ID, nil
}

// SubmitTaskWithCallback submits a task and calls the callback when done
func (qm *QueueManager) SubmitTaskWithCallback(task *models.Task, callback models.TaskCallback) (string, error) {
	taskID, err := qm.SubmitTask(task)
	if err != nil {
		return "", err
	}
	
	// Start goroutine to wait for completion
	go qm.waitForTaskCompletion(taskID, callback)
	
	return taskID, nil
}

// SubmitTaskWithChannel submits a task and sends result to channel
func (qm *QueueManager) SubmitTaskWithChannel(task *models.Task, resultChan chan *models.TaskResult) (string, error) {
	return qm.SubmitTaskWithCallback(task, func(result *models.TaskResult) {
		select {
		case resultChan <- result:
		default:
			log.Printf("Result channel full for task %s", result.TaskID)
		}
	})
}

// SubmitStreamingTask submits a task for streaming execution
func (qm *QueueManager) SubmitStreamingTask(task *models.Task) (<-chan *models.StreamChunk, error) {
	// For streaming tasks, we execute immediately
	if !qm.supportsStreaming(task.Type) {
		return nil, fmt.Errorf("task type %s does not support streaming", task.Type)
	}
	
	if task.ID == "" {
		task.ID = uuid.New().String()
	}
	
	task.Status = models.StatusRunning
	task.CreatedAt = time.Now()
	now := time.Now()
	task.StartedAt = &now
	
	// Save to storage
	if err := qm.storage.SaveTask(task); err != nil {
		return nil, fmt.Errorf("failed to save streaming task: %w", err)
	}
	
	// Execute immediately
	streamChan, err := qm.executor.ExecuteStream(qm.ctx, task)
	if err != nil {
		task.Status = models.StatusFailed
		task.Error = err.Error()
		qm.storage.SaveTask(task)
		return nil, fmt.Errorf("failed to start streaming task: %w", err)
	}
	
	// Monitor stream completion
	go qm.monitorStreamingTask(task, streamChan)
	
	return streamChan, nil
}

// GetTask retrieves a task by ID
func (qm *QueueManager) GetTask(taskID string) (*models.Task, error) {
	// Check running tasks first
	qm.tasksMu.RLock()
	if task, exists := qm.runningTasks[taskID]; exists {
		qm.tasksMu.RUnlock()
		return task, nil
	}
	qm.tasksMu.RUnlock()
	
	// Load from storage
	return qm.storage.LoadTask(taskID)
}

// GetTaskStatus retrieves the status of a task
func (qm *QueueManager) GetTaskStatus(taskID string) (models.TaskStatus, error) {
	task, err := qm.GetTask(taskID)
	if err != nil {
		return "", err
	}
	
	return task.Status, nil
}

// CancelTask cancels a task
func (qm *QueueManager) CancelTask(taskID string) error {
	// Check if task is running
	qm.tasksMu.RLock()
	if task, exists := qm.runningTasks[taskID]; exists {
		qm.tasksMu.RUnlock()
		
		// Cancel the task's context
		if task.CancelFunc != nil {
			task.CancelFunc()
		}
		
		// Update status
		task.Status = models.StatusCancelled
		now := time.Now()
		task.CompletedAt = &now
		
		if err := qm.storage.SaveTask(task); err != nil {
			return fmt.Errorf("failed to save cancelled task: %w", err)
		}
		
		qm.publishEvent(&models.TaskEvent{
			TaskID:    taskID,
			Type:      "task_cancelled",
			Status:    task.Status,
			Timestamp: time.Now(),
		})
		
		return nil
	}
	qm.tasksMu.RUnlock()
	
	// Try to remove from scheduler
	if err := qm.scheduler.RemoveTask(taskID); err != nil {
		return fmt.Errorf("failed to remove task from scheduler: %w", err)
	}
	
	// Update status in storage
	if err := qm.storage.UpdateTaskStatus(taskID, models.StatusCancelled); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}
	
	qm.publishEvent(&models.TaskEvent{
		TaskID:    taskID,
		Type:      "task_cancelled",
		Status:    models.StatusCancelled,
		Timestamp: time.Now(),
	})
	
	return nil
}

// UpdateTaskPriority updates the priority of a task
func (qm *QueueManager) UpdateTaskPriority(taskID string, priority models.Priority) error {
	// Try to update in scheduler first
	if err := qm.scheduler.UpdateTaskPriority(taskID, priority); err != nil {
		return fmt.Errorf("failed to update task priority in scheduler: %w", err)
	}
	
	// Update in storage
	task, err := qm.storage.LoadTask(taskID)
	if err != nil {
		return fmt.Errorf("failed to load task: %w", err)
	}
	
	task.Priority = priority
	if err := qm.storage.SaveTask(task); err != nil {
		return fmt.Errorf("failed to save updated task: %w", err)
	}
	
	qm.publishEvent(&models.TaskEvent{
		TaskID:    taskID,
		Type:      "priority_updated",
		Status:    task.Status,
		Timestamp: time.Now(),
		Data:      priority,
	})
	
	return nil
}

// ListTasks retrieves tasks based on filter criteria
func (qm *QueueManager) ListTasks(filter models.TaskFilter) ([]*models.Task, error) {
	return qm.storage.ListTasks(filter)
}

// GetQueueStats retrieves queue statistics
func (qm *QueueManager) GetQueueStats() (*models.QueueStats, error) {
	stats, err := qm.storage.GetStats()
	if err != nil {
		return nil, err
	}
	
	// Add scheduler stats
	schedulerStats := qm.scheduler.GetQueueStats()
	for priority, count := range schedulerStats {
		stats.QueuesByPriority[priority] = count
	}
	
	// Add worker stats
	qm.workersMu.RLock()
	stats.WorkersActive = len(qm.activeWorkers)
	stats.WorkersIdle = qm.config.MaxWorkers - len(qm.activeWorkers)
	qm.workersMu.RUnlock()
	
	return stats, nil
}

// Subscribe to task events
func (qm *QueueManager) Subscribe(eventTypes []string) (<-chan *models.TaskEvent, error) {
	eventChan := make(chan *models.TaskEvent, 100)
	
	qm.eventsMu.Lock()
	qm.eventSubscribers[eventChan] = true
	qm.eventsMu.Unlock()
	
	return eventChan, nil
}

// Unsubscribe from task events
func (qm *QueueManager) Unsubscribe(eventChan <-chan *models.TaskEvent) error {
	qm.eventsMu.Lock()
	defer qm.eventsMu.Unlock()
	
	// Convert read-only channel to the map key type
	for ch := range qm.eventSubscribers {
		if ch == eventChan {
			delete(qm.eventSubscribers, ch)
			close(ch)
			break
		}
	}
	
	return nil
}

// Background loops and helper methods

func (qm *QueueManager) schedulerLoop() {
	ticker := time.NewTicker(qm.config.SchedulingInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-qm.ctx.Done():
			return
		case <-ticker.C:
			qm.processNextTasks()
		}
	}
}

func (qm *QueueManager) retryLoop() {
	ticker := time.NewTicker(qm.config.SchedulingInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-qm.ctx.Done():
			return
		case <-ticker.C:
			qm.processRetryTasks()
		}
	}
}

func (qm *QueueManager) cleanupLoop() {
	ticker := time.NewTicker(qm.config.CleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-qm.ctx.Done():
			return
		case <-ticker.C:
			qm.cleanupCompletedTasks()
		}
	}
}

func (qm *QueueManager) processNextTasks() {
	for {
		// Check if we have available workers
		select {
		case <-qm.workerPool:
			// Get next task
			task := qm.scheduler.GetNextTask()
			if task == nil {
				// No tasks available, return worker to pool
				qm.workerPool <- struct{}{}
				return
			}
			
			// Start worker for this task
			go qm.executeTask(task)
		default:
			// No workers available
			return
		}
	}
}

func (qm *QueueManager) processRetryTasks() {
	readyTasks := qm.retryScheduler.GetReadyTasks()
	for _, task := range readyTasks {
		// Add back to scheduler
		if err := qm.scheduler.AddTask(task); err != nil {
			log.Printf("Failed to re-add retry task %s: %v", task.ID, err)
		} else {
			// Save updated task
			if err := qm.storage.SaveTask(task); err != nil {
				log.Printf("Failed to save retry task %s: %v", task.ID, err)
			}
		}
	}
}

func (qm *QueueManager) executeTask(task *models.Task) {
	defer func() {
		// Return worker to pool
		qm.workerPool <- struct{}{}
		
		// Remove from running tasks
		qm.tasksMu.Lock()
		delete(qm.runningTasks, task.ID)
		qm.tasksMu.Unlock()
		
		// Remove from active workers
		qm.workersMu.Lock()
		delete(qm.activeWorkers, task.ID)
		qm.workersMu.Unlock()
	}()
	
	// Create task context
	taskCtx, cancel := context.WithCancel(qm.ctx)
	task.Context = taskCtx
	task.CancelFunc = cancel
	
	// Add to running tasks and active workers
	qm.tasksMu.Lock()
	qm.runningTasks[task.ID] = task
	qm.tasksMu.Unlock()
	
	qm.workersMu.Lock()
	qm.activeWorkers[task.ID] = cancel
	qm.workersMu.Unlock()
	
	// Update task status
	task.Status = models.StatusRunning
	now := time.Now()
	task.StartedAt = &now
	
	if err := qm.storage.SaveTask(task); err != nil {
		log.Printf("Failed to save task %s: %v", task.ID, err)
	}
	
	qm.publishEvent(&models.TaskEvent{
		TaskID:    task.ID,
		Type:      "task_started",
		Status:    task.Status,
		Timestamp: now,
	})
	
	log.Printf("Executing task %s (type: %s, priority: %d)", task.ID, task.Type, task.Priority)
	
	// Execute task
	result, err := qm.executor.Execute(taskCtx, task)
	
	// Update task with result
	completedAt := time.Now()
	task.CompletedAt = &completedAt
	
	if err != nil {
		task.Status = models.StatusFailed
		task.Error = err.Error()
		
		// Check if should retry
		if task.RetryCount < task.MaxRetries {
			log.Printf("Task %s failed, will retry (attempt %d/%d): %v", 
				task.ID, task.RetryCount+1, task.MaxRetries, err)
			qm.retryScheduler.AddFailedTask(task)
		} else {
			log.Printf("Task %s failed permanently after %d retries: %v", 
				task.ID, task.RetryCount, err)
		}
	} else {
		task.Status = models.StatusCompleted
		task.Result = result.Data
		log.Printf("Task %s completed successfully", task.ID)
	}
	
	// Save final state
	if err := qm.storage.SaveTask(task); err != nil {
		log.Printf("Failed to save completed task %s: %v", task.ID, err)
	}
	
	// Publish completion event
	qm.publishEvent(&models.TaskEvent{
		TaskID:    task.ID,
		Type:      "task_completed",
		Status:    task.Status,
		Timestamp: completedAt,
		Data:      result,
	})
}

func (qm *QueueManager) loadPendingTasks() error {
	pendingTasks, err := qm.storage.GetTasksByStatus(models.StatusPending)
	if err != nil {
		return fmt.Errorf("failed to load pending tasks: %w", err)
	}

	runningTasks, err := qm.storage.GetTasksByStatus(models.StatusRunning)
	if err != nil {
		return fmt.Errorf("failed to load running tasks: %w", err)
	}

	allTasks := append(pendingTasks, runningTasks...)

	for _, task := range allTasks {
		if task.Status == models.StatusRunning {
			// 重新加载任务以获取最新状态
			latestTask, err := qm.storage.LoadTask(task.ID)
			if err != nil {
				log.Printf("Failed to load task %s: %v", task.ID, err)
				continue
			}
			
			// 如果任务已经完成、失败或被取消，跳过不重新执行
			if latestTask.Status == models.StatusCompleted || 
			   latestTask.Status == models.StatusFailed || 
			   latestTask.Status == models.StatusCancelled {
				log.Printf("Task %s already finished with status %s, skipping recovery", latestTask.ID, latestTask.Status)
				continue
			}
			
			// 只有真正被中断的任务才重置为pending
			latestTask.Status = models.StatusPending
			if err := qm.storage.SaveTask(latestTask); err != nil {
				log.Printf("Failed to reset running task %s to pending: %v", latestTask.ID, err)
				continue
			}
			task = latestTask // 更新循环中的task变量
		}
		
		// 只添加pending状态的任务到调度器
		if task.Status == models.StatusPending {
			if err := qm.scheduler.AddTask(task); err != nil {
				log.Printf("Failed to add pending task %s to scheduler: %v", task.ID, err)
			}
		}
	}

	log.Printf("Loaded %d pending tasks from storage", len(allTasks))
	return nil
}

func (qm *QueueManager) cleanupCompletedTasks() {
	err := qm.storage.CleanupCompletedTasks(
		qm.config.MaxCompletedTasks,
		qm.config.CleanupInterval*24, // Keep tasks for 24 cleanup intervals
	)
	if err != nil {
		log.Printf("Failed to cleanup completed tasks: %v", err)
	}
}

func (qm *QueueManager) waitForTaskCompletion(taskID string, callback models.TaskCallback) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-qm.ctx.Done():
			return
		case <-ticker.C:
			task, err := qm.GetTask(taskID)
			if err != nil {
				callback(&models.TaskResult{
					TaskID:  taskID,
					Success: false,
					Error:   err.Error(),
				})
				return
			}
			
			if task.Status == models.StatusCompleted || 
			   task.Status == models.StatusFailed || 
			   task.Status == models.StatusCancelled {
				
				result := &models.TaskResult{
					TaskID:  taskID,
					Success: task.Status == models.StatusCompleted,
					Data:    task.Result,
					Error:   task.Error,
				}
				callback(result)
				return
			}
		}
	}
}

func (qm *QueueManager) monitorStreamingTask(task *models.Task, streamChan <-chan *models.StreamChunk) {
	for chunk := range streamChan {
		if chunk.Done {
			if chunk.Error != nil {
				task.Status = models.StatusFailed
				task.Error = chunk.Error.Error()
			} else {
				task.Status = models.StatusCompleted
			}
			
			now := time.Now()
			task.CompletedAt = &now
			qm.storage.SaveTask(task)
			
			qm.publishEvent(&models.TaskEvent{
				TaskID:    task.ID,
				Type:      "task_completed",
				Status:    task.Status,
				Timestamp: now,
			})
			
			break
		}
	}
}

func (qm *QueueManager) publishEvent(event *models.TaskEvent) {
	qm.eventsMu.RLock()
	defer qm.eventsMu.RUnlock()
	
	for subscriber := range qm.eventSubscribers {
		select {
		case subscriber <- event:
		default:
			// Subscriber channel is full, skip
		}
	}
}

func (qm *QueueManager) validateTask(task *models.Task) error {
	if task.Type == "" {
		return fmt.Errorf("task type is required")
	}
	
	if task.Model == "" {
		return fmt.Errorf("model is required")
	}
	
	if task.Payload == nil {
		return fmt.Errorf("payload is required")
	}
	
	if !qm.executor.CanExecute(task.Type) {
		return fmt.Errorf("unsupported task type: %s", task.Type)
	}
	
	return nil
}

func (qm *QueueManager) supportsStreaming(taskType models.TaskType) bool {
	return taskType == models.TaskTypeChat || taskType == models.TaskTypeGenerate
}