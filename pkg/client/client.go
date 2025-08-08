package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

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
