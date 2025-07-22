package repository

import (
	"context"
	"fmt"
	"time"
	
	"taskc/backend/internal/model"
	"gorm.io/gorm"
)

// ProbeRepository provides data access methods for probe configurations and results.
// It handles CRUD operations for both probe configs and execution results.
type ProbeRepository struct {
	db *gorm.DB
}

// NewProbeRepository creates a new probe repository instance.
func NewProbeRepository(db *gorm.DB) *ProbeRepository {
	return &ProbeRepository{db: db}
}

// CreateConfig creates a new probe configuration.
func (r *ProbeRepository) CreateConfig(ctx context.Context, config *model.ProbeConfig) error {
	if err := r.db.WithContext(ctx).Create(config).Error; err != nil {
		return fmt.Errorf("failed to create probe config: %w", err)
	}
	return nil
}

// GetConfigByID retrieves a probe configuration by its ID.
func (r *ProbeRepository) GetConfigByID(ctx context.Context, id uint) (*model.ProbeConfig, error) {
	var config model.ProbeConfig
	err := r.db.WithContext(ctx).First(&config, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("probe config with ID %d not found", id)
		}
		return nil, fmt.Errorf("failed to get probe config %d: %w", id, err)
	}
	return &config, nil
}

// GetConfigsByTaskID retrieves all enabled probe configurations for a task.
func (r *ProbeRepository) GetConfigsByTaskID(ctx context.Context, taskID string) ([]*model.ProbeConfig, error) {
	var configs []*model.ProbeConfig
	err := r.db.WithContext(ctx).Where("task_id = ? AND enabled = ?", taskID, true).
		Order("created_at DESC").
		Find(&configs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get probe configs for task %s: %w", taskID, err)
	}
	return configs, nil
}

// UpdateConfig updates an existing probe configuration.
func (r *ProbeRepository) UpdateConfig(ctx context.Context, config *model.ProbeConfig) error {
	if err := r.db.WithContext(ctx).Save(config).Error; err != nil {
		return fmt.Errorf("failed to update probe config %d: %w", config.ID, err)
	}
	return nil
}

// DeleteConfig removes a probe configuration.
func (r *ProbeRepository) DeleteConfig(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&model.ProbeConfig{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete probe config %d: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("probe config %d not found", id)
	}
	return nil
}

// SaveResult saves a probe execution result.
func (r *ProbeRepository) SaveResult(ctx context.Context, result *model.ProbeResult) error {
	if err := r.db.WithContext(ctx).Create(result).Error; err != nil {
		return fmt.Errorf("failed to save probe result: %w", err)
	}
	return nil
}

// GetResultsByConfigID retrieves probe results for a specific configuration.
func (r *ProbeRepository) GetResultsByConfigID(ctx context.Context, configID uint, limit int) ([]*model.ProbeResult, error) {
	var results []*model.ProbeResult
	query := r.db.WithContext(ctx).Where("probe_config_id = ?", configID).Order("timestamp DESC")
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	
	err := query.Find(&results).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get probe results for config %d: %w", configID, err)
	}
	return results, nil
}

// GetResultsByTaskID retrieves probe results for all configurations of a task.
func (r *ProbeRepository) GetResultsByTaskID(ctx context.Context, taskID string, limit int) ([]*model.ProbeResult, error) {
	var results []*model.ProbeResult
	query := r.db.WithContext(ctx).Table("probe_results").
		Joins("JOIN probe_configs ON probe_results.probe_config_id = probe_configs.id").
		Where("probe_configs.task_id = ?", taskID).
		Order("probe_results.timestamp DESC")
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	
	err := query.Find(&results).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get probe results for task %s: %w", taskID, err)
	}
	return results, nil
}

// GetLatestResult retrieves the most recent probe result for a configuration.
func (r *ProbeRepository) GetLatestResult(ctx context.Context, configID uint) (*model.ProbeResult, error) {
	var result model.ProbeResult
	err := r.db.WithContext(ctx).Where("probe_config_id = ?", configID).
		Order("timestamp DESC").
		First(&result).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest result for config %d: %w", configID, err)
	}
	return &result, nil
}

// DeleteResult removes a probe result by ID.
func (r *ProbeRepository) DeleteResult(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&model.ProbeResult{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete probe result %d: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("probe result %d not found", id)
	}
	return nil
}

