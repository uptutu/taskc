package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"taskc/backend/internal/model"
	"taskc/backend/internal/repository"
	"taskc/backend/pkg/logger"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// TaskStatusConfig contains configuration for task status monitoring.
type TaskStatusConfig struct {
	HeartbeatTimeout  time.Duration // Time after which a task is considered suspicious
	MaxMissedBeats   int           // Number of missed heartbeats before marking as failed
	CheckInterval    time.Duration // How often to check task statuses
	RecoveryWindow   time.Duration // Time window for status recovery validation
}

// DefaultTaskStatusConfig returns default configuration for task status monitoring.
func DefaultTaskStatusConfig() TaskStatusConfig {
	return TaskStatusConfig{
		HeartbeatTimeout: 30 * time.Second,
		MaxMissedBeats:   3,
		CheckInterval:    10 * time.Second,
		RecoveryWindow:   5 * time.Minute,
	}
}

// TaskStatusService manages the task status state machine.
// It handles transitions between HEALTHY, SUSPECTED, and FAILED states
// based on heartbeat patterns and active probing results.
type TaskStatusService struct {
	taskRepo          *repository.TaskRepository
	heartbeatRepo     *repository.HeartbeatRepository
	alertService      *AlertService
	probeService      *ProbeService
	config            TaskStatusConfig
	cron              *cron.Cron
	running           bool
	mu                sync.RWMutex
}

// NewTaskStatusService creates a new task status service instance.
func NewTaskStatusService(
	taskRepo *repository.TaskRepository,
	heartbeatRepo *repository.HeartbeatRepository,
	alertService *AlertService,
	probeService *ProbeService,
	config TaskStatusConfig,
) *TaskStatusService {
	return &TaskStatusService{
		taskRepo:      taskRepo,
		heartbeatRepo: heartbeatRepo,
		alertService:  alertService,
		probeService:  probeService,
		config:        config,
		cron:          cron.New(),
		running:       false,
	}
}

// Start begins the task status monitoring service.
func (s *TaskStatusService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("task status service is already running")
	}

	// Schedule periodic status checks
	_, err := s.cron.AddFunc(fmt.Sprintf("@every %s", s.config.CheckInterval), func() {
		if err := s.checkAllTaskStatuses(ctx); err != nil {
			logger.Error("failed to check task statuses", zap.Error(err))
		}
	})
	if err != nil {
		return fmt.Errorf("failed to schedule status check job: %w", err)
	}

	s.cron.Start()
	s.running = true

	logger.Info("task status service started",
		zap.Duration("check_interval", s.config.CheckInterval),
		zap.Duration("heartbeat_timeout", s.config.HeartbeatTimeout),
		zap.Int("max_missed_beats", s.config.MaxMissedBeats),
	)

	return nil
}

// Stop stops the task status monitoring service.
func (s *TaskStatusService) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("task status service is not running")
	}

	s.cron.Stop()
	s.running = false

	logger.Info("task status service stopped")
	return nil
}

// checkAllTaskStatuses checks and updates status for all active tasks.
func (s *TaskStatusService) checkAllTaskStatuses(ctx context.Context) error {
	tasks, err := s.taskRepo.GetActiveTasks(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active tasks: %w", err)
	}

	logger.Debug("checking task statuses",
		zap.Int("task_count", len(tasks)),
	)

	for _, task := range tasks {
		if err := s.evaluateTaskStatus(ctx, task); err != nil {
			logger.Error("failed to evaluate task status",
				zap.String("task_id", task.TaskID),
				zap.Error(err),
			)
		}
	}

	return nil
}

// evaluateTaskStatus evaluates and updates a single task's status based on heartbeat data.
func (s *TaskStatusService) evaluateTaskStatus(ctx context.Context, task *model.Task) error {
	// Get the latest heartbeat
	lastHeartbeat, err := s.heartbeatRepo.GetLatest(ctx, task.TaskID)
	if err != nil {
		return fmt.Errorf("failed to get latest heartbeat: %w", err)
	}

	// If no heartbeat exists, keep current status (task might be newly created)
	if lastHeartbeat == nil {
		return nil
	}

	// Calculate time since last heartbeat
	timeSinceLastHeartbeat := time.Since(lastHeartbeat.Timestamp)
	missedBeats := int(timeSinceLastHeartbeat / s.config.HeartbeatTimeout)

	// Determine new status based on state machine rules
	newStatus := s.calculateNewStatus(task.Status, missedBeats)

	// Only update if status has changed
	if newStatus != task.Status {
		if err := s.transitionTaskStatus(ctx, task, newStatus, missedBeats); err != nil {
			return fmt.Errorf("failed to transition task status: %w", err)
		}
	}

	return nil
}

// calculateNewStatus implements the task status state machine logic.
// State transitions:
// HEALTHY -> SUSPECTED (when heartbeats start missing)
// SUSPECTED -> FAILED (when too many heartbeats are missed)
// SUSPECTED -> HEALTHY (when heartbeats resume)
// FAILED -> SUSPECTED (when heartbeats resume, requires validation)
func (s *TaskStatusService) calculateNewStatus(currentStatus model.TaskStatus, missedBeats int) model.TaskStatus {
	switch currentStatus {
	case model.TaskStatusHealthy:
		if missedBeats >= 1 {
			return model.TaskStatusSuspected
		}
		return model.TaskStatusHealthy

	case model.TaskStatusSuspected:
		if missedBeats >= s.config.MaxMissedBeats {
			return model.TaskStatusFailed
		}
		if missedBeats == 0 {
			return model.TaskStatusHealthy
		}
		return model.TaskStatusSuspected

	case model.TaskStatusFailed:
		// Failed tasks require active probing to confirm recovery
		if missedBeats == 0 {
			return model.TaskStatusSuspected // Move to suspected first for validation
		}
		return model.TaskStatusFailed

	default:
		return model.TaskStatusHealthy
	}
}

