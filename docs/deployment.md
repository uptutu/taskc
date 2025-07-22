# 部署运维指南

本文档介绍TaskC在生产环境中的部署、配置和运维最佳实践。

## 生产环境部署

### 系统要求

**最低配置**:
- CPU: 4核
- 内存: 8GB
- 硬盘: 100GB SSD
- 网络: 1Gbps

**推荐配置**:
- CPU: 8核
- 内存: 16GB  
- 硬盘: 200GB SSD
- 网络: 10Gbps

**操作系统**:
- Ubuntu 20.04+ / CentOS 8+ / RHEL 8+
- Docker 20.10+
- Docker Compose 2.0+

### 架构部署

#### 单机部署

```yaml
# docker-compose.prod.yml
version: '3.8'

services:
  mysql:
    image: mysql:8.0
    restart: always
    environment:
      MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD}
      MYSQL_DATABASE: taskc
    volumes:
      - mysql_data:/var/lib/mysql
      - ./mysql/conf.d:/etc/mysql/conf.d
    ports:
      - "3306:3306"
    command: --default-authentication-plugin=mysql_native_password

  redis:
    image: redis:7-alpine
    restart: always
    command: redis-server --requirepass ${REDIS_PASSWORD}
    volumes:
      - redis_data:/data
    ports:
      - "6379:6379"

  backend:
    image: taskc-backend:latest
    restart: always
    depends_on:
      - mysql
      - redis
    environment:
      - DB_HOST=mysql
      - DB_PASSWORD=${MYSQL_ROOT_PASSWORD}
      - REDIS_HOST=redis
      - REDIS_PASSWORD=${REDIS_PASSWORD}
    volumes:
      - ./configs:/app/configs
      - logs:/app/logs
    ports:
      - "8080:8080"

  frontend:
    image: taskc-frontend:latest
    restart: always
    depends_on:
      - backend
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/ssl:/etc/nginx/ssl

volumes:
  mysql_data:
  redis_data:
  logs:
```

#### 集群部署

```yaml
# kubernetes/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: taskc-backend
spec:
  replicas: 3
  selector:
    matchLabels:
      app: taskc-backend
  template:
    metadata:
      labels:
        app: taskc-backend
    spec:
      containers:
      - name: backend
        image: taskc-backend:latest
        ports:
        - containerPort: 8080
        env:
        - name: DB_HOST
          value: "mysql-service"
        - name: REDIS_HOST
          value: "redis-service"
        resources:
          requests:
            memory: "512Mi"
            cpu: "250m"
          limits:
            memory: "1Gi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
```

### 环境变量配置

```bash
# .env.production
# 数据库配置
MYSQL_ROOT_PASSWORD=your_strong_password
DB_HOST=mysql
DB_PORT=3306
DB_NAME=taskc
DB_USER=root

# Redis配置  
REDIS_PASSWORD=your_redis_password
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_DB=0

# 应用配置
APP_ENV=production
APP_DEBUG=false
LOG_LEVEL=info
JWT_SECRET=your_jwt_secret_key

# 告警配置
SMTP_HOST=smtp.company.com
SMTP_USER=alerts@company.com
SMTP_PASSWORD=email_password
TWILIO_ACCOUNT_SID=your_twilio_sid
TWILIO_AUTH_TOKEN=your_twilio_token
SLACK_WEBHOOK_URL=https://hooks.slack.com/your/webhook

# SSL证书配置
SSL_CERT_PATH=/etc/ssl/certs/taskc.crt
SSL_KEY_PATH=/etc/ssl/private/taskc.key
```

## 数据库优化

### MySQL配置优化

```ini
# /etc/mysql/conf.d/taskc.cnf
[mysqld]
# 基础配置
default-storage-engine = InnoDB
character-set-server = utf8mb4
collation-server = utf8mb4_unicode_ci

# 性能优化
innodb_buffer_pool_size = 4G
innodb_log_file_size = 256M
innodb_log_buffer_size = 16M
innodb_flush_log_at_trx_commit = 2
innodb_flush_method = O_DIRECT

# 连接配置
max_connections = 1000
max_connect_errors = 100000
wait_timeout = 300
interactive_timeout = 300

# 查询缓存
query_cache_type = 1
query_cache_size = 64M
query_cache_limit = 2M

# 慢查询日志
slow_query_log = 1
slow_query_log_file = /var/log/mysql/slow.log
long_query_time = 2
```

### 索引优化

