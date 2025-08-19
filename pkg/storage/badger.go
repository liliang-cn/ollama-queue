package storage

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/liliang-cn/ollama-queue/pkg/models"
)

// Storage interface defines the operations for task storage
type Storage interface {
	Open() error
	Close() error
	SaveTask(task *models.Task) error
	LoadTask(taskID string) (*models.Task, error)
	DeleteTask(taskID string) error
	ListTasks(filter models.TaskFilter) ([]*models.Task, error)
	UpdateTaskStatus(taskID string, status models.TaskStatus) error
	GetTasksByStatus(status models.TaskStatus) ([]*models.Task, error)
	CleanupCompletedTasks(maxTasks int, olderThan time.Duration) error
	GetStats() (*models.QueueStats, error)
	
	// Cron task operations
	SaveCronTask(cronTask *models.CronTask) error
	LoadCronTask(cronID string) (*models.CronTask, error)
	DeleteCronTask(cronID string) error
	ListCronTasks(filter models.CronFilter) ([]*models.CronTask, error)
}

// BadgerStorage implements Storage using BadgerDB
type BadgerStorage struct {
	db   *badger.DB
	path string
}

// NewBadgerStorage creates a new BadgerDB storage instance
func NewBadgerStorage(storagePath string) *BadgerStorage {
	return &BadgerStorage{
		path: filepath.Clean(storagePath),
	}
}

// Open opens the BadgerDB database
func (s *BadgerStorage) Open() error {
	opts := badger.DefaultOptions(s.path).WithLoggingLevel(badger.WARNING)
	
	db, err := badger.Open(opts)
	if err != nil {
		return fmt.Errorf("failed to open badger database: %w", err)
	}
	
	s.db = db
	return nil
}

// Close closes the BadgerDB database
func (s *BadgerStorage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// SaveTask saves a task to the database
func (s *BadgerStorage) SaveTask(task *models.Task) error {
	return s.db.Update(func(txn *badger.Txn) error {
		taskData, err := json.Marshal(task)
		if err != nil {
			return fmt.Errorf("failed to marshal task: %w", err)
		}
		
		key := fmt.Sprintf("task:%s", task.ID)
		err = txn.Set([]byte(key), taskData)
		if err != nil {
			return fmt.Errorf("failed to save task: %w", err)
		}
		
		// Create index entries for efficient querying
		statusKey := fmt.Sprintf("status:%s:%s", task.Status, task.ID)
		err = txn.Set([]byte(statusKey), []byte(task.ID))
		if err != nil {
			return fmt.Errorf("failed to create status index: %w", err)
		}
		
		priorityKey := fmt.Sprintf("priority:%d:%s", task.Priority, task.ID)
		err = txn.Set([]byte(priorityKey), []byte(task.ID))
		if err != nil {
			return fmt.Errorf("failed to create priority index: %w", err)
		}
		
		return nil
	})
}

// LoadTask loads a task from the database
func (s *BadgerStorage) LoadTask(taskID string) (*models.Task, error) {
	var task models.Task
	
	err := s.db.View(func(txn *badger.Txn) error {
		key := fmt.Sprintf("task:%s", taskID)
		item, err := txn.Get([]byte(key))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return fmt.Errorf("task not found: %s", taskID)
			}
			return fmt.Errorf("failed to get task: %w", err)
		}
		
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &task)
		})
	})
	
	return &task, err
}

// DeleteTask deletes a task from the database
func (s *BadgerStorage) DeleteTask(taskID string) error {
	return s.db.Update(func(txn *badger.Txn) error {
		// First, get the task to obtain its status and priority for index cleanup
		key := fmt.Sprintf("task:%s", taskID)
		item, err := txn.Get([]byte(key))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return nil // Task already deleted
			}
			return fmt.Errorf("failed to get task for deletion: %w", err)
		}
		
		var task models.Task
		err = item.Value(func(val []byte) error {
			return json.Unmarshal(val, &task)
		})
		if err != nil {
			return fmt.Errorf("failed to unmarshal task for deletion: %w", err)
		}
		
		// Delete the task
		err = txn.Delete([]byte(key))
		if err != nil {
			return fmt.Errorf("failed to delete task: %w", err)
		}
		
		// Delete index entries
		statusKey := fmt.Sprintf("status:%s:%s", task.Status, taskID)
		txn.Delete([]byte(statusKey))
		
		priorityKey := fmt.Sprintf("priority:%d:%s", task.Priority, taskID)
		txn.Delete([]byte(priorityKey))
		
		return nil
	})
}

