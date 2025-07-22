package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"taskc/backend/internal/model"
	"taskc/backend/internal/repository"
	"taskc/backend/pkg/logger"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

type AlertService struct {
	alertRepo   *repository.AlertRepository
	redisClient *redis.Client
	channels    map[string]AlertChannel
}

type AlertChannel interface {
	Send(ctx context.Context, alert *model.Alert) error
	GetRateLimit() int
}

func NewAlertService(alertRepo *repository.AlertRepository, redisClient *redis.Client) *AlertService {
	service := &AlertService{
		alertRepo:   alertRepo,
		redisClient: redisClient,
		channels:    make(map[string]AlertChannel),
	}

	// 注册告警通道
	service.registerChannels()
	
	// 启动告警处理器
	go service.startAlertProcessor()

	return service
}

func (s *AlertService) registerChannels() {
	// Use default configurations - these should be loaded from config file
	smsConfig := SMSConfig{
		RateLimit: 30,
		Provider:  "default",
		APIKey:    "",
		APISecret: "",
	}
	emailConfig := EmailConfig{
		RateLimit: 100,
		SMTPHost:  "smtp.gmail.com",
		SMTPPort:  587,
		Username:  "",
		Password:  "",
		FromEmail: "noreply@taskc.com",
		ToEmails:  []string{"admin@example.com"},
	}
	slackConfig := SlackConfig{
		RateLimit:  20,
		WebhookURL: "",
	}

	s.channels["sms"] = NewSMSChannel(smsConfig)
	s.channels["email"] = NewEmailChannel(emailConfig)
	s.channels["slack"] = NewSlackChannel(slackConfig)
}

// CreateAlert 创建告警
func (s *AlertService) CreateAlert(ctx context.Context, alert *model.Alert) error {
	if err := s.alertRepo.Create(ctx, alert); err != nil {
		return fmt.Errorf("failed to create alert: %w", err)
	}

	// 发布告警事件到Redis
	if err := s.publishAlertEvent(ctx, alert); err != nil {
		logger.Warn("Failed to publish alert event", zap.Error(err))
	}

	return nil
}

// SendAlert 发送告警
func (s *AlertService) SendAlert(ctx context.Context, alert *model.Alert) error {
	var channels []string
	if err := json.Unmarshal([]byte(alert.Channels), &channels); err != nil {
		return fmt.Errorf("invalid channels format: %w", err)
	}

	var errors []error
	for _, channelName := range channels {
		channel, exists := s.channels[channelName]
		if !exists {
			logger.Warn("Unknown alert channel", zap.String("channel", channelName))
			continue
		}

		// 检查速率限制
		if !s.checkRateLimit(ctx, channelName, channel.GetRateLimit()) {
			logger.Warn("Rate limit exceeded for channel", zap.String("channel", channelName))
			continue
		}

		if err := channel.Send(ctx, alert); err != nil {
			logger.Error("Failed to send alert", 
				zap.String("channel", channelName),
				zap.Error(err),
			)
			errors = append(errors, err)
		} else {
			logger.Info("Alert sent successfully",
				zap.String("channel", channelName),
				zap.Uint("alert_id", alert.ID),
			)
		}
	}

	// 标记为已发送
	if err := s.alertRepo.MarkAsSent(ctx, alert.ID); err != nil {
		logger.Error("Failed to mark alert as sent", zap.Error(err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to send alert to some channels: %v", errors)
	}

	return nil
}

// checkRateLimit 检查速率限制
func (s *AlertService) checkRateLimit(ctx context.Context, channel string, limit int) bool {
	key := fmt.Sprintf("rate_limit:%s", channel)
	
	// 使用Redis实现令牌桶算法
	current, err := s.redisClient.Get(ctx, key).Int()
	if err != nil && err != redis.Nil {
		logger.Warn("Failed to get rate limit", zap.Error(err))
		return true // 如果Redis失败，允许发送
	}

	if current >= limit {
		return false
	}

	// 增加计数器
	pipe := s.redisClient.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, time.Minute)
	pipe.Exec(ctx)

	return true
}

// publishAlertEvent 发布告警事件
func (s *AlertService) publishAlertEvent(ctx context.Context, alert *model.Alert) error {
	event := map[string]interface{}{
		"type":     "alert_created",
		"alert_id": alert.ID,
		"task_id":  alert.TaskID,
		"level":    alert.Level,
		"title":    alert.Title,
		"message":  alert.Message,
		"timestamp": time.Now().Unix(),
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return s.redisClient.XAdd(ctx, &redis.XAddArgs{
		Stream: "alert_events",
		Values: map[string]interface{}{
			"data": eventJSON,
		},
	}).Err()
}

// startAlertProcessor 启动告警处理器
func (s *AlertService) startAlertProcessor() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.processUnsentAlerts()
	}
}

// processUnsentAlerts 处理未发送的告警
func (s *AlertService) processUnsentAlerts() {
	ctx := context.Background()
	alerts, err := s.alertRepo.GetUnsentAlerts(ctx)
	if err != nil {
		logger.Error("Failed to get unsent alerts", zap.Error(err))
		return
	}

	for _, alert := range alerts {
		if err := s.SendAlert(ctx, alert); err != nil {
			logger.Error("Failed to send alert",
				zap.Uint("alert_id", alert.ID),
				zap.Error(err),
			)
		}
	}
}

// GetAlerts 获取告警列表
func (s *AlertService) GetAlerts(ctx context.Context, page, limit int, level string) ([]*model.Alert, int64, error) {
	offset := (page - 1) * limit
	return s.alertRepo.List(ctx, offset, limit, level)
}

// GetAlertsByTaskID 根据任务ID获取告警
func (s *AlertService) GetAlertsByTaskID(ctx context.Context, taskID string, limit int) ([]*model.Alert, error) {
	return s.alertRepo.GetByTaskID(ctx, taskID, limit)
}

// GetAlertStatistics 获取告警统计
func (s *AlertService) GetAlertStatistics(ctx context.Context, hours int) (map[string]interface{}, error) {
	return s.alertRepo.GetAlertStatistics(ctx, hours)
}