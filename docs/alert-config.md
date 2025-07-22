# 告警配置指南

TaskC提供多通道、分级的告警机制，支持短信、邮件、Slack等多种通知方式，确保故障能够及时触达相关人员。

## 告警通道配置

### 短信通道 (SMS)

#### Twilio集成

```yaml
alert:
  channels:
    sms:
      enabled: true
      provider: twilio
      config:
        account_sid: "${TWILIO_ACCOUNT_SID}"
        auth_token: "${TWILIO_AUTH_TOKEN}" 
        from_number: "+1234567890"
        rate_limit: 30  # 每分钟最大发送数
      recipients:
        - phone: "+86138xxxxxxxx"
          name: "运维负责人"
          severity_levels: ["CRITICAL", "WARNING"]
        - phone: "+86139xxxxxxxx" 
          name: "值班工程师"
          severity_levels: ["CRITICAL"]
```

#### 阿里云短信服务

```yaml
alert:
  channels:
    sms:
      enabled: true
      provider: aliyun
      config:
        access_key_id: "${ALIYUN_ACCESS_KEY}"
        access_key_secret: "${ALIYUN_ACCESS_SECRET}"
        region: "cn-hangzhou"
        sign_name: "任务中台"
        template_code: "SMS_123456789"
        rate_limit: 50
      recipients:
        - phone: "138xxxxxxxx"
          name: "系统管理员"
          severity_levels: ["CRITICAL"]
```

### 邮件通道 (Email)

#### SMTP配置

```yaml
alert:
  channels:
    email:
      enabled: true
      provider: smtp
      config:
        host: "smtp.example.com"
        port: 587
        username: "alerts@company.com"
        password: "${EMAIL_PASSWORD}"
        use_tls: true
        from_address: "alerts@company.com"
        from_name: "TaskC告警系统"
        rate_limit: 100  # 每分钟最大发送数
      recipients:
        - email: "ops-team@company.com"
          name: "运维团队"
          severity_levels: ["CRITICAL", "WARNING", "INFO"]
        - email: "dev-team@company.com"
          name: "开发团队" 
          severity_levels: ["CRITICAL", "WARNING"]
```

#### 企业邮箱配置

```yaml
# 腾讯企业邮箱
email:
  provider: smtp
  config:
    host: "smtp.exmail.qq.com"
    port: 465
    use_ssl: true
    
# 阿里企业邮箱  
email:
  provider: smtp
  config:
    host: "smtp.qiye.aliyun.com"
    port: 465
    use_ssl: true
```

### Slack通道

```yaml
alert:
  channels:
    slack:
      enabled: true
      provider: webhook
      config:
        webhook_url: "${SLACK_WEBHOOK_URL}"
        channel: "#alerts"
        username: "TaskC Bot"
        icon_emoji: ":warning:"
        rate_limit: 20  # 每秒最大发送数
      recipients:
        - channel: "#ops-alerts"
          severity_levels: ["CRITICAL", "WARNING"]
        - channel: "#dev-notifications"
          severity_levels: ["WARNING", "INFO"]
        - user: "@john.doe"
          severity_levels: ["CRITICAL"]
```

### 钉钉通道

```yaml
alert:
  channels:
    dingtalk:
      enabled: true
      provider: webhook
      config:
        webhook_url: "${DINGTALK_WEBHOOK_URL}"
        secret: "${DINGTALK_SECRET}"
        at_mobiles: ["138xxxxxxxx"]
        at_all: false
        rate_limit: 20
```

### 企业微信通道

```yaml
alert:
  channels:
    wechat_work:
      enabled: true
      provider: webhook  
      config:
        webhook_url: "${WECHAT_WORK_WEBHOOK_URL}"
        mentioned_list: ["@all"]
        mentioned_mobile_list: ["138xxxxxxxx"]
```

## 告警级别配置

### 级别定义

```yaml
alert:
  severity_levels:
    CRITICAL:
      priority: 1
      color: "#FF0000"
      channels: ["sms", "slack", "email"]
      escalation:
        enabled: true
        delay_minutes: 5
        max_escalations: 3
    WARNING:
      priority: 2  
      color: "#FFA500"
      channels: ["slack", "email"]
      escalation:
        enabled: true
        delay_minutes: 15
        max_escalations: 2
    INFO:
      priority: 3
      color: "#00FF00" 
      channels: ["email"]
      escalation:
        enabled: false
```

