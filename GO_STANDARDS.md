# Go 开发规范

**项目**: TaskC 任务中台系统
**创建时间**: 2025-07-22
**适用版本**: Go 1.21+

---

## 📁 项目结构规范

### 目录组织
```
backend/
├── cmd/                    # 应用程序入口
│   └── server/
│       └── main.go
├── internal/               # 私有应用代码
│   ├── api/               # API层
│   │   ├── handlers/      # HTTP处理器
│   │   └── routes/        # 路由定义
│   ├── service/           # 业务逻辑层
│   ├── repository/        # 数据访问层
│   ├── model/             # 数据模型
│   ├── middleware/        # 中间件
│   └── config/            # 配置管理
├── logger/
├── redis/
├── utils/
├── configs/               # 配置文件
├── migrations/            # 数据库迁移
└── go.mod
```

### 包命名规范
- 使用小写字母，避免下划线和驼峰
- 包名应简洁且具有描述性
- 避免 `util`、`common`、`base` 等通用名称

---

## 🏷️ 命名约定

### 变量命名
```go
// ✅ 推荐
var userID int64
var httpClient *http.Client
var dbConnection *sql.DB

// ❌ 避免
var user_id int64
var httpClient *http.Client
var db_connection *sql.DB
```

### 函数命名
```go
// ✅ 推荐 - 动词开头，清晰描述功能
func GetUserByID(id int64) (*User, error)
func CreateTask(task *Task) error
func UpdateTaskStatus(id int64, status TaskStatus) error

// ❌ 避免 - 模糊或不准确
func User(id int64) (*User, error)
func TaskCreate(task *Task) error
```

### 常量命名
```go
// ✅ 推荐 -
const (
    DefautTimeout = 30 * time.Second
    MaxRetryCount = 3
    APIVersion     = "v1"
)

// 枚举类型常量
type TaskStatus int

const (
    TaskStatusHealthy TaskStatus = iota
    TaskStatusSuspected
    TaskStatusFailed
)

// 全局非导出的全局变量
var _db *DB
var _port string
```

### 接口命名
```go
// ✅ 推荐 - 以 -er 结尾
type Reader interface {
    Read([]byte) (int, error)
}

type TaskRepository interface {
    Create(task *Task) error
    GetByID(id int64) (*Task, error)
    Update(task *Task) error
}

type AlertSender interface {
    Send(alert *Alert) error
}
```

---

## 🔧 代码风格

### 错误处理
```go
// ✅ 推荐 - 明确的错误处理
func GetTask(id int64) (*Task, error) {
    task, err := repo.FindByID(id)
    if err != nil {
        return nil, fmt.Errorf("failed to get task %d: %w", id, err)
    }

    if task == nil {
        return nil, ErrTaskNotFound
    }

    return task, nil
}

// ✅ 推荐 - 自定义错误类型
var (
    ErrTaskNotFound = errors.New("task not found")
    ErrInvalidInput = errors.New("invalid input")
)
```

### 结构体定义
```go
// ✅ 推荐 - 结构体字段标签完整
type Task struct {
    ID          int64     `json:"id" db:"id" validate:"required"`
    Name        string    `json:"name" db:"name" validate:"required,min=1,max=100"`
    Status      TaskStatus `json:"status" db:"status"`
    CreatedAt   time.Time `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// ✅ 推荐 - 构造函数模式
func NewTask(name string) *Task {
    return &Task{
        Name:      name,
        Status:    TaskStatusHealthy,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }
}
```

### 函数参数
```go
// ✅ 推荐 - 参数过多时使用配置结构体
type CreateTaskOptions struct {
    Name        string
    Description string
    Timeout     time.Duration
    RetryCount  int
}

func CreateTask(opts CreateTaskOptions) (*Task, error) {
    // 实现逻辑
}

// ✅ 推荐 - 使用上下文
func GetTaskWithContext(ctx context.Context, id int64) (*Task, error) {
    // 实现逻辑
}
```

---

## 📝 注释规范

### 包注释
```go
// Package handlers provides HTTP request handlers for the task management API.
// It implements REST endpoints for task CRUD operations, health monitoring,
// and alert management.
package handlers
```

### 函数注释
```go
// GetTaskByID retrieves a task by its unique identifier.
// Returns ErrTaskNotFound if the task does not exist.
// Returns other errors if database operation fails.
func GetTaskByID(id int64) (*Task, error) {
    // 实现逻辑
}

// CreateHeartbeat processes incoming heartbeat data and updates task status.
// It performs the following operations:
//   1. Validates heartbeat data
//   2. Updates task last_seen timestamp
//   3. Triggers status evaluation if needed
func CreateHeartbeat(heartbeat *Heartbeat) error {
    // 实现逻辑
}
```

### 结构体注释
```go
// Task represents a monitored task in the system.
// Each task has a unique ID and maintains health status.
type Task struct {
    // ID is the unique identifier for the task
    ID int64 `json:"id"`

    // Name is the human-readable task name
    Name string `json:"name"`

    // Status indicates current health status
    Status TaskStatus `json:"status"`
}
```

---

## 🧪 测试规范

### 测试文件组织
```go
// task_service_test.go
package service

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func TestTaskService_GetByID(t *testing.T) {
    tests := []struct {
        name    string
        taskID  int64
        want    *Task
        wantErr bool
    }{
        {
            name:   "success",
            taskID: 1,
            want:   &Task{ID: 1, Name: "test"},
            wantErr: false,
        },
        {
            name:    "not found",
            taskID:  999,
            want:    nil,
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 测试逻辑
        })
    }
}
```

### Mock 使用
```go
//go:generate mockery --name=TaskRepository --output=mocks
type TaskRepository interface {
    GetByID(id int64) (*Task, error)
    Create(task *Task) error
}
```

---

## 🚀 性能规范

### 内存管理
```go
// ✅ 推荐 - 预分配切片容量
func ProcessTasks(tasks []Task) []Result {
    results := make([]Result, 0, len(tasks))
    for _, task := range tasks {
        // 处理逻辑
    }
    return results
}

