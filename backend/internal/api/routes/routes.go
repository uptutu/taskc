package routes

import (
	"taskc/backend/internal/api/handlers"
	"taskc/backend/internal/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

func SetupRoutes(app *fiber.App, taskHandler *handlers.TaskHandler) {
	// API版本路由
	api := app.Group("/api/v1")

	// 应用中间件
	api.Use(middleware.CORS())
	api.Use(middleware.Logger())
	api.Use(middleware.RateLimiter())

	// 健康检查
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"time":   c.Context().Time(),
		})
	})

	// 任务管理路由
	tasks := api.Group("/tasks")
	{
		tasks.Post("/", taskHandler.CreateTask)
		tasks.Get("/", taskHandler.ListTasks)
		tasks.Get("/:task_id", taskHandler.GetTask)
		tasks.Put("/:task_id/status", taskHandler.UpdateTaskStatus)
		tasks.Delete("/:task_id", taskHandler.DeleteTask)
		
		// 获取任务健康状态和指标
		tasks.Get("/:task_id/health", taskHandler.GetTaskHealth)
		tasks.Get("/:task_id/metrics", taskHandler.GetTaskMetrics)
	}

	// 心跳管理路由
	heartbeat := api.Group("/heartbeat")
	{
		heartbeat.Post("/:task_id", taskHandler.ReceiveHeartbeat)
	}

	// 探测管理路由
	probe := api.Group("/probe")
	{
		probe.Post("/", taskHandler.TriggerProbe)
	}

	// WebSocket路由（用于实时更新）
	ws := api.Group("/ws")
	ws.Get("/tasks", websocket.New(handlers.HandleTaskWebSocket))
}