### 自动级别提升

```yaml
alert:
  auto_escalation:
    rules:
      - condition: "duration > 30m AND severity == 'WARNING'"
        action: "upgrade_to_critical"
      - condition: "consecutive_failures >= 5"
        action: "upgrade_to_critical"
      - condition: "affected_tasks_count > 10"
        action: "upgrade_to_critical"
```

## 告警规则配置

### 基础规则

```yaml
alert:
  rules:
    heartbeat_missed:
      enabled: true
      condition: "missed_heartbeats >= 3"
      severity: "WARNING"
      message: "任务 {{task_name}} 心跳丢失"
      cooldown: "10m"
      
    task_failed:
      enabled: true
      condition: "status == 'FAILED'"
      severity: "CRITICAL"
      message: "任务 {{task_name}} 已故障"
      immediate: true
      
    probe_failed:
      enabled: true
      condition: "probe_result == 'FAILED' AND consecutive_failures >= 2"
      severity: "CRITICAL" 
      message: "任务 {{task_name}} 探测失败"
      cooldown: "5m"
      
    high_resource_usage:
      enabled: true
      condition: "cpu_usage > 90 OR memory_usage > 95"
      severity: "WARNING"
      message: "任务 {{task_name}} 资源使用率过高"
      cooldown: "15m"
```

### 复合条件规则

```yaml
alert:
  rules:
    service_degraded:
      enabled: true
      condition: |
        (response_time > 2000 AND success_rate < 95) OR
        (queue_size > 1000 AND processing_rate < 50)
      severity: "WARNING"
      message: "服务 {{task_name}} 性能下降"
      evaluation_period: "5m"
      
    cluster_failure:
      enabled: true
      condition: |
        failed_tasks_count > total_tasks_count * 0.3 AND
        duration > 300
      severity: "CRITICAL"
      message: "集群故障：{{failed_tasks_count}}/{{total_tasks_count}} 任务失败"
      immediate: true
```

## 告警抑制和去重

### 抑制规则

```yaml
alert:
  suppression:
    rules:
      - name: "maintenance_window"
        condition: "maintenance_mode == true"
        suppress_all: true
        
      - name: "dependency_failure" 
        condition: "dependency.database.status == 'FAILED'"
        suppress_labels: ["task_type=web_service"]
        
      - name: "known_issue"
        condition: "task_id in ['task_001', 'task_002']"
        suppress_severity: ["WARNING"]
        until: "2023-12-01T00:00:00Z"
```

### 去重配置

```yaml
alert:
  deduplication:
    enabled: true
    window: "30m"
    group_by: ["task_id", "severity", "alert_type"]
    max_alerts_per_group: 5
    summary_threshold: 10  # 超过10个相似告警时发送汇总
```

## 告警模板配置

### 短信模板

```yaml
alert:
  templates:
    sms:
      task_failed: |
        【任务告警】
        任务：{{task_name}}
        状态：{{status}}
        时间：{{timestamp}}
        详情：{{dashboard_url}}
        
      high_error_rate: |
        【性能告警】
        任务：{{task_name}} 
        错误率：{{error_rate}}%
        持续时间：{{duration}}
        链接：{{dashboard_url}}
```

### 邮件模板

```yaml
alert:
  templates:
    email:
      subject_template: "[{{severity}}] TaskC告警 - {{task_name}}"
      html_template: |
        <html>
        <head>
          <style>
            .critical { color: #d32f2f; }
            .warning { color: #f57c00; }
            .info { color: #388e3c; }
          </style>
        </head>
        <body>
          <h2 class="{{severity_class}}">任务告警通知</h2>
          <table>
            <tr><td>任务名称</td><td>{{task_name}}</td></tr>
            <tr><td>告警级别</td><td class="{{severity_class}}">{{severity}}</td></tr>
            <tr><td>当前状态</td><td>{{status}}</td></tr>
            <tr><td>发生时间</td><td>{{timestamp}}</td></tr>
            <tr><td>持续时间</td><td>{{duration}}</td></tr>
          </table>
          
          <h3>详细信息</h3>
          <pre>{{details}}</pre>
          
          <p><a href="{{dashboard_url}}">查看控制台</a></p>
        </body>
        </html>
```