// UpdateTaskStatus updates the status of a task
func (s *BadgerStorage) UpdateTaskStatus(taskID string, newStatus models.TaskStatus) error {
	return s.db.Update(func(txn *badger.Txn) error {
		// Load existing task
		key := fmt.Sprintf("task:%s", taskID)
		item, err := txn.Get([]byte(key))
		if err != nil {
			return fmt.Errorf("task not found: %s", taskID)
		}
		
		var task models.Task
		err = item.Value(func(val []byte) error {
			return json.Unmarshal(val, &task)
		})
		if err != nil {
			return fmt.Errorf("failed to unmarshal task: %w", err)
		}
		
		oldStatus := task.Status
		task.Status = newStatus
		
		// Update timestamps
		now := time.Now()
		switch newStatus {
		case models.StatusRunning:
			task.StartedAt = &now
		case models.StatusCompleted, models.StatusFailed, models.StatusCancelled:
			task.CompletedAt = &now
		}
		
		// Save updated task
		taskData, err := json.Marshal(task)
		if err != nil {
			return fmt.Errorf("failed to marshal updated task: %w", err)
		}
		
		err = txn.Set([]byte(key), taskData)
		if err != nil {
			return fmt.Errorf("failed to save updated task: %w", err)
		}
		
		// Update status index
		oldStatusKey := fmt.Sprintf("status:%s:%s", oldStatus, taskID)
		txn.Delete([]byte(oldStatusKey))
		
		newStatusKey := fmt.Sprintf("status:%s:%s", newStatus, taskID)
		err = txn.Set([]byte(newStatusKey), []byte(taskID))
		if err != nil {
			return fmt.Errorf("failed to update status index: %w", err)
		}
		
		return nil
	})
}

// GetTasksByStatus retrieves tasks by their status
func (s *BadgerStorage) GetTasksByStatus(status models.TaskStatus) ([]*models.Task, error) {
	var tasks []*models.Task
	
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		
		prefix := []byte(fmt.Sprintf("status:%s:", status))
		
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			
			err := item.Value(func(taskIDBytes []byte) error {
				taskID := string(taskIDBytes)
				task, err := s.loadTaskInTxn(txn, taskID)
				if err != nil {
					return err
				}
				tasks = append(tasks, task)
				return nil
			})
			if err != nil {
				return err
			}
		}
		
		return nil
	})
	
	return tasks, err
}

// ListTasks retrieves tasks based on filter criteria
func (s *BadgerStorage) ListTasks(filter models.TaskFilter) ([]*models.Task, error) {
	var tasks []*models.Task
	
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()
		
		prefix := []byte("task:")
		count := 0
		
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			if filter.Offset > 0 && count < filter.Offset {
				count++
				continue
			}
			
			if filter.Limit > 0 && len(tasks) >= filter.Limit {
				break
			}
			
			item := it.Item()
			
			var task models.Task
			err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &task)
			})
			if err != nil {
				continue
			}
			
			// Apply filters
			if !s.matchesFilter(&task, filter) {
				count++
				continue
			}
			
			tasks = append(tasks, &task)
			count++
		}
		
		return nil
	})
	
	return tasks, err
}

