package scheduler

import (
	"container/heap"
	"sync"
	"time"

	"github.com/liliang-cn/ollama-queue/internal/models"
)

// Scheduler interface defines task scheduling operations
type Scheduler interface {
	AddTask(task *models.Task) error
	GetNextTask() *models.Task
	RemoveTask(taskID string) error
	UpdateTaskPriority(taskID string, priority models.Priority) error
	GetQueueLength() int
	GetQueueStats() map[models.Priority]int
}

// PriorityScheduler implements priority-based task scheduling
type PriorityScheduler struct {
	queues map[models.Priority]*taskQueue
	tasks  map[string]*models.Task
	mu     sync.RWMutex
}

// taskQueue implements a priority queue for tasks
type taskQueue struct {
	items []*taskItem
	mu    sync.Mutex
}

// taskItem represents a task in the priority queue
type taskItem struct {
	task     *models.Task
	priority models.Priority
	index    int
}

// NewPriorityScheduler creates a new priority scheduler
func NewPriorityScheduler() *PriorityScheduler {
	scheduler := &PriorityScheduler{
		queues: make(map[models.Priority]*taskQueue),
		tasks:  make(map[string]*models.Task),
	}
	
	// Initialize queues for all priority levels
	priorities := []models.Priority{
		models.PriorityLow,
		models.PriorityNormal,
		models.PriorityHigh,
		models.PriorityCritical,
	}
	
	for _, priority := range priorities {
		scheduler.queues[priority] = &taskQueue{
			items: make([]*taskItem, 0),
		}
	}
	
	return scheduler
}

// AddTask adds a task to the appropriate priority queue
func (ps *PriorityScheduler) AddTask(task *models.Task) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	
	// Store task reference
	ps.tasks[task.ID] = task
	
	// Add to priority queue
	queue, exists := ps.queues[task.Priority]
	if !exists {
		queue = &taskQueue{items: make([]*taskItem, 0)}
		ps.queues[task.Priority] = queue
	}
	
	queue.mu.Lock()
	defer queue.mu.Unlock()
	
	item := &taskItem{
		task:     task,
		priority: task.Priority,
	}
	
	heap.Push(queue, item)
	return nil
}

// GetNextTask retrieves the next task to be executed based on priority
func (ps *PriorityScheduler) GetNextTask() *models.Task {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	
	// First check all queues by priority (highest to lowest)
	// Collect all priorities and sort them in descending order
	var allPriorities []models.Priority
	for priority := range ps.queues {
		allPriorities = append(allPriorities, priority)
	}
	
	// Sort priorities in descending order (highest first)
	for i := 0; i < len(allPriorities); i++ {
		for j := i + 1; j < len(allPriorities); j++ {
			if allPriorities[i] < allPriorities[j] {
				allPriorities[i], allPriorities[j] = allPriorities[j], allPriorities[i]
			}
		}
	}
	
	for _, priority := range allPriorities {
		queue := ps.queues[priority]
		if queue == nil {
			continue
		}
		
		queue.mu.Lock()
		if queue.Len() > 0 {
			item := heap.Pop(queue).(*taskItem)
			queue.mu.Unlock()
			
			// Remove from tasks map
			delete(ps.tasks, item.task.ID)
			return item.task
		}
		queue.mu.Unlock()
	}
	
	return nil
}

// RemoveTask removes a task from the scheduler
func (ps *PriorityScheduler) RemoveTask(taskID string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	
	task, exists := ps.tasks[taskID]
	if !exists {
		return nil // Task not found, already removed
	}
	
	queue := ps.queues[task.Priority]
	queue.mu.Lock()
	defer queue.mu.Unlock()
	
	// Find and remove the task from the queue
	for i, item := range queue.items {
		if item.task.ID == taskID {
			heap.Remove(queue, i)
			break
		}
	}
	
	// Remove from tasks map
	delete(ps.tasks, taskID)
	return nil
}

