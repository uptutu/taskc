CREATE DATABASE IF NOT EXISTS taskc CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE taskc;

-- 创建用户表
CREATE TABLE IF NOT EXISTS users (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(100) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    role VARCHAR(20) DEFAULT 'user',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    INDEX idx_users_username (username),
    INDEX idx_users_email (email),
    INDEX idx_users_deleted_at (deleted_at)
);

-- 插入默认管理员用户
INSERT IGNORE INTO users (username, email, password, role) VALUES 
('admin', 'admin@taskc.com', '$2a$10$rO1FmGXJ5xtpRQv7GFzQ8OKN.7YBqiU.0VqFGIzEZqf0hXJPnQU8m', 'admin');

-- 插入一些测试任务
INSERT IGNORE INTO tasks (task_id, name, description, status) VALUES 
('user-service', '用户服务', '负责用户认证和管理的核心服务', 'HEALTHY'),
('order-service', '订单服务', '处理订单创建和支付的服务', 'HEALTHY'),
('notification-service', '通知服务', '发送邮件和短信通知的服务', 'SUSPECTED'),
('payment-service', '支付服务', '处理支付相关逻辑的服务', 'FAILED');

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_task_id ON tasks(task_id);
CREATE INDEX IF NOT EXISTS idx_heartbeats_task_timestamp ON heartbeats(task_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_alerts_task_created ON alerts(task_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_task_logs_task_created ON task_logs(task_id, created_at DESC);