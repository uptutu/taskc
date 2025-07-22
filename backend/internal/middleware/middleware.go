package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/websocket/v2"
)

// CORS 中间件
func CORS() fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowMethods:     "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
		AllowCredentials: true,
	})
}

// Logger 中间件
func Logger() fiber.Handler {
	return logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} - ${latency}\n",
	})
}

// RateLimiter 限流中间件
func RateLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        100,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(429).JSON(fiber.Map{
				"error": "Too many requests",
			})
		},
	})
}

// WebSocketUpgrade WebSocket升级中间件
func WebSocketUpgrade() fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		// WebSocket连接处理逻辑在handlers中实现
	})
}

// Auth JWT认证中间件
func Auth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// JWT验证逻辑
		token := c.Get("Authorization")
		if token == "" {
			return c.Status(401).JSON(fiber.Map{
				"error": "Authorization token required",
			})
		}

		// 验证JWT token
		// 这里简化处理，实际应该验证JWT
		
		return c.Next()
	}
}