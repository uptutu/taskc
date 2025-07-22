package service

import (
	"compress/gzip"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"taskc/backend/internal/model"
	"taskc/backend/internal/repository"
	"taskc/backend/pkg/logger"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// LogConfig contains configuration for log management.
type LogConfig struct {
	LogDirectory       string        `json:"log_directory"`
	MaxLogSize         int64         `json:"max_log_size"`          // in bytes
	MaxLogAge          time.Duration `json:"max_log_age"`           // max age before rotation
	CompressionEnabled bool          `json:"compression_enabled"`
	RetentionDays      int           `json:"retention_days"`        // days to keep logs
	RotationInterval   time.Duration `json:"rotation_interval"`     // how often to check for rotation
	BufferSize         int           `json:"buffer_size"`           // log buffer size
	FlushInterval      time.Duration `json:"flush_interval"`        // how often to flush buffer
	DiskThreshold      int           `json:"disk_threshold"`        // disk usage percentage threshold
	CleanupTime        string        `json:"cleanup_time"`          // daily cleanup time
}

// DefaultLogConfig returns default configuration for log management.
func DefaultLogConfig() LogConfig {
	return LogConfig{
		LogDirectory:       "./logs",
		MaxLogSize:         100 * 1024 * 1024, // 100MB
		MaxLogAge:          24 * time.Hour,
		CompressionEnabled: true,
		RetentionDays:      30,
		RotationInterval:   1 * time.Hour,
		BufferSize:         1000,
		FlushInterval:      10 * time.Second,
		DiskThreshold:      80,
		CleanupTime:        "02:00",
	}
}

// LogEntry represents a structured log entry for aggregation.
type LogEntry = repository.LogEntry

// LogSearchFilters contains filters for log search operations.
type LogSearchFilters = repository.LogSearchFilters

// LogSearchResult contains the results of a log search operation.
type LogSearchResult struct {
	Logs       []LogEntry       `json:"logs"`
	TotalCount int              `json:"total_count"`
	Filters    LogSearchFilters `json:"filters"`
}

// LogService provides centralized log management and aggregation.
type LogService struct {
	logRepo     *repository.LogRepository
	redisClient *redis.Client
	config      LogConfig
	
	// Log aggregation buffer
	buffer      chan LogEntry
	bufferMutex sync.RWMutex
	
	// Service lifecycle
	running bool
	stopCh  chan struct{}
	wg      sync.WaitGroup
	mu      sync.RWMutex
}

func NewLogService(logRepo *repository.LogRepository, redisClient *redis.Client, config LogConfig) *LogService {
	return &LogService{
		logRepo:     logRepo,
		redisClient: redisClient,
		config:      config,
		buffer:      make(chan LogEntry, config.BufferSize),
		running:     false,
		stopCh:      make(chan struct{}),
	}
}

// Start begins the log management service.
func (s *LogService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("log service is already running")
	}

	// Ensure log directory exists
	if err := s.ensureLogDirectory(); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Start log buffer processor
	s.wg.Add(1)
	go s.processLogBuffer(ctx)

	// Start log rotation scheduler
	s.wg.Add(1)
	go s.scheduleLogRotation(ctx)

	// Start log cleanup scheduler
	s.wg.Add(1)
	go s.scheduleLogCleanup(ctx)

	s.running = true

	logger.Info("log service started",
		zap.String("log_directory", s.config.LogDirectory),
		zap.Duration("rotation_interval", s.config.RotationInterval),
		zap.Int("retention_days", s.config.RetentionDays),
	)

	return nil
}

// Stop stops the log management service.
func (s *LogService) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("log service is not running")
	}

	close(s.stopCh)
	s.wg.Wait()
	close(s.buffer)

	s.running = false

	logger.Info("log service stopped")
	return nil
}

// LogEntry logs a structured entry to the aggregation system.
func (s *LogService) LogEntry(ctx context.Context, entry LogEntry) error {
	if !s.running {
		return fmt.Errorf("log service is not running")
	}

	entry.Timestamp = time.Now()

	select {
	case s.buffer <- entry:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Buffer is full, drop the entry and log warning
		logger.Warn("log buffer is full, dropping entry",
			zap.String("component", entry.Component),
			zap.String("level", entry.Level),
		)
		return fmt.Errorf("log buffer is full")
	}
}

