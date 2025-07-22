package model

import (
	"time"
	"gorm.io/gorm"
)

type TaskStatus string

const (
	TaskStatusHealthy   TaskStatus = "healthy"
	TaskStatusSuspected TaskStatus = "suspected"
	TaskStatusFailed    TaskStatus = "failed"
)

type AlertLevel string

const (
	AlertLevelCritical AlertLevel = "critical"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelInfo     AlertLevel = "info"
)

type ProbeType string

const (
	ProbeTypeHTTP ProbeType = "http"
	ProbeTypeTCP  ProbeType = "tcp"
	ProbeTypePing ProbeType = "ping"
)

type Task struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	TaskID      string         `json:"task_id" gorm:"uniqueIndex;not null" validate:"required"`
	Name        string         `json:"name" gorm:"not null" validate:"required"`
	Description string         `json:"description"`
	Status      TaskStatus     `json:"status" gorm:"default:healthy"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	Heartbeats   []Heartbeat   `json:"heartbeats,omitempty" gorm:"foreignKey:TaskID;references:TaskID"`
	ProbeConfigs []ProbeConfig `json:"probe_configs,omitempty" gorm:"foreignKey:TaskID;references:TaskID"`
	Alerts       []Alert       `json:"alerts,omitempty" gorm:"foreignKey:TaskID;references:TaskID"`
}

type Heartbeat struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	TaskID    string    `json:"task_id" gorm:"index;not null"`
	Timestamp time.Time `json:"timestamp" gorm:"not null"`
	Metadata  string    `json:"metadata" gorm:"type:json"`
	CreatedAt time.Time `json:"created_at"`

	Task Task `json:"task,omitempty" gorm:"foreignKey:TaskID;references:TaskID"`
}

type HeartbeatMetadata struct {
	CPULoad    float64 `json:"cpu_load"`
	MemUsedMB  int64   `json:"mem_used_mb"`
	QueueSize  int     `json:"queue_size"`
	CustomData map[string]interface{} `json:"custom_data,omitempty"`
}

type ProbeConfig struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	TaskID    string         `json:"task_id" gorm:"index;not null"`
	Type      ProbeType      `json:"type" gorm:"not null"`
	Config    string         `json:"config" gorm:"type:json;not null"`
	Enabled   bool           `json:"enabled" gorm:"default:true"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	Task         Task           `json:"task,omitempty" gorm:"foreignKey:TaskID;references:TaskID"`
	ProbeResults []ProbeResult  `json:"probe_results,omitempty" gorm:"foreignKey:ProbeConfigID"`
}

type ProbeConfigData struct {
	Endpoint       string   `json:"endpoint"`
	TimeoutMs      int      `json:"timeout_ms"`
	ExpectedStatus int      `json:"expected_status"`
	Headers        map[string]string `json:"headers,omitempty"`
	SuccessConditions []string `json:"success_conditions,omitempty"`
}

type ProbeResult struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	ProbeConfigID  uint      `json:"probe_config_id" gorm:"index;not null"`
	Result         string    `json:"result" gorm:"not null"`
	LatencyMs      int64     `json:"latency_ms"`
	ErrorMessage   string    `json:"error_message"`
	Timestamp      time.Time `json:"timestamp" gorm:"not null"`

	ProbeConfig ProbeConfig `json:"probe_config,omitempty" gorm:"foreignKey:ProbeConfigID"`
}

type Alert struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	TaskID    string         `json:"task_id" gorm:"index;not null"`
	Level     AlertLevel     `json:"level" gorm:"not null"`
	Title     string         `json:"title" gorm:"not null"`
	Message   string         `json:"message" gorm:"not null"`
	Channels  string         `json:"channels" gorm:"type:json"`
	Sent      bool           `json:"sent" gorm:"default:false"`
	SentAt    *time.Time     `json:"sent_at"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	Task Task `json:"task,omitempty" gorm:"foreignKey:TaskID;references:TaskID"`
}

type TaskLog struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	TaskID    string         `json:"task_id" gorm:"index;not null"`
	Level     string         `json:"level" gorm:"not null"`
	Message   string         `json:"message" gorm:"not null"`
	Component string         `json:"component" gorm:"index"`
	Timestamp time.Time      `json:"timestamp" gorm:"not null;index"`
	Metadata  string         `json:"metadata" gorm:"type:json"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	Task Task `json:"task,omitempty" gorm:"foreignKey:TaskID;references:TaskID"`
}

type User struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Username  string         `json:"username" gorm:"uniqueIndex;not null" validate:"required"`
	Email     string         `json:"email" gorm:"uniqueIndex;not null" validate:"required,email"`
	Password  string         `json:"-" gorm:"not null"`
	Role      string         `json:"role" gorm:"default:user"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}