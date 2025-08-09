package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/liliang-cn/ollama-queue/internal/models"
)

func parsePriority(s string) (models.Priority, error) {
	// Try to parse as numeric value first
	if num, err := strconv.Atoi(s); err == nil {
		switch models.Priority(num) {
		case models.PriorityLow:
			return models.PriorityLow, nil
		case models.PriorityNormal:
			return models.PriorityNormal, nil
		case models.PriorityHigh:
			return models.PriorityHigh, nil
		case models.PriorityCritical:
			return models.PriorityCritical, nil
		default:
			return 0, fmt.Errorf("invalid priority value: %d (must be 1, 5, 10, or 15)", num)
		}
	}
	
	// Parse as string value
	switch strings.ToLower(s) {
	case "low":
		return models.PriorityLow, nil
	case "normal":
		return models.PriorityNormal, nil
	case "high":
		return models.PriorityHigh, nil
	case "critical":
		return models.PriorityCritical, nil
	default:
		return 0, fmt.Errorf("invalid priority: %s (must be 'low', 'normal', 'high', 'critical', 1, 5, 10, or 15)", s)
	}
}

// safeTaskIDShort safely truncates a task ID to 8 characters for display
func safeTaskIDShort(taskID string) string {
	if len(taskID) >= 8 {
		return taskID[:8]
	}
	return taskID
}