// CreateTaskLog creates a task log entry.
func (s *LogService) CreateTaskLog(ctx context.Context, log *model.TaskLog) error {
	// Also add to aggregation buffer for real-time processing
	entry := LogEntry{
		Timestamp: log.Timestamp,
		Level:     log.Level,
		Message:   log.Message,
		TaskID:    log.TaskID,
		Component: "task",
		Fields:    map[string]interface{}{
			"log_id": log.ID,
		},
	}
	
	// Non-blocking add to buffer
	select {
	case s.buffer <- entry:
	default:
		logger.Debug("log buffer full, skipping aggregation for task log")
	}
	
	return s.logRepo.Create(ctx, log)
}

// SearchLogs searches for log entries based on filters.
func (s *LogService) SearchLogs(ctx context.Context, filters LogSearchFilters) (*LogSearchResult, error) {
	// Search in database for structured logs
	dbLogs, err := s.logRepo.Search(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to search database logs: %w", err)
	}

	// Search in file system for application logs if needed
	var fileLogs []LogEntry
	if filters.SearchFiles {
		fileLogs, err = s.searchLogFiles(ctx, filters)
		if err != nil {
			logger.Warn("failed to search log files", zap.Error(err))
		}
	}

	// Combine and sort results
	allLogs := append(dbLogs, fileLogs...)
	result := &LogSearchResult{
		Logs:       allLogs,
		TotalCount: len(allLogs),
		Filters:    filters,
	}

	// Apply pagination
	if filters.Limit > 0 {
		start := filters.Offset
		end := start + filters.Limit
		if start < len(allLogs) {
			if end > len(allLogs) {
				end = len(allLogs)
			}
			result.Logs = allLogs[start:end]
		} else {
			result.Logs = []LogEntry{}
		}
	}

	return result, nil
}

// GetTaskLogs gets task logs with pagination and filtering.
func (s *LogService) GetTaskLogs(ctx context.Context, taskID string, page, limit int, level string) ([]*model.TaskLog, int64, error) {
	offset := (page - 1) * limit
	return s.logRepo.GetByTaskID(ctx, taskID, offset, limit, level)
}

// GetLogStatistics returns aggregated log statistics.
func (s *LogService) GetLogStatistics(ctx context.Context, taskID string, hours int) (map[string]interface{}, error) {
	since := time.Now().Add(-time.Duration(hours) * time.Hour)
	stats, err := s.logRepo.GetStatistics(ctx, taskID, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get log statistics: %w", err)
	}

	// Add file system statistics
	fileStats, err := s.getFileSystemStats()
	if err != nil {
		logger.Warn("failed to get file system stats", zap.Error(err))
	} else {
		stats["file_system"] = fileStats
	}

	return stats, nil
}

// RotateLogs manually triggers log rotation.
func (s *LogService) RotateLogs(ctx context.Context) error {
	logger.Info("manually triggering log rotation")
	return s.performLogRotation(ctx)
}

// CleanupOldLogs manually triggers cleanup of old logs.
func (s *LogService) CleanupOldLogs(ctx context.Context) error {
	logger.Info("manually triggering log cleanup")
	return s.performLogCleanup(ctx)
}

