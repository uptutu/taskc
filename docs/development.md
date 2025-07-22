# 开发指南

本文档介绍TaskC项目的开发环境搭建、代码结构、开发规范和贡献流程。

## 开发环境搭建

### 系统要求

- **操作系统**: macOS / Linux / Windows (WSL2)
- **Go**: 1.21+
- **Node.js**: 18+
- **Docker**: 20.10+
- **Docker Compose**: 2.0+
- **Git**: 2.30+

### 开发工具推荐

**IDE/编辑器**:
- VS Code + Go插件 + React插件
- GoLand (JetBrains)
- WebStorm (前端)

**必装插件**:
- Go语言支持
- ESLint / Prettier
- Docker支持
- Git History

### 环境配置

#### 1. 克隆项目

```bash
git clone https://github.com/company/taskc.git
cd taskc
```

#### 2. 后端开发环境

```bash
cd backend

# 安装Go依赖
go mod download

# 创建开发配置
cp configs/config.example.yaml configs/config.dev.yaml

# 启动依赖服务
cd ../docker
docker-compose up -d mysql redis

# 运行数据库迁移
cd ../backend
go run cmd/migrate/main.go

# 启动开发服务器
go run cmd/server/main.go
```

#### 3. 前端开发环境

```bash
cd frontend

# 安装依赖
npm install

# 启动开发服务器
npm run dev
```

#### 4. 开发配置文件

```yaml
# backend/configs/config.dev.yaml
server:
  host: "0.0.0.0"
  port: 8080
  debug: true

database:
  host: "localhost"
  port: 3306
  username: "root"
  password: "password"
  dbname: "taskc_dev"
  debug: true

redis:
  host: "localhost"
  port: 6379
  password: ""
  db: 0

log:
  level: "debug"
  format: "console"
```

## 项目结构

### 后端目录结构

```
backend/
├── cmd/                    # 应用入口
│   ├── server/            # 主服务器
│   ├── migrate/           # 数据库迁移工具
│   └── worker/            # 后台任务处理器
├── internal/              # 内部包
│   ├── api/              # HTTP API处理层
│   │   ├── handler/      # 请求处理器
│   │   ├── middleware/   # 中间件
│   │   └── router/       # 路由配置
│   ├── service/          # 业务逻辑层
│   │   ├── heartbeat/    # 心跳服务
│   │   ├── task/         # 任务管理
│   │   ├── alert/        # 告警服务
│   │   └── probe/        # 探测服务
│   ├── repository/       # 数据访问层
│   │   ├── mysql/        # MySQL实现
│   │   └── redis/        # Redis实现
│   └── model/           # 数据模型
├── pkg/                  # 公共包
│   ├── logger/          # 日志组件
│   ├── config/          # 配置管理
│   ├── database/        # 数据库连接
│   ├── cache/           # 缓存组件
│   └── utils/           # 工具函数
├── configs/             # 配置文件
├── migrations/          # 数据库迁移文件
├── docs/               # API文档
└── tests/              # 测试文件
```

### 前端目录结构

```
frontend/
├── public/              # 静态资源
├── src/
│   ├── components/      # 通用组件
│   │   ├── Layout/     # 布局组件
│   │   ├── Charts/     # 图表组件
│   │   └── Forms/      # 表单组件
│   ├── pages/          # 页面组件
│   │   ├── Dashboard/  # 仪表板
│   │   ├── TaskList/   # 任务列表
│   │   ├── AlertList/  # 告警列表
│   │   └── Settings/   # 设置页面
│   ├── store/          # 状态管理
│   │   ├── taskStore.ts
│   │   └── dashboardStore.ts
│   ├── api/            # API客户端
│   ├── hooks/          # 自定义Hooks
│   ├── utils/          # 工具函数
│   ├── types/          # TypeScript类型定义
│   └── styles/         # 样式文件
├── package.json
└── vite.config.ts
```

## 开发规范

### Go代码规范

#### 1. 命名约定

```go
// 包名使用小写
package heartbeat

// 公共函数使用大驼峰
func ProcessHeartbeat(taskID string) error {}

// 私有函数使用小驼峰
func validateHeartbeat(data *HeartbeatData) error {}

// 常量使用大写加下划线
const MAX_RETRY_COUNT = 3

// 接口名以-er结尾
type HeartbeatProcessor interface {
    Process(data *HeartbeatData) error
}
```

#### 2. 错误处理

```go
// 使用自定义错误类型
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation failed for field %s: %s", e.Field, e.Message)
}

// 错误包装
func processTask(taskID string) error {
    task, err := repo.GetTask(taskID)
    if err != nil {
        return fmt.Errorf("failed to get task %s: %w", taskID, err)
    }
    return nil
}
```

#### 3. 日志记录

