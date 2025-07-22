package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"taskc/backend/internal/api/handlers"
	"taskc/backend/internal/api/routes"
	"taskc/backend/internal/config"
	"taskc/backend/internal/repository"
	"taskc/backend/internal/service"
	"taskc/backend/migrations"
	"taskc/backend/pkg/logger"

	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// 初始化日志
	if err := logger.Init(cfg.Log.Level, cfg.Log.Format, cfg.Log.Output); err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer logger.Sync()

	// 连接数据库
	db, err := connectDatabase(cfg)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}

	// 运行数据库迁移
	if err := migration.AutoMigrate(db); err != nil {
		logger.Fatal("Failed to migrate database", zap.Error(err))
	}

	if err := migration.CreateIndexes(db); err != nil {
		logger.Warn("Failed to create indexes", zap.Error(err))
	}

	// 连接Redis
	redisClient, err := connectRedis(cfg)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}

	// 初始化Repository
	taskRepo := repository.NewTaskRepository(db)
	heartbeatRepo := repository.NewHeartbeatRepository(db)
	alertRepo := repository.NewAlertRepository(db)
	probeRepo := repository.NewProbeRepository(db)

	// 初始化Service
	taskService := service.NewTaskService(taskRepo, heartbeatRepo, alertRepo)
	heartbeatService := service.NewHeartbeatService(heartbeatRepo, taskRepo, redisClient, service.DefaultHeartbeatConfig())
	probeService := service.NewProbeService(probeRepo, taskRepo, service.DefaultProbeConfig())

	// 初始化Handler
	taskHandler := handlers.NewTaskHandler(taskService, heartbeatService, probeService)

	// 创建Fiber应用
	app := fiber.New(fiber.Config{
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// 设置路由
	routes.SetupRoutes(app, taskHandler)

	// 启动定时任务
	startCronJobs(cfg, heartbeatService)

	// 启动服务器
	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
		logger.Info("Starting server", zap.String("address", addr))
		if err := app.Listen(addr); err != nil {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// 等待信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

func connectDatabase(cfg *config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(cfg.GetDSN()), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	return db, nil
}

func connectRedis(cfg *config.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:       cfg.GetRedisAddr(),
		Password:   cfg.Redis.Password,
		DB:         cfg.Redis.DB,
		PoolSize:   cfg.Redis.PoolSize,
		MaxRetries: cfg.Redis.MaxRetries,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return client, nil
}

func startCronJobs(cfg *config.Config, heartbeatService *service.HeartbeatService) {
	c := cron.New()

	// 心跳检查任务
	c.AddFunc("@every 10s", func() {
		ctx := context.Background()
		if err := heartbeatService.CheckMissedHeartbeats(
			ctx,
			cfg.Heartbeat.Timeout,
			cfg.Heartbeat.MaxMissedBeats,
		); err != nil {
			logger.Error("Failed to check missed heartbeats", zap.Error(err))
		}
	})

	// 日志清理任务
	c.AddFunc(fmt.Sprintf("0 %s * * *", cfg.Log.CleanupTime), func() {
		// 实现日志清理逻辑
		logger.Info("Starting log cleanup job")
	})

	c.Start()
	logger.Info("Cron jobs started")
}