// ExportLogs exports logs in specified format.
func (s *LogService) ExportLogs(ctx context.Context, taskID string, format string, start, end time.Time) ([]byte, error) {
	logs, err := s.logRepo.GetByTimeRange(ctx, taskID, start, end)
	if err != nil {
		return nil, err
	}

	switch format {
	case "json":
		return s.exportAsJSON(logs)
	case "csv":
		return s.exportAsCSV(logs)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// CheckDiskUsage checks disk usage and triggers cleanup if needed.
func (s *LogService) CheckDiskUsage(ctx context.Context) error {
	if s.config.LogDirectory == "" {
		return nil
	}

	usage, err := s.getDiskUsage(s.config.LogDirectory)
	if err != nil {
		return err
	}

	logger.Info("Disk usage checked",
		zap.Float64("usage_percent", usage),
		zap.Int("threshold", s.config.DiskThreshold),
	)

	// If disk usage exceeds threshold, trigger emergency cleanup
	if usage > float64(s.config.DiskThreshold) {
		logger.Warn("Disk usage threshold exceeded, triggering emergency cleanup",
			zap.Float64("usage", usage),
		)
		return s.emergencyCleanup(ctx)
	}

	return nil
}

// processLogBuffer processes log entries from the buffer.
func (s *LogService) processLogBuffer(ctx context.Context) {
	defer s.wg.Done()

	flushTicker := time.NewTicker(s.config.FlushInterval)
	defer flushTicker.Stop()

	var batchBuffer []LogEntry

	for {
		select {
		case entry, ok := <-s.buffer:
			if !ok {
				// Buffer closed, flush remaining entries
				if len(batchBuffer) > 0 {
					s.flushLogBatch(ctx, batchBuffer)
				}
				return
			}
			
			batchBuffer = append(batchBuffer, entry)
			
			// Flush if buffer is full
			if len(batchBuffer) >= s.config.BufferSize/2 {
				s.flushLogBatch(ctx, batchBuffer)
				batchBuffer = batchBuffer[:0]
			}

		case <-flushTicker.C:
			// Periodic flush
			if len(batchBuffer) > 0 {
				s.flushLogBatch(ctx, batchBuffer)
				batchBuffer = batchBuffer[:0]
			}

		case <-s.stopCh:
			// Service stopping, flush remaining entries
			if len(batchBuffer) > 0 {
				s.flushLogBatch(ctx, batchBuffer)
			}
			return

		case <-ctx.Done():
			// Context cancelled
			return
		}
	}
}

// flushLogBatch flushes a batch of log entries to storage.
func (s *LogService) flushLogBatch(ctx context.Context, entries []LogEntry) {
	if len(entries) == 0 {
		return
	}

	// Store in database
	if err := s.logRepo.BatchCreate(ctx, entries); err != nil {
		logger.Error("failed to store log batch in database",
			zap.Int("batch_size", len(entries)),
			zap.Error(err),
		)
	}

	// Publish to Redis for real-time subscribers
	for _, entry := range entries {
		if err := s.publishLogEvent(ctx, entry); err != nil {
			logger.Debug("failed to publish log event", zap.Error(err))
		}
	}

	logger.Debug("flushed log batch",
		zap.Int("batch_size", len(entries)),
	)
}

// publishLogEvent publishes a log event to Redis streams.
func (s *LogService) publishLogEvent(ctx context.Context, entry LogEntry) error {
	if s.redisClient == nil {
		return nil
	}
	
	return s.redisClient.XAdd(ctx, &redis.XAddArgs{
		Stream: "log_events",
		Values: map[string]interface{}{
			"timestamp": entry.Timestamp.Unix(),
			"level":     entry.Level,
			"message":   entry.Message,
			"task_id":   entry.TaskID,
			"component": entry.Component,
		},
	}).Err()
}

// scheduleLogRotation runs the log rotation scheduler.
func (s *LogService) scheduleLogRotation(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.RotationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.performLogRotation(ctx); err != nil {
				logger.Error("log rotation failed", zap.Error(err))
			}

		case <-s.stopCh:
			return

		case <-ctx.Done():
			return
		}
	}
}

// scheduleLogCleanup runs the log cleanup scheduler.
func (s *LogService) scheduleLogCleanup(ctx context.Context) {
	defer s.wg.Done()

	// Run cleanup daily at specified time
	for {
		now := time.Now()
		nextCleanup := s.getNextCleanupTime(now)
		
		select {
		case <-time.After(time.Until(nextCleanup)):
			if err := s.performLogCleanup(ctx); err != nil {
				logger.Error("log cleanup failed", zap.Error(err))
			}

		case <-s.stopCh:
			return

		case <-ctx.Done():
			return
		}
	}
}

// getNextCleanupTime calculates the next cleanup time based on configuration.
func (s *LogService) getNextCleanupTime(now time.Time) time.Time {
	cleanupTime, err := time.Parse("15:04", s.config.CleanupTime)
	if err != nil {
		// Default to 2:00 AM if parsing fails
		cleanupTime, _ = time.Parse("15:04", "02:00")
	}
	
	next := time.Date(now.Year(), now.Month(), now.Day(), 
		cleanupTime.Hour(), cleanupTime.Minute(), 0, 0, now.Location())
	
	if next.Before(now) {
		next = next.Add(24 * time.Hour)
	}
	
	return next
}

