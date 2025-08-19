package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseCronExpression(t *testing.T) {
	tests := []struct {
		name        string
		expression  string
		shouldError bool
		description string
	}{
		{
			name:        "valid_every_minute",
			expression:  "* * * * *",
			shouldError: false,
			description: "Every minute",
		},
		{
			name:        "valid_hourly",
			expression:  "0 * * * *",
			shouldError: false,
			description: "Every hour",
		},
		{
			name:        "valid_daily",
			expression:  "0 9 * * *",
			shouldError: false,
			description: "Daily at 9 AM",
		},
		{
			name:        "valid_weekdays",
			expression:  "0 9 * * 1-5",
			shouldError: false,
			description: "Weekdays at 9 AM",
		},
		{
			name:        "valid_every_15_minutes",
			expression:  "*/15 * * * *",
			shouldError: false,
			description: "Every 15 minutes",
		},
		{
			name:        "valid_specific_days",
			expression:  "0 12 * * 1,3,5",
			shouldError: false,
			description: "Monday, Wednesday, Friday at noon",
		},
		{
			name:        "invalid_too_few_fields",
			expression:  "0 9 * *",
			shouldError: true,
			description: "Only 4 fields",
		},
		{
			name:        "invalid_too_many_fields",
			expression:  "0 9 * * * *",
			shouldError: true,
			description: "6 fields",
		},
		{
			name:        "invalid_minute_range",
			expression:  "60 * * * *",
			shouldError: true,
			description: "Minute out of range",
		},
		{
			name:        "invalid_hour_range",
			expression:  "0 24 * * *",
			shouldError: true,
			description: "Hour out of range",
		},
		{
			name:        "invalid_day_range",
			expression:  "0 0 32 * *",
			shouldError: true,
			description: "Day out of range",
		},
		{
			name:        "invalid_month_range",
			expression:  "0 0 1 13 *",
			shouldError: true,
			description: "Month out of range",
		},
		{
			name:        "invalid_dow_range",
			expression:  "0 0 * * 7",
			shouldError: true,
			description: "Day of week out of range",
		},
		{
			name:        "invalid_step_format",
			expression:  "*/abc * * * *",
			shouldError: true,
			description: "Invalid step format",
		},
		{
			name:        "invalid_range_format",
			expression:  "0 9-25 * * *",
			shouldError: true,
			description: "Invalid hour range",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cronExpr, err := ParseCronExpression(tt.expression)
			
			if tt.shouldError {
				assert.Error(t, err, "Expected error for expression: %s", tt.expression)
				assert.Nil(t, cronExpr, "CronExpression should be nil on error")
			} else {
				assert.NoError(t, err, "Unexpected error for expression: %s", tt.expression)
				assert.NotNil(t, cronExpr, "CronExpression should not be nil")
			}
		})
	}
}

func TestCronExpressionNextTime(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		baseTime   string
		expected   []string
	}{
		{
			name:       "every_minute",
			expression: "* * * * *",
			baseTime:   "2025-08-19 10:30:00",
			expected: []string{
				"2025-08-19 10:31:00",
				"2025-08-19 10:32:00",
				"2025-08-19 10:33:00",
			},
		},
		{
			name:       "hourly_at_zero",
			expression: "0 * * * *",
			baseTime:   "2025-08-19 10:30:00",
			expected: []string{
				"2025-08-19 11:00:00",
				"2025-08-19 12:00:00",
				"2025-08-19 13:00:00",
			},
		},
		{
			name:       "daily_at_9am",
			expression: "0 9 * * *",
			baseTime:   "2025-08-19 10:30:00",
			expected: []string{
				"2025-08-20 09:00:00",
				"2025-08-21 09:00:00",
				"2025-08-22 09:00:00",
			},
		},
		{
			name:       "weekdays_9am",
			expression: "0 9 * * 1-5",
			baseTime:   "2025-08-19 10:30:00", // Tuesday
			expected: []string{
				"2025-08-20 09:00:00", // Wednesday
				"2025-08-21 09:00:00", // Thursday
				"2025-08-22 09:00:00", // Friday
			},
		},
		{
			name:       "every_15_minutes",
			expression: "*/15 * * * *",
			baseTime:   "2025-08-19 10:32:00",
			expected: []string{
				"2025-08-19 10:45:00",
				"2025-08-19 11:00:00",
				"2025-08-19 11:15:00",
			},
		},
		{
			name:       "specific_days_mon_wed_fri",
			expression: "0 12 * * 1,3,5",
			baseTime:   "2025-08-19 13:00:00", // Tuesday
			expected: []string{
				"2025-08-20 12:00:00", // Wednesday
				"2025-08-22 12:00:00", // Friday
				"2025-08-25 12:00:00", // Monday
			},
		},
		{
			name:       "first_day_of_month",
			expression: "0 0 1 * *",
			baseTime:   "2025-08-19 10:30:00",
			expected: []string{
				"2025-09-01 00:00:00",
				"2025-10-01 00:00:00",
				"2025-11-01 00:00:00",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cronExpr, err := ParseCronExpression(tt.expression)
			assert.NoError(t, err)

			baseTime, err := time.Parse("2006-01-02 15:04:05", tt.baseTime)
			assert.NoError(t, err)

			current := baseTime
			for i, expectedStr := range tt.expected {
				nextTime := cronExpr.NextTime(current)
				expected, err := time.Parse("2006-01-02 15:04:05", expectedStr)
				assert.NoError(t, err)

				assert.Equal(t, expected, nextTime, "Next time %d mismatch for expression %s", i+1, tt.expression)
				current = nextTime
			}
		})
	}
}

