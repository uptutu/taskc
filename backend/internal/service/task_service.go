package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"taskc/backend/internal/model"
	"taskc/backend/internal/repository"
	"taskc/backend/pkg/logger"

	"go.uber.org/zap"
)

// TaskService provides business logic for task management operations.
// It coordinates between different repositories and handles task lifecycle.
type TaskService struct {
	taskRepo      *repository.TaskRepository
	heartbeatRepo *repository.HeartbeatRepository
	alertRepo     *repository.AlertRepository
}

// NewTaskService creates a new task service instance.
func NewTaskService(taskRepo *repository.TaskRepository, heartbeatRepo *repository.HeartbeatRepository, alertRepo *repository.AlertRepository) *TaskService {
	return &TaskService{
		taskRepo:      taskRepo,
		heartbeatRepo: heartbeatRepo,
		alertRepo:     alertRepo,
	}
}

// CreateTask creates a new task after validating it doesn't already exist.
func (s *TaskService) CreateTask(ctx context.Context, task *model.Task) error {
	// Check if task ID already exists
	existing, err := s.taskRepo.GetByTaskID(ctx, task.TaskID)
	if err == nil && existing != nil {
		return fmt.Errorf("task with ID %s already exists", task.TaskID)
	}

	// Set initial status if not specified
	if task.Status == "" {
		task.Status = model.TaskStatusHealthy
	}

	if err := s.taskRepo.Create(ctx, task); err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	logger.Info("task created successfully",
		zap.String("task_id", task.TaskID),
		zap.String("name", task.Name),
		zap.String("status", string(task.Status)),
	)

	return nil
}

// GetTaskByID retrieves a task by its task ID.
func (s *TaskService) GetTaskByID(ctx context.Context, taskID string) (*model.Task, error) {
	task, err := s.taskRepo.GetByTaskID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task %s: %w", taskID, err)
	}
	return task, nil
}

// ListTasks retrieves a paginated list of tasks with optional status filtering.
func (s *TaskService) ListTasks(ctx context.Context, page, limit int, status string) ([]*model.Task, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit
	tasks, total, err := s.taskRepo.List(ctx, offset, limit, status)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list tasks: %w", err)
	}

	return tasks, total, nil
}

// UpdateTaskStatus updates a task's status and handles related side effects.
func (s *TaskService) UpdateTaskStatus(ctx context.Context, taskID string, status model.TaskStatus) error {
	task, err := s.taskRepo.GetByTaskID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task for status update: %w", err)
	}

	oldStatus := task.Status
	if oldStatus == status {
		return nil // No change needed
	}

	task.Status = status

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	logger.Info("task status updated",
		zap.String("task_id", taskID),
		zap.String("old_status", string(oldStatus)),
		zap.String("new_status", string(status)),
	)

	// Handle status-specific actions
	if status == model.TaskStatusFailed && oldStatus != model.TaskStatusFailed {
		if err := s.createFailureAlert(ctx, task); err != nil {
			logger.Error("failed to create failure alert",
				zap.String("task_id", taskID),
				zap.Error(err),
			)
		}
	}

	return nil
}

// DeleteTask removes a task and related data.
func (s *TaskService) DeleteTask(ctx context.Context, taskID string) error {
	// Verify task exists before deletion
	if _, err := s.taskRepo.GetByTaskID(ctx, taskID); err != nil {
		return fmt.Errorf("task not found for deletion: %w", err)
	}

	if err := s.taskRepo.Delete(ctx, taskID); err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	logger.Info("task deleted successfully",
		zap.String("task_id", taskID),
	)

	return nil
}

// TaskHealthResponse represents the health status response for a task.
type TaskHealthResponse struct {
	Status          model.TaskStatus `json:"status"`
	LastHeartbeat   *string          `json:"last_heartbeat"`
	ProbeHistory    []ProbeResult    `json:"probe_history"`
	ResourceUsage   ResourceUsage    `json:"resource_usage"`
	UptimeRatio     float64          `json:"uptime_ratio"`
}

// ProbeResult represents a probe execution result.
type ProbeResult struct {
	Timestamp string `json:"timestamp"`
	Result    string `json:"result"`
	LatencyMs int64  `json:"latency_ms"`
}

