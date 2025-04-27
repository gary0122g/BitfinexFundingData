package scheduler

import (
	"context"
	"time"
)

// Task represents a schedulable task
type Task interface {
	// Execute runs the task, accepting a context parameter to support cancellation and timeout
	Execute(ctx context.Context) error
	GetName() string
	GetPriority() int
	GetRetryPolicy() RetryPolicy
}

// RetryPolicy defines the retry strategy for tasks
type RetryPolicy struct {
	MaxRetries  int           // Maximum number of retry attempts
	BackoffBase time.Duration // Base backoff duration
}

// BaseTask provides a basic implementation of a task
type BaseTask struct {
	Name        string      // Task name
	Priority    int         // Task priority, higher numbers mean higher priority
	RetryPolicy RetryPolicy // Retry strategy
}

// GetName returns the task name
func (t *BaseTask) GetName() string {
	return t.Name
}

// GetPriority returns the task priority
func (t *BaseTask) GetPriority() int {
	return t.Priority
}

// GetRetryPolicy returns the task retry strategy
func (t *BaseTask) GetRetryPolicy() RetryPolicy {
	return t.RetryPolicy
}

// TaskScheduler defines the task scheduler interface
type TaskScheduler interface {
	// Schedule schedules a task, using context to support cancellation
	Schedule(ctx context.Context, task Task) error

	// ScheduleWithDelay schedules a task to be executed after a delay
	ScheduleWithDelay(ctx context.Context, task Task, delay time.Duration) error

	// ScheduleRecurring schedules a task to be executed periodically
	ScheduleRecurring(ctx context.Context, task Task, interval time.Duration) error

	// Cancel cancels a scheduled task
	Cancel(taskName string) error

	// Start starts the scheduler
	Start(ctx context.Context) error

	// Stop stops the scheduler
	Stop() error
}
