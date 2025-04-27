package scheduler

import (
	"context"
	"sync"
	"time"
)

// Scheduler implements the TaskScheduler interface
type Scheduler struct {
	workers      int
	queueSize    int
	taskQueue    chan Task
	periodicTask map[string]*PeriodicTask
	mu           sync.Mutex
	wg           sync.WaitGroup
	quit         chan struct{}
}

// NewScheduler creates a new task scheduler
func NewScheduler(workers, queueSize int) *Scheduler {
	return &Scheduler{
		workers:      workers,
		queueSize:    queueSize,
		taskQueue:    make(chan Task, queueSize),
		periodicTask: make(map[string]*PeriodicTask),
		quit:         make(chan struct{}),
	}
}

// Start launches the scheduler
func (s *Scheduler) Start() {
	// Start workers
	for i := 0; i < s.workers; i++ {
		s.wg.Add(1)
		go s.worker()
	}
}

// worker processes tasks from the task queue
func (s *Scheduler) worker() {
	defer s.wg.Done()

	for {
		select {
		case task := <-s.taskQueue:
			// Execute task
			ctx := context.Background()
			err := task.Execute(ctx)

			// If task execution fails and there's a retry policy, handle retry logic here
			if err != nil {
				policy := task.GetRetryPolicy()
				if policy.MaxRetries > 0 {
					// Actual retry logic can be added here
				}
			}
		case <-s.quit:
			return
		}
	}
}

// SubmitTask submits a task to the scheduler
func (s *Scheduler) SubmitTask(task Task) {
	select {
	case s.taskQueue <- task:
		// Task successfully submitted
	default:
		// Queue is full, can add handling logic here
	}
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	close(s.quit)
	s.wg.Wait()
}

// PeriodicTask represents a task that runs periodically
type PeriodicTask struct {
	BaseTask
	interval time.Duration
	lastRun  time.Time
	runFunc  func(ctx context.Context) error
	mu       sync.Mutex
}

// NewPeriodicTask creates a new periodic task
func (s *Scheduler) NewPeriodicTask(name string, interval time.Duration, runFunc func(ctx context.Context) error, priority int) *PeriodicTask {
	task := &PeriodicTask{
		BaseTask: BaseTask{
			Name:     name,
			Priority: priority,
			RetryPolicy: RetryPolicy{
				MaxRetries:  3,
				BackoffBase: 500 * time.Millisecond,
			},
		},
		interval: interval,
		lastRun:  time.Now(),
		runFunc:  runFunc,
	}

	s.mu.Lock()
	s.periodicTask[name] = task
	s.mu.Unlock()

	return task
}

// Execute runs the periodic task
func (p *PeriodicTask) Execute(ctx context.Context) error {
	p.mu.Lock()
	p.lastRun = time.Now()
	p.mu.Unlock()

	return p.runFunc(ctx)
}

// ShouldRun checks if the task should be executed
func (p *PeriodicTask) ShouldRun() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	return time.Since(p.lastRun) >= p.interval
}

// Schedule implements the TaskScheduler interface
func (s *Scheduler) Schedule(ctx context.Context, task Task) error {
	s.SubmitTask(task)
	return nil
}

// ScheduleWithDelay implements the TaskScheduler interface
func (s *Scheduler) ScheduleWithDelay(ctx context.Context, task Task, delay time.Duration) error {
	go func() {
		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
			s.SubmitTask(task)
		case <-ctx.Done():
			timer.Stop()
			return
		}
	}()
	return nil
}

// ScheduleRecurring implements the TaskScheduler interface
func (s *Scheduler) ScheduleRecurring(ctx context.Context, task Task, interval time.Duration) error {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.SubmitTask(task)
			case <-ctx.Done():
				return
			case <-s.quit:
				return
			}
		}
	}()
	return nil
}

// Cancel implements the TaskScheduler interface
func (s *Scheduler) Cancel(taskName string) error {
	// Logic for canceling tasks can be implemented here
	return nil
}

// StartWithContext implements the Start method of the TaskScheduler interface, but accepts a context parameter
func (s *Scheduler) StartWithContext(ctx context.Context) error {
	s.Start()
	return nil
}
