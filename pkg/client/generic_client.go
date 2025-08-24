package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	
	"github.com/liliang-cn/ollama-queue/pkg/models"
	"github.com/liliang-cn/ollama-queue/pkg/executor"
)

// Extended client methods for generic task support

// SubmitGenericTask submits a generic task to the queue
func (c *Client) SubmitGenericTask(task *models.GenericTask) (string, error) {
	body, err := json.Marshal(task)
	if err != nil {
		return "", err
	}
	
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s/api/generic-tasks", c.addr), bytes.NewBuffer(body))
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
		return "", fmt.Errorf("failed to submit generic task: %s", resp.Status)
	}
	
	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	
	return result["task_id"], nil
}

// GetGenericTask retrieves a generic task by ID
func (c *Client) GetGenericTask(taskID string) (*models.GenericTask, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/api/generic-tasks/%s", c.addr, taskID), nil)
	if err != nil {
		return nil, err
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get generic task: %s", resp.Status)
	}
	
	var task models.GenericTask
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, err
	}
	
	return &task, nil
}

// ListExecutors returns all available executors
func (c *Client) ListExecutors() ([]*executor.ExecutorMetadata, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/api/executors", c.addr), nil)
	if err != nil {
		return nil, err
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list executors: %s", resp.Status)
	}
	
	var executors []*executor.ExecutorMetadata
	if err := json.NewDecoder(resp.Body).Decode(&executors); err != nil {
		return nil, err
	}
	
	return executors, nil
}

// GetExecutorInfo returns detailed information about a specific executor
func (c *Client) GetExecutorInfo(executorType string) (*executor.ExecutorMetadata, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/api/executors/%s", c.addr, executorType), nil)
	if err != nil {
		return nil, err
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get executor info: %s", resp.Status)
	}
	
	var executor executor.ExecutorMetadata
	if err := json.NewDecoder(resp.Body).Decode(&executor); err != nil {
		return nil, err
	}
	
	return &executor, nil
}

// ListGenericTasks lists generic tasks with filtering
func (c *Client) ListGenericTasks(filter models.GenericTaskFilter) ([]*models.GenericTask, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/api/generic-tasks", c.addr), nil)
	if err != nil {
		return nil, err
	}
	
	// Add query parameters for filtering
	q := req.URL.Query()
	if len(filter.ExecutorTypes) > 0 {
		for _, et := range filter.ExecutorTypes {
			q.Add("executor_type", string(et))
		}
	}
	if len(filter.Actions) > 0 {
		for _, action := range filter.Actions {
			q.Add("action", string(action))
		}
	}
	if len(filter.Status) > 0 {
		for _, status := range filter.Status {
			q.Add("status", string(status))
		}
	}
	if filter.Limit > 0 {
		q.Add("limit", fmt.Sprintf("%d", filter.Limit))
	}
	req.URL.RawQuery = q.Encode()
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list generic tasks: %s", resp.Status)
	}
	
	var tasks []*models.GenericTask
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		return nil, err
	}
	
	return tasks, nil
}