```sql
-- 心跳表索引
CREATE INDEX idx_heartbeat_task_time ON heartbeats(task_id, created_at);
CREATE INDEX idx_heartbeat_status ON heartbeats(status, created_at);

-- 任务表索引  
CREATE INDEX idx_task_status ON tasks(status);
CREATE INDEX idx_task_created ON tasks(created_at);
CREATE INDEX idx_task_name ON tasks(name);

-- 告警表索引
CREATE INDEX idx_alert_task_time ON alerts(task_id, created_at);
CREATE INDEX idx_alert_severity ON alerts(severity, created_at);
CREATE INDEX idx_alert_status ON alerts(status);

-- 日志表分区
ALTER TABLE logs PARTITION BY RANGE (UNIX_TIMESTAMP(created_at)) (
  PARTITION p202311 VALUES LESS THAN (UNIX_TIMESTAMP('2023-12-01')),
  PARTITION p202312 VALUES LESS THAN (UNIX_TIMESTAMP('2024-01-01')),
  PARTITION p202401 VALUES LESS THAN (UNIX_TIMESTAMP('2024-02-01'))
);
```

## 缓存优化

### Redis配置

```conf
# redis.conf
# 内存配置
maxmemory 2gb
maxmemory-policy allkeys-lru

# 持久化配置
save 900 1
save 300 10
save 60 10000

# 网络配置
tcp-keepalive 300
timeout 300

# 日志配置
logfile /var/log/redis/redis-server.log
loglevel notice

# 安全配置
requirepass your_strong_password
rename-command FLUSHDB ""
rename-command FLUSHALL ""
```

### 缓存策略

```go
// 心跳数据缓存策略
type HeartbeatCache struct {
    // 最近心跳缓存 - 1分钟TTL
    recent_heartbeats: "heartbeat:recent:{task_id}" // TTL: 60s
    
    // 任务状态缓存 - 5分钟TTL  
    task_status: "status:{task_id}" // TTL: 300s
    
    // 统计数据缓存 - 15分钟TTL
    statistics: "stats:dashboard" // TTL: 900s
}
```

## 监控和告警

### Prometheus配置

```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  
scrape_configs:
  - job_name: 'taskc-backend'
    static_configs:
      - targets: ['taskc-backend:8080']
    metrics_path: /metrics
    scrape_interval: 5s
    
  - job_name: 'mysql'
    static_configs:
      - targets: ['mysql-exporter:9104']
      
  - job_name: 'redis'
    static_configs:
      - targets: ['redis-exporter:9121']
```

### Grafana Dashboard

```json
{
  "dashboard": {
    "title": "TaskC监控面板",
    "panels": [
      {
        "title": "任务状态分布",
        "type": "piechart",
        "targets": [{
          "expr": "sum by (status) (taskc_task_status_total)"
        }]
      },
      {
        "title": "心跳接收速率",
        "type": "graph", 
        "targets": [{
          "expr": "rate(taskc_heartbeat_received_total[5m])"
        }]
      },
      {
        "title": "告警发送统计",
        "type": "stat",
        "targets": [{
          "expr": "increase(taskc_alerts_sent_total[1h])"
        }]
      }
    ]
  }
}
```

### 健康检查

```yaml
# docker-compose健康检查
healthcheck:
  test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 40s
```

## 备份和恢复

### 数据备份策略

```bash
#!/bin/bash
# backup.sh - 每日数据备份脚本

BACKUP_DIR="/data/backups"
DATE=$(date +%Y%m%d_%H%M%S)

# MySQL备份
mysqldump -h mysql -u root -p${MYSQL_ROOT_PASSWORD} \
  --single-transaction --routines --triggers \
  taskc > ${BACKUP_DIR}/mysql_${DATE}.sql

# Redis备份
redis-cli -h redis -a ${REDIS_PASSWORD} --rdb ${BACKUP_DIR}/redis_${DATE}.rdb

# 压缩备份文件
tar -czf ${BACKUP_DIR}/taskc_backup_${DATE}.tar.gz \
  ${BACKUP_DIR}/mysql_${DATE}.sql \
  ${BACKUP_DIR}/redis_${DATE}.rdb

# 删除7天前的备份
find ${BACKUP_DIR} -name "taskc_backup_*.tar.gz" -mtime +7 -delete

# 上传到云存储
aws s3 cp ${BACKUP_DIR}/taskc_backup_${DATE}.tar.gz \
  s3://company-backups/taskc/
```

### 恢复流程

```bash
# 1. 停止服务
docker-compose down

# 2. 恢复MySQL数据
docker run --rm -v mysql_data:/var/lib/mysql \
  -v /data/backups:/backup \
  mysql:8.0 sh -c "
  mysql -h mysql -u root -p${MYSQL_ROOT_PASSWORD} < /backup/mysql_backup.sql
"

# 3. 恢复Redis数据
docker run --rm -v redis_data:/data \
  -v /data/backups:/backup \
  redis:7-alpine sh -c "
  cp /backup/redis_backup.rdb /data/dump.rdb
"

# 4. 重启服务
docker-compose up -d
```

## 日志管理

### 日志轮转配置

