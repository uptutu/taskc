-- TaskC Database Initialization Script
-- Create database and user if they don't exist

CREATE DATABASE IF NOT EXISTS taskc CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE taskc;

-- Tasks table
CREATE TABLE IF NOT EXISTS tasks (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    task_id VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    status ENUM('healthy', 'suspected', 'failed') DEFAULT 'healthy',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_task_id (task_id),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at)
);

-- Heartbeats table
CREATE TABLE IF NOT EXISTS heartbeats (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    task_id VARCHAR(100) NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    metadata JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_task_id (task_id),
    INDEX idx_timestamp (timestamp),
    INDEX idx_created_at (created_at),
    FOREIGN KEY (task_id) REFERENCES tasks(task_id) ON DELETE CASCADE
);

-- Probe configurations table
CREATE TABLE IF NOT EXISTS probe_configs (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    task_id VARCHAR(100) NOT NULL,
    type ENUM('http', 'tcp', 'ping') NOT NULL,
    config JSON NOT NULL,
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_task_id (task_id),
    INDEX idx_type (type),
    INDEX idx_enabled (enabled),
    FOREIGN KEY (task_id) REFERENCES tasks(task_id) ON DELETE CASCADE
);

-- Probe results table
CREATE TABLE IF NOT EXISTS probe_results (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    probe_config_id BIGINT NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    result ENUM('SUCCESS', 'FAILURE') NOT NULL,
    latency_ms INT DEFAULT 0,
    error_message TEXT,
    response_data JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_probe_config_id (probe_config_id),
    INDEX idx_timestamp (timestamp),
    INDEX idx_result (result),
    INDEX idx_created_at (created_at),
    FOREIGN KEY (probe_config_id) REFERENCES probe_configs(id) ON DELETE CASCADE
);

-- Alerts table
CREATE TABLE IF NOT EXISTS alerts (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    task_id VARCHAR(100) NOT NULL,
    level ENUM('info', 'warning', 'critical') NOT NULL,
    title VARCHAR(200) NOT NULL,
    message TEXT NOT NULL,
    channels JSON NOT NULL,
    sent BOOLEAN DEFAULT FALSE,
    sent_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_task_id (task_id),
    INDEX idx_level (level),
    INDEX idx_sent (sent),
    INDEX idx_created_at (created_at),
    FOREIGN KEY (task_id) REFERENCES tasks(task_id) ON DELETE CASCADE
);

-- Task logs table
CREATE TABLE IF NOT EXISTS task_logs (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    task_id VARCHAR(100) NOT NULL,
    level ENUM('debug', 'info', 'warning', 'error', 'fatal') NOT NULL,
    message TEXT NOT NULL,
    component VARCHAR(100),
    timestamp TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_task_id (task_id),
    INDEX idx_level (level),
    INDEX idx_component (component),
    INDEX idx_timestamp (timestamp),
    INDEX idx_created_at (created_at),
    FOREIGN KEY (task_id) REFERENCES tasks(task_id) ON DELETE CASCADE
);

-- Insert sample data for testing
INSERT IGNORE INTO tasks (task_id, name, description, status) VALUES
('web-server-01', 'Web Server 01', 'Main web server instance', 'healthy'),
('database-01', 'Database Server 01', 'Primary MySQL database server', 'healthy'),
('api-gateway', 'API Gateway', 'Main API gateway service', 'healthy'),
('worker-service', 'Background Worker', 'Background job processing service', 'healthy');

-- Insert sample probe configurations
INSERT IGNORE INTO probe_configs (task_id, type, config, enabled) VALUES
('web-server-01', 'http', '{"url": "http://localhost:8080/health", "timeout": 10, "expected_status": 200}', true),
('database-01', 'tcp', '{"host": "localhost", "port": 3306, "timeout": 5}', true),
('api-gateway', 'http', '{"url": "http://localhost:8080/api/v1/health", "timeout": 10, "expected_status": 200}', true);

-- Create database user and grant permissions
CREATE USER IF NOT EXISTS 'taskc_user'@'%' IDENTIFIED BY 'taskc_password_2024';
GRANT ALL PRIVILEGES ON taskc.* TO 'taskc_user'@'%';
FLUSH PRIVILEGES;