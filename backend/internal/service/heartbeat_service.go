package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"taskc/backend/internal/model"
	"taskc/backend/internal/repository"
	"taskc/backend/pkg/logger"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// HeartbeatService manages heartbeat processing and task status monitoring.
// It handles heartbeat recording, caching, and status state machine transitions.
type HeartbeatService struct {
	heartbeatRepo *repository.HeartbeatRepository
	taskRepo      *repository.TaskRepository
	redisClient   *redis.Client

	// Configuration
	heartbeatTimeout  time.Duration
	maxMissedBeats   int
	cacheExpiration  time.Duration
}

// HeartbeatConfig contains configuration for heartbeat service.
type HeartbeatConfig struct {
	Timeout        time.Duration
	MaxMissedBeats int
	CacheExpiry    time.Duration
}

// DefaultHeartbeatConfig returns optimized heartbeat configuration for PRD requirements.
func DefaultHeartbeatConfig() HeartbeatConfig {
	return HeartbeatConfig{
		Timeout:        20 * time.Second,  // 优化：减少超时时间
		MaxMissedBeats: 2,                 // 优化：减少最大丢失次数以更快检测故障
		CacheExpiry:    3 * time.Minute,   // 优化：减少缓存过期时间
	}
}

// NewHeartbeatService creates a new heartbeat service instance.
func NewHeartbeatService(heartbeatRepo *repository.HeartbeatRepository, taskRepo *repository.TaskRepository, redisClient *redis.Client, config HeartbeatConfig) *HeartbeatService {
	return &HeartbeatService{
		heartbeatRepo:     heartbeatRepo,
		taskRepo:          taskRepo,
		redisClient:       redisClient,
		heartbeatTimeout:  config.Timeout,
		maxMissedBeats:   config.MaxMissedBeats,
		cacheExpiration:  config.CacheExpiry,
	}
}

// RecordHeartbeat processes and records incoming heartbeat data.
// It performs the following operations:
//   1. Validates and saves heartbeat to database
//   2. Updates Redis cache for fast access
//   3. Triggers task status evaluation
//   4. Publishes heartbeat event for real-time processing
func (s *HeartbeatService) RecordHeartbeat(ctx context.Context, heartbeat *model.Heartbeat) error {
	logger.Debug("recording heartbeat",
		zap.String("task_id", heartbeat.TaskID),
		zap.Time("timestamp", heartbeat.Timestamp),
	)

	// 1. Save to database
	if err := s.heartbeatRepo.Create(ctx, heartbeat); err != nil {
		return fmt.Errorf("failed to save heartbeat to database: %w", err)
	}

	// 2. Update Redis cache (non-blocking)
	if err := s.updateHeartbeatCache(ctx, heartbeat); err != nil {
		logger.Warn("failed to update heartbeat cache",
			zap.String("task_id", heartbeat.TaskID),
			zap.Error(err),
		)
		// Don't fail the entire operation if cache update fails
	}

	// 3. Update task status based on heartbeat
	if err := s.updateTaskStatusFromHeartbeat(ctx, heartbeat.TaskID); err != nil {
		logger.Warn("failed to update task status from heartbeat",
			zap.String("task_id", heartbeat.TaskID),
			zap.Error(err),
		)
		// Don't fail the entire operation if status update fails
	}

	// 4. Publish heartbeat event for real-time processing (non-blocking)
	if err := s.publishHeartbeatEvent(ctx, heartbeat); err != nil {
		logger.Warn("failed to publish heartbeat event",
			zap.String("task_id", heartbeat.TaskID),
			zap.Error(err),
		)
		// Don't fail the entire operation if event publishing fails
	}

	logger.Info("heartbeat recorded successfully",
		zap.String("task_id", heartbeat.TaskID),
	)

	return nil
}

// updateHeartbeatCache updates the heartbeat cache in Redis for fast access.
func (s *HeartbeatService) updateHeartbeatCache(ctx context.Context, heartbeat *model.Heartbeat) error {
	key := fmt.Sprintf("heartbeat:%s", heartbeat.TaskID)
	
	data := map[string]interface{}{
		"timestamp": heartbeat.Timestamp.Unix(),
		"metadata":  heartbeat.Metadata,
		"updated_at": time.Now().Unix(),
	}

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal heartbeat cache data: %w", err)
	}

	// Set heartbeat cache with configured expiration
	if err := s.redisClient.Set(ctx, key, dataJSON, s.cacheExpiration).Err(); err != nil {
		return fmt.Errorf("failed to set heartbeat cache: %w", err)
	}

	return nil
}