// ProbeStatistics represents aggregated probe execution statistics.
type ProbeStatistics struct {
	TotalExecutions  int64   `json:"total_executions"`
	SuccessCount     int64   `json:"success_count"`
	FailureCount     int64   `json:"failure_count"`
	SuccessRate      float64 `json:"success_rate"`
	AverageLatencyMs float64 `json:"average_latency_ms"`
	MinLatencyMs     int64   `json:"min_latency_ms"`
	MaxLatencyMs     int64   `json:"max_latency_ms"`
}

// GetSuccessRate calculates the success rate for a probe configuration over a time period.
func (r *ProbeRepository) GetSuccessRate(ctx context.Context, configID uint, hours int) (float64, error) {
	since := time.Now().Add(-time.Duration(hours) * time.Hour)
	
	var total, success int64
	
	// Get total count
	err := r.db.WithContext(ctx).Model(&model.ProbeResult{}).
		Where("probe_config_id = ? AND timestamp >= ?", configID, since).
		Count(&total).Error
	if err != nil {
		return 0, fmt.Errorf("failed to count total probe results: %w", err)
	}
	
	if total == 0 {
		return 0, nil
	}
	
	// Get success count
	err = r.db.WithContext(ctx).Model(&model.ProbeResult{}).
		Where("probe_config_id = ? AND timestamp >= ? AND result = ?", configID, since, "SUCCESS").
		Count(&success).Error
	if err != nil {
		return 0, fmt.Errorf("failed to count successful probe results: %w", err)
	}
	
	return float64(success) / float64(total), nil
}

// GetAverageLatency calculates the average latency for successful probes.
func (r *ProbeRepository) GetAverageLatency(ctx context.Context, configID uint, hours int) (float64, error) {
	since := time.Now().Add(-time.Duration(hours) * time.Hour)
	
	var avgLatency float64
	err := r.db.WithContext(ctx).Model(&model.ProbeResult{}).
		Select("AVG(latency_ms)").
		Where("probe_config_id = ? AND timestamp >= ? AND result = ?", configID, since, "SUCCESS").
		Row().Scan(&avgLatency)
	
	if err != nil {
		return 0, fmt.Errorf("failed to calculate average latency: %w", err)
	}
	
	return avgLatency, nil
}

// GetStatistics returns comprehensive statistics for probe executions.
func (r *ProbeRepository) GetStatistics(ctx context.Context, taskID string, since time.Time) (map[string]interface{}, error) {
	// Get all results for the task since the specified time
	var results []struct {
		Result     string `json:"result"`
		LatencyMs  int64  `json:"latency_ms"`
	}
	
	err := r.db.WithContext(ctx).Table("probe_results").
		Select("probe_results.result, probe_results.latency_ms").
		Joins("JOIN probe_configs ON probe_results.probe_config_id = probe_configs.id").
		Where("probe_configs.task_id = ? AND probe_results.timestamp >= ?", taskID, since).
		Find(&results).Error
		
	if err != nil {
		return nil, fmt.Errorf("failed to get probe statistics: %w", err)
	}
	
	if len(results) == 0 {
		return map[string]interface{}{
			"total_executions":    0,
			"success_count":       0,
			"failure_count":       0,
			"success_rate":        0.0,
			"average_latency_ms":  0.0,
			"min_latency_ms":      0,
			"max_latency_ms":      0,
		}, nil
	}
	
	// Calculate statistics
	var successCount, failureCount int64
	var totalLatency, minLatency, maxLatency int64
	minLatency = results[0].LatencyMs
	
	for _, result := range results {
		if result.Result == "SUCCESS" {
			successCount++
			totalLatency += result.LatencyMs
		} else {
			failureCount++
		}
		
		if result.LatencyMs < minLatency {
			minLatency = result.LatencyMs
		}
		if result.LatencyMs > maxLatency {
			maxLatency = result.LatencyMs
		}
	}
	
	totalExecutions := successCount + failureCount
	successRate := float64(successCount) / float64(totalExecutions)
	
	avgLatency := float64(0)
	if successCount > 0 {
		avgLatency = float64(totalLatency) / float64(successCount)
	}
	
	return map[string]interface{}{
		"total_executions":   totalExecutions,
		"success_count":      successCount,
		"failure_count":      failureCount,
		"success_rate":       successRate,
		"average_latency_ms": avgLatency,
		"min_latency_ms":     minLatency,
		"max_latency_ms":     maxLatency,
	}, nil
}

// DeleteOldResults removes probe results older than the specified time.
func (r *ProbeRepository) DeleteOldResults(ctx context.Context, before time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Where("timestamp < ?", before).Delete(&model.ProbeResult{})
	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete old probe results: %w", result.Error)
	}
	return result.RowsAffected, nil
}