```go
import "github.com/taskc/pkg/logger"

func ProcessHeartbeat(ctx context.Context, data *HeartbeatData) error {
    log := logger.FromContext(ctx).With(
        "task_id", data.TaskID,
        "timestamp", data.Timestamp,
    )
    
    log.Info("processing heartbeat")
    
    if err := validateData(data); err != nil {
        log.Error("heartbeat validation failed", "error", err)
        return err
    }
    
    log.Info("heartbeat processed successfully")
    return nil
}
```

#### 4. 单元测试

```go
func TestProcessHeartbeat(t *testing.T) {
    tests := []struct {
        name    string
        input   *HeartbeatData
        wantErr bool
    }{
        {
            name: "valid heartbeat",
            input: &HeartbeatData{
                TaskID:    "task-001",
                Timestamp: time.Now().Unix(),
                Status:    "healthy",
            },
            wantErr: false,
        },
        {
            name: "invalid task id",
            input: &HeartbeatData{
                TaskID:    "",
                Timestamp: time.Now().Unix(),
                Status:    "healthy",
            },
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ProcessHeartbeat(context.Background(), tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("ProcessHeartbeat() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### TypeScript代码规范

#### 1. 类型定义

```typescript
// src/types/index.ts
export interface Task {
  id: string;
  name: string;
  status: TaskStatus;
  createdAt: string;
  lastHeartbeat?: string;
}

export type TaskStatus = 'HEALTHY' | 'SUSPECTED' | 'FAILED';

export interface HeartbeatData {
  taskId: string;
  timestamp: number;
  metadata: {
    cpuLoad: number;
    memUsedMb: number;
    queueSize: number;
    [key: string]: unknown;
  };
}
```

#### 2. React组件规范

```typescript
// src/components/TaskCard/TaskCard.tsx
import React from 'react';
import { Card, Tag, Typography } from 'antd';
import { Task } from '@/types';
import './TaskCard.css';

interface TaskCardProps {
  task: Task;
  onStatusClick?: (taskId: string) => void;
}

export const TaskCard: React.FC<TaskCardProps> = ({ task, onStatusClick }) => {
  const handleStatusClick = () => {
    onStatusClick?.(task.id);
  };

  const getStatusColor = (status: Task['status']) => {
    switch (status) {
      case 'HEALTHY': return 'green';
      case 'SUSPECTED': return 'orange';
      case 'FAILED': return 'red';
      default: return 'default';
    }
  };

  return (
    <Card 
      className="task-card"
      title={task.name}
      extra={
        <Tag 
          color={getStatusColor(task.status)}
          onClick={handleStatusClick}
          style={{ cursor: 'pointer' }}
        >
          {task.status}
        </Tag>
      }
    >
      <Typography.Text type="secondary">
        Last heartbeat: {task.lastHeartbeat || 'Never'}
      </Typography.Text>
    </Card>
  );
};
```

#### 3. 状态管理

```typescript
// src/store/taskStore.ts
import { create } from 'zustand';
import { Task } from '@/types';
import { taskAPI } from '@/api';

interface TaskState {
  tasks: Task[];
  loading: boolean;
  error: string | null;
  
  // Actions
  fetchTasks: () => Promise<void>;
  addTask: (task: Omit<Task, 'id'>) => Promise<void>;
  updateTaskStatus: (taskId: string, status: Task['status']) => void;
}

export const useTaskStore = create<TaskState>((set, get) => ({
  tasks: [],
  loading: false,
  error: null,

  fetchTasks: async () => {
    set({ loading: true, error: null });
    try {
      const tasks = await taskAPI.getTasks();
      set({ tasks, loading: false });
    } catch (error) {
      set({ 
        error: error instanceof Error ? error.message : 'Unknown error',
        loading: false 
      });
    }
  },

  addTask: async (taskData) => {
    try {
      const newTask = await taskAPI.createTask(taskData);
      set(state => ({ 
        tasks: [...state.tasks, newTask] 
      }));
    } catch (error) {
      set({ 
        error: error instanceof Error ? error.message : 'Failed to create task' 
      });
    }
  },

  updateTaskStatus: (taskId, status) => {
    set(state => ({
      tasks: state.tasks.map(task => 
        task.id === taskId ? { ...task, status } : task
      )
    }));
  },
}));
```

## API设计规范

### RESTful API设计

```go
// internal/api/handler/task.go
type TaskHandler struct {
    taskService service.TaskService
}

