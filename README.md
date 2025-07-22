# TaskC - 分布式任务调度中台

一个高可用、可观测的分布式任务调度和监控平台，支持秒级健康检测和智能告警机制。

## 🚀 核心特性

- **实时健康监控**: <5秒延迟的任务状态感知
- **智能告警系统**: 多通道告警（短信/邮件/Slack），99.99%触达率
- **智能日志管理**: 自动保留策略，存储成本降低40%
- **主动探测机制**: HTTP/TCP/Ping多协议探测
- **可视化面板**: 实时任务状态监控和历史数据分析

## 🏗️ 系统架构

```
任务节点 → 心跳上报 → TaskC中台 → 健康状态引擎 → Redis流处理
                    ↓
               状态判断引擎
                    ↓
            [健康] → MySQL存储
            [异常] → 告警引擎 → 多通道推送
```

**技术栈:**
- **后端**: Go + Fiber框架
- **前端**: React + TypeScript + Ant Design  
- **数据库**: MySQL 8.0 + Redis 7.0
- **部署**: Docker + Docker Compose

## 📦 快速开始

### 环境要求
- Docker & Docker Compose
- Node.js 18+ (开发)
- Go 1.21+ (开发)

### 一键部署
```bash
# 克隆项目
git clone <repository-url>
cd taskc

# 部署完整环境
./scripts/deploy.sh

# 访问前端面板
open http://localhost:3000
```

### 开发模式
```bash
# 启动数据库服务
cd docker && docker-compose up -d mysql redis

# 后端开发
cd backend && go run cmd/server/main.go

# 前端开发  
cd frontend && npm run dev
```

## 📋 核心功能

### 健康监控
- 心跳接收: `POST /api/v1/heartbeat/{task_id}`
- 状态查询: `GET /api/v1/tasks/{task_id}/health`
- 主动探测: `POST /api/v1/probe`

### 告警机制
- **紧急级别**: 短信 + 电话通知
- **重要级别**: Slack + 邮件通知  
- **警告级别**: 控制台展示

### 日志管理
- 默认保留90天，可配置[7-365]天
- 磁盘占用>85%时自动清理
- 审计日志永久保留

## 🔧 配置说明

主要配置文件: `backend/configs/config.yaml`

```yaml
heartbeat:
  check_interval: 5s        # 健康检查间隔
  failure_threshold: 3      # 失败阈值

alert:
  channels: ["sms", "email", "slack"]
  rate_limit: 30           # 短信速率限制(条/分钟)
  
log:
  retention_days: 90       # 日志保留天数
  cleanup_time: "02:00"    # 清理执行时间
```

## 📊 监控指标

| 指标名称 | 类型 | 说明 |
|---------|------|------|
| task_heartbeat_received | counter | 心跳接收次数 |
| task_probe_latency_ms | gauge | 探测延迟 |
| task_status | enum | 任务状态(0:健康 1:疑似 2:故障) |
| task_resource_cpu | gauge | CPU占用百分比 |

## 📚 详细文档

复杂的配置和使用细节请参考以下文档：

- [API接口文档](docs/api.md) - 完整的接口规范和示例
- [探测配置指南](docs/probe-config.md) - 主动探测配置详解
- [告警配置指南](docs/alert-config.md) - 多通道告警配置
- [部署运维指南](docs/deployment.md) - 生产环境部署最佳实践
- [开发指南](docs/development.md) - 开发环境搭建和贡献指南

## 🤝 贡献指南

1. Fork项目
2. 创建特性分支: `git checkout -b feature/new-feature`
3. 提交更改: `git commit -am 'Add new feature'`
4. 推送分支: `git push origin feature/new-feature`
5. 提交Pull Request

## 📄 许可证

本项目采用MIT许可证 - 详见[LICENSE](LICENSE)文件

## 🆘 支持

如遇问题，请通过以下方式获取帮助：
- 提交[Issue](../../issues)
- 查看[Wiki文档](../../wiki)
- 联系维护团队