package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"taskc/backend/internal/model"
	"taskc/backend/internal/repository"
	"taskc/backend/pkg/logger"

	"go.uber.org/zap"
)

// ProbeConfig contains configuration for the probe service.
type ProbeConfig struct {
	DefaultTimeout    time.Duration
	MaxConcurrentProbes int
	RetryAttempts     int
	RetryDelay        time.Duration
}

// DefaultProbeConfig returns default probe service configuration.
func DefaultProbeConfig() ProbeConfig {
	return ProbeConfig{
		DefaultTimeout:      5 * time.Second,
		MaxConcurrentProbes: 50,
		RetryAttempts:       3,
		RetryDelay:          1 * time.Second,
	}
}

// ProbeService manages task health probing operations.
// It supports multiple probe types (HTTP, TCP, Ping) and handles
// probe execution, result storage, and configuration management.
type ProbeService struct {
	probeRepo *repository.ProbeRepository
	taskRepo  *repository.TaskRepository
	config    ProbeConfig
	httpClient *http.Client
}

// NewProbeService creates a new probe service instance.
func NewProbeService(probeRepo *repository.ProbeRepository, taskRepo *repository.TaskRepository, config ProbeConfig) *ProbeService {
	return &ProbeService{
		probeRepo: probeRepo,
		taskRepo:  taskRepo,
		config:    config,
		httpClient: &http.Client{
			Timeout: config.DefaultTimeout,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   5 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				MaxIdleConns:       100,
				IdleConnTimeout:    90 * time.Second,
				TLSHandshakeTimeout: 10 * time.Second,
			},
		},
	}
}