// GET /api/v1/tasks
func (h *TaskHandler) GetTasks(c *fiber.Ctx) error {
    ctx := c.Context()
    
    // 解析查询参数
    params := struct {
        Status string `query:"status"`
        Page   int    `query:"page" default:"1"`
        Size   int    `query:"size" default:"20"`
    }{}
    
    if err := c.QueryParser(&params); err != nil {
        return c.Status(400).JSON(fiber.Map{
            "error": "invalid query parameters",
        })
    }
    
    // 业务逻辑处理
    tasks, total, err := h.taskService.GetTasks(ctx, service.GetTasksParams{
        Status: params.Status,
        Page:   params.Page,
        Size:   params.Size,
    })
    if err != nil {
        return c.Status(500).JSON(fiber.Map{
            "error": "failed to get tasks",
        })
    }
    
    // 返回响应
    return c.JSON(fiber.Map{
        "data": tasks,
        "pagination": fiber.Map{
            "page":  params.Page,
            "size":  params.Size,
            "total": total,
            "pages": (total + params.Size - 1) / params.Size,
        },
    })
}
```

### 响应格式标准

```go
// 成功响应
{
  "data": { /* 业务数据 */ },
  "meta": {
    "timestamp": "2023-11-05T08:15:30Z",
    "request_id": "req_123456"
  }
}

// 错误响应
{
  "error": {
    "code": 400,
    "message": "Validation failed",
    "details": [
      {
        "field": "task_id",
        "message": "task_id is required"
      }
    ]
  },
  "meta": {
    "timestamp": "2023-11-05T08:15:30Z", 
    "request_id": "req_123456"
  }
}
```

## 测试策略

### 后端测试

#### 单元测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/service/heartbeat

# 生成测试覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

#### 集成测试

```go
// tests/integration/heartbeat_test.go
func TestHeartbeatEndToEnd(t *testing.T) {
    // 启动测试数据库
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)
    
    // 启动测试服务器
    app := setupTestApp(db)
    
    // 发送心跳请求
    req := httptest.NewRequest("POST", "/api/v1/heartbeat/task-001", 
        strings.NewReader(`{"timestamp": 1699174530, "status": "healthy"}`))
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := app.Test(req)
    require.NoError(t, err)
    require.Equal(t, 202, resp.StatusCode)
    
    // 验证数据库状态
    var heartbeat model.Heartbeat
    err = db.Where("task_id = ?", "task-001").First(&heartbeat).Error
    require.NoError(t, err)
    assert.Equal(t, "healthy", heartbeat.Status)
}
```

### 前端测试

#### 组件测试

```typescript
// src/components/TaskCard/TaskCard.test.tsx
import { render, screen, fireEvent } from '@testing-library/react';
import { TaskCard } from './TaskCard';
import { Task } from '@/types';

const mockTask: Task = {
  id: 'task-001',
  name: 'Test Task',
  status: 'HEALTHY',
  createdAt: '2023-11-05T08:15:30Z',
  lastHeartbeat: '2023-11-05T08:15:30Z',
};

