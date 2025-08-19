package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/liliang-cn/ollama-queue/internal/models"
)

// Client is a client for the Ollama Queue server.

type Client struct {
	addr       string
	httpClient *http.Client
}

// New creates a new Client.
func New(addr string) *Client {
	return &Client{
		addr:       addr,
		httpClient: &http.Client{},
	}
}

// ListTasks lists tasks in the queue.
func (c *Client) ListTasks(filter models.TaskFilter) ([]*models.Task, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/api/tasks", c.addr), nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	if len(filter.Status) > 0 {
		for _, status := range filter.Status {
			q.Add("status", string(status))
		}
	}
	if len(filter.Type) > 0 {
		for _, taskType := range filter.Type {
			q.Add("type", string(taskType))
		}
	}
	if len(filter.Priority) > 0 {
		for _, priority := range filter.Priority {
			q.Add("priority", fmt.Sprintf("%d", priority))
		}
	}
	if filter.Limit > 0 {
		q.Add("limit", fmt.Sprintf("%d", filter.Limit))
	}
	if filter.Offset > 0 {
		q.Add("offset", fmt.Sprintf("%d", filter.Offset))
	}
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tasks []*models.Task
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		return nil, err
	}

	return tasks, nil
}

// SubmitTask submits a new task to the queue.
func (c *Client) SubmitTask(task *models.Task) (string, error) {
	data, err := json.Marshal(task)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s/api/tasks", c.addr), bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		TaskID string `json:"task_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.TaskID, nil
}

// CancelTask cancels a task.
func (c *Client) CancelTask(taskID string) error {
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s/api/tasks/%s/cancel", c.addr, taskID), nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// UpdateTaskPriority updates the priority of a task.
func (c *Client) UpdateTaskPriority(taskID string, priority models.Priority) error {
	data, err := json.Marshal(map[string]models.Priority{"priority": priority})
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s/api/tasks/%s/priority", c.addr, taskID), bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// GetTask retrieves a task by ID.
func (c *Client) GetTask(taskID string) (*models.Task, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/api/tasks/%s", c.addr, taskID), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var task models.Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, err
	}

	return &task, nil
}

// GetQueueStats retrieves queue statistics.
func (c *Client) GetQueueStats() (*models.QueueStats, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/api/status", c.addr), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var stats models.QueueStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, err
	}

	return &stats, nil
}

// SubmitBatchTasks submits multiple tasks and waits for all to complete
func (c *Client) SubmitBatchTasks(tasks []*models.Task) ([]*models.TaskResult, error) {
	if len(tasks) == 0 {
		return []*models.TaskResult{}, nil
	}
	
	results := make([]*models.TaskResult, len(tasks))
	taskIDs := make([]string, len(tasks))
	
	// Submit all tasks
	for i, task := range tasks {
		taskID, err := c.SubmitTask(task)
		if err != nil {
			return nil, fmt.Errorf("failed to submit task %d: %w", i, err)
		}
		taskIDs[i] = taskID
	}
	
	// Poll for completion
	for {
		completed := 0
		for i, taskID := range taskIDs {
			if results[i] != nil {
				completed++
				continue
			}
			
			task, err := c.GetTask(taskID)
			if err != nil {
				continue // Skip errors and retry
			}
			
			if task.Status == models.StatusCompleted || task.Status == models.StatusFailed {
				results[i] = &models.TaskResult{
					TaskID:  taskID,
					Success: task.Status == models.StatusCompleted,
					Data:    task.Result,
					Error:   task.Error,
				}
				completed++
			}
		}
		
		if completed == len(taskIDs) {
			break
		}
		
		time.Sleep(500 * time.Millisecond)
	}
	
	return results, nil
}