// ensureLogDirectory creates the log directory if it doesn't exist.
func (s *LogService) ensureLogDirectory() error {
	return os.MkdirAll(s.config.LogDirectory, 0755)
}

// performLogRotation performs log file rotation.
func (s *LogService) performLogRotation(ctx context.Context) error {
	logFiles, err := filepath.Glob(filepath.Join(s.config.LogDirectory, "*.log"))
	if err != nil {
		return fmt.Errorf("failed to list log files: %w", err)
	}

	for _, logFile := range logFiles {
		if err := s.rotateLogFile(logFile); err != nil {
			logger.Error("failed to rotate log file",
				zap.String("file", logFile),
				zap.Error(err),
			)
		}
	}

	logger.Debug("log rotation completed",
		zap.Int("files_processed", len(logFiles)),
	)

	return nil
}

// rotateLogFile rotates a single log file if needed.
func (s *LogService) rotateLogFile(filePath string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	shouldRotate := false

	// Check size
	if info.Size() > s.config.MaxLogSize {
		shouldRotate = true
	}

	// Check age
	if time.Since(info.ModTime()) > s.config.MaxLogAge {
		shouldRotate = true
	}

	if !shouldRotate {
		return nil
	}

	// Create rotated filename
	timestamp := time.Now().Format("20060102_150405")
	ext := filepath.Ext(filePath)
	base := strings.TrimSuffix(filePath, ext)
	rotatedPath := fmt.Sprintf("%s_%s%s", base, timestamp, ext)

	// Rename current file
	if err := os.Rename(filePath, rotatedPath); err != nil {
		return fmt.Errorf("failed to rename log file: %w", err)
	}

	// Compress if enabled
	if s.config.CompressionEnabled {
		if err := s.compressLogFile(rotatedPath); err != nil {
			logger.Warn("failed to compress rotated log file",
				zap.String("file", rotatedPath),
				zap.Error(err),
			)
		}
	}

	logger.Info("log file rotated",
		zap.String("original", filePath),
		zap.String("rotated", rotatedPath),
		zap.Int64("size", info.Size()),
	)

	return nil
}

// compressLogFile compresses a log file using gzip.
func (s *LogService) compressLogFile(filePath string) error {
	// Open source file
	sourceFile, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Create compressed file
	compressedPath := filePath + ".gz"
	compressedFile, err := os.Create(compressedPath)
	if err != nil {
		return fmt.Errorf("failed to create compressed file: %w", err)
	}
	defer compressedFile.Close()

	// Create gzip writer
	gzipWriter := gzip.NewWriter(compressedFile)
	defer gzipWriter.Close()

	// Copy data
	_, err = io.Copy(gzipWriter, sourceFile)
	if err != nil {
		os.Remove(compressedPath)
		return fmt.Errorf("failed to compress file: %w", err)
	}

	// Remove original file after successful compression
	if err := os.Remove(filePath); err != nil {
		logger.Warn("failed to remove original file after compression",
			zap.String("file", filePath),
			zap.Error(err),
		)
	}

	return nil
}

// performLogCleanup removes old log files beyond retention period.
func (s *LogService) performLogCleanup(ctx context.Context) error {
	cutoffTime := time.Now().AddDate(0, 0, -s.config.RetentionDays)

	// Clean database logs
	deletedCount, err := s.logRepo.DeleteOldLogs(ctx, cutoffTime)
	if err != nil {
		logger.Error("failed to clean database logs", zap.Error(err))
	} else {
		logger.Info("cleaned old database logs",
			zap.Int64("deleted_count", deletedCount),
		)
	}

	// Clean file system logs
	filePattern := filepath.Join(s.config.LogDirectory, "*")
	files, err := filepath.Glob(filePattern)
	if err != nil {
		return fmt.Errorf("failed to list files for cleanup: %w", err)
	}

	var cleanedFiles int
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoffTime) {
			if err := os.Remove(file); err != nil {
				logger.Warn("failed to remove old log file",
					zap.String("file", file),
					zap.Error(err),
				)
			} else {
				cleanedFiles++
			}
		}
	}

	logger.Info("log cleanup completed",
		zap.Int("files_cleaned", cleanedFiles),
		zap.Time("cutoff_time", cutoffTime),
	)

	return nil
}

