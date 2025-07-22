package handlers

import (
	"encoding/json"
	"strconv"
	"time"

	"taskc/backend/internal/model"
	"taskc/backend/internal/service"
	"taskc/backend/pkg/logger"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// TaskHandler handles HTTP requests for task management operations.
// It provides REST endpoints for task CRUD operations, heartbeat processing,
// and health monitoring.
type TaskHandler struct {
	taskService      *service.TaskService
	heartbeatService *service.HeartbeatService
	probeService     *service.ProbeService
}

// HeartbeatRequest represents the incoming heartbeat data structure.
type HeartbeatRequest struct {
	Timestamp int64                    `json:"timestamp" validate:"required"`
	Metadata  model.HeartbeatMetadata `json:"metadata"`
}

// HeartbeatResponse represents the heartbeat processing response.
type HeartbeatResponse struct {
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

// CreateTaskRequest represents the task creation request.
type CreateTaskRequest struct {
	TaskID      string `json:"task_id" validate:"required,min=1,max=100"`
	Name        string `json:"name" validate:"required,min=1,max=200"`
	Description string `json:"description" validate:"max=1000"`
}

// UpdateTaskStatusRequest represents the task status update request.
type UpdateTaskStatusRequest struct {
	Status model.TaskStatus `json:"status" validate:"required"`
}

// ProbeRequest represents the probe execution request.
type ProbeRequest struct {
	TaskID string                 `json:"task_id" validate:"required"`
	Type   model.ProbeType        `json:"type" validate:"required"`
	Config map[string]interface{} `json:"config" validate:"required"`
}

func NewTaskHandler(taskService *service.TaskService, heartbeatService *service.HeartbeatService, probeService *service.ProbeService) *TaskHandler {
	return &TaskHandler{
		taskService:      taskService,
		heartbeatService: heartbeatService,
		probeService:     probeService,
	}
}

// CreateTask creates a new task with validation.
func (h *TaskHandler) CreateTask(c *fiber.Ctx) error {
	var req CreateTaskRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Warn("invalid create task request body", zap.Error(err))
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	// TODO: Add request validation
	// if err := h.validator.Struct(&req); err != nil {
	//     return fiber.NewError(fiber.StatusBadRequest, err.Error())
	// }

	task := &model.Task{
		TaskID:      req.TaskID,
		Name:        req.Name,
		Description: req.Description,
		Status:      model.TaskStatusHealthy,
	}

	if err := h.taskService.CreateTask(c.Context(), task); err != nil {
		logger.Error("failed to create task",
			zap.String("task_id", req.TaskID),
			zap.Error(err),
		)
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create task")
	}

	return c.Status(fiber.StatusCreated).JSON(task)
}

// GetTask retrieves a task by its task ID.
func (h *TaskHandler) GetTask(c *fiber.Ctx) error {
	taskID := c.Params("task_id")
	if taskID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "task_id parameter is required")
	}

	task, err := h.taskService.GetTaskByID(c.Context(), taskID)
	if err != nil {
		logger.Warn("task not found",
			zap.String("task_id", taskID),
			zap.Error(err),
		)
		return fiber.NewError(fiber.StatusNotFound, "task not found")
	}

	return c.JSON(task)
}

