package repository

import (
	"context"
	"fmt"
	"time"
	"taskc/backend/internal/model"
	"gorm.io/gorm"
)

type AlertRepository struct {
	db *gorm.DB
}

func NewAlertRepository(db *gorm.DB) *AlertRepository {
	return &AlertRepository{db: db}
}

// Create 创建告警
func (r *AlertRepository) Create(ctx context.Context, alert *model.Alert) error {
	return r.db.WithContext(ctx).Create(alert).Error
}

// GetByID 根据ID获取告警
func (r *AlertRepository) GetByID(ctx context.Context, id uint) (*model.Alert, error) {
	var alert model.Alert
	err := r.db.WithContext(ctx).First(&alert, id).Error
	if err != nil {
		return nil, err
	}
	return &alert, nil
}

// GetByTaskID 根据任务ID获取告警
func (r *AlertRepository) GetByTaskID(ctx context.Context, taskID string, limit int) ([]*model.Alert, error) {
	var alerts []*model.Alert
	query := r.db.WithContext(ctx).Where("task_id = ?", taskID).Order("created_at DESC")
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	
	err := query.Find(&alerts).Error
	return alerts, err
}

// List 获取告警列表
func (r *AlertRepository) List(ctx context.Context, offset, limit int, level string) ([]*model.Alert, int64, error) {
	var alerts []*model.Alert
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Alert{})
	
	if level != "" {
		query = query.Where("level = ?", level)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&alerts).Error
	return alerts, total, err
}

// Update 更新告警
func (r *AlertRepository) Update(alert *model.Alert) error {
	return r.db.Save(alert).Error
}

// MarkAsSent 标记告警为已发送
func (r *AlertRepository) MarkAsSent(ctx context.Context, id uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.Alert{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"sent":    true,
			"sent_at": &now,
		}).Error
}

// GetUnsentAlerts 获取未发送的告警
func (r *AlertRepository) GetUnsentAlerts(ctx context.Context) ([]*model.Alert, error) {
	var alerts []*model.Alert
	err := r.db.WithContext(ctx).Where("sent = ?", false).
		Order("created_at ASC").
		Find(&alerts).Error
	return alerts, err
}

// CountSince 统计指定时间之后的告警数量
func (r *AlertRepository) CountSince(ctx context.Context, taskID string, since time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Alert{}).
		Where("task_id = ? AND created_at >= ?", taskID, since).
		Count(&count).Error
	return count, err
}

// GetAlertsByLevel 根据级别获取告警
func (r *AlertRepository) GetAlertsByLevel(level model.AlertLevel) ([]*model.Alert, error) {
	var alerts []*model.Alert
	err := r.db.Where("level = ?", level).
		Order("created_at DESC").
		Find(&alerts).Error
	return alerts, err
}

// Delete 删除告警
func (r *AlertRepository) Delete(id uint) error {
	return r.db.Delete(&model.Alert{}, id).Error
}

// DeleteOld 删除旧的告警记录
func (r *AlertRepository) DeleteOld(before time.Time) error {
	return r.db.Where("created_at < ?", before).Delete(&model.Alert{}).Error
}

// GetAlertStatistics 获取告警统计信息
func (r *AlertRepository) GetAlertStatistics(ctx context.Context, hours int) (map[string]interface{}, error) {
	since := time.Now().Add(-time.Duration(hours) * time.Hour)
	
	var results []struct {
		Level string
		Count int64
	}

	err := r.db.WithContext(ctx).Model(&model.Alert{}).
		Select("level, COUNT(*) as count").
		Where("created_at >= ?", since).
		Group("level").
		Scan(&results).Error

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
	statistics["period"] = fmt.Sprintf("%d hours", hours)

	return statistics, nil
}