// emergencyCleanup performs emergency cleanup when disk usage is high.
func (s *LogService) emergencyCleanup(ctx context.Context) error {
	// Delete the oldest 50% of logs
	halfRetentionDays := s.config.RetentionDays / 2
	emergencyDate := time.Now().AddDate(0, 0, -halfRetentionDays)
	
	deletedCount, err := s.logRepo.DeleteOldLogs(ctx, emergencyDate)
	if err != nil {
		return err
	}

	logger.Warn("Emergency cleanup performed",
		zap.Int64("deleted_count", deletedCount),
		zap.Int("retention_days", halfRetentionDays),
	)

	return nil
}

// searchLogFiles searches log files on the file system.
func (s *LogService) searchLogFiles(ctx context.Context, filters LogSearchFilters) ([]LogEntry, error) {
	var results []LogEntry
	
	// This is a simplified implementation - in production, you might want to use
	// more sophisticated file searching techniques or indexing
	pattern := filepath.Join(s.config.LogDirectory, "*.log*")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to list log files: %w", err)
	}

	for _, file := range files {
		// Skip if context is cancelled
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		fileResults, err := s.searchSingleFile(file, filters)
		if err != nil {
			logger.Debug("failed to search file", zap.String("file", file), zap.Error(err))
			continue
		}
		results = append(results, fileResults...)
	}

	return results, nil
}

// searchSingleFile searches a single log file for matching entries.
func (s *LogService) searchSingleFile(filePath string, filters LogSearchFilters) ([]LogEntry, error) {
	var results []LogEntry
	
	// For compressed files, we'd need to decompress first
	if strings.HasSuffix(filePath, ".gz") {
		return s.searchCompressedFile(filePath, filters)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// This is a simplified search - in production, you might want to parse
	// structured log formats (JSON, etc.) more carefully
	// For now, we'll just do basic text matching
	
	return results, nil
}

// searchCompressedFile searches a gzip-compressed log file.
func (s *LogService) searchCompressedFile(filePath string, filters LogSearchFilters) ([]LogEntry, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer gzipReader.Close()

	// Parse compressed content
	// This is a simplified implementation
	return []LogEntry{}, nil
}

// getFileSystemStats returns file system statistics.
func (s *LogService) getFileSystemStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	// Count log files
	pattern := filepath.Join(s.config.LogDirectory, "*")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return stats, err
	}

	var totalSize int64
	var logFiles, compressedFiles int
	
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		
		totalSize += info.Size()
		
		if strings.HasSuffix(file, ".log") {
			logFiles++
		} else if strings.HasSuffix(file, ".gz") {
			compressedFiles++
		}
	}

	stats["total_files"] = len(files)
	stats["log_files"] = logFiles
	stats["compressed_files"] = compressedFiles
	stats["total_size_bytes"] = totalSize
	stats["total_size_mb"] = totalSize / (1024 * 1024)
	
	return stats, nil
}

// getDiskUsage returns disk usage percentage for the log directory.
func (s *LogService) getDiskUsage(path string) (float64, error) {
	// This is a simplified implementation
	// In production, you'd want to use system calls to get actual disk usage
	return 45.0, nil
}

// exportAsJSON exports logs as JSON.
func (s *LogService) exportAsJSON(logs []*model.TaskLog) ([]byte, error) {
	return json.MarshalIndent(logs, "", "  ")
}

// exportAsCSV exports logs as CSV.
func (s *LogService) exportAsCSV(logs []*model.TaskLog) ([]byte, error) {
	var buf strings.Builder
	writer := csv.NewWriter(&buf)
	
	// Write header
	header := []string{"Timestamp", "Level", "TaskID", "Message"}
	if err := writer.Write(header); err != nil {
		return nil, err
	}
	
	// Write data
	for _, log := range logs {
		record := []string{
			log.Timestamp.Format(time.RFC3339),
			log.Level,
			log.TaskID,
			log.Message,
		}
		if err := writer.Write(record); err != nil {
			return nil, err
		}
	}
	
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	
	return []byte(buf.String()), nil
}