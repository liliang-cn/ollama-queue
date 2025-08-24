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

// GenericQueueManager represents the main queue management system with pluggable executors
type GenericQueueManager struct {
	config         *models.Config
	storage        storage.Storage
	scheduler      scheduler.Scheduler
	cronScheduler  *scheduler.CronScheduler
	
	// Executor registry for pluggable executors
	executorRegistry *executor.ExecutorRegistry
	taskAdapter      *models.TaskAdapter
	
	// Worker management
	workerPool    chan struct{}
	activeWorkers map[string]context.CancelFunc
	workersMu     sync.RWMutex
	
	// Task management
	runningTasks  map[string]*models.GenericTask
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

// NewGenericQueueManager creates a new generic queue manager
func NewGenericQueueManager(config *models.Config) (*GenericQueueManager, error) {
	if config == nil {
		config = models.DefaultConfig()
	}
	
	// Initialize storage
	badgerStorage := storage.NewBadgerStorage(config.StoragePath)
	if err := badgerStorage.Open(); err != nil {
		return nil, fmt.Errorf("failed to open storage: %w", err)
	}
	
	// Initialize executor registry
	executorRegistry := executor.NewExecutorRegistry()
	
	// Register default Ollama executor
	ollamaPlugin, err := executor.NewOllamaPlugin(config)
	if err != nil {
		badgerStorage.Close()
		return nil, fmt.Errorf("failed to create Ollama executor: %w", err)
	}
	
	if err := executorRegistry.Register(models.ExecutorTypeOllama, ollamaPlugin); err != nil {
		badgerStorage.Close()
		return nil, fmt.Errorf("failed to register Ollama executor: %w", err)
	}
	
	// Initialize scheduler
	var taskScheduler scheduler.Scheduler
	localScheduler := scheduler.NewPriorityScheduler()
	
	if config.RemoteScheduling.Enabled {
		// Remote scheduling logic
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
	
	// Initialize cron scheduler
	cronScheduler := scheduler.NewCronScheduler(badgerStorage, nil) // Will be set later
	
	ctx, cancel := context.WithCancel(context.Background())
	
	qm := &GenericQueueManager{
		config:           config,
		storage:          badgerStorage,
		scheduler:        taskScheduler,
		cronScheduler:    cronScheduler,
		executorRegistry: executorRegistry,
		taskAdapter:      models.NewTaskAdapter(),
		workerPool:       make(chan struct{}, config.MaxWorkers),
		activeWorkers:    make(map[string]context.CancelFunc),
		runningTasks:     make(map[string]*models.GenericTask),
		eventSubscribers: make(map[chan *models.TaskEvent]bool),
		ctx:              ctx,
		cancel:           cancel,
		stopped:          make(chan struct{}),
	}
	
	// Fill worker pool
	for i := 0; i < config.MaxWorkers; i++ {
		qm.workerPool <- struct{}{}
	}
	
	return qm, nil
}

// RegisterExecutor registers a new executor plugin
func (qm *GenericQueueManager) RegisterExecutor(executorType models.ExecutorType, executor executor.GenericExecutor) error {
	return qm.executorRegistry.Register(executorType, executor)
}

// UnregisterExecutor removes an executor plugin
func (qm *GenericQueueManager) UnregisterExecutor(executorType models.ExecutorType) {
	qm.executorRegistry.Unregister(executorType)
}

// ListExecutors returns metadata for all registered executors
func (qm *GenericQueueManager) ListExecutors() []*executor.ExecutorMetadata {
	return qm.executorRegistry.ListExecutors()
}

// SubmitGenericTask submits a generic task to the queue
func (qm *GenericQueueManager) SubmitGenericTask(task *models.GenericTask) (string, error) {
	if task.ID == "" {
		task.ID = uuid.New().String()
	}
	
	// Validate that we have an executor for this task
	if !qm.executorRegistry.CanExecuteTask(task) {
		return "", fmt.Errorf("no executor available for type %s with action %s", 
			task.ExecutorType, task.Action)
	}
	
	// Set task timestamps
	task.CreatedAt = time.Now()
	task.Status = models.StatusPending
	
	// Convert to legacy task for storage compatibility
	legacyTask, err := qm.taskAdapter.ConvertToLegacy(task)
	if err != nil {
		return "", fmt.Errorf("failed to convert task: %w", err)
	}
	
	// Save task to storage
	if err := qm.storage.SaveTask(legacyTask); err != nil {
		return "", fmt.Errorf("failed to save task: %w", err)
	}
	
	// Add to scheduler
	if err := qm.scheduler.AddTask(legacyTask); err != nil {
		qm.storage.DeleteTask(task.ID)
		return "", fmt.Errorf("failed to schedule task: %w", err)
	}
	
	// Emit event
	qm.emitEvent(&models.TaskEvent{
		Type:   "task_submitted",
		TaskID: task.ID,
		Status: models.StatusPending,
	})
	
	return task.ID, nil
}

// SubmitTask provides backward compatibility with legacy Task structure
func (qm *GenericQueueManager) SubmitTask(task *models.Task) (string, error) {
	if task.ID == "" {
		task.ID = uuid.New().String()
	}
	
	// Convert legacy task to generic task
	genericTask, err := qm.taskAdapter.ConvertToGeneric(task)
	if err != nil {
		return "", fmt.Errorf("failed to convert legacy task: %w", err)
	}
	
	return qm.SubmitGenericTask(genericTask)
}

// GetTask gets a task by ID (backward compatibility)
func (qm *GenericQueueManager) GetTask(taskID string) (*models.Task, error) {
	return qm.storage.LoadTask(taskID)
}

// GetGenericTask gets a generic task by ID
func (qm *GenericQueueManager) GetGenericTask(taskID string) (*models.GenericTask, error) {
	legacyTask, err := qm.storage.LoadTask(taskID)
	if err != nil {
		return nil, err
	}
	
	return qm.taskAdapter.ConvertToGeneric(legacyTask)
}

// CancelTask cancels a task by ID
func (qm *GenericQueueManager) CancelTask(taskID string) error {
	// Load the task
	task, err := qm.storage.LoadTask(taskID)
	if err != nil {
		return fmt.Errorf("failed to load task: %w", err)
	}
	
	// Update status
	task.Status = models.StatusCancelled
	completedAt := time.Now()
	task.CompletedAt = &completedAt
	
	// Save updated task
	if err := qm.storage.SaveTask(task); err != nil {
		return fmt.Errorf("failed to save cancelled task: %w", err)
	}
	
	// Remove from scheduler
	qm.scheduler.RemoveTask(taskID)
	
	// Emit event
	qm.emitEvent(&models.TaskEvent{
		Type:   "task_cancelled",
		TaskID: taskID,
		Status: models.StatusCancelled,
	})
	
	return nil
}

// GetQueueStats returns queue statistics
func (qm *GenericQueueManager) GetQueueStats() (*models.QueueStats, error) {
	// This would need to be implemented based on storage queries
	return &models.QueueStats{
		PendingTasks:   0,
		RunningTasks:   len(qm.runningTasks),
		CompletedTasks: 0,
		FailedTasks:    0,
	}, nil
}

func (qm *GenericQueueManager) emitEvent(event *models.TaskEvent) {
	qm.eventsMu.RLock()
	defer qm.eventsMu.RUnlock()
	
	for subscriber := range qm.eventSubscribers {
		select {
		case subscriber <- event:
		default:
			// Subscriber is not reading, skip
		}
	}
}

// Close closes the queue manager
func (qm *GenericQueueManager) Close() error {
	qm.stopOnce.Do(func() {
		log.Println("Stopping generic queue manager...")
		
		qm.cancel()
		
		// Close executor registry
		qm.executorRegistry.Close()
		
		// Close storage
		qm.storage.Close()
		
		close(qm.stopped)
		log.Println("Generic queue manager stopped")
	})
	
	return nil
}