func TestCronExpressionMatches(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		testTime   string
		shouldMatch bool
	}{
		{
			name:        "every_minute_matches",
			expression:  "* * * * *",
			testTime:    "2025-08-19 10:30:00",
			shouldMatch: true,
		},
		{
			name:        "hourly_matches",
			expression:  "0 * * * *",
			testTime:    "2025-08-19 10:00:00",
			shouldMatch: true,
		},
		{
			name:        "hourly_no_match",
			expression:  "0 * * * *",
			testTime:    "2025-08-19 10:30:00",
			shouldMatch: false,
		},
		{
			name:        "daily_9am_matches",
			expression:  "0 9 * * *",
			testTime:    "2025-08-19 09:00:00",
			shouldMatch: true,
		},
		{
			name:        "daily_9am_no_match",
			expression:  "0 9 * * *",
			testTime:    "2025-08-19 10:00:00",
			shouldMatch: false,
		},
		{
			name:        "weekdays_matches_monday",
			expression:  "0 9 * * 1-5",
			testTime:    "2025-08-18 09:00:00", // Monday
			shouldMatch: true,
		},
		{
			name:        "weekdays_no_match_sunday",
			expression:  "0 9 * * 1-5",
			testTime:    "2025-08-17 09:00:00", // Sunday
			shouldMatch: false,
		},
		{
			name:        "every_15min_matches",
			expression:  "*/15 * * * *",
			testTime:    "2025-08-19 10:15:00",
			shouldMatch: true,
		},
		{
			name:        "every_15min_no_match",
			expression:  "*/15 * * * *",
			testTime:    "2025-08-19 10:13:00",
			shouldMatch: false,
		},
		{
			name:        "specific_days_matches_wednesday",
			expression:  "0 12 * * 1,3,5",
			testTime:    "2025-08-20 12:00:00", // Wednesday
			shouldMatch: true,
		},
		{
			name:        "specific_days_no_match_tuesday",
			expression:  "0 12 * * 1,3,5",
			testTime:    "2025-08-19 12:00:00", // Tuesday
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cronExpr, err := ParseCronExpression(tt.expression)
			assert.NoError(t, err)

			testTime, err := time.Parse("2006-01-02 15:04:05", tt.testTime)
			assert.NoError(t, err)

			matches := cronExpr.matches(testTime)
			assert.Equal(t, tt.shouldMatch, matches, "Match result mismatch for expression %s at time %s", tt.expression, tt.testTime)
		})
	}
}

