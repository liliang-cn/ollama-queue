package cmd

import (
	"fmt"
	"strings"

	"github.com/liliang-cn/ollama-queue/internal/models"
)

func parsePriority(s string) (models.Priority, error) {
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
		return 0, fmt.Errorf("invalid priority: %s", s)
	}
}