### Slack模板

```yaml
alert:
  templates:
    slack:
      message_template: |
        {
          "blocks": [
            {
              "type": "header",
              "text": {
                "type": "plain_text",
                "text": "🚨 TaskC告警 - {{severity}}"
              }
            },
            {
              "type": "section",
              "fields": [
                {"type": "mrkdwn", "text": "*任务名称:*\n{{task_name}}"},
                {"type": "mrkdwn", "text": "*状态:*\n{{status}}"},
                {"type": "mrkdwn", "text": "*时间:*\n{{timestamp}}"},
                {"type": "mrkdwn", "text": "*持续:*\n{{duration}}"}
              ]
            },
            {
              "type": "actions",
              "elements": [
                {
                  "type": "button",
                  "text": {"type": "plain_text", "text": "查看详情"},
                  "url": "{{dashboard_url}}"
                },
                {
                  "type": "button", 
                  "text": {"type": "plain_text", "text": "确认告警"},
                  "url": "{{acknowledge_url}}"
                }
              ]
            }
          ]
        }
```

## 告警时间窗口

### 静默时间配置

```yaml
alert:
  silence_periods:
    - name: "night_shift"
      enabled: true
      start_time: "23:00"
      end_time: "07:00"
      timezone: "Asia/Shanghai"
      affected_channels: ["sms"]
      exceptions:
        severity_levels: ["CRITICAL"]
        tasks: ["production_critical_*"]
        
    - name: "weekend"
      enabled: true
      days: ["saturday", "sunday"]
      start_time: "00:00" 
      end_time: "23:59"
      affected_channels: ["sms", "slack"]
      exceptions:
        severity_levels: ["CRITICAL"]
```

### 维护窗口

```yaml
alert:
  maintenance_windows:
    - name: "weekly_maintenance"
      schedule: "0 2 * * 1"  # 每周一凌晨2点
      duration: "2h"
      suppress_all: true
      notification_channels: ["email"]
      advance_notice: "1h"
      
    - name: "database_upgrade"
      start_time: "2023-12-01T02:00:00Z"
      end_time: "2023-12-01T06:00:00Z"
      affected_tasks: ["db_*", "data_processor_*"]
      custom_message: "数据库升级维护中"
```

## 告警确认和处理

### 自动确认规则

```yaml
alert:
  auto_acknowledge:
    rules:
      - condition: "status == 'HEALTHY' AND duration < '5m'"
        action: "auto_resolve"
        message: "任务已自动恢复"
        
      - condition: "severity == 'INFO'"
        action: "auto_acknowledge"
        delay: "1h"
```

### 告警升级

```yaml
alert:
  escalation:
    rules:
      - level: 1
        delay: "5m"
        channels: ["slack", "email"]
        recipients: ["ops-team@company.com"]
        
      - level: 2
        delay: "15m" 
        channels: ["sms", "slack", "email"]
        recipients: ["manager@company.com"]
        require_acknowledgment: true
        
      - level: 3
        delay: "30m"
        channels: ["sms"]
        recipients: ["cto@company.com"]
        escalation_message: "严重告警未得到处理"
```

## 最佳实践

### 1. 告警设计原则

- **可操作性**: 每个告警都应该有明确的处理步骤
- **相关性**: 告警信息应该包含足够的上下文
- **及时性**: 关键告警应该立即触达相关人员
- **避免疲劳**: 合理设置频率限制和去重机制

### 2. 通道选择策略

```bash
CRITICAL级别: SMS + Slack + Email (立即)
WARNING级别: Slack + Email (5分钟内)
INFO级别: Email (批量发送)
```

### 3. 告警内容优化

- 包含足够的上下文信息
- 提供快速访问链接
- 使用清晰的错误描述
- 避免技术术语，使用业务语言

### 4. 监控告警系统本身

```yaml
meta_alerts:
  channel_health:
    enabled: true
    check_interval: "5m"
    failure_threshold: 3
    
  template_validation:
    enabled: true
    validate_on_config_change: true
    
  rate_limit_monitoring:
    enabled: true
    alert_when_limit_reached: true
```