package migration

import (
	"taskc/backend/internal/model"
	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.Task{},
		&model.Heartbeat{},
		&model.ProbeConfig{},
		&model.ProbeResult{},
		&model.Alert{},
		&model.TaskLog{},
		&model.User{},
	)
}

func CreateIndexes(db *gorm.DB) error {
	// 创建心跳查询索引
	if err := db.Exec("CREATE INDEX idx_heartbeats_task_timestamp ON heartbeats(task_id, timestamp DESC)").Error; err != nil {
		// 忽略已存在的索引错误
	}

	// 创建任务状态索引
	if err := db.Exec("CREATE INDEX idx_tasks_status ON tasks(status)").Error; err != nil {
		// 忽略已存在的索引错误
	}

	// 创建告警查询索引
	if err := db.Exec("CREATE INDEX idx_alerts_task_created ON alerts(task_id, created_at DESC)").Error; err != nil {
		// 忽略已存在的索引错误
	}

	// 创建日志查询索引
	if err := db.Exec("CREATE INDEX idx_task_logs_task_created ON task_logs(task_id, created_at DESC)").Error; err != nil {
		// 忽略已存在的索引错误
	}

	return nil
}