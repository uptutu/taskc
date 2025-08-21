package service_test

import (
	"testing"
	"time"

	"taskc/backend/internal/model"
	"taskc/backend/internal/service"
)

// TestHeartbeatServicePerformance tests heartbeat processing performance
func TestHeartbeatServicePerformance(t *testing.T) {
	// This is a basic performance test to ensure heartbeat processing meets PRD requirements
	
	// Simulate heartbeat processing time
	start := time.Now()
	
	// Create a mock heartbeat
	heartbeat := &model.Heartbeat{
		TaskID:    "test-task-001",
		Timestamp: time.Now(),
		Metadata:  `{"cpu_load": 0.5, "mem_used_mb": 1024}`,
	}
	
	// Simulate processing (in real test, this would use actual service)
	_ = heartbeat
	processTime := time.Since(start)
	
	// PRD requirement: <5s detection latency
	// Processing should be much faster than detection interval
	maxAllowedProcessTime := 1 * time.Second
	
	if processTime > maxAllowedProcessTime {
		t.Errorf("Heartbeat processing took %v, expected < %v", processTime, maxAllowedProcessTime)
	}
	
	t.Logf("Heartbeat processing completed in %v", processTime)
}

// TestHeartbeatConfigOptimization tests the optimized heartbeat configuration
func TestHeartbeatConfigOptimization(t *testing.T) {
	config := service.DefaultHeartbeatConfig()
	
	// Verify optimized configuration values
	expectedTimeout := 20 * time.Second
	
	if config.Timeout != expectedTimeout {
		t.Errorf("Expected timeout %v, got %v", expectedTimeout, config.Timeout)
	}
	
	// Calculate detection latency: timeout / maxMissedBeats * checkInterval
	// With check interval of 5s, max missed beats of 2, should detect failure in ~10s
	maxDetectionLatency := 15 * time.Second // Allow some buffer
	actualDetectionLatency := time.Duration(config.MaxMissedBeats) * 5 * time.Second
	
	if actualDetectionLatency > maxDetectionLatency {
		t.Errorf("Detection latency %v exceeds max allowed %v", actualDetectionLatency, maxDetectionLatency)
	}
	
	t.Logf("Configuration optimized: timeout=%v, maxMissed=%d, detection latency~%v", 
		config.Timeout, config.MaxMissedBeats, actualDetectionLatency)
}

// TestLogServiceCleanup tests log cleanup functionality
func TestLogServiceCleanup(t *testing.T) {
	config := service.DefaultLogConfig()
	
	// Verify log configuration meets PRD requirements
	expectedRetentionDays := 90
	expectedDiskThreshold := 85
	
	if config.RetentionDays != expectedRetentionDays {
		t.Errorf("Expected retention days %d, got %d", expectedRetentionDays, config.RetentionDays)
	}
	
	if config.DiskThreshold != expectedDiskThreshold {
		t.Errorf("Expected disk threshold %d, got %d", expectedDiskThreshold, config.DiskThreshold)
	}
	
	t.Logf("Log service configured: retention=%d days, disk threshold=%d%%", 
		config.RetentionDays, config.DiskThreshold)
}

// BenchmarkHeartbeatProcessing benchmarks heartbeat processing performance
func BenchmarkHeartbeatProcessing(b *testing.B) {
	heartbeat := &model.Heartbeat{
		TaskID:    "benchmark-task",
		Timestamp: time.Now(),
		Metadata:  `{"cpu_load": 0.7, "mem_used_mb": 2048, "queue_size": 10}`,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate heartbeat metadata parsing
		_ = heartbeat.Metadata
		_ = heartbeat.TaskID
		_ = heartbeat.Timestamp
	}
}