// SubmitBatchTasksAsync submits multiple tasks and calls callback when all complete
func (c *Client) SubmitBatchTasksAsync(tasks []*models.Task, callback models.BatchCallback) ([]string, error) {
	if len(tasks) == 0 {
		go callback([]*models.TaskResult{})
		return []string{}, nil
	}
	
	taskIDs := make([]string, len(tasks))
	
	// Submit all tasks
	for i, task := range tasks {
		taskID, err := c.SubmitTask(task)
		if err != nil {
			return nil, fmt.Errorf("failed to submit task %d: %w", i, err)
		}
		taskIDs[i] = taskID
	}
	
	// Start monitoring in background
	go func() {
		results := make([]*models.TaskResult, len(tasks))
		completed := make([]bool, len(tasks))
		var completedCount int
		var mu sync.Mutex
		
		for {
			mu.Lock()
			currentCompleted := completedCount
			mu.Unlock()
			
			if currentCompleted == len(tasks) {
				callback(results)
				break
			}
			
			for i, taskID := range taskIDs {
				mu.Lock()
				if completed[i] {
					mu.Unlock()
					continue
				}
				mu.Unlock()
				
				task, err := c.GetTask(taskID)
				if err != nil {
					continue // Skip errors and retry
				}
				
				if task.Status == models.StatusCompleted || task.Status == models.StatusFailed {
					result := &models.TaskResult{
						TaskID:  taskID,
						Success: task.Status == models.StatusCompleted,
						Data:    task.Result,
						Error:   task.Error,
					}
					
					mu.Lock()
					if !completed[i] {
						results[i] = result
						completed[i] = true
						completedCount++
					}
					mu.Unlock()
				}
			}
			
			time.Sleep(500 * time.Millisecond)
		}
	}()
	
	return taskIDs, nil
}

// SubmitScheduledTask submits a task to be executed at a specific time
func (c *Client) SubmitScheduledTask(task *models.Task, scheduledAt time.Time) (string, error) {
	task.ScheduledAt = &scheduledAt
	return c.SubmitTask(task)
}

// SubmitScheduledBatchTasks submits multiple tasks with scheduled execution times
func (c *Client) SubmitScheduledBatchTasks(tasks []*models.Task, scheduledAt time.Time) ([]*models.TaskResult, error) {
	// Set scheduled time for all tasks
	for _, task := range tasks {
		task.ScheduledAt = &scheduledAt
	}
	return c.SubmitBatchTasks(tasks)
}

// SubmitScheduledBatchTasksAsync submits multiple scheduled tasks asynchronously
func (c *Client) SubmitScheduledBatchTasksAsync(tasks []*models.Task, scheduledAt time.Time, callback models.BatchCallback) ([]string, error) {
	// Set scheduled time for all tasks
	for _, task := range tasks {
		task.ScheduledAt = &scheduledAt
	}
	return c.SubmitBatchTasksAsync(tasks, callback)
}

// Cron Task Management Methods

// AddCronTask adds a new cron task
func (c *Client) AddCronTask(cronTask *models.CronTask) (string, error) {
	body, err := json.Marshal(cronTask)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s/api/cron", c.addr), bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var response struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}

	return response.ID, nil
}

// GetCronTask retrieves a cron task by ID
func (c *Client) GetCronTask(cronID string) (*models.CronTask, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/api/cron/%s", c.addr, cronID), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var cronTask models.CronTask
	if err := json.NewDecoder(resp.Body).Decode(&cronTask); err != nil {
		return nil, err
	}

	return &cronTask, nil
}

// UpdateCronTask updates an existing cron task
func (c *Client) UpdateCronTask(cronTask *models.CronTask) error {
	body, err := json.Marshal(cronTask)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", fmt.Sprintf("http://%s/api/cron/%s", c.addr, cronTask.ID), bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return nil
}

// RemoveCronTask removes a cron task
func (c *Client) RemoveCronTask(cronID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("http://%s/api/cron/%s", c.addr, cronID), nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return nil
}

// ListCronTasks lists cron tasks with optional filters
func (c *Client) ListCronTasks(filter models.CronFilter) ([]*models.CronTask, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/api/cron", c.addr), nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	if filter.Enabled != nil {
		q.Set("enabled", fmt.Sprintf("%t", *filter.Enabled))
	}
	if len(filter.Names) > 0 {
		for _, name := range filter.Names {
			q.Add("name", name)
		}
	}
	if filter.Limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", filter.Limit))
	}
	if filter.Offset > 0 {
		q.Set("offset", fmt.Sprintf("%d", filter.Offset))
	}
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var cronTasks []*models.CronTask
	if err := json.NewDecoder(resp.Body).Decode(&cronTasks); err != nil {
		return nil, err
	}

	return cronTasks, nil
}

// EnableCronTask enables a cron task
func (c *Client) EnableCronTask(cronID string) error {
	req, err := http.NewRequest("PUT", fmt.Sprintf("http://%s/api/cron/%s/enable", c.addr, cronID), nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return nil
}

// DisableCronTask disables a cron task
func (c *Client) DisableCronTask(cronID string) error {
	req, err := http.NewRequest("PUT", fmt.Sprintf("http://%s/api/cron/%s/disable", c.addr, cronID), nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return nil
}