// updateTaskStatusFromHeartbeat updates task status when heartbeat is received.
// Tasks in SUSPECTED or FAILED state will be recovered to HEALTHY.
func (s *HeartbeatService) updateTaskStatusFromHeartbeat(ctx context.Context, taskID string) error {
	task, err := s.taskRepo.GetByTaskID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task %s: %w", taskID, err)
	}

	// Only update status if task was previously unhealthy
	if task.Status == model.TaskStatusSuspected || task.Status == model.TaskStatusFailed {
		oldStatus := task.Status
		task.Status = model.TaskStatusHealthy
		
		if err := s.taskRepo.Update(ctx, task); err != nil {
			return fmt.Errorf("failed to update task status: %w", err)
		}

		logger.Info("task recovered from heartbeat",
			zap.String("task_id", taskID),
			zap.String("old_status", string(oldStatus)),
			zap.String("new_status", string(task.Status)),
		)

		// Publish recovery event
		s.publishStatusChangeEvent(taskID, task.Status, 0)
	}

	return nil
}

// publishHeartbeatEvent publishes heartbeat event to Redis Stream for real-time processing.
func (s *HeartbeatService) publishHeartbeatEvent(ctx context.Context, heartbeat *model.Heartbeat) error {
	event := map[string]interface{}{
		"type":       "heartbeat",
		"task_id":    heartbeat.TaskID,
		"timestamp":  heartbeat.Timestamp.Unix(),
		"metadata":   heartbeat.Metadata,
		"created_at": time.Now().Unix(),
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal heartbeat event: %w", err)
	}

	// Publish to Redis Stream
	if err := s.redisClient.XAdd(ctx, &redis.XAddArgs{
		Stream: "task_events",
		Values: map[string]interface{}{
			"data": eventJSON,
		},
	}).Err(); err != nil {
		return fmt.Errorf("failed to publish heartbeat event to stream: %w", err)
	}

	return nil
}

// CheckMissedHeartbeats 检查丢失的心跳
func (s *HeartbeatService) CheckMissedHeartbeats(ctx context.Context, timeout time.Duration, maxMissed int) error {
	tasks, err := s.taskRepo.ListActive(ctx)
	if err != nil {
		return err
	}

	for _, task := range tasks {
		if err := s.checkTaskHeartbeat(ctx, task.TaskID, timeout, maxMissed); err != nil {
			logger.Error("Failed to check task heartbeat",
				zap.String("task_id", task.TaskID),
				zap.Error(err),
			)
		}
	}

	return nil
}

// checkTaskHeartbeat 检查单个任务的心跳
func (s *HeartbeatService) checkTaskHeartbeat(ctx context.Context, taskID string, timeout time.Duration, maxMissed int) error {
	lastHeartbeat, err := s.heartbeatRepo.GetLatest(ctx, taskID)
	if err != nil {
		return err
	}

	if lastHeartbeat == nil {
		return nil // 新任务，没有心跳记录
	}

	timeSinceLastHeartbeat := time.Since(lastHeartbeat.Timestamp)
	
	task, err := s.taskRepo.GetByTaskID(ctx, taskID)
	if err != nil {
		return err
	}

	// 计算丢失的心跳次数
	missedBeats := int(timeSinceLastHeartbeat / timeout)

	var newStatus model.TaskStatus
	switch {
	case missedBeats >= maxMissed:
		newStatus = model.TaskStatusFailed
	case missedBeats >= maxMissed/2:
		newStatus = model.TaskStatusSuspected
	default:
		newStatus = model.TaskStatusHealthy
	}

	// 只在状态发生变化时更新
	if task.Status != newStatus {
		task.Status = newStatus
		if err := s.taskRepo.Update(ctx, task); err != nil {
			return err
		}

		logger.Info("Task status updated due to missed heartbeats",
			zap.String("task_id", taskID),
			zap.String("old_status", string(task.Status)),
			zap.String("new_status", string(newStatus)),
			zap.Int("missed_beats", missedBeats),
		)

		// 如果状态变为SUSPECTED或FAILED，发布事件
		if newStatus != model.TaskStatusHealthy {
			s.publishStatusChangeEvent(taskID, newStatus, missedBeats)
		}
	}

	return nil
}

// publishStatusChangeEvent 发布状态变更事件
func (s *HeartbeatService) publishStatusChangeEvent(taskID string, status model.TaskStatus, missedBeats int) {
	ctx := context.Background()
	
	event := map[string]interface{}{
		"type":         "status_change",
		"task_id":      taskID,
		"status":       status,
		"missed_beats": missedBeats,
		"timestamp":    time.Now().Unix(),
	}

	eventJSON, _ := json.Marshal(event)
	
	s.redisClient.XAdd(ctx, &redis.XAddArgs{
		Stream: "task_events",
		Values: map[string]interface{}{
			"data": eventJSON,
		},
	})
}