// ✅ 推荐 - 使用字符串构建器
func BuildMessage(parts []string) string {
    var builder strings.Builder
    builder.Grow(estimatedSize) // 预估容量
    for _, part := range parts {
        builder.WriteString(part)
    }
    return builder.String()
}
```

### 并发控制
```go
// ✅ 推荐 - 使用工作池模式
func ProcessHeartbeats(heartbeats <-chan Heartbeat) {
    const numWorkers = 10
    var wg sync.WaitGroup

    for i := 0; i < numWorkers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for heartbeat := range heartbeats {
                processHeartbeat(heartbeat)
            }
        }()
    }

    wg.Wait()
}
```

---

## 🔒 安全规范

### 输入验证
```go
// ✅ 推荐 - 使用验证库
type CreateTaskRequest struct {
    Name    string `json:"name" validate:"required,min=1,max=100"`
    Timeout int    `json:"timeout" validate:"min=1,max=3600"`
}

func (h *TaskHandler) Create(c *fiber.Ctx) error {
    var req CreateTaskRequest
    if err := c.BodyParser(&req); err != nil {
        return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
    }

    if err := h.validator.Struct(&req); err != nil {
        return fiber.NewError(fiber.StatusBadRequest, err.Error())
    }

    // 处理逻辑
}
```

### 敏感信息处理
```go
// ✅ 推荐 - 不记录敏感信息
func LogUserAction(userID int64, action string) {
    logger.Info("user action",
        zap.Int64("user_id", userID),
        zap.String("action", action),
        // ❌ 不要记录: zap.String("password", password)
    )
}
```

---

## 📊 日志规范

### 日志级别使用
```go
// ✅ 推荐 - 合理使用日志级别
func ProcessHeartbeat(heartbeat *Heartbeat) error {
    // Debug - 调试信息
    logger.Debug("processing heartbeat",
        zap.String("task_id", heartbeat.TaskID),
    )

    // Info - 重要业务事件
    logger.Info("heartbeat processed successfully",
        zap.String("task_id", heartbeat.TaskID),
        zap.String("status", heartbeat.Status),
    )

    // Warn - 需要关注但不影响功能
    if heartbeat.Latency > threshold {
        logger.Warn("high heartbeat latency detected",
            zap.Duration("latency", heartbeat.Latency),
        )
    }

    // Error - 错误情况
    if err != nil {
        logger.Error("failed to process heartbeat",
            zap.Error(err),
            zap.String("task_id", heartbeat.TaskID),
        )
        return err
    }

    return nil
}
```

---

## 🔧 依赖管理

### Go Modules
```go
// ✅ 推荐 - 明确版本约束
require (
    github.com/gofiber/fiber/v2 v2.52.0
    github.com/go-redis/redis/v8 v8.11.5
    gorm.io/gorm v1.25.5
)

// ✅ 推荐 - 分组依赖
require (
    // Web框架
    github.com/gofiber/fiber/v2 v2.52.0
    github.com/gofiber/websocket/v2 v2.2.1

    // 数据库
    gorm.io/driver/mysql v1.5.2
    gorm.io/gorm v1.25.5

    // 缓存和消息队列
    github.com/go-redis/redis/v8 v8.11.5
)
```

---

## 🎯 配置管理

### 配置结构
```go
// ✅ 推荐 - 结构化配置
type Config struct {
    Server   ServerConfig   `mapstructure:"server"`
    Database DatabaseConfig `mapstructure:"database"`
    Redis    RedisConfig    `mapstructure:"redis"`
    Log      LogConfig      `mapstructure:"log"`
}

type ServerConfig struct {
    Host         string        `mapstructure:"host" default:"0.0.0.0"`
    Port         int           `mapstructure:"port" default:"8080"`
    ReadTimeout  time.Duration `mapstructure:"read_timeout" default:"30s"`
    WriteTimeout time.Duration `mapstructure:"write_timeout" default:"30s"`
}
```

---

## 📋 代码审查清单

### 提交前检查
- [ ] 所有公共函数都有注释
- [ ] 错误处理完整且适当
- [ ] 使用了上下文传递
- [ ] 没有硬编码的魔法数字
- [ ] 测试覆盖率满足要求
- [ ] 没有竞态条件
- [ ] 内存泄漏检查
- [ ] 日志记录适当且不包含敏感信息

### 性能检查
- [ ] 避免不必要的内存分配
- [ ] 合理使用缓存
- [ ] 数据库查询优化
- [ ] 并发控制得当

---

**最后更新**: 2025-07-22
**维护者**: TaskC 开发团队
