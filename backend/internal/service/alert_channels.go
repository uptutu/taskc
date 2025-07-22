package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"taskc/backend/internal/model"
	"taskc/backend/pkg/logger"
	"time"

	"go.uber.org/zap"
)

// AlertChannelConfig represents configuration for alert channels.
type AlertChannelConfig struct {
	SMSConfig   SMSConfig   `json:"sms"`
	EmailConfig EmailConfig `json:"email"`
	SlackConfig SlackConfig `json:"slack"`
}

// SMSConfig contains SMS channel configuration.
type SMSConfig struct {
	RateLimit int    `json:"rate_limit"`
	Provider  string `json:"provider"`
	APIKey    string `json:"api_key"`
	APISecret string `json:"api_secret"`
}

// EmailConfig contains email channel configuration.
type EmailConfig struct {
	RateLimit int    `json:"rate_limit"`
	SMTPHost  string `json:"smtp_host"`
	SMTPPort  int    `json:"smtp_port"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	FromEmail string `json:"from_email"`
	ToEmails  []string `json:"to_emails"`
}

// SlackConfig contains Slack channel configuration.
type SlackConfig struct {
	RateLimit  int    `json:"rate_limit"`
	WebhookURL string `json:"webhook_url"`
}

// SMSChannel SMS告警通道
type SMSChannel struct {
	config SMSConfig
}

func NewSMSChannel(config SMSConfig) *SMSChannel {
	return &SMSChannel{config: config}
}

func (c *SMSChannel) Send(ctx context.Context, alert *model.Alert) error {
	// 这里实现SMS发送逻辑
	// 可以集成Twilio、阿里云短信等服务
	logger.Info("SMS alert sent",
		zap.String("task_id", alert.TaskID),
		zap.String("title", alert.Title),
		zap.String("provider", c.config.Provider),
	)
	return nil
}

func (c *SMSChannel) GetRateLimit() int {
	return c.config.RateLimit
}

// EmailChannel 邮件告警通道
type EmailChannel struct {
	config EmailConfig
}

func NewEmailChannel(config EmailConfig) *EmailChannel {
	return &EmailChannel{config: config}
}

func (c *EmailChannel) Send(ctx context.Context, alert *model.Alert) error {
	// 构造邮件内容
	subject := fmt.Sprintf("[任务告警] %s", alert.Title)
	body := c.buildEmailBody(alert)

	// 发送邮件
	auth := smtp.PlainAuth("", c.config.Username, c.config.Password, c.config.SMTPHost)
	toEmails := c.config.ToEmails
	if len(toEmails) == 0 {
		toEmails = []string{"admin@example.com"} // 默认邮箱
	}
	
	for _, toEmail := range toEmails {
		msg := fmt.Sprintf("To: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
			toEmail, subject, body)

		addr := fmt.Sprintf("%s:%d", c.config.SMTPHost, c.config.SMTPPort)
		err := smtp.SendMail(addr, auth, c.config.FromEmail, []string{toEmail}, []byte(msg))
		
		if err != nil {
			return fmt.Errorf("failed to send email to %s: %w", toEmail, err)
		}
	}

	logger.Info("Email alert sent",
		zap.String("task_id", alert.TaskID),
		zap.String("title", alert.Title),
		zap.Int("recipients", len(toEmails)),
	)
	return nil
}

func (c *EmailChannel) buildEmailBody(alert *model.Alert) string {
	return fmt.Sprintf(`
		<h3>任务 %s 告警通知</h3>
		<p><b>当前状态：</b> <span style="color:red">%s</span></p>
		<p><b>告警级别：</b> %s</p>
		<p><b>告警信息：</b> %s</p>
		<p><b>发生时间：</b> %s</p>
		<a href="http://localhost:3000/tasks/%s">查看控制台</a>
	`,
		alert.TaskID,
		alert.Title,
		alert.Level,
		alert.Message,
		alert.CreatedAt.Format("2006-01-02 15:04:05"),
		alert.TaskID,
	)
}

func (c *EmailChannel) GetRateLimit() int {
	return c.config.RateLimit
}

// SlackChannel Slack告警通道
type SlackChannel struct {
	config SlackConfig
}

func NewSlackChannel(config SlackConfig) *SlackChannel {
	return &SlackChannel{config: config}
}

func (c *SlackChannel) Send(ctx context.Context, alert *model.Alert) error {
	if c.config.WebhookURL == "" {
		return fmt.Errorf("slack webhook URL not configured")
	}

	// 构造Slack消息
	message := c.buildSlackMessage(alert)
	messageJSON, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal slack message: %w", err)
	}

	// 发送到Slack
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "POST", c.config.WebhookURL, bytes.NewBuffer(messageJSON))
	if err != nil {
		return fmt.Errorf("failed to create slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send slack message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}

	logger.Info("Slack alert sent",
		zap.String("task_id", alert.TaskID),
		zap.String("title", alert.Title),
	)
	return nil
}

func (c *SlackChannel) buildSlackMessage(alert *model.Alert) map[string]interface{} {
	color := "danger"
	if alert.Level == model.AlertLevelWarning {
		color = "warning"
	} else if alert.Level == model.AlertLevelInfo {
		color = "good"
	}

	return map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"color":      color,
				"title":      fmt.Sprintf("任务告警: %s", alert.TaskID),
				"title_link": fmt.Sprintf("http://localhost:3000/tasks/%s", alert.TaskID),
				"fields": []map[string]interface{}{
					{
						"title": "状态",
						"value": alert.Title,
						"short": true,
					},
					{
						"title": "级别",
						"value": string(alert.Level),
						"short": true,
					},
					{
						"title": "消息",
						"value": alert.Message,
						"short": false,
					},
					{
						"title": "时间",
						"value": alert.CreatedAt.Format("2006-01-02 15:04:05"),
						"short": true,
					},
				},
			},
		},
	}
}

func (c *SlackChannel) GetRateLimit() int {
	return c.config.RateLimit
}