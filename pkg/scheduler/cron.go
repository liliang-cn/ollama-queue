package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/liliang-cn/ollama-queue/internal/models"
	"github.com/liliang-cn/ollama-queue/pkg/storage"
)

// CronScheduler manages recurring tasks based on cron expressions
type CronScheduler struct {
	storage     storage.Storage
	cronTasks   map[string]*models.CronTask
	nextRuns    map[string]time.Time
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	ticker      *time.Ticker
	taskSubmitter TaskSubmitter
}

// TaskSubmitter interface for submitting tasks
type TaskSubmitter interface {
	SubmitTask(task *models.Task) (string, error)
}

// NewCronScheduler creates a new cron scheduler
func NewCronScheduler(storage storage.Storage, submitter TaskSubmitter) *CronScheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &CronScheduler{
		storage:       storage,
		cronTasks:     make(map[string]*models.CronTask),
		nextRuns:      make(map[string]time.Time),
		ctx:           ctx,
		cancel:        cancel,
		ticker:        time.NewTicker(time.Minute),
		taskSubmitter: submitter,
	}
}

// Start begins the cron scheduler
func (cs *CronScheduler) Start() error {
	// Load existing cron tasks from storage
	if err := cs.loadCronTasks(); err != nil {
		return fmt.Errorf("failed to load cron tasks: %w", err)
	}

	// Start the scheduler loop
	go cs.run()
	
	log.Printf("Cron scheduler started with %d tasks", len(cs.cronTasks))
	return nil
}

// Stop stops the cron scheduler
func (cs *CronScheduler) Stop() error {
	cs.cancel()
	cs.ticker.Stop()
	log.Println("Cron scheduler stopped")
	return nil
}

// AddCronTask adds a new cron task
func (cs *CronScheduler) AddCronTask(cronTask *models.CronTask) error {
	// Validate cron expression
	cronExpr, err := models.ParseCronExpression(cronTask.CronExpr)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	// Set defaults
	if cronTask.ID == "" {
		cronTask.ID = uuid.New().String()
	}
	
	now := time.Now()
	if cronTask.CreatedAt.IsZero() {
		cronTask.CreatedAt = now
	}
	cronTask.UpdatedAt = now

	// Calculate next run time
	nextRun := cronExpr.NextTime(now)
	cronTask.NextRun = &nextRun

	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Save to storage
	if err := cs.saveCronTask(cronTask); err != nil {
		return fmt.Errorf("failed to save cron task: %w", err)
	}

	// Add to memory
	cs.cronTasks[cronTask.ID] = cronTask
	cs.nextRuns[cronTask.ID] = nextRun

	log.Printf("Added cron task %s (%s) with expression %s, next run: %s", 
		cronTask.ID, cronTask.Name, cronTask.CronExpr, nextRun.Format(time.RFC3339))

	return nil
}

// UpdateCronTask updates an existing cron task
func (cs *CronScheduler) UpdateCronTask(cronTask *models.CronTask) error {
	// Validate cron expression
	cronExpr, err := models.ParseCronExpression(cronTask.CronExpr)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Check if task exists
	existing, exists := cs.cronTasks[cronTask.ID]
	if !exists {
		return fmt.Errorf("cron task %s not found", cronTask.ID)
	}

	// Update fields
	cronTask.CreatedAt = existing.CreatedAt
	cronTask.UpdatedAt = time.Now()
	cronTask.RunCount = existing.RunCount

	// Recalculate next run time if expression or enabled status changed
	if existing.CronExpr != cronTask.CronExpr || existing.Enabled != cronTask.Enabled {
		if cronTask.Enabled {
			nextRun := cronExpr.NextTime(time.Now())
			cronTask.NextRun = &nextRun
			cs.nextRuns[cronTask.ID] = nextRun
		} else {
			cronTask.NextRun = nil
			delete(cs.nextRuns, cronTask.ID)
		}
	}

	// Save to storage
	if err := cs.saveCronTask(cronTask); err != nil {
		return fmt.Errorf("failed to update cron task: %w", err)
	}

	// Update in memory
	cs.cronTasks[cronTask.ID] = cronTask

	log.Printf("Updated cron task %s (%s)", cronTask.ID, cronTask.Name)
	return nil
}

// RemoveCronTask removes a cron task
func (cs *CronScheduler) RemoveCronTask(cronID string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Check if task exists
	if _, exists := cs.cronTasks[cronID]; !exists {
		return fmt.Errorf("cron task %s not found", cronID)
	}

	// Remove from storage
	if err := cs.deleteCronTask(cronID); err != nil {
		return fmt.Errorf("failed to delete cron task: %w", err)
	}

	// Remove from memory
	delete(cs.cronTasks, cronID)
	delete(cs.nextRuns, cronID)

	log.Printf("Removed cron task %s", cronID)
	return nil
}

// GetCronTask retrieves a cron task by ID
func (cs *CronScheduler) GetCronTask(cronID string) (*models.CronTask, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	cronTask, exists := cs.cronTasks[cronID]
	if !exists {
		return nil, fmt.Errorf("cron task %s not found", cronID)
	}

	// Return a copy
	taskCopy := *cronTask
	return &taskCopy, nil
}