// CleanupCompletedTasks removes old completed tasks
func (s *BadgerStorage) CleanupCompletedTasks(maxTasks int, olderThan time.Duration) error {
	cutoffTime := time.Now().Add(-olderThan)
	
	return s.db.Update(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()
		
		var toDelete []string
		completedCount := 0
		
		// Count completed tasks and find old ones to delete
		prefix := []byte("task:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			
			var task models.Task
			err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &task)
			})
			if err != nil {
				continue
			}
			
			if task.Status == models.StatusCompleted {
				completedCount++
				
				if task.CompletedAt != nil && task.CompletedAt.Before(cutoffTime) {
					toDelete = append(toDelete, task.ID)
				}
			}
		}
		
		// If we have too many completed tasks, delete oldest ones
		if completedCount > maxTasks {
			excessCount := completedCount - maxTasks
			if len(toDelete) < excessCount {
				// Get more completed tasks to delete (oldest first)
				// This is a simplified approach - in production, you might want to sort by completion time
			}
		}
		
		// Delete the tasks
		for _, taskID := range toDelete {
			err := s.deleteTaskInTxn(txn, taskID)
			if err != nil {
				return fmt.Errorf("failed to delete task %s: %w", taskID, err)
			}
		}
		
		return nil
	})
}

// GetStats retrieves queue statistics
func (s *BadgerStorage) GetStats() (*models.QueueStats, error) {
	stats := &models.QueueStats{
		QueuesByPriority: make(map[models.Priority]int),
	}
	
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()
		
		prefix := []byte("task:")
		
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			
			var task models.Task
			err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &task)
			})
			if err != nil {
				continue
			}
			
			// Count by status
			switch task.Status {
			case models.StatusPending:
				stats.PendingTasks++
			case models.StatusRunning:
				stats.RunningTasks++
			case models.StatusCompleted:
				stats.CompletedTasks++
			case models.StatusFailed:
				stats.FailedTasks++
			case models.StatusCancelled:
				stats.CancelledTasks++
			}
			
			// Count by priority for pending tasks
			if task.Status == models.StatusPending {
				stats.QueuesByPriority[task.Priority]++
			}
			
			stats.TotalTasks++
		}
		
		return nil
	})
	
	return stats, err
}

// SaveCronTask saves a cron task to the database
func (s *BadgerStorage) SaveCronTask(cronTask *models.CronTask) error {
	return s.db.Update(func(txn *badger.Txn) error {
		cronData, err := json.Marshal(cronTask)
		if err != nil {
			return fmt.Errorf("failed to marshal cron task: %w", err)
		}
		
		key := fmt.Sprintf("cron:%s", cronTask.ID)
		err = txn.Set([]byte(key), cronData)
		if err != nil {
			return fmt.Errorf("failed to save cron task: %w", err)
		}
		
		// Create index entries for efficient querying
		enabledKey := fmt.Sprintf("cron_enabled:%t:%s", cronTask.Enabled, cronTask.ID)
		err = txn.Set([]byte(enabledKey), []byte(cronTask.ID))
		if err != nil {
			return fmt.Errorf("failed to create enabled index: %w", err)
		}
		
		nameKey := fmt.Sprintf("cron_name:%s:%s", cronTask.Name, cronTask.ID)
		err = txn.Set([]byte(nameKey), []byte(cronTask.ID))
		if err != nil {
			return fmt.Errorf("failed to create name index: %w", err)
		}
		
		return nil
	})
}

// LoadCronTask loads a cron task from the database
func (s *BadgerStorage) LoadCronTask(cronID string) (*models.CronTask, error) {
	var cronTask models.CronTask
	
	err := s.db.View(func(txn *badger.Txn) error {
		key := fmt.Sprintf("cron:%s", cronID)
		item, err := txn.Get([]byte(key))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return fmt.Errorf("cron task not found: %s", cronID)
			}
			return fmt.Errorf("failed to get cron task: %w", err)
		}
		
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &cronTask)
		})
	})
	
	if err != nil {
		return nil, err
	}
	
	return &cronTask, nil
}

