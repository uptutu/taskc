package repository

import (
	"context"
	"fmt"
	"time"
	"taskc/backend/internal/model"
	"gorm.io/gorm"
)

// LogEntry represents a structured log entry (avoiding circular import).
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	TaskID    string                 `json:"task_id,omitempty"`
	Component string                 `json:"component"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// LogSearchFilters contains filters for log search operations.
type LogSearchFilters struct {
	TaskID      string    `json:"task_id,omitempty"`
	Level       string    `json:"level,omitempty"`
	Component   string    `json:"component,omitempty"`
	Message     string    `json:"message,omitempty"`
	StartTime   time.Time `json:"start_time,omitempty"`
	EndTime     time.Time `json:"end_time,omitempty"`
	Limit       int       `json:"limit"`
	Offset      int       `json:"offset"`
	SearchFiles bool      `json:"search_files"`
}

type LogRepository struct {
	db *gorm.DB
}

func NewLogRepository(db *gorm.DB) *LogRepository {
	return &LogRepository{db: db}
}

// Create creates a log entry.
func (r *LogRepository) Create(ctx context.Context, log *model.TaskLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// BatchCreate creates multiple log entries in a single transaction.
func (r *LogRepository) BatchCreate(ctx context.Context, entries []LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	// Convert LogEntry to TaskLog for database storage
	var taskLogs []*model.TaskLog
	for _, entry := range entries {
		taskLog := &model.TaskLog{
			TaskID:    entry.TaskID,
			Level:     entry.Level,
			Message:   entry.Message,
			Timestamp: entry.Timestamp,
			Component: entry.Component,
		}
		taskLogs = append(taskLogs, taskLog)
	}

	return r.db.WithContext(ctx).CreateInBatches(taskLogs, 100).Error
}

// GetByTaskID gets logs by task ID with pagination and filtering.
func (r *LogRepository) GetByTaskID(ctx context.Context, taskID string, offset, limit int, level string) ([]*model.TaskLog, int64, error) {
	var logs []*model.TaskLog
	var total int64

	query := r.db.WithContext(ctx).Model(&model.TaskLog{}).Where("task_id = ?", taskID)
	
	if level != "" {
		query = query.Where("level = ?", level)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated data
	err := query.Offset(offset).Limit(limit).Order("timestamp DESC").Find(&logs).Error
	return logs, total, err
}

// Search searches for log entries based on filters.
func (r *LogRepository) Search(ctx context.Context, filters LogSearchFilters) ([]LogEntry, error) {
	var taskLogs []*model.TaskLog

	query := r.db.WithContext(ctx).Model(&model.TaskLog{})

	// Apply filters
	if filters.TaskID != "" {
		query = query.Where("task_id = ?", filters.TaskID)
	}
	if filters.Level != "" {
		query = query.Where("level = ?", filters.Level)
	}
	if filters.Component != "" {
		query = query.Where("component = ?", filters.Component)
	}
	if filters.Message != "" {
		query = query.Where("message LIKE ?", "%"+filters.Message+"%")
	}
	if !filters.StartTime.IsZero() {
		query = query.Where("timestamp >= ?", filters.StartTime)
	}
	if !filters.EndTime.IsZero() {
		query = query.Where("timestamp <= ?", filters.EndTime)
	}

	// Apply pagination and ordering
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}
	query = query.Order("timestamp DESC")

	err := query.Find(&taskLogs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to search logs: %w", err)
	}

	// Convert to LogEntry
	var entries []LogEntry
	for _, log := range taskLogs {
		entry := LogEntry{
			Timestamp: log.Timestamp,
			Level:     log.Level,
			Message:   log.Message,
			TaskID:    log.TaskID,
			Component: log.Component,
			Fields:    make(map[string]interface{}),
		}
		// Add database-specific fields
		entry.Fields["id"] = log.ID
		entries = append(entries, entry)
	}

	return entries, nil
}

// GetByTimeRange gets logs by time range.
func (r *LogRepository) GetByTimeRange(ctx context.Context, taskID string, start, end time.Time) ([]*model.TaskLog, error) {
	var logs []*model.TaskLog
	err := r.db.WithContext(ctx).Where("task_id = ? AND timestamp BETWEEN ? AND ?", taskID, start, end).
		Order("timestamp DESC").
		Find(&logs).Error
	return logs, err
}

// DeleteOldLogs deletes old logs.
func (r *LogRepository) DeleteOldLogs(ctx context.Context, before time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Where("timestamp < ?", before).Delete(&model.TaskLog{})
	return result.RowsAffected, result.Error
}

// GetStatistics gets log statistics.
func (r *LogRepository) GetStatistics(ctx context.Context, taskID string, since time.Time) (map[string]interface{}, error) {
	var results []struct {
		Level string
		Count int64
	}

	query := r.db.WithContext(ctx).Model(&model.TaskLog{}).
		Select("level, COUNT(*) as count").
		Where("timestamp >= ?", since)
	
	if taskID != "" {
		query = query.Where("task_id = ?", taskID)
	}

	err := query.Group("level").Scan(&results).Error
	if err != nil {
		return nil, err
	}

	statistics := make(map[string]interface{})
	var total int64
	
	for _, result := range results {
		statistics[result.Level] = result.Count
		total += result.Count
	}
	
	statistics["total"] = total
	statistics["period_hours"] = int(time.Since(since).Hours())

	return statistics, nil
}

// Count gets total log count.
func (r *LogRepository) Count(ctx context.Context, taskID string) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&model.TaskLog{})
	
	if taskID != "" {
		query = query.Where("task_id = ?", taskID)
	}
	
	err := query.Count(&count).Error
	return count, err
}

// GetRecentLogs gets recent logs.
func (r *LogRepository) GetRecentLogs(ctx context.Context, taskID string, limit int) ([]*model.TaskLog, error) {
	var logs []*model.TaskLog
	query := r.db.WithContext(ctx).Where("task_id = ?", taskID).
		Order("timestamp DESC").
		Limit(limit)
	
	err := query.Find(&logs).Error
	return logs, err
}