func TestCreateTaskFromTemplate(t *testing.T) {
	cronTask := &CronTask{
		ID:       "test-cron-123",
		Name:     "Test Cron Task",
		RunCount: 5,
		TaskTemplate: TaskTemplate{
			Type:     TaskTypeChat,
			Model:    "qwen3",
			Priority: PriorityHigh,
			Payload: []ChatMessage{
				{Role: "user", Content: "Test message"},
			},
			Options: map[string]interface{}{
				"stream": false,
			},
		},
	}

	task := cronTask.CreateTaskFromTemplate()

	// Verify basic fields
	assert.Equal(t, TaskTypeChat, task.Type)
	assert.Equal(t, "qwen3", task.Model)
	assert.Equal(t, PriorityHigh, task.Priority)
	assert.Equal(t, StatusPending, task.Status)
	assert.NotEmpty(t, task.CreatedAt)

	// Verify payload
	assert.NotNil(t, task.Payload)
	messages, ok := task.Payload.([]ChatMessage)
	assert.True(t, ok)
	assert.Len(t, messages, 1)
	assert.Equal(t, "user", messages[0].Role)
	assert.Equal(t, "Test message", messages[0].Content)

	// Verify options with cron metadata
	assert.NotNil(t, task.Options)
	assert.Equal(t, false, task.Options["stream"])
	assert.Equal(t, "test-cron-123", task.Options["cron_id"])
	assert.Equal(t, "Test Cron Task", task.Options["cron_name"])
	assert.Equal(t, 6, task.Options["cron_run_count"]) // RunCount + 1
}

func TestCronTaskValidation(t *testing.T) {
	t.Run("valid_chat_task", func(t *testing.T) {
		cronTask := &CronTask{
			Name:     "Valid Chat Task",
			CronExpr: "0 9 * * *",
			Enabled:  true,
			TaskTemplate: TaskTemplate{
				Type:     TaskTypeChat,
				Model:    "qwen3",
				Priority: PriorityNormal,
				Payload: []ChatMessage{
					{Role: "user", Content: "Hello"},
				},
			},
		}

		// This should not panic or fail when creating a task
		task := cronTask.CreateTaskFromTemplate()
		assert.NotNil(t, task)
		assert.Equal(t, TaskTypeChat, task.Type)
	})

	t.Run("valid_generate_task", func(t *testing.T) {
		cronTask := &CronTask{
			Name:     "Valid Generate Task",
			CronExpr: "*/5 * * * *",
			Enabled:  true,
			TaskTemplate: TaskTemplate{
				Type:     TaskTypeGenerate,
				Model:    "qwen3",
				Priority: PriorityHigh,
				Payload: map[string]interface{}{
					"prompt": "Generate something",
				},
			},
		}

		task := cronTask.CreateTaskFromTemplate()
		assert.NotNil(t, task)
		assert.Equal(t, TaskTypeGenerate, task.Type)
	})

	t.Run("valid_embed_task", func(t *testing.T) {
		cronTask := &CronTask{
			Name:     "Valid Embed Task",
			CronExpr: "0 */2 * * *",
			Enabled:  true,
			TaskTemplate: TaskTemplate{
				Type:     TaskTypeEmbed,
				Model:    "qwen3",
				Priority: PriorityLow,
				Payload: map[string]interface{}{
					"input": "Text to embed",
				},
			},
		}

		task := cronTask.CreateTaskFromTemplate()
		assert.NotNil(t, task)
		assert.Equal(t, TaskTypeEmbed, task.Type)
	})
}

func TestCronFieldParsing(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		min      int
		max      int
		expected []int
		hasError bool
	}{
		{
			name:     "wildcard",
			field:    "*",
			min:      0,
			max:      59,
			expected: nil, // All=true
			hasError: false,
		},
		{
			name:     "single_value",
			field:    "15",
			min:      0,
			max:      59,
			expected: []int{15},
			hasError: false,
		},
		{
			name:     "range",
			field:    "1-5",
			min:      0,
			max:      6,
			expected: []int{1, 2, 3, 4, 5},
			hasError: false,
		},
		{
			name:     "list",
			field:    "1,3,5",
			min:      0,
			max:      6,
			expected: []int{1, 3, 5},
			hasError: false,
		},
		{
			name:     "step_values",
			field:    "*/15",
			min:      0,
			max:      59,
			expected: []int{0, 15, 30, 45},
			hasError: false,
		},
		{
			name:     "value_out_of_range",
			field:    "60",
			min:      0,
			max:      59,
			expected: nil,
			hasError: true,
		},
		{
			name:     "invalid_range",
			field:    "5-1",
			min:      0,
			max:      59,
			expected: nil,
			hasError: true,
		},
		{
			name:     "invalid_step",
			field:    "*/0",
			min:      0,
			max:      59,
			expected: nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field, err := parseCronField(tt.field, tt.min, tt.max)

			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.expected == nil {
					assert.True(t, field.All)
				} else {
					assert.False(t, field.All)
					assert.Equal(t, tt.expected, field.Values)
				}
			}
		})
	}
}