// ListTasks retrieves a paginated list of tasks with optional filtering.
func (h *TaskHandler) ListTasks(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	status := c.Query("status")

	tasks, total, err := h.taskService.ListTasks(c.Context(), page, limit, status)
	if err != nil {
		logger.Error("failed to list tasks",
			zap.Int("page", page),
			zap.Int("limit", limit),
			zap.String("status", status),
			zap.Error(err),
		)
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list tasks")
	}

	return c.JSON(fiber.Map{
		"tasks": tasks,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// UpdateTaskStatus updates the status of a task.
func (h *TaskHandler) UpdateTaskStatus(c *fiber.Ctx) error {
	taskID := c.Params("task_id")
	if taskID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "task_id parameter is required")
	}

	var req UpdateTaskStatusRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Warn("invalid update task status request body",
			zap.String("task_id", taskID),
			zap.Error(err),
		)
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if err := h.taskService.UpdateTaskStatus(c.Context(), taskID, req.Status); err != nil {
		logger.Error("failed to update task status",
			zap.String("task_id", taskID),
			zap.String("status", string(req.Status)),
			zap.Error(err),
		)
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update task status")
	}

	return c.JSON(fiber.Map{
		"message": "task status updated successfully",
	})
}

// ReceiveHeartbeat processes incoming heartbeat data and updates task status.
// It performs the following operations:
//   1. Validates heartbeat data
//   2. Records heartbeat to database  
//   3. Triggers status evaluation if needed
//   4. Returns current task status
func (h *TaskHandler) ReceiveHeartbeat(c *fiber.Ctx) error {
	taskID := c.Params("task_id")
	if taskID == "" {
		logger.Warn("heartbeat request missing task_id")
		return fiber.NewError(fiber.StatusBadRequest, "task_id is required")
	}

	var req HeartbeatRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Warn("invalid heartbeat request body",
			zap.String("task_id", taskID),
			zap.Error(err),
		)
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	// Validate timestamp is not too old or in future
	now := time.Now()
	heartbeatTime := time.Unix(req.Timestamp/1000, 0)
	if heartbeatTime.Before(now.Add(-5*time.Minute)) || heartbeatTime.After(now.Add(1*time.Minute)) {
		logger.Warn("heartbeat timestamp out of range",
			zap.String("task_id", taskID),
			zap.Time("heartbeat_time", heartbeatTime),
			zap.Time("current_time", now),
		)
		return fiber.NewError(fiber.StatusBadRequest, "timestamp out of acceptable range")
	}

	metadataJSON, err := json.Marshal(req.Metadata)
	if err != nil {
		logger.Error("failed to marshal heartbeat metadata",
			zap.String("task_id", taskID),
			zap.Error(err),
		)
		return fiber.NewError(fiber.StatusInternalServerError, "failed to process metadata")
	}

	heartbeat := &model.Heartbeat{
		TaskID:    taskID,
		Timestamp: heartbeatTime,
		Metadata:  string(metadataJSON),
	}

	logger.Debug("processing heartbeat",
		zap.String("task_id", taskID),
		zap.Time("timestamp", heartbeatTime),
	)

	if err := h.heartbeatService.RecordHeartbeat(c.Context(), heartbeat); err != nil {
		logger.Error("failed to record heartbeat",
			zap.String("task_id", taskID),
			zap.Error(err),
		)
		return fiber.NewError(fiber.StatusInternalServerError, "failed to record heartbeat")
	}

	// Get current task status for response header
	task, err := h.taskService.GetTaskByID(c.Context(), taskID)
	if err != nil {
		logger.Warn("failed to get task status for heartbeat response",
			zap.String("task_id", taskID),
			zap.Error(err),
		)
		// Don't fail the heartbeat if we can't get status
	} else {
		c.Set("X-Task-Status", string(task.Status))
		logger.Info("heartbeat processed successfully",
			zap.String("task_id", taskID),
			zap.String("status", string(task.Status)),
		)
	}

	return c.Status(fiber.StatusAccepted).JSON(HeartbeatResponse{
		Message:   "heartbeat received",
		Timestamp: now.Unix(),
	})
}

// TriggerProbe executes a probe for task health validation.
func (h *TaskHandler) TriggerProbe(c *fiber.Ctx) error {
	var req ProbeRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Warn("invalid trigger probe request body", zap.Error(err))
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		logger.Error("failed to marshal probe config",
			zap.String("task_id", req.TaskID),
			zap.Error(err),
		)
		return fiber.NewError(fiber.StatusInternalServerError, "failed to process probe config")
	}

	probeConfig := &model.ProbeConfig{
		TaskID: req.TaskID,
		Type:   req.Type,
		Config: string(configJSON),
	}

	result, err := h.probeService.ExecuteProbe(c.Context(), probeConfig)
	if err != nil {
		logger.Error("failed to execute probe",
			zap.String("task_id", req.TaskID),
			zap.String("probe_type", string(req.Type)),
			zap.Error(err),
		)
		return fiber.NewError(fiber.StatusInternalServerError, "failed to execute probe")
	}

	return c.JSON(result)
}

// GetTaskHealth retrieves comprehensive health information for a task.
func (h *TaskHandler) GetTaskHealth(c *fiber.Ctx) error {
	taskID := c.Params("task_id")
	if taskID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "task_id parameter is required")
	}

	health, err := h.taskService.GetTaskHealth(c.Context(), taskID)
	if err != nil {
		logger.Error("failed to get task health",
			zap.String("task_id", taskID),
			zap.Error(err),
		)
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get task health")
	}

	return c.JSON(health)
}

// GetTaskMetrics retrieves aggregated metrics for a task over a time period.
func (h *TaskHandler) GetTaskMetrics(c *fiber.Ctx) error {
	taskID := c.Params("task_id")
	if taskID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "task_id parameter is required")
	}

	hours, _ := strconv.Atoi(c.Query("hours", "24"))

	metrics, err := h.taskService.GetTaskMetrics(c.Context(), taskID, hours)
	if err != nil {
		logger.Error("failed to get task metrics",
			zap.String("task_id", taskID),
			zap.Int("hours", hours),
			zap.Error(err),
		)
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get task metrics")
	}

	return c.JSON(metrics)
}

// DeleteTask removes a task and its related data.
func (h *TaskHandler) DeleteTask(c *fiber.Ctx) error {
	taskID := c.Params("task_id")
	if taskID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "task_id parameter is required")
	}

	if err := h.taskService.DeleteTask(c.Context(), taskID); err != nil {
		logger.Error("failed to delete task",
			zap.String("task_id", taskID),
			zap.Error(err),
		)
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete task")
	}

	return c.JSON(fiber.Map{
		"message": "task deleted successfully",
	})
}