package repository

import (
	"context"
	"fmt"
	
	"taskc/backend/internal/model"
	"gorm.io/gorm"
)

// TaskRepository provides data access methods for task records.
// It implements CRUD operations and specialized queries for task management.
type TaskRepository struct {
	db *gorm.DB
}

// NewTaskRepository creates a new task repository instance.
func NewTaskRepository(db *gorm.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

// Create saves a new task to the database.
func (r *TaskRepository) Create(ctx context.Context, task *model.Task) error {
	if err := r.db.WithContext(ctx).Create(task).Error; err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}
	return nil
}

// GetByID retrieves a task by its database ID.
func (r *TaskRepository) GetByID(ctx context.Context, id uint) (*model.Task, error) {
	var task model.Task
	err := r.db.WithContext(ctx).First(&task, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("task with ID %d not found", id)
		}
		return nil, fmt.Errorf("failed to get task by ID %d: %w", id, err)
	}
	return &task, nil
}

// GetByTaskID retrieves a task by its task ID string.
func (r *TaskRepository) GetByTaskID(ctx context.Context, taskID string) (*model.Task, error) {
	var task model.Task
	err := r.db.WithContext(ctx).Where("task_id = ?", taskID).First(&task).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("task with task_id %s not found", taskID)
		}
		return nil, fmt.Errorf("failed to get task by task_id %s: %w", taskID, err)
	}
	return &task, nil
}

// List retrieves a paginated list of tasks with optional status filtering.
func (r *TaskRepository) List(ctx context.Context, offset, limit int, status string) ([]*model.Task, int64, error) {
	var tasks []*model.Task
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Task{})
	
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count tasks: %w", err)
	}

	// Get paginated data
	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&tasks).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list tasks: %w", err)
	}

	return tasks, total, nil
}

// Update saves changes to an existing task.
func (r *TaskRepository) Update(ctx context.Context, task *model.Task) error {
	if err := r.db.WithContext(ctx).Save(task).Error; err != nil {
		return fmt.Errorf("failed to update task %s: %w", task.TaskID, err)
	}
	return nil
}

// Delete removes a task by its task ID.
func (r *TaskRepository) Delete(ctx context.Context, taskID string) error {
	result := r.db.WithContext(ctx).Where("task_id = ?", taskID).Delete(&model.Task{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete task %s: %w", taskID, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("task %s not found", taskID)
	}
	return nil
}

// GetActiveTasks retrieves all tasks that are not in FAILED status.
// This is used by the status monitoring service.
func (r *TaskRepository) GetActiveTasks(ctx context.Context) ([]*model.Task, error) {
	var tasks []*model.Task
	err := r.db.WithContext(ctx).Where("status != ?", model.TaskStatusFailed).Find(&tasks).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get active tasks: %w", err)
	}
	return tasks, nil
}

// ListActive is an alias for GetActiveTasks to maintain compatibility
func (r *TaskRepository) ListActive(ctx context.Context) ([]*model.Task, error) {
	return r.GetActiveTasks(ctx)
}

// GetTasksByStatus retrieves all tasks with a specific status.
func (r *TaskRepository) GetTasksByStatus(ctx context.Context, status model.TaskStatus) ([]*model.Task, error) {
	var tasks []*model.Task
	err := r.db.WithContext(ctx).Where("status = ?", status).Find(&tasks).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks by status %s: %w", status, err)
	}
	return tasks, nil
}

// UpdateStatus updates only the status field of a task.
func (r *TaskRepository) UpdateStatus(ctx context.Context, taskID string, status model.TaskStatus) error {
	result := r.db.WithContext(ctx).Model(&model.Task{}).
		Where("task_id = ?", taskID).
		Update("status", status)
	if result.Error != nil {
		return fmt.Errorf("failed to update task status for %s: %w", taskID, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("task %s not found", taskID)
	}
	return nil
}

// GetStatusCounts returns count of tasks grouped by status.
// This is used for metrics and dashboard displays.
func (r *TaskRepository) GetStatusCounts(ctx context.Context) (map[string]int64, error) {
	var results []struct {
		Status string
		Count  int64
	}

	err := r.db.WithContext(ctx).Model(&model.Task{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&results).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get task status counts: %w", err)
	}

	counts := make(map[string]int64)
	for _, result := range results {
		counts[result.Status] = result.Count
	}

	// Ensure all status types are present with zero values
	statuses := []string{string(model.TaskStatusHealthy), string(model.TaskStatusSuspected), string(model.TaskStatusFailed)}
	for _, status := range statuses {
		if _, exists := counts[status]; !exists {
			counts[status] = 0
		}
	}

	return counts, nil
}