// ProbeExecutionResult represents the result of a probe execution.
type ProbeExecutionResult struct {
	Success      bool          `json:"success"`
	LatencyMs    int64         `json:"latency_ms"`
	ErrorMessage string        `json:"error_message,omitempty"`
	StatusCode   int           `json:"status_code,omitempty"`
	ResponseSize int64         `json:"response_size,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ExecuteProbe executes a probe based on the provided configuration.
// It supports HTTP, TCP, and Ping probe types with comprehensive error handling.
func (s *ProbeService) ExecuteProbe(ctx context.Context, config *model.ProbeConfig) (*model.ProbeResult, error) {
	logger.Debug("executing probe",
		zap.String("task_id", config.TaskID),
		zap.String("probe_type", string(config.Type)),
		zap.Uint("config_id", config.ID),
	)

	startTime := time.Now()
	var execResult *ProbeExecutionResult
	var err error

	// Execute probe based on type
	switch config.Type {
	case model.ProbeTypeHTTP:
		execResult, err = s.executeHTTPProbe(ctx, config)
	case model.ProbeTypeTCP:
		execResult, err = s.executeTCPProbe(ctx, config)
	case model.ProbeTypePing:
		execResult, err = s.executePingProbe(ctx, config)
	default:
		return nil, fmt.Errorf("unsupported probe type: %s", config.Type)
	}

	// Create probe result
	result := &model.ProbeResult{
		ProbeConfigID: config.ID,
		Result:        "FAILED",
		LatencyMs:     time.Since(startTime).Milliseconds(),
		Timestamp:     time.Now(),
	}

	if execResult != nil {
		result.LatencyMs = execResult.LatencyMs
		if execResult.Success {
			result.Result = "SUCCESS"
		} else {
			result.ErrorMessage = execResult.ErrorMessage
		}
	}

	if err != nil {
		result.ErrorMessage = err.Error()
		logger.Warn("probe execution failed",
			zap.String("task_id", config.TaskID),
			zap.String("probe_type", string(config.Type)),
			zap.Error(err),
		)
	} else {
		logger.Info("probe executed successfully",
			zap.String("task_id", config.TaskID),
			zap.String("result", result.Result),
			zap.Int64("latency_ms", result.LatencyMs),
		)
	}

	// Save probe result
	if saveErr := s.probeRepo.SaveResult(ctx, result); saveErr != nil {
		logger.Error("failed to save probe result",
			zap.String("task_id", config.TaskID),
			zap.Error(saveErr),
		)
	}

	return result, err
}

// HTTPProbeConfig represents HTTP-specific probe configuration.
type HTTPProbeConfig struct {
	Endpoint       string            `json:"endpoint"`
	Method         string            `json:"method,omitempty"`
	TimeoutMs      int               `json:"timeout_ms,omitempty"`
	ExpectedStatus int               `json:"expected_status,omitempty"`
	Headers        map[string]string `json:"headers,omitempty"`
	Body           string            `json:"body,omitempty"`
	SuccessConditions []string       `json:"success_conditions,omitempty"`
	FollowRedirects   bool           `json:"follow_redirects,omitempty"`
	VerifySSL         bool           `json:"verify_ssl,omitempty"`
}

// executeHTTPProbe executes an HTTP probe with comprehensive validation.
func (s *ProbeService) executeHTTPProbe(ctx context.Context, config *model.ProbeConfig) (*ProbeExecutionResult, error) {
	var httpConfig HTTPProbeConfig
	if err := json.Unmarshal([]byte(config.Config), &httpConfig); err != nil {
		return nil, fmt.Errorf("invalid HTTP probe config: %w", err)
	}

	// Validate endpoint
	if httpConfig.Endpoint == "" {
		return nil, fmt.Errorf("endpoint is required for HTTP probe")
	}

	// Set defaults
	if httpConfig.Method == "" {
		httpConfig.Method = "GET"
	}
	if httpConfig.ExpectedStatus == 0 {
		httpConfig.ExpectedStatus = 200
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, httpConfig.Method, httpConfig.Endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Add request body if provided
	if httpConfig.Body != "" {
		req.Body = io.NopCloser(strings.NewReader(httpConfig.Body))
		req.ContentLength = int64(len(httpConfig.Body))
	}

	// Add custom headers
	for key, value := range httpConfig.Headers {
		req.Header.Set(key, value)
	}

	// Set User-Agent if not provided
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "TaskC-Probe/1.0")
	}

	// Configure client for this specific probe
	client := s.httpClient
	if httpConfig.TimeoutMs > 0 {
		client = &http.Client{
			Timeout: time.Duration(httpConfig.TimeoutMs) * time.Millisecond,
			Transport: s.httpClient.Transport,
		}
	}

	// Execute request
	startTime := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(startTime).Milliseconds()

	result := &ProbeExecutionResult{
		LatencyMs: latency,
		Metadata:  make(map[string]interface{}),
	}

	if err != nil {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("HTTP request failed: %v", err)
		return result, nil // Return result, not error, so we can save the failed attempt
	}
	defer resp.Body.Close()

	// Record response metadata
	result.StatusCode = resp.StatusCode
	result.Metadata["status_code"] = resp.StatusCode
	result.Metadata["content_type"] = resp.Header.Get("Content-Type")
	result.Metadata["content_length"] = resp.ContentLength

	// Read response body for size calculation and content validation
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // Limit to 1MB
	if err != nil {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("Failed to read response body: %v", err)
		return result, nil
	}
	result.ResponseSize = int64(len(body))

	// Validate status code
	if resp.StatusCode != httpConfig.ExpectedStatus {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("Expected status %d, got %d", httpConfig.ExpectedStatus, resp.StatusCode)
		return result, nil
	}

	// Validate success conditions
	if len(httpConfig.SuccessConditions) > 0 {
		if err := s.validateSuccessConditions(httpConfig.SuccessConditions, resp, body); err != nil {
			result.Success = false
			result.ErrorMessage = fmt.Sprintf("Success condition failed: %v", err)
			return result, nil
		}
	}

	result.Success = true
	return result, nil
}

// validateSuccessConditions validates HTTP response against success conditions.
func (s *ProbeService) validateSuccessConditions(conditions []string, resp *http.Response, body []byte) error {
	bodyStr := string(body)
	
	for _, condition := range conditions {
		switch {
		case strings.HasPrefix(condition, "status_code=="):
			expected := strings.TrimPrefix(condition, "status_code==")
			if fmt.Sprintf("%d", resp.StatusCode) != expected {
				return fmt.Errorf("status code condition failed: expected %s, got %d", expected, resp.StatusCode)
			}
			
		case strings.HasPrefix(condition, "body_contains:"):
			expected := strings.TrimPrefix(condition, "body_contains:")
			if !strings.Contains(bodyStr, expected) {
				return fmt.Errorf("body content condition failed: '%s' not found in response", expected)
			}
			
		case strings.HasPrefix(condition, "response_time<"):
			// This would be handled at a higher level with latency data
			continue
			
		case strings.HasPrefix(condition, "header_exists:"):
			headerName := strings.TrimPrefix(condition, "header_exists:")
			if resp.Header.Get(headerName) == "" {
				return fmt.Errorf("header condition failed: header '%s' not found", headerName)
			}
			
		default:
			return fmt.Errorf("unknown success condition: %s", condition)
		}
	}
	
	return nil
}

// TCPProbeConfig represents TCP-specific probe configuration.
type TCPProbeConfig struct {
	Host      string `json:"host"`
	Port      int    `json:"port"`
	TimeoutMs int    `json:"timeout_ms,omitempty"`
}

// executeTCPProbe executes a TCP connection probe.
func (s *ProbeService) executeTCPProbe(ctx context.Context, config *model.ProbeConfig) (*ProbeExecutionResult, error) {
	var tcpConfig TCPProbeConfig
	if err := json.Unmarshal([]byte(config.Config), &tcpConfig); err != nil {
		return nil, fmt.Errorf("invalid TCP probe config: %w", err)
	}

	// Validate configuration
	if tcpConfig.Host == "" {
		return nil, fmt.Errorf("host is required for TCP probe")
	}
	if tcpConfig.Port <= 0 || tcpConfig.Port > 65535 {
		return nil, fmt.Errorf("valid port (1-65535) is required for TCP probe")
	}

	// Set default timeout
	timeout := s.config.DefaultTimeout
	if tcpConfig.TimeoutMs > 0 {
		timeout = time.Duration(tcpConfig.TimeoutMs) * time.Millisecond
	}

	// Create dialer with timeout
	dialer := &net.Dialer{
		Timeout: timeout,
	}

	// Execute TCP connection
	startTime := time.Now()
	address := fmt.Sprintf("%s:%d", tcpConfig.Host, tcpConfig.Port)
	conn, err := dialer.DialContext(ctx, "tcp", address)
	latency := time.Since(startTime).Milliseconds()

	result := &ProbeExecutionResult{
		LatencyMs: latency,
		Metadata: map[string]interface{}{
			"host": tcpConfig.Host,
			"port": tcpConfig.Port,
			"address": address,
		},
	}

	if err != nil {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("TCP connection failed: %v", err)
		return result, nil
	}

	// Connection successful, close it
	conn.Close()
	result.Success = true
	return result, nil
}

// PingProbeConfig represents Ping-specific probe configuration.
type PingProbeConfig struct {
	Host      string `json:"host"`
	Count     int    `json:"count,omitempty"`
	TimeoutMs int    `json:"timeout_ms,omitempty"`
}

// executePingProbe executes an ICMP ping probe.
// Note: This implementation uses TCP ping as ICMP requires root privileges.
func (s *ProbeService) executePingProbe(ctx context.Context, config *model.ProbeConfig) (*ProbeExecutionResult, error) {
	var pingConfig PingProbeConfig
	if err := json.Unmarshal([]byte(config.Config), &pingConfig); err != nil {
		return nil, fmt.Errorf("invalid ping probe config: %w", err)
	}

	// Validate configuration
	if pingConfig.Host == "" {
		return nil, fmt.Errorf("host is required for ping probe")
	}

	// Set defaults
	if pingConfig.Count <= 0 {
		pingConfig.Count = 1
	}
	timeout := s.config.DefaultTimeout
	if pingConfig.TimeoutMs > 0 {
		timeout = time.Duration(pingConfig.TimeoutMs) * time.Millisecond
	}

	// Resolve host address
	startTime := time.Now()
	addrs, err := net.LookupHost(pingConfig.Host)
	resolveLatency := time.Since(startTime).Milliseconds()

	result := &ProbeExecutionResult{
		Metadata: map[string]interface{}{
			"host": pingConfig.Host,
			"count": pingConfig.Count,
			"resolve_latency_ms": resolveLatency,
		},
	}

	if err != nil {
		result.Success = false
		result.LatencyMs = resolveLatency
		result.ErrorMessage = fmt.Sprintf("DNS resolution failed: %v", err)
		return result, nil
	}

	if len(addrs) == 0 {
		result.Success = false
		result.LatencyMs = resolveLatency
		result.ErrorMessage = "no IP addresses found for host"
		return result, nil
	}

	// Use first resolved address
	targetIP := addrs[0]
	result.Metadata["resolved_ip"] = targetIP

	// Perform TCP "ping" to port 80 (fallback connectivity test)
	// This is a simplified implementation since ICMP requires privileges
	dialer := &net.Dialer{
		Timeout: timeout,
	}

	var totalLatency int64
	successCount := 0

	for i := 0; i < pingConfig.Count; i++ {
		pingStart := time.Now()
		
		// Try common ports for connectivity test
		ports := []int{80, 443, 22, 53}
		connected := false
		
		for _, port := range ports {
			address := fmt.Sprintf("%s:%d", targetIP, port)
			conn, err := dialer.DialContext(ctx, "tcp", address)
			if err == nil {
				conn.Close()
				connected = true
				break
			}
		}
		
		pingLatency := time.Since(pingStart).Milliseconds()
		totalLatency += pingLatency
		
		if connected {
			successCount++
		}
		
		// Small delay between pings
		if i < pingConfig.Count-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	// Calculate average latency
	avgLatency := totalLatency / int64(pingConfig.Count)
	result.LatencyMs = avgLatency
	result.Metadata["success_count"] = successCount
	result.Metadata["total_count"] = pingConfig.Count
	result.Metadata["success_rate"] = float64(successCount) / float64(pingConfig.Count)

	// Consider probe successful if at least one ping succeeded
	if successCount > 0 {
		result.Success = true
	} else {
		result.Success = false
		result.ErrorMessage = "no connectivity detected on common ports"
	}

	return result, nil
}

// CreateProbeConfig creates a new probe configuration.
func (s *ProbeService) CreateProbeConfig(ctx context.Context, config *model.ProbeConfig) error {
	// Validate probe configuration before saving
	if err := s.validateProbeConfig(config); err != nil {
		return fmt.Errorf("probe config validation failed: %w", err)
	}

	if err := s.probeRepo.CreateConfig(ctx, config); err != nil {
		return fmt.Errorf("failed to create probe config: %w", err)
	}

	logger.Info("probe config created",
		zap.String("task_id", config.TaskID),
		zap.String("probe_type", string(config.Type)),
		zap.Uint("config_id", config.ID),
	)

	return nil
}

// GetTaskProbeConfigs retrieves probe configurations for a specific task.
func (s *ProbeService) GetTaskProbeConfigs(ctx context.Context, taskID string) ([]*model.ProbeConfig, error) {
	configs, err := s.probeRepo.GetConfigsByTaskID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get probe configs for task %s: %w", taskID, err)
	}
	return configs, nil
}

// UpdateProbeConfig updates an existing probe configuration.
func (s *ProbeService) UpdateProbeConfig(ctx context.Context, config *model.ProbeConfig) error {
	// Validate updated configuration
	if err := s.validateProbeConfig(config); err != nil {
		return fmt.Errorf("probe config validation failed: %w", err)
	}

	if err := s.probeRepo.UpdateConfig(ctx, config); err != nil {
		return fmt.Errorf("failed to update probe config: %w", err)
	}

	logger.Info("probe config updated",
		zap.String("task_id", config.TaskID),
		zap.String("probe_type", string(config.Type)),
		zap.Uint("config_id", config.ID),
	)

	return nil
}

// DeleteProbeConfig removes a probe configuration.
func (s *ProbeService) DeleteProbeConfig(ctx context.Context, configID uint) error {
	if err := s.probeRepo.DeleteConfig(ctx, configID); err != nil {
		return fmt.Errorf("failed to delete probe config %d: %w", configID, err)
	}

	logger.Info("probe config deleted",
		zap.Uint("config_id", configID),
	)

	return nil
}

// GetProbeResults retrieves probe execution results for a task.
func (s *ProbeService) GetProbeResults(ctx context.Context, taskID string, limit int) ([]*model.ProbeResult, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	results, err := s.probeRepo.GetResultsByTaskID(ctx, taskID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get probe results for task %s: %w", taskID, err)
	}

	return results, nil
}

// validateProbeConfig validates probe configuration based on type.
func (s *ProbeService) validateProbeConfig(config *model.ProbeConfig) error {
	if config.TaskID == "" {
		return fmt.Errorf("task_id is required")
	}

	switch config.Type {
	case model.ProbeTypeHTTP:
		var httpConfig HTTPProbeConfig
		if err := json.Unmarshal([]byte(config.Config), &httpConfig); err != nil {
			return fmt.Errorf("invalid HTTP config: %w", err)
		}
		if httpConfig.Endpoint == "" {
			return fmt.Errorf("endpoint is required for HTTP probe")
		}

	case model.ProbeTypeTCP:
		var tcpConfig TCPProbeConfig
		if err := json.Unmarshal([]byte(config.Config), &tcpConfig); err != nil {
			return fmt.Errorf("invalid TCP config: %w", err)
		}
		if tcpConfig.Host == "" {
			return fmt.Errorf("host is required for TCP probe")
		}
		if tcpConfig.Port <= 0 || tcpConfig.Port > 65535 {
			return fmt.Errorf("valid port (1-65535) is required for TCP probe")
		}

	case model.ProbeTypePing:
		var pingConfig PingProbeConfig
		if err := json.Unmarshal([]byte(config.Config), &pingConfig); err != nil {
			return fmt.Errorf("invalid Ping config: %w", err)
		}
		if pingConfig.Host == "" {
			return fmt.Errorf("host is required for Ping probe")
		}

	default:
		return fmt.Errorf("unsupported probe type: %s", config.Type)
	}

	return nil
}

// GetProbeStatistics returns statistics about probe executions for a task.
func (s *ProbeService) GetProbeStatistics(ctx context.Context, taskID string, hours int) (map[string]interface{}, error) {
	if hours <= 0 || hours > 24*7 {
		hours = 24
	}

	since := time.Now().Add(-time.Duration(hours) * time.Hour)
	stats, err := s.probeRepo.GetStatistics(ctx, taskID, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get probe statistics: %w", err)
	}

	return stats, nil
}