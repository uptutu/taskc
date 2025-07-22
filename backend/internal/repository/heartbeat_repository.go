package repository

import (
	"context"
	"fmt"
	"time"
	
	"taskc/backend/internal/model"
	"gorm.io/gorm"
)

// HeartbeatRepository provides data access methods for heartbeat records.
// It implements CRUD operations and specialized queries for heartbeat monitoring.
type HeartbeatRepository struct {
	db *gorm.DB
}

// NewHeartbeatRepository creates a new heartbeat repository instance.
func NewHeartbeatRepository(db *gorm.DB) *HeartbeatRepository {
	return &HeartbeatRepository{db: db}
}

// Create saves a new heartbeat record to the database.
func (r *HeartbeatRepository) Create(ctx context.Context, heartbeat *model.Heartbeat) error {
	if err := r.db.WithContext(ctx).Create(heartbeat).Error; err != nil {
		return fmt.Errorf("failed to create heartbeat record: %w", err)
	}
	return nil
}

// GetLatest retrieves the most recent heartbeat for a given task.
// Returns nil if no heartbeat records exist for the task.
func (r *HeartbeatRepository) GetLatest(ctx context.Context, taskID string) (*model.Heartbeat, error) {
	var heartbeat model.Heartbeat
	err := r.db.WithContext(ctx).Where("task_id = ?", taskID).
		Order("timestamp DESC").
		First(&heartbeat).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest heartbeat for task %s: %w", taskID, err)
	}
	return &heartbeat, nil
}

// GetSince retrieves all heartbeats for a task since the specified time.
func (r *HeartbeatRepository) GetSince(ctx context.Context, taskID string, since time.Time) ([]*model.Heartbeat, error) {
	var heartbeats []*model.Heartbeat
	err := r.db.WithContext(ctx).Where("task_id = ? AND timestamp >= ?", taskID, since).
		Order("timestamp DESC").
		Find(&heartbeats).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get heartbeats since %v for task %s: %w", since, taskID, err)
	}
	return heartbeats, nil
}

// CountSince counts heartbeats for a task since the specified time.
func (r *HeartbeatRepository) CountSince(ctx context.Context, taskID string, since time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Heartbeat{}).
		Where("task_id = ? AND timestamp >= ?", taskID, since).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("failed to count heartbeats since %v for task %s: %w", since, taskID, err)
	}
	return count, nil
}

// GetByTimeRange retrieves heartbeats within a specific time range.
func (r *HeartbeatRepository) GetByTimeRange(ctx context.Context, taskID string, start, end time.Time) ([]*model.Heartbeat, error) {
	var heartbeats []*model.Heartbeat
	err := r.db.WithContext(ctx).Where("task_id = ? AND timestamp BETWEEN ? AND ?", taskID, start, end).
		Order("timestamp DESC").
		Find(&heartbeats).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get heartbeats in range %v-%v for task %s: %w", start, end, taskID, err)
	}
	return heartbeats, nil
}

// DeleteByID removes a heartbeat record by its ID.
func (r *HeartbeatRepository) DeleteByID(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&model.Heartbeat{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete heartbeat record %d: %w", id, err)
	}
	return nil
}

// DeleteOldRecords removes heartbeat records older than the specified time.
// This is used for data retention and cleanup.
func (r *HeartbeatRepository) DeleteOldRecords(ctx context.Context, before time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Where("timestamp < ?", before).Delete(&model.Heartbeat{})
	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete old heartbeat records before %v: %w", before, result.Error)
	}
	return result.RowsAffected, nil
}

// HeartbeatStatistics contains aggregated heartbeat metrics.
type HeartbeatStatistics struct {
	TotalCount int64   `json:"total_count"`
	Frequency  float64 `json:"frequency"` // heartbeats per hour
	Period     string  `json:"period"`
	AvgLatency float64 `json:"avg_latency"`
}

// GetStatistics calculates heartbeat statistics for a task over a time period.
func (r *HeartbeatRepository) GetStatistics(ctx context.Context, taskID string, hours int) (*HeartbeatStatistics, error) {
	since := time.Now().Add(-time.Duration(hours) * time.Hour)
	
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Heartbeat{}).
		Where("task_id = ? AND timestamp >= ?", taskID, since).
		Count(&count).Error
	
	if err != nil {
		return nil, fmt.Errorf("failed to get heartbeat statistics for task %s: %w", taskID, err)
	}

	// Calculate heartbeat frequency (per hour)
	duration := time.Duration(hours) * time.Hour
	frequency := float64(count) / duration.Hours()

	return &HeartbeatStatistics{
		TotalCount: count,
		Frequency:  frequency,
		Period:     fmt.Sprintf("%d hours", hours),
		AvgLatency: 0, // TODO: Calculate average latency if needed
	}, nil
}

// GetRecent retrieves the most recent heartbeats for a task, limited by count.
func (r *HeartbeatRepository) GetRecent(ctx context.Context, taskID string, limit int) ([]*model.Heartbeat, error) {
	var heartbeats []*model.Heartbeat
	err := r.db.WithContext(ctx).Where("task_id = ?", taskID).
		Order("timestamp DESC").
		Limit(limit).
		Find(&heartbeats).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get recent heartbeats for task %s: %w", taskID, err)
	}
	return heartbeats, nil
}