// ResourceUsage represents task resource utilization metrics.
type ResourceUsage struct {
	AvgCPU    float64 `json:"avg_cpu"`
	MaxMemMB  int64   `json:"max_mem_mb"`
}

// GetTaskHealth retrieves comprehensive health information for a task.
func (s *TaskService) GetTaskHealth(ctx context.Context, taskID string) (*TaskHealthResponse, error) {
	task, err := s.taskRepo.GetByTaskID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task for health check: %w", err)
	}

	response := &TaskHealthResponse{
		Status:        task.Status,
		ProbeHistory:  []ProbeResult{},
		ResourceUsage: ResourceUsage{},
	}

	// Get latest heartbeat
	lastHeartbeat, err := s.heartbeatRepo.GetLatest(ctx, taskID)
	if err != nil {
		logger.Warn("failed to get latest heartbeat for health check",
			zap.String("task_id", taskID),
			zap.Error(err),
		)
	} else if lastHeartbeat != nil {
		heartbeatTime := lastHeartbeat.Timestamp.Format(time.RFC3339)
		response.LastHeartbeat = &heartbeatTime
	}

	// Calculate resource usage
	resourceUsage, err := s.calculateResourceUsage(ctx, taskID)
	if err != nil {
		logger.Warn("failed to calculate resource usage for health check",
			zap.String("task_id", taskID),
			zap.Error(err),
		)
	} else {
		response.ResourceUsage = resourceUsage
	}

	// Calculate uptime ratio
	uptimeRatio := s.calculateUptimeRatio(ctx, taskID, 24*time.Hour)
	response.UptimeRatio = uptimeRatio

	return response, nil
}

// TaskMetrics represents aggregated metrics for a task.
type TaskMetrics struct {
	HeartbeatCount int64                    `json:"heartbeat_count"`
	AlertCount     int64                    `json:"alert_count"`
	StatusHistory  []StatusHistoryEntry     `json:"status_history"`
	UptimeRatio    float64                  `json:"uptime_ratio"`
	ResourceTrend  ResourceTrend            `json:"resource_trend"`
}

// StatusHistoryEntry represents a status change event.
type StatusHistoryEntry struct {
	Timestamp string           `json:"timestamp"`
	Status    model.TaskStatus `json:"status"`
	Duration  int64            `json:"duration_seconds"`
}

// ResourceTrend represents resource usage trends.
type ResourceTrend struct {
	AvgCPU      float64 `json:"avg_cpu"`
	MaxCPU      float64 `json:"max_cpu"`
	AvgMemory   int64   `json:"avg_memory_mb"`
	MaxMemory   int64   `json:"max_memory_mb"`
}

// GetTaskMetrics retrieves comprehensive metrics for a task over a time period.
func (s *TaskService) GetTaskMetrics(ctx context.Context, taskID string, hours int) (*TaskMetrics, error) {
	if hours <= 0 || hours > 24*7 { // Limit to 1 week
		hours = 24
	}

	since := time.Now().Add(-time.Duration(hours) * time.Hour)
	
	// Get heartbeat statistics
	heartbeatCount, err := s.heartbeatRepo.CountSince(ctx, taskID, since)
	if err != nil {
		return nil, fmt.Errorf("failed to count heartbeats: %w", err)
	}

	// Get alert statistics
	alertCount, err := s.alertRepo.CountSince(ctx, taskID, since)
	if err != nil {
		return nil, fmt.Errorf("failed to count alerts: %w", err)
	}

	// Calculate resource trends
	resourceTrend, err := s.calculateResourceTrend(ctx, taskID, since)
	if err != nil {
		logger.Warn("failed to calculate resource trend",
			zap.String("task_id", taskID),
			zap.Error(err),
		)
		resourceTrend = ResourceTrend{}
	}

	// Calculate uptime ratio
	uptimeRatio := s.calculateUptimeRatio(ctx, taskID, time.Duration(hours)*time.Hour)

	metrics := &TaskMetrics{
		HeartbeatCount: heartbeatCount,
		AlertCount:     alertCount,
		StatusHistory:  []StatusHistoryEntry{}, // TODO: Implement status history
		UptimeRatio:    uptimeRatio,
		ResourceTrend:  resourceTrend,
	}

	return metrics, nil
}