// transitionTaskStatus handles the actual status transition and side effects.
func (s *TaskStatusService) transitionTaskStatus(ctx context.Context, task *model.Task, newStatus model.TaskStatus, missedBeats int) error {
	oldStatus := task.Status
	task.Status = newStatus

	// Update task in database
	if err := s.taskRepo.Update(ctx, task); err != nil {
		return fmt.Errorf("failed to update task status in database: %w", err)
	}

	logger.Info("task status transition",
		zap.String("task_id", task.TaskID),
		zap.String("old_status", string(oldStatus)),
		zap.String("new_status", string(newStatus)),
		zap.Int("missed_beats", missedBeats),
	)

	// Handle status-specific actions
	switch newStatus {
	case model.TaskStatusSuspected:
		// When transitioning to suspected, trigger active probing
		if err := s.triggerProbeForTask(ctx, task); err != nil {
			logger.Warn("failed to trigger probe for suspected task",
				zap.String("task_id", task.TaskID),
				zap.Error(err),
			)
		}

	case model.TaskStatusFailed:
		// When transitioning to failed, send critical alert
		if err := s.sendTaskFailureAlert(ctx, task, missedBeats); err != nil {
			logger.Error("failed to send task failure alert",
				zap.String("task_id", task.TaskID),
				zap.Error(err),
			)
		}

	case model.TaskStatusHealthy:
		// When recovering to healthy, send recovery notification
		if oldStatus != model.TaskStatusHealthy {
			if err := s.sendTaskRecoveryAlert(ctx, task, oldStatus); err != nil {
				logger.Warn("failed to send task recovery alert",
					zap.String("task_id", task.TaskID),
					zap.Error(err),
				)
			}
		}
	}

	return nil
}

// triggerProbeForTask initiates active probing for a suspected task.
func (s *TaskStatusService) triggerProbeForTask(ctx context.Context, task *model.Task) error {
	// Get probe configurations for this task
	probeConfigs, err := s.probeService.GetTaskProbeConfigs(ctx, task.TaskID)
	if err != nil {
		return fmt.Errorf("failed to get probe configs: %w", err)
	}

	// If no probe configs exist, skip probing
	if len(probeConfigs) == 0 {
		logger.Debug("no probe configs found for task, skipping active probing",
			zap.String("task_id", task.TaskID),
		)
		return nil
	}

	// Execute probes asynchronously
	go func() {
		for _, config := range probeConfigs {
			if _, err := s.probeService.ExecuteProbe(context.Background(), config); err != nil {
				logger.Error("probe execution failed",
					zap.String("task_id", task.TaskID),
					zap.Uint("probe_config_id", config.ID),
					zap.Error(err),
				)
			}
		}
	}()

	return nil
}

// sendTaskFailureAlert sends a critical alert when a task fails.
func (s *TaskStatusService) sendTaskFailureAlert(ctx context.Context, task *model.Task, missedBeats int) error {
	alert := &model.Alert{
		TaskID:  task.TaskID,
		Level:   model.AlertLevelCritical,
		Title:   fmt.Sprintf("Task %s Failed", task.TaskID),
		Message: fmt.Sprintf("Task '%s' has failed after missing %d heartbeats. Last heartbeat timeout exceeded.", task.Name, missedBeats),
		Channels: `["sms", "email", "slack"]`, // Critical alerts go to all channels
	}

	return s.alertService.SendAlert(ctx, alert)
}

// sendTaskRecoveryAlert sends a notification when a task recovers.
func (s *TaskStatusService) sendTaskRecoveryAlert(ctx context.Context, task *model.Task, previousStatus model.TaskStatus) error {
	alert := &model.Alert{
		TaskID:  task.TaskID,
		Level:   model.AlertLevelInfo,
		Title:   fmt.Sprintf("Task %s Recovered", task.TaskID),
		Message: fmt.Sprintf("Task '%s' has recovered from %s status and is now healthy.", task.Name, previousStatus),
		Channels: `["email", "slack"]`, // Recovery notifications to non-critical channels
	}

	return s.alertService.SendAlert(ctx, alert)
}

// ForceStatusTransition manually forces a task to a specific status.
// This should be used carefully and only for administrative purposes.
func (s *TaskStatusService) ForceStatusTransition(ctx context.Context, taskID string, newStatus model.TaskStatus, reason string) error {
	task, err := s.taskRepo.GetByTaskID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	oldStatus := task.Status
	task.Status = newStatus

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	logger.Warn("forced task status transition",
		zap.String("task_id", taskID),
		zap.String("old_status", string(oldStatus)),
		zap.String("new_status", string(newStatus)),
		zap.String("reason", reason),
	)

	return nil
}

// GetTaskStatusMetrics returns metrics about task statuses.
func (s *TaskStatusService) GetTaskStatusMetrics(ctx context.Context) (map[string]int64, error) {
	metrics, err := s.taskRepo.GetStatusCounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get task status metrics: %w", err)
	}

	return metrics, nil
}