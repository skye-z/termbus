package command

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/termbus/termbus/internal/eventbus"
	"github.com/termbus/termbus/internal/logger"
	"go.uber.org/zap"
)

// ScheduledTask 定时任务
type ScheduledTask struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Command    string    `json:"command"`
	Schedule   string    `json:"schedule"`
	SessionIDs []string  `json:"session_ids"`
	GroupName  string    `json:"group_name"`
	Enabled    bool      `json:"enabled"`
	CreatedAt  time.Time `json:"created_at"`
	NextRun    time.Time `json:"next_run"`
	LastRun    time.Time `json:"last_run"`
	LastStatus string    `json:"last_status"`
}

// TaskScheduler 任务调度器
type TaskScheduler struct {
	tasks    map[string]*ScheduledTask
	executor *BatchExecutor
	eventBus *eventbus.Manager
	ticker   *time.Ticker
	mu       sync.RWMutex
	running  bool
}

// NewTaskScheduler 创建任务调度器
func NewTaskScheduler(executor *BatchExecutor, eventBus *eventbus.Manager) *TaskScheduler {
	return &TaskScheduler{
		tasks:    make(map[string]*ScheduledTask),
		executor: executor,
		eventBus: eventBus,
		running:  false,
	}
}

// Add 添加任务
func (s *TaskScheduler) Add(task *ScheduledTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tasks[task.ID]; exists {
		return fmt.Errorf("task already exists: %s", task.ID)
	}

	if task.ID == "" {
		task.ID = fmt.Sprintf("task_%d", time.Now().UnixNano())
	}

	task.CreatedAt = time.Now()
	task.NextRun = s.calculateNextRun(task.Schedule)
	s.tasks[task.ID] = task

	logger.GetLogger().Info("scheduled task added",
		zap.String("id", task.ID),
		zap.String("name", task.Name),
		zap.String("schedule", task.Schedule),
	)

	return s.Save()
}

// Remove 删除任务
func (s *TaskScheduler) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tasks[id]; !exists {
		return fmt.Errorf("task not found: %s", id)
	}

	delete(s.tasks, id)

	logger.GetLogger().Info("scheduled task removed", zap.String("id", id))

	return s.Save()
}

// List 列出所有任务
func (s *TaskScheduler) List() []*ScheduledTask {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*ScheduledTask, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}

	return tasks
}

// Start 启动调度器
func (s *TaskScheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("scheduler already running")
	}

	s.ticker = time.NewTicker(1 * time.Minute)
	s.running = true

	go s.run()

	logger.GetLogger().Info("task scheduler started")

	return nil
}

// Stop 停止调度器
func (s *TaskScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.running = false
	s.ticker.Stop()

	logger.GetLogger().Info("task scheduler stopped")
}

// run 运行调度器
func (s *TaskScheduler) run() {
	for s.running {
		<-s.ticker.C

		now := time.Now()
		for _, task := range s.tasks {
			if !task.Enabled {
				continue
			}

			if now.After(task.NextRun) || now.Equal(task.NextRun) {
				s.ExecuteTask(task.ID)
				task.NextRun = s.calculateNextRun(task.Schedule)
			}
		}
	}
}

// ExecuteTask 执行任务
func (s *TaskScheduler) ExecuteTask(id string) (*BatchResult, error) {
	s.mu.RLock()
	task, exists := s.tasks[id]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("task not found: %s", id)
	}

	if !task.Enabled {
		return nil, fmt.Errorf("task is disabled: %s", id)
	}

	logger.GetLogger().Info("executing scheduled task",
		zap.String("id", id),
		zap.String("command", task.Command),
	)

	sessionIDs := task.SessionIDs
	if len(sessionIDs) == 0 && task.GroupName != "" {
		sessions := getSessionIDsFromGroup(task.GroupName, "")
		if len(sessions) > 0 {
			sessionIDs = sessions
		}
	}

	batch := &BatchCommand{
		Command:    task.Command,
		SessionIDs: sessionIDs,
		Parallel:   5,
		Timeout:    60,
	}

	results, err := s.executor.Execute(batch)
	if err != nil {
		task.LastStatus = "failed"
	} else {
		task.LastStatus = "success"
	}

	task.LastRun = time.Now()
	s.Save()

	s.eventBus.Publish("scheduled.executed", task, results)

	if len(results) > 0 {
		return results[0], err
	}

	return nil, err
}

// calculateNextRun 计算下次运行时间
func (s *TaskScheduler) calculateNextRun(schedule string) time.Time {
	now := time.Now()

	if schedule == "" || schedule == "* * * * *" {
		return now.Add(1 * time.Hour)
	}

	next := now

	parts := strings.Split(schedule, " ")
	if len(parts) >= 5 {
		minStr := parts[0]
		hourStr := parts[1]

		var min, hour int
		fmt.Sscanf(minStr, "%d", &min)
		fmt.Sscanf(hourStr, "%d", &hour)

		next = time.Date(now.Year(), now.Month(), now.Day(), hour, min, 0, 0, now.Location())

		if now.After(next) {
			next = next.Add(24 * time.Hour)
		}
	}

	return next
}

// Save 保存任务
func (s *TaskScheduler) Save() error {
	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, ".termbus", "scheduled_tasks.json")

	data, err := json.MarshalIndent(s.tasks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tasks: %w", err)
	}

	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create tasks directory: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to save tasks: %w", err)
	}

	return nil
}

// Load 加载任务
func (s *TaskScheduler) Load() error {
	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, ".termbus", "scheduled_tasks.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	if err := json.Unmarshal(data, &s.tasks); err != nil {
		return fmt.Errorf("failed to unmarshal tasks: %w", err)
	}

	return nil
}