describe('TaskCard', () => {
  it('renders task information correctly', () => {
    render(<TaskCard task={mockTask} />);
    
    expect(screen.getByText('Test Task')).toBeInTheDocument();
    expect(screen.getByText('HEALTHY')).toBeInTheDocument();
    expect(screen.getByText(/Last heartbeat/)).toBeInTheDocument();
  });

  it('calls onStatusClick when status tag is clicked', () => {
    const mockOnStatusClick = jest.fn();
    render(<TaskCard task={mockTask} onStatusClick={mockOnStatusClick} />);
    
    fireEvent.click(screen.getByText('HEALTHY'));
    expect(mockOnStatusClick).toHaveBeenCalledWith('task-001');
  });
});
```

#### E2E测试

```typescript
// cypress/e2e/dashboard.cy.ts
describe('Dashboard', () => {
  beforeEach(() => {
    cy.visit('/dashboard');
  });

  it('displays task statistics', () => {
    cy.get('[data-testid="total-tasks"]').should('contain', '0');
    cy.get('[data-testid="healthy-tasks"]').should('contain', '0');
    cy.get('[data-testid="failed-tasks"]').should('contain', '0');
  });

  it('can create a new task', () => {
    cy.get('[data-testid="add-task-button"]').click();
    cy.get('[data-testid="task-name-input"]').type('New Test Task');
    cy.get('[data-testid="submit-button"]').click();
    
    cy.get('.task-card').should('contain', 'New Test Task');
  });
});
```

## Git工作流

### 分支策略

```bash
main          # 主分支，用于生产环境
├── develop   # 开发分支，用于测试环境  
├── feature/* # 功能分支
├── hotfix/*  # 热修复分支
└── release/* # 发布分支
```

### 提交规范

```bash
# 提交信息格式
<type>(<scope>): <subject>

# 类型说明
feat:     新功能
fix:      bug修复  
docs:     文档更新
style:    代码格式化
refactor: 重构
test:     测试相关
chore:    构建/工具链相关

# 示例
feat(heartbeat): add heartbeat validation
fix(alert): resolve email notification bug
docs(api): update API documentation
```

### Pull Request流程

1. **创建功能分支**
```bash
git checkout -b feature/add-probe-timeout
```

2. **开发和提交**
```bash
git add .
git commit -m "feat(probe): add timeout configuration"
git push origin feature/add-probe-timeout
```

3. **创建Pull Request**
- 标题要清晰描述改动
- 在描述中说明改动原因和测试情况
- 关联相关Issue
- 添加适当的标签

4. **代码审查**
- 至少需要一人审查
- 通过所有自动化检查
- 解决审查意见

5. **合并代码**
```bash
# 使用squash merge保持历史清洁
git checkout develop
git pull origin develop
git merge --squash feature/add-probe-timeout
git commit -m "feat(probe): add timeout configuration"
git push origin develop
```

## 性能优化

### 后端优化

```go
// 使用连接池
func setupDatabase() *gorm.DB {
    db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})
    sqlDB, _ := db.DB()
    
    sqlDB.SetMaxIdleConns(10)
    sqlDB.SetMaxOpenConns(100)
    sqlDB.SetConnMaxLifetime(time.Hour)
    
    return db
}

// 批量处理
func (r *HeartbeatRepository) BatchInsert(heartbeats []model.Heartbeat) error {
    return r.db.CreateInBatches(heartbeats, 100).Error
}

// 缓存查询结果
func (s *TaskService) GetTaskStatus(ctx context.Context, taskID string) (string, error) {
    cacheKey := fmt.Sprintf("task:status:%s", taskID)
    
    if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
        return cached, nil
    }
    
    status, err := s.repo.GetTaskStatus(ctx, taskID)
    if err != nil {
        return "", err
    }
    
    s.cache.Set(ctx, cacheKey, status, 5*time.Minute)
    return status, nil
}
```

### 前端优化

```typescript
// 使用React.memo优化渲染
export const TaskCard = React.memo<TaskCardProps>(({ task, onStatusClick }) => {
  // 组件逻辑
});

// 使用useMemo优化计算
const Dashboard: React.FC = () => {
  const { tasks } = useTaskStore();
  
  const statistics = useMemo(() => ({
    total: tasks.length,
    healthy: tasks.filter(t => t.status === 'HEALTHY').length,
    failed: tasks.filter(t => t.status === 'FAILED').length,
  }), [tasks]);
  
  return (
    <div>
      <StatisticsCard data={statistics} />
      {/* 其他内容 */}
    </div>
  );
};

// 使用虚拟化处理大列表
import { FixedSizeList as List } from 'react-window';

const TaskList: React.FC = () => {
  const { tasks } = useTaskStore();
  
  const Row = ({ index, style }) => (
    <div style={style}>
      <TaskCard task={tasks[index]} />
    </div>
  );
  
  return (
    <List
      height={600}
      itemCount={tasks.length}
      itemSize={120}
    >
      {Row}
    </List>
  );
};
```

## 调试技巧

### 后端调试

```go
// 使用delve调试器
// 安装: go install github.com/go-delve/delve/cmd/dlv@latest
// 运行: dlv debug cmd/server/main.go

// 日志调试
func ProcessHeartbeat(data *HeartbeatData) error {
    log.Debug("processing heartbeat", 
        "task_id", data.TaskID,
        "payload_size", len(data.Metadata))
    
    // 业务逻辑
    
    log.Debug("heartbeat processed",
        "duration_ms", time.Since(start).Milliseconds())
}

// 性能分析
import _ "net/http/pprof"

func main() {
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
    
    // 启动主服务
}
```

### 前端调试

```typescript
// React Developer Tools
// Chrome扩展安装后可以查看组件树和状态

// 使用console.log调试
const TaskCard: React.FC<TaskCardProps> = ({ task }) => {
  console.log('TaskCard render:', { task });
  
  useEffect(() => {
    console.log('Task status changed:', task.status);
  }, [task.status]);
};

// 使用Redux DevTools (如果使用Redux)
// 安装Chrome扩展后可以查看状态变化

// 错误边界
class ErrorBoundary extends React.Component {
  constructor(props) {
    super(props);
    this.state = { hasError: false };
  }
  
  static getDerivedStateFromError(error) {
    return { hasError: true };
  }
  
  componentDidCatch(error, errorInfo) {
    console.error('Error caught by boundary:', error, errorInfo);
  }
  
  render() {
    if (this.state.hasError) {
      return <h1>Something went wrong.</h1>;
    }
    return this.props.children;
  }
}
```

## 贡献流程

1. **Fork项目** - 点击GitHub上的Fork按钮
2. **创建功能分支** - `git checkout -b feature/new-feature`
3. **编写代码** - 遵循代码规范
4. **编写测试** - 确保测试覆盖率
5. **运行测试** - 确保所有测试通过
6. **提交代码** - 使用规范的提交信息
7. **创建PR** - 详细描述变更内容
8. **代码审查** - 响应审查意见
9. **合并代码** - 审查通过后合并