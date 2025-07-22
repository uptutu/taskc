# 探测配置指南

TaskC支持多种探测协议，用于主动检查任务服务的可用性。当心跳丢失时，系统会自动触发探测来确认任务状态。

## HTTP/HTTPS探测

### 基础配置

```json
{
  "type": "HTTP_GET",
  "config": {
    "endpoint": "http://127.0.0.1:8080/health",
    "timeout_ms": 2000,
    "expected_status": 200,
    "follow_redirects": true,
    "max_redirects": 5
  }
}
```

### 高级配置

```json
{
  "type": "HTTP_POST", 
  "config": {
    "endpoint": "https://api.example.com/healthcheck",
    "timeout_ms": 5000,
    "headers": {
      "Content-Type": "application/json",
      "Authorization": "Bearer token123",
      "X-API-Key": "your-api-key"
    },
    "body": {
      "check_type": "full",
      "components": ["database", "redis", "queue"]
    },
    "tls_config": {
      "verify_cert": true,
      "ca_cert_file": "/path/to/ca.crt",
      "client_cert_file": "/path/to/client.crt", 
      "client_key_file": "/path/to/client.key"
    },
    "success_conditions": [
      "status_code == 200",
      "response.body.contains('healthy')",
      "response.time < 1000",
      "response.headers['content-type'].contains('json')"
    ]
  }
}
```

### 成功条件语法

支持以下条件表达式：

**状态码检查**:
```javascript
status_code == 200
status_code >= 200 && status_code < 300
status_code in [200, 201, 204]
```

**响应体检查**:
```javascript
response.body.contains('OK')
response.body.matches('^{"status":"healthy"}$')
response.body.json.status == 'up'
response.body.json.services.database == true
```

**响应头检查**:
```javascript  
response.headers['server'].contains('nginx')
response.headers['content-length'] > 0
```

**性能检查**:
```javascript
response.time < 2000
response.size < 1048576  // 1MB
```

## TCP连接探测

### 基础TCP探测

```json
{
  "type": "TCP_CONNECT",
  "config": {
    "host": "127.0.0.1",
    "port": 8080,
    "timeout_ms": 3000
  }
}
```

### TCP数据传输探测

```json
{
  "type": "TCP_DATA",
  "config": {
    "host": "redis.example.com",
    "port": 6379,
    "timeout_ms": 5000,
    "send_data": "PING\r\n",
    "expected_response": "+PONG\r\n",
    "close_after_check": true
  }
}
```

## UDP探测

```json
{
  "type": "UDP",
  "config": {
    "host": "dns.example.com",
    "port": 53,
    "timeout_ms": 2000,
    "send_data": "ping_payload",
    "expected_response": "pong_payload"
  }
}
```

## PING探测

### ICMP Ping

```json
{
  "type": "PING",
  "config": {
    "host": "192.168.1.100",
    "count": 3,
    "timeout_ms": 1000,
    "packet_size": 32,
    "interval_ms": 100
  }
}
```

### IPv6支持

```json
{
  "type": "PING",
  "config": {
    "host": "2001:db8::1",
    "ip_version": 6,
    "count": 3,
    "timeout_ms": 2000
  }
}
```

## 数据库探测

### MySQL探测

```json
{
  "type": "MYSQL",
  "config": {
    "host": "mysql.example.com",
    "port": 3306,
    "username": "health_check",
    "password": "check_pass",
    "database": "test_db",
    "timeout_ms": 3000,
    "query": "SELECT 1 as status",
    "expected_result": [{"status": 1}]
  }
}
```

### Redis探测

```json
{
  "type": "REDIS",
  "config": {
    "host": "redis.example.com",
    "port": 6379,
    "password": "redis_pass",
    "database": 0,
    "timeout_ms": 2000,
    "command": "PING",
    "expected_response": "PONG"
  }
}
```

### PostgreSQL探测

```json
{
  "type": "POSTGRESQL",
  "config": {
    "host": "postgres.example.com",
    "port": 5432,
    "username": "health_user",
    "password": "health_pass", 
    "database": "health_db",
    "sslmode": "require",
    "timeout_ms": 5000,
    "query": "SELECT current_timestamp",
    "max_rows": 1
  }
}
```

## 自定义脚本探测

### Shell脚本探测

```json
{
  "type": "SCRIPT",
  "config": {
    "script_type": "shell",
    "script_content": "#!/bin/bash\ncurl -f http://service/health || exit 1",
    "timeout_ms": 10000,
    "working_directory": "/tmp",
    "environment": {
      "SERVICE_URL": "http://127.0.0.1:8080",
      "API_KEY": "your-key"
    },
    "expected_exit_code": 0
  }
}
```

### Python脚本探测

```json
{
  "type": "SCRIPT", 
  "config": {
    "script_type": "python3",
    "script_content": "import requests\nr = requests.get('http://service/health')\nassert r.status_code == 200",
    "timeout_ms": 15000,
    "requirements": ["requests==2.28.0"],
    "expected_exit_code": 0
  }
}
```

## 探测策略配置

### 重试策略

```json
{
  "retry_policy": {
    "max_attempts": 3,
    "backoff_strategy": "exponential",
    "initial_delay_ms": 1000,
    "max_delay_ms": 10000,
    "retry_on_errors": [
      "TIMEOUT",
      "CONNECTION_REFUSED", 
      "DNS_RESOLUTION_FAILED"
    ]
  }
}
```

### 探测调度

```json
{
  "schedule": {
    "trigger_on_heartbeat_miss": true,
    "periodic_probe": {
      "enabled": true,
      "interval": "5m",
      "only_when_suspected": false
    },
    "probe_window": {
      "start_time": "09:00",
      "end_time": "18:00", 
      "timezone": "Asia/Shanghai"
    }
  }
}
```

## 探测结果处理

### 结果解析配置

```json
{
  "result_processing": {
    "success_threshold": 1,
    "failure_threshold": 2,
    "consecutive_checks": true,
    "status_mapping": {
      "0-199": "FAILED",
      "200-299": "HEALTHY", 
      "300-399": "SUSPECTED",
      "400-599": "FAILED"
    }
  }
}
```

### 告警触发规则

```json
{
  "alert_rules": {
    "on_first_failure": false,
    "on_consecutive_failures": 2,
    "on_status_change": true,
    "cooldown_period": "10m",
    "escalation_rules": [
      {
        "after_minutes": 5,
        "severity": "WARNING"
      },
      {
        "after_minutes": 15,
        "severity": "CRITICAL"
      }
    ]
  }
}
```

## 最佳实践

### 1. 探测设计原则

- **轻量级**: 探测检查应该快速且资源消耗小
- **幂等性**: 探测不应对目标服务产生副作用
- **专用端点**: 为健康检查提供专用的轻量级接口

### 2. 超时时间设置

```bash
# 推荐的超时时间设置
TCP探测: 3-5秒
HTTP探测: 5-10秒  
数据库探测: 5-15秒
脚本探测: 10-30秒
```

### 3. 探测频率控制

```json
{
  "frequency_control": {
    "min_interval_between_probes": "30s",
    "max_concurrent_probes": 10,
    "rate_limit_per_target": "1/10s"
  }
}
```

### 4. 错误处理

```json
{
  "error_handling": {
    "ignore_ssl_errors": false,
    "treat_timeout_as_failure": true,
    "log_response_body_on_error": true,
    "max_response_size": "1MB"
  }
}
```

### 5. 安全考虑

- 探测凭据应使用最小权限原则
- 敏感信息使用环境变量或密钥管理服务
- 定期轮换探测用的API密钥
- 限制探测源IP地址