// DeleteCronTask deletes a cron task from the database
func (s *BadgerStorage) DeleteCronTask(cronID string) error {
	return s.db.Update(func(txn *badger.Txn) error {
		// First, get the cron task to obtain its properties for index cleanup
		key := fmt.Sprintf("cron:%s", cronID)
		item, err := txn.Get([]byte(key))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return nil // Cron task already deleted
			}
			return fmt.Errorf("failed to get cron task for deletion: %w", err)
		}
		
		var cronTask models.CronTask
		err = item.Value(func(val []byte) error {
			return json.Unmarshal(val, &cronTask)
		})
		if err != nil {
			return fmt.Errorf("failed to unmarshal cron task for deletion: %w", err)
		}
		
		// Delete the cron task
		err = txn.Delete([]byte(key))
		if err != nil {
			return fmt.Errorf("failed to delete cron task: %w", err)
		}
		
		// Delete index entries
		enabledKey := fmt.Sprintf("cron_enabled:%t:%s", cronTask.Enabled, cronID)
		txn.Delete([]byte(enabledKey))
		
		nameKey := fmt.Sprintf("cron_name:%s:%s", cronTask.Name, cronID)
		txn.Delete([]byte(nameKey))
		
		return nil
	})
}

// ListCronTasks retrieves cron tasks based on filter criteria
func (s *BadgerStorage) ListCronTasks(filter models.CronFilter) ([]*models.CronTask, error) {
	var cronTasks []*models.CronTask
	
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()
		
		prefix := []byte("cron:")
		count := 0
		
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			if filter.Offset > 0 && count < filter.Offset {
				count++
				continue
			}
			
			if filter.Limit > 0 && len(cronTasks) >= filter.Limit {
				break
			}
			
			item := it.Item()
			
			var cronTask models.CronTask
			err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &cronTask)
			})
			if err != nil {
				continue
			}
			
			// Apply filters
			if !s.matchesCronFilter(&cronTask, filter) {
				count++
				continue
			}
			
			cronTasks = append(cronTasks, &cronTask)
			count++
		}
		
		return nil
	})
	
	return cronTasks, err
}

// Helper methods

func (s *BadgerStorage) loadTaskInTxn(txn *badger.Txn, taskID string) (*models.Task, error) {
	key := fmt.Sprintf("task:%s", taskID)
	item, err := txn.Get([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	
	var task models.Task
	err = item.Value(func(val []byte) error {
		return json.Unmarshal(val, &task)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}
	
	return &task, nil
}

func (s *BadgerStorage) deleteTaskInTxn(txn *badger.Txn, taskID string) error {
	// Get task first to clean up indexes
	task, err := s.loadTaskInTxn(txn, taskID)
	if err != nil {
		return err
	}
	
	// Delete task
	taskKey := fmt.Sprintf("task:%s", taskID)
	err = txn.Delete([]byte(taskKey))
	if err != nil {
		return err
	}
	
	// Delete indexes
	statusKey := fmt.Sprintf("status:%s:%s", task.Status, taskID)
	txn.Delete([]byte(statusKey))
	
	priorityKey := fmt.Sprintf("priority:%d:%s", task.Priority, taskID)
	txn.Delete([]byte(priorityKey))
	
	return nil
}

func (s *BadgerStorage) matchesFilter(task *models.Task, filter models.TaskFilter) bool {
	// Check status filter
	if len(filter.Status) > 0 {
		found := false
		for _, status := range filter.Status {
			if task.Status == status {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check type filter
	if len(filter.Type) > 0 {
		found := false
		for _, taskType := range filter.Type {
			if task.Type == taskType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check priority filter
	if len(filter.Priority) > 0 {
		found := false
		for _, priority := range filter.Priority {
			if task.Priority == priority {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	return true
}

func (s *BadgerStorage) matchesCronFilter(cronTask *models.CronTask, filter models.CronFilter) bool {
	// Check enabled filter
	if filter.Enabled != nil && cronTask.Enabled != *filter.Enabled {
		return false
	}
	
	// Check names filter
	if len(filter.Names) > 0 {
		found := false
		for _, name := range filter.Names {
			if cronTask.Name == name {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	return true
}