```bash
# /etc/logrotate.d/taskc
/var/log/taskc/*.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    create 0644 taskc taskc
    postrotate
        docker kill -s USR1 taskc-backend
    endscript
}
```

### 集中日志收集

```yaml
# filebeat.yml
filebeat.inputs:
- type: container
  paths:
    - '/var/lib/docker/containers/*/*.log'
  processors:
  - add_docker_metadata:
      host: "unix:///var/run/docker.sock"

output.elasticsearch:
  hosts: ["elasticsearch:9200"]
  index: "taskc-logs-%{+yyyy.MM.dd}"
```

## 性能调优

### 应用层优化

```go
// 连接池配置
db.SetMaxOpenConns(100)
db.SetMaxIdleConns(20)
db.SetConnMaxLifetime(time.Hour)

// Redis连接池
redis := &redis.Pool{
    MaxIdle:     20,
    MaxActive:   100,
    IdleTimeout: 240 * time.Second,
}
```

### 系统层优化

```bash
# /etc/sysctl.conf
# 网络优化
net.core.somaxconn = 65535
net.ipv4.tcp_max_syn_backlog = 8192
net.ipv4.tcp_tw_reuse = 1

# 文件描述符限制
fs.file-max = 1000000

# 内存配置
vm.swappiness = 10
vm.dirty_ratio = 15
```

## 安全配置

### SSL/TLS配置

```nginx
# nginx SSL配置
server {
    listen 443 ssl http2;
    server_name taskc.company.com;
    
    ssl_certificate /etc/ssl/certs/taskc.crt;
    ssl_certificate_key /etc/ssl/private/taskc.key;
    
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512;
    ssl_prefer_server_ciphers off;
    
    # HSTS
    add_header Strict-Transport-Security "max-age=63072000" always;
    
    # 其他安全头
    add_header X-Content-Type-Options nosniff;
    add_header X-Frame-Options DENY;
    add_header X-XSS-Protection "1; mode=block";
}
```

### 防火墙配置

```bash
# UFW防火墙规则
ufw default deny incoming
ufw default allow outgoing
ufw allow 22/tcp    # SSH
ufw allow 80/tcp    # HTTP
ufw allow 443/tcp   # HTTPS
ufw allow from 10.0.0.0/8 to any port 3306  # MySQL内网访问
ufw enable
```

## 故障排除

### 常见问题

1. **数据库连接超时**
```bash
# 检查连接数
SHOW PROCESSLIST;
SHOW STATUS LIKE 'Threads_connected';

# 优化配置
SET GLOBAL max_connections = 1000;
SET GLOBAL wait_timeout = 300;
```

2. **Redis内存不足**
```bash
# 检查内存使用
redis-cli INFO memory

# 清理过期键
redis-cli --latency-history -i 1
```

3. **心跳处理延迟**
```bash
# 检查队列积压
redis-cli XLEN heartbeat_stream

# 检查处理速度
curl http://localhost:8080/metrics | grep heartbeat_processing_duration
```

### 日志分析

```bash
# 错误日志统计
grep "ERROR" /var/log/taskc/app.log | wc -l

# 慢查询分析
mysqldumpslow /var/log/mysql/slow.log

# 接口响应时间分析  
awk '/api_duration_ms/ {sum+=$NF; count++} END {print sum/count}' /var/log/taskc/access.log
```

## 运维自动化

### 健康检查脚本

```bash
#!/bin/bash
# health_check.sh
set -e

echo "检查TaskC系统健康状态..."

# 检查服务状态
services=("taskc-backend" "taskc-frontend" "taskc-mysql" "taskc-redis")
for service in "${services[@]}"; do
    if ! docker ps | grep -q $service; then
        echo "❌ $service 服务未运行"
        exit 1
    fi
done

# 检查API健康
if ! curl -f http://localhost:8080/health >/dev/null 2>&1; then
    echo "❌ 后端API健康检查失败"
    exit 1
fi

# 检查数据库连接
if ! docker exec taskc-mysql mysqladmin -u root -p${MYSQL_ROOT_PASSWORD} ping >/dev/null 2>&1; then
    echo "❌ 数据库连接失败"
    exit 1
fi

echo "✅ 系统健康检查通过"
```

### 自动部署脚本

```bash
#!/bin/bash
# deploy.sh
set -e

VERSION=$1
if [ -z "$VERSION" ]; then
    echo "用法: $0 <version>"
    exit 1
fi

echo "开始部署TaskC $VERSION..."

# 拉取最新镜像
docker pull taskc-backend:$VERSION
docker pull taskc-frontend:$VERSION

# 停止旧服务
docker-compose down

# 备份数据
./backup.sh

# 启动新版本
VERSION=$VERSION docker-compose up -d

# 健康检查
sleep 30
./health_check.sh

echo "✅ 部署完成"
```