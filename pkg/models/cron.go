package models

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// CronTask represents a recurring task definition
type CronTask struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	CronExpr     string                 `json:"cron_expr"`
	Enabled      bool                   `json:"enabled"`
	TaskTemplate TaskTemplate           `json:"task_template"`
	NextRun      *time.Time            `json:"next_run,omitempty"`
	LastRun      *time.Time            `json:"last_run,omitempty"`
	RunCount     int                   `json:"run_count"`
	CreatedAt    time.Time             `json:"created_at"`
	UpdatedAt    time.Time             `json:"updated_at"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// TaskTemplate defines the template for tasks created by cron
type TaskTemplate struct {
	Type     TaskType           `json:"type"`
	Model    string             `json:"model"`
	Priority Priority           `json:"priority"`
	Payload  interface{}        `json:"payload"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

// CronExpression represents parsed cron fields
type CronExpression struct {
	Minute     CronField
	Hour       CronField
	DayOfMonth CronField
	Month      CronField
	DayOfWeek  CronField
}

// CronField represents a single cron field (minute, hour, etc.)
type CronField struct {
	Values []int
	All    bool
}

// CronFilter for querying cron tasks
type CronFilter struct {
	Enabled *bool     `json:"enabled,omitempty"`
	Names   []string  `json:"names,omitempty"`
	Limit   int       `json:"limit,omitempty"`
	Offset  int       `json:"offset,omitempty"`
}

// ParseCronExpression parses a cron expression
// Supports standard 5-field format: "* * * * *" (min hour day month dow)
// Examples: "0 9 * * 1-5" (9 AM on weekdays), "*/15 * * * *" (every 15 minutes)
func ParseCronExpression(expr string) (*CronExpression, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return nil, fmt.Errorf("cron expression must have 5 fields, got %d", len(fields))
	}

	cron := &CronExpression{}
	var err error

	// Parse each field
	if cron.Minute, err = parseCronField(fields[0], 0, 59); err != nil {
		return nil, fmt.Errorf("invalid minute field: %w", err)
	}
	if cron.Hour, err = parseCronField(fields[1], 0, 23); err != nil {
		return nil, fmt.Errorf("invalid hour field: %w", err)
	}
	if cron.DayOfMonth, err = parseCronField(fields[2], 1, 31); err != nil {
		return nil, fmt.Errorf("invalid day field: %w", err)
	}
	if cron.Month, err = parseCronField(fields[3], 1, 12); err != nil {
		return nil, fmt.Errorf("invalid month field: %w", err)
	}
	if cron.DayOfWeek, err = parseCronField(fields[4], 0, 6); err != nil {
		return nil, fmt.Errorf("invalid day of week field: %w", err)
	}

	return cron, nil
}

// parseCronField parses a single cron field
func parseCronField(field string, min, max int) (CronField, error) {
	if field == "*" {
		return CronField{All: true}, nil
	}

	var values []int

	// Handle step values (*/n)
	if strings.HasPrefix(field, "*/") {
		stepStr := strings.TrimPrefix(field, "*/")
		step, err := strconv.Atoi(stepStr)
		if err != nil || step <= 0 {
			return CronField{}, fmt.Errorf("invalid step value: %s", stepStr)
		}
		for i := min; i <= max; i += step {
			values = append(values, i)
		}
		return CronField{Values: values}, nil
	}

	// Handle comma-separated values
	parts := strings.Split(field, ",")
	for _, part := range parts {
		// Handle ranges (n-m)
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return CronField{}, fmt.Errorf("invalid range format: %s", part)
			}
			start, err1 := strconv.Atoi(rangeParts[0])
			end, err2 := strconv.Atoi(rangeParts[1])
			if err1 != nil || err2 != nil {
				return CronField{}, fmt.Errorf("invalid range values: %s", part)
			}
			if start < min || end > max || start > end {
				return CronField{}, fmt.Errorf("range out of bounds: %s (min=%d, max=%d)", part, min, max)
			}
			for i := start; i <= end; i++ {
				values = append(values, i)
			}
		} else {
			// Single value
			value, err := strconv.Atoi(part)
			if err != nil {
				return CronField{}, fmt.Errorf("invalid value: %s", part)
			}
			if value < min || value > max {
				return CronField{}, fmt.Errorf("value out of bounds: %d (min=%d, max=%d)", value, min, max)
			}
			values = append(values, value)
		}
	}

	return CronField{Values: values}, nil
}

// NextTime calculates the next execution time after the given time
func (ce *CronExpression) NextTime(after time.Time) time.Time {
	// Start from the minute after 'after'
	next := after.Add(time.Minute).Truncate(time.Minute)
	
	// Find next valid time
	for i := 0; i < 366*24*60; i++ { // Max 1 year ahead
		if ce.matches(next) {
			return next
		}
		next = next.Add(time.Minute)
	}
	
	// Fallback if no match found (shouldn't happen with valid expressions)
	return after.Add(time.Hour)
}

// matches checks if the given time matches the cron expression
func (ce *CronExpression) matches(t time.Time) bool {
	return ce.matchesField(ce.Minute, t.Minute()) &&
		ce.matchesField(ce.Hour, t.Hour()) &&
		ce.matchesField(ce.DayOfMonth, t.Day()) &&
		ce.matchesField(ce.Month, int(t.Month())) &&
		ce.matchesField(ce.DayOfWeek, int(t.Weekday()))
}

// matchesField checks if a value matches a cron field
func (ce *CronExpression) matchesField(field CronField, value int) bool {
	if field.All {
		return true
	}
	for _, v := range field.Values {
		if v == value {
			return true
		}
	}
	return false
}

// CreateTaskFromTemplate creates a new task from the cron task template
func (ct *CronTask) CreateTaskFromTemplate() *Task {
	task := &Task{
		Type:      ct.TaskTemplate.Type,
		Model:     ct.TaskTemplate.Model,
		Priority:  ct.TaskTemplate.Priority,
		Payload:   ct.TaskTemplate.Payload,
		Options:   ct.TaskTemplate.Options,
		Status:    StatusPending,
		CreatedAt: time.Now(),
	}
	
	// Add cron metadata
	if task.Options == nil {
		task.Options = make(map[string]interface{})
	}
	task.Options["cron_id"] = ct.ID
	task.Options["cron_name"] = ct.Name
	task.Options["cron_run_count"] = ct.RunCount + 1
	
	return task
}