// ListCronTasks lists cron tasks with optional filters
func (cs *CronScheduler) ListCronTasks(filter models.CronFilter) ([]*models.CronTask, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	var result []*models.CronTask

	for _, cronTask := range cs.cronTasks {
		// Apply filters
		if filter.Enabled != nil && cronTask.Enabled != *filter.Enabled {
			continue
		}

		if len(filter.Names) > 0 {
			found := false
			for _, name := range filter.Names {
				if cronTask.Name == name {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Create copy and add to result
		taskCopy := *cronTask
		result = append(result, &taskCopy)
	}

	// Apply limit and offset
	if filter.Offset > 0 {
		if filter.Offset >= len(result) {
			return []*models.CronTask{}, nil
		}
		result = result[filter.Offset:]
	}

	if filter.Limit > 0 && filter.Limit < len(result) {
		result = result[:filter.Limit]
	}

	return result, nil
}

// EnableCronTask enables a cron task
func (cs *CronScheduler) EnableCronTask(cronID string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cronTask, exists := cs.cronTasks[cronID]
	if !exists {
		return fmt.Errorf("cron task %s not found", cronID)
	}

	if cronTask.Enabled {
		return nil // Already enabled
	}

	// Parse expression and calculate next run
	cronExpr, err := models.ParseCronExpression(cronTask.CronExpr)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	nextRun := cronExpr.NextTime(time.Now())
	cronTask.Enabled = true
	cronTask.NextRun = &nextRun
	cronTask.UpdatedAt = time.Now()

	// Save to storage
	if err := cs.saveCronTask(cronTask); err != nil {
		return fmt.Errorf("failed to save cron task: %w", err)
	}

	// Update memory
	cs.nextRuns[cronID] = nextRun

	log.Printf("Enabled cron task %s, next run: %s", cronID, nextRun.Format(time.RFC3339))
	return nil
}

// DisableCronTask disables a cron task
func (cs *CronScheduler) DisableCronTask(cronID string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cronTask, exists := cs.cronTasks[cronID]
	if !exists {
		return fmt.Errorf("cron task %s not found", cronID)
	}

	if !cronTask.Enabled {
		return nil // Already disabled
	}

	cronTask.Enabled = false
	cronTask.NextRun = nil
	cronTask.UpdatedAt = time.Now()

	// Save to storage
	if err := cs.saveCronTask(cronTask); err != nil {
		return fmt.Errorf("failed to save cron task: %w", err)
	}

	// Update memory
	delete(cs.nextRuns, cronID)

	log.Printf("Disabled cron task %s", cronID)
	return nil
}

// GetNextRuns returns the next run times for all active cron tasks
func (cs *CronScheduler) GetNextRuns() map[string]time.Time {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	result := make(map[string]time.Time)
	for id, nextRun := range cs.nextRuns {
		result[id] = nextRun
	}
	return result
}

// Private methods

func (cs *CronScheduler) run() {
	for {
		select {
		case <-cs.ctx.Done():
			return
		case <-cs.ticker.C:
			cs.checkAndRunTasks()
		}
	}
}

func (cs *CronScheduler) checkAndRunTasks() {
	now := time.Now()
	cs.mu.Lock()
	defer cs.mu.Unlock()

	for cronID, nextRun := range cs.nextRuns {
		if now.After(nextRun) || now.Equal(nextRun) {
			cronTask := cs.cronTasks[cronID]
			if cronTask == nil || !cronTask.Enabled {
				continue
			}

			// Create and submit task
			task := cronTask.CreateTaskFromTemplate()
			taskID, err := cs.taskSubmitter.SubmitTask(task)
			if err != nil {
				log.Printf("Failed to submit task from cron %s (%s): %v", cronID, cronTask.Name, err)
				continue
			}

			log.Printf("Submitted task %s from cron %s (%s)", taskID, cronID, cronTask.Name)

			// Update cron task
			cronTask.RunCount++
			cronTask.LastRun = &now
			cronTask.UpdatedAt = now

			// Calculate next run
			cronExpr, err := models.ParseCronExpression(cronTask.CronExpr)
			if err != nil {
				log.Printf("Error parsing cron expression for %s: %v", cronID, err)
				continue
			}

			nextRun := cronExpr.NextTime(now)
			cronTask.NextRun = &nextRun
			cs.nextRuns[cronID] = nextRun

			// Save updated cron task
			if err := cs.saveCronTask(cronTask); err != nil {
				log.Printf("Failed to save updated cron task %s: %v", cronID, err)
			}

			log.Printf("Cron task %s (%s) completed run #%d, next run: %s", 
				cronID, cronTask.Name, cronTask.RunCount, nextRun.Format(time.RFC3339))
		}
	}
}

func (cs *CronScheduler) loadCronTasks() error {
	// This would typically load from storage
	// For now, we'll implement a simple approach
	cronTasks, err := cs.listCronTasksFromStorage()
	if err != nil {
		return err
	}

	now := time.Now()
	for _, cronTask := range cronTasks {
		cs.cronTasks[cronTask.ID] = cronTask

		if cronTask.Enabled {
			// Parse expression and calculate next run
			cronExpr, err := models.ParseCronExpression(cronTask.CronExpr)
			if err != nil {
				log.Printf("Invalid cron expression for task %s: %v", cronTask.ID, err)
				continue
			}

			var nextRun time.Time
			if cronTask.NextRun != nil && cronTask.NextRun.After(now) {
				nextRun = *cronTask.NextRun
			} else {
				nextRun = cronExpr.NextTime(now)
				cronTask.NextRun = &nextRun
				cronTask.UpdatedAt = now
				cs.saveCronTask(cronTask) // Update next run time
			}

			cs.nextRuns[cronTask.ID] = nextRun
		}
	}

	return nil
}

// Storage operations - these would use the actual storage implementation
func (cs *CronScheduler) saveCronTask(cronTask *models.CronTask) error {
	return cs.storage.SaveCronTask(cronTask)
}

func (cs *CronScheduler) deleteCronTask(cronID string) error {
	return cs.storage.DeleteCronTask(cronID)
}

func (cs *CronScheduler) listCronTasksFromStorage() ([]*models.CronTask, error) {
	return cs.storage.ListCronTasks(models.CronFilter{})
}