// UpdateTaskPriority updates the priority of a task
func (ps *PriorityScheduler) UpdateTaskPriority(taskID string, newPriority models.Priority) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	
	task, exists := ps.tasks[taskID]
	if !exists {
		return nil // Task not found
	}
	
	oldPriority := task.Priority
	if oldPriority == newPriority {
		return nil // No change needed
	}
	
	// Remove from old queue
	oldQueue := ps.queues[oldPriority]
	oldQueue.mu.Lock()
	for i, item := range oldQueue.items {
		if item.task.ID == taskID {
			heap.Remove(oldQueue, i)
			break
		}
	}
	oldQueue.mu.Unlock()
	
	// Update task priority
	task.Priority = newPriority
	
	// Add to new queue
	newQueue, exists := ps.queues[newPriority]
	if !exists {
		newQueue = &taskQueue{items: make([]*taskItem, 0)}
		ps.queues[newPriority] = newQueue
	}
	
	newQueue.mu.Lock()
	item := &taskItem{
		task:     task,
		priority: newPriority,
	}
	heap.Push(newQueue, item)
	newQueue.mu.Unlock()
	
	return nil
}

// GetQueueLength returns the total number of tasks in all queues
func (ps *PriorityScheduler) GetQueueLength() int {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	
	return len(ps.tasks)
}

// GetQueueStats returns statistics about tasks in each priority queue
func (ps *PriorityScheduler) GetQueueStats() map[models.Priority]int {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	
	stats := make(map[models.Priority]int)
	
	for priority, queue := range ps.queues {
		queue.mu.Lock()
		stats[priority] = queue.Len()
		queue.mu.Unlock()
	}
	
	return stats
}

// Priority queue implementation for taskQueue

func (tq *taskQueue) Len() int { return len(tq.items) }

func (tq *taskQueue) Less(i, j int) bool {
	// Higher priority values come first, then FIFO for same priority
	if tq.items[i].priority == tq.items[j].priority {
		return tq.items[i].task.CreatedAt.Before(tq.items[j].task.CreatedAt)
	}
	return tq.items[i].priority > tq.items[j].priority
}

func (tq *taskQueue) Swap(i, j int) {
	tq.items[i], tq.items[j] = tq.items[j], tq.items[i]
	tq.items[i].index = i
	tq.items[j].index = j
}

func (tq *taskQueue) Push(x any) {
	item := x.(*taskItem)
	item.index = len(tq.items)
	tq.items = append(tq.items, item)
}

func (tq *taskQueue) Pop() any {
	old := tq.items
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	tq.items = old[0 : n-1]
	return item
}

// RetryScheduler handles task retry logic with exponential backoff
type RetryScheduler struct {
	retryQueue map[string]*retryItem
	config     models.RetryConfig
	mu         sync.RWMutex
}

type retryItem struct {
	task      *models.Task
	nextRetry time.Time
	delay     time.Duration
}

// NewRetryScheduler creates a new retry scheduler
func NewRetryScheduler(config models.RetryConfig) *RetryScheduler {
	return &RetryScheduler{
		retryQueue: make(map[string]*retryItem),
		config:     config,
	}
}

// AddFailedTask adds a failed task to the retry queue
func (rs *RetryScheduler) AddFailedTask(task *models.Task) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	
	if task.RetryCount >= task.MaxRetries {
		return nil // Max retries exceeded
	}
	
	// Calculate delay with exponential backoff
	delay := rs.config.InitialDelay
	for i := 0; i < task.RetryCount; i++ {
		delay = time.Duration(float64(delay) * rs.config.BackoffFactor)
	}
	
	if delay > rs.config.MaxDelay {
		delay = rs.config.MaxDelay
	}
	
	rs.retryQueue[task.ID] = &retryItem{
		task:      task,
		nextRetry: time.Now().Add(delay),
		delay:     delay,
	}
	
	return nil
}

// GetReadyTasks returns tasks that are ready to be retried
func (rs *RetryScheduler) GetReadyTasks() []*models.Task {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	
	now := time.Now()
	var readyTasks []*models.Task
	
	for taskID, item := range rs.retryQueue {
		if now.After(item.nextRetry) {
			item.task.RetryCount++
			item.task.Status = models.StatusPending
			readyTasks = append(readyTasks, item.task)
			delete(rs.retryQueue, taskID)
		}
	}
	
	return readyTasks
}

// RemoveTask removes a task from the retry queue
func (rs *RetryScheduler) RemoveTask(taskID string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	
	delete(rs.retryQueue, taskID)
}

// GetRetryQueueLength returns the number of tasks in the retry queue
func (rs *RetryScheduler) GetRetryQueueLength() int {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	
	return len(rs.retryQueue)
}