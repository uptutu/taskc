# API接口文档

## 认证方式

所有API请求需要在Header中包含认证信息：

```http
Authorization: Bearer <token>
Content-Type: application/json
```

## 心跳管理

### 提交心跳

**接口**: `POST /api/v1/heartbeat/{task_id}`

**请求参数**:
```json
{
  "timestamp": 1630000000000,
  "metadata": {
    "cpu_load": 0.72,
    "mem_used_mb": 1024,
    "queue_size": 15,
    "custom_fields": {
      "version": "1.0.0",
      "environment": "production"
    }
  }
}
```

**响应**:
```http
HTTP/1.1 202 Accepted
X-Task-Status: Healthy | Suspected | Failed
X-Task-Last-Check: 2023-11-05T08:15:30Z

{
  "status": "accepted",
  "task_status": "HEALTHY",
  "next_heartbeat_expected": "2023-11-05T08:15:35Z"
}
```

### 查询任务健康状态

**接口**: `GET /api/v1/tasks/{task_id}/health`

**响应**:
```json
{
  "task_id": "task_001",
  "status": "HEALTHY",
  "last_heartbeat": "2023-11-05T08:15:30Z",
  "missed_beats": 0,
  "probe_history": [
    {
      "timestamp": "2023-11-05T08:14:25Z",
      "type": "HTTP_GET",
      "result": "SUCCESS",
      "latency_ms": 48,
      "details": {
        "status_code": 200,
        "response_size": 156
      }
    }
  ],
  "resource_usage": {
    "avg_cpu": 32.5,
    "max_mem_mb": 1024,
    "avg_queue_size": 12.3
  },
  "uptime_percentage": 99.95
}
```

## 主动探测

### 触发探测

**接口**: `POST /api/v1/probe`

**HTTP探测请求**:
```json
{
  "task_id": "task_001",
  "type": "HTTP_GET",
  "config": {
    "endpoint": "http://127.0.0.1:8080/health",
    "timeout_ms": 1500,
    "expected_status": 200,
    "headers": {
      "User-Agent": "TaskC-Probe/1.0"
    },
    "success_conditions": [
      "status_code == 200",
      "response.body.contains('OK')"
    ]
  }
}
```

**TCP探测请求**:
```json
{
  "task_id": "task_002", 
  "type": "TCP_CONNECT",
  "config": {
    "host": "127.0.0.1",
    "port": 8080,
    "timeout_ms": 2000
  }
}
```

**Ping探测请求**:
```json
{
  "task_id": "task_003",
  "type": "PING",
  "config": {
    "host": "127.0.0.1", 
    "count": 3,
    "timeout_ms": 1000,
    "packet_size": 32
  }
}
```

**响应**:
```json
{
  "probe_id": "probe_123456",
  "status": "STARTED",
  "estimated_completion": "2023-11-05T08:15:32Z"
}
```

### 查询探测结果

**接口**: `GET /api/v1/probe/{probe_id}`

**响应**:
```json
{
  "probe_id": "probe_123456",
  "task_id": "task_001",
  "type": "HTTP_GET",
  "status": "COMPLETED",
  "result": "SUCCESS",
  "start_time": "2023-11-05T08:15:30Z",
  "end_time": "2023-11-05T08:15:32Z",
  "latency_ms": 245,
  "details": {
    "status_code": 200,
    "response_headers": {
      "content-type": "application/json"
    },
    "response_body": "{\"status\":\"OK\"}",
    "dns_resolve_ms": 5,
    "tcp_connect_ms": 12,
    "tls_handshake_ms": 45,
    "response_time_ms": 183
  }
}
```

## 任务管理

### 创建任务

**接口**: `POST /api/v1/tasks`

**请求**:
```json
{
  "name": "Web服务监控",
  "description": "监控Web服务健康状态",
  "heartbeat_config": {
    "interval": 30,
    "timeout": 90,
    "max_missed": 3
  },
  "probe_config": {
    "enabled": true,
    "type": "HTTP_GET",
    "config": {
      "endpoint": "http://service.example.com/health",
      "timeout_ms": 5000
    }
  },
  "alert_config": {
    "enabled": true,
    "channels": ["email", "slack"],
    "severity": "CRITICAL"
  }
}
```

### 获取任务列表

**接口**: `GET /api/v1/tasks`

**查询参数**:
- `status`: 过滤状态 (HEALTHY|SUSPECTED|FAILED)
- `page`: 页码 (默认1)
- `size`: 每页数量 (默认20)
- `sort`: 排序字段 (name|created_at|last_heartbeat)

**响应**:
```json
{
  "tasks": [
    {
      "id": "task_001",
      "name": "Web服务监控", 
      "status": "HEALTHY",
      "created_at": "2023-11-01T10:00:00Z",
      "last_heartbeat": "2023-11-05T08:15:30Z"
    }
  ],
  "pagination": {
    "page": 1,
    "size": 20,
    "total": 45,
    "pages": 3
  }
}
```

## 告警管理

### 获取告警列表

**接口**: `GET /api/v1/alerts`

**响应**:
```json
{
  "alerts": [
    {
      "id": "alert_001",
      "task_id": "task_001",
      "task_name": "Web服务监控",
      "severity": "CRITICAL",
      "message": "任务已故障",
      "created_at": "2023-11-05T08:10:00Z",
      "acknowledged": false,
      "channels_sent": ["email", "slack"],
      "retry_count": 2
    }
  ]
}
```

### 确认告警

**接口**: `PUT /api/v1/alerts/{alert_id}/acknowledge`

**请求**:
```json
{
  "message": "已知悉，正在处理"
}
```

## 系统信息

### 获取系统状态

**接口**: `GET /api/v1/system/health`

**响应**:
```json
{
  "status": "HEALTHY",
  "version": "1.0.0",
  "uptime": 86400,
  "components": {
    "database": {
      "status": "HEALTHY",
      "connections": 15,
      "latency_ms": 2.3
    },
    "redis": {
      "status": "HEALTHY", 
      "connections": 8,
      "memory_usage": "45MB"
    }
  },
  "statistics": {
    "tasks_total": 120,
    "tasks_healthy": 115,
    "tasks_suspected": 3,
    "tasks_failed": 2,
    "heartbeats_last_hour": 14400,
    "alerts_last_24h": 5
  }
}
```

## 错误码说明

| 错误码 | 说明 |
|-------|------|
| 400 | 请求参数错误 |
| 401 | 认证失败 |
| 403 | 权限不足 |
| 404 | 资源不存在 |
| 409 | 资源冲突 |
| 429 | 请求频率超限 |
| 500 | 服务器内部错误 |
| 503 | 服务暂不可用 |

**错误响应格式**:
```json
{
  "error": {
    "code": 400,
    "message": "参数validation失败",
    "details": [
      {
        "field": "timeout_ms",
        "message": "必须大于0且小于60000"
      }
    ],
    "request_id": "req_123456"
  }
}
```