// createFailureAlert creates an alert when a task fails.
func (s *TaskService) createFailureAlert(ctx context.Context, task *model.Task) error {
	alert := &model.Alert{
		TaskID:   task.TaskID,
		Level:    model.AlertLevelCritical,
		Title:    "Task Failed",
		Message:  fmt.Sprintf("Task '%s' has failed and requires attention.", task.Name),
		Channels: `["sms", "email", "slack"]`,
	}

	if err := s.alertRepo.Create(ctx, alert); err != nil {
		return fmt.Errorf("failed to create failure alert: %w", err)
	}

	return nil
}

// calculateResourceUsage calculates resource usage metrics from recent heartbeats.
func (s *TaskService) calculateResourceUsage(ctx context.Context, taskID string) (ResourceUsage, error) {
	since := time.Now().Add(-1 * time.Hour)
	heartbeats, err := s.heartbeatRepo.GetSince(ctx, taskID, since)
	if err != nil || len(heartbeats) == 0 {
		return ResourceUsage{}, nil
	}

	var totalCPU float64
	var maxMem int64
	var cpuCount int

	for _, hb := range heartbeats {
		var metadata model.HeartbeatMetadata
		if err := json.Unmarshal([]byte(hb.Metadata), &metadata); err == nil {
			if metadata.CPULoad > 0 {
				totalCPU += metadata.CPULoad
				cpuCount++
			}
			if metadata.MemUsedMB > maxMem {
				maxMem = metadata.MemUsedMB
			}
		}
	}

	avgCPU := float64(0)
	if cpuCount > 0 {
		avgCPU = (totalCPU / float64(cpuCount)) * 100
	}

	return ResourceUsage{
		AvgCPU:   avgCPU,
		MaxMemMB: maxMem,
	}, nil
}

// calculateResourceTrend calculates resource usage trends over time.
func (s *TaskService) calculateResourceTrend(ctx context.Context, taskID string, since time.Time) (ResourceTrend, error) {
	heartbeats, err := s.heartbeatRepo.GetSince(ctx, taskID, since)
	if err != nil || len(heartbeats) == 0 {
		return ResourceTrend{}, nil
	}

	var totalCPU, maxCPU float64
	var totalMem, maxMem int64
	var cpuCount, memCount int

	for _, hb := range heartbeats {
		var metadata model.HeartbeatMetadata
		if err := json.Unmarshal([]byte(hb.Metadata), &metadata); err == nil {
			if metadata.CPULoad > 0 {
				totalCPU += metadata.CPULoad
				cpuCount++
				if metadata.CPULoad > maxCPU {
					maxCPU = metadata.CPULoad
				}
			}
			if metadata.MemUsedMB > 0 {
				totalMem += metadata.MemUsedMB
				memCount++
				if metadata.MemUsedMB > maxMem {
					maxMem = metadata.MemUsedMB
				}
			}
		}
	}

	avgCPU := float64(0)
	if cpuCount > 0 {
		avgCPU = (totalCPU / float64(cpuCount)) * 100
		maxCPU = maxCPU * 100
	}

	avgMem := int64(0)
	if memCount > 0 {
		avgMem = totalMem / int64(memCount)
	}

	return ResourceTrend{
		AvgCPU:    avgCPU,
		MaxCPU:    maxCPU,
		AvgMemory: avgMem,
		MaxMemory: maxMem,
	}, nil
}

// calculateUptimeRatio calculates the uptime ratio for a task over a given period.
// This is based on heartbeat frequency and task status changes.
func (s *TaskService) calculateUptimeRatio(ctx context.Context, taskID string, period time.Duration) float64 {
	since := time.Now().Add(-period)
	
	// Get heartbeat count for the period
	heartbeatCount, err := s.heartbeatRepo.CountSince(ctx, taskID, since)
	if err != nil {
		logger.Warn("failed to get heartbeat count for uptime calculation",
			zap.String("task_id", taskID),
			zap.Error(err),
		)
		return 0.0
	}

	// Expected heartbeats (assuming 30-second intervals)
	expectedHeartbeats := int64(period.Seconds() / 30)
	if expectedHeartbeats == 0 {
		return 1.0
	}

	// Calculate uptime ratio based on heartbeat frequency
	uptime := float64(heartbeatCount) / float64(expectedHeartbeats)
	if uptime > 1.0 {
		uptime = 1.0
	}

	return uptime
}