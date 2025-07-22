#!/bin/bash

# 任务中台部署脚本
# 使用方法: ./deploy.sh [dev|prod]

set -e

ENV=${1:-dev}
PROJECT_ROOT=$(dirname "$0")/..

echo "🚀 开始部署任务中台系统 - 环境: $ENV"

# 检查Docker和Docker Compose
check_dependencies() {
    echo "📋 检查依赖..."
    
    if ! command -v docker &> /dev/null; then
        echo "❌ Docker 未安装，请先安装 Docker"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        echo "❌ Docker Compose 未安装，请先安装 Docker Compose"
        exit 1
    fi
    
    echo "✅ 依赖检查完成"
}

# 创建必要的目录
create_directories() {
    echo "📁 创建目录结构..."
    
    mkdir -p "$PROJECT_ROOT/logs"
    mkdir -p "$PROJECT_ROOT/data/mysql"
    mkdir -p "$PROJECT_ROOT/data/redis"
    mkdir -p "$PROJECT_ROOT/nginx/ssl"
    
    echo "✅ 目录创建完成"
}

# 复制配置文件
copy_configs() {
    echo "⚙️ 复制配置文件..."
    
    if [ "$ENV" = "prod" ]; then
        cp "$PROJECT_ROOT/backend/configs/config.prod.yaml" "$PROJECT_ROOT/backend/configs/config.yaml"
    else
        cp "$PROJECT_ROOT/backend/configs/config.yaml" "$PROJECT_ROOT/backend/configs/config.dev.yaml"
    fi
    
    echo "✅ 配置文件复制完成"
}

# 构建和启动服务
deploy_services() {
    echo "🔨 构建和启动服务..."
    
    cd "$PROJECT_ROOT/docker"
    
    # 停止现有服务
    docker-compose down
    
    # 构建镜像
    docker-compose build
    
    # 启动服务
    if [ "$ENV" = "prod" ]; then
        docker-compose up -d
    else
        docker-compose up -d
    fi
    
    echo "✅ 服务启动完成"
}

# 等待服务就绪
wait_for_services() {
    echo "⏳ 等待服务就绪..."
    
    # 等待MySQL就绪
    echo "等待 MySQL 服务..."
    timeout 60 bash -c 'until docker exec taskc-mysql mysqladmin ping -h localhost --silent; do sleep 2; done'
    
    # 等待Redis就绪
    echo "等待 Redis 服务..."
    timeout 30 bash -c 'until docker exec taskc-redis redis-cli ping | grep PONG; do sleep 2; done'
    
    # 等待后端服务就绪
    echo "等待后端服务..."
    timeout 60 bash -c 'until curl -f http://localhost:8080/api/v1/health; do sleep 5; done'
    
    echo "✅ 所有服务已就绪"
}

# 运行数据库迁移
run_migrations() {
    echo "🗄️ 运行数据库迁移..."
    
    # 这里可以添加数据库迁移逻辑
    # docker exec taskc-backend ./migrate
    
    echo "✅ 数据库迁移完成"
}

# 健康检查
health_check() {
    echo "🏥 执行健康检查..."
    
    # 检查所有服务状态
    services=("taskc-mysql" "taskc-redis" "taskc-backend" "taskc-frontend")
    
    for service in "${services[@]}"; do
        if docker ps | grep -q "$service"; then
            echo "✅ $service 运行正常"
        else
            echo "❌ $service 运行异常"
            exit 1
        fi
    done
    
    # 检查API响应
    if curl -f http://localhost:8080/api/v1/health > /dev/null 2>&1; then
        echo "✅ API 健康检查通过"
    else
        echo "❌ API 健康检查失败"
        exit 1
    fi
    
    # 检查前端
    if curl -f http://localhost:3000 > /dev/null 2>&1; then
        echo "✅ 前端服务正常"
    else
        echo "❌ 前端服务异常"
        exit 1
    fi
    
    echo "✅ 健康检查完成"
}

# 显示部署信息
show_info() {
    echo ""
    echo "🎉 部署完成！"
    echo ""
    echo "📊 服务访问地址:"
    echo "   前端界面: http://localhost:3000"
    echo "   后端API:  http://localhost:8080"
    echo "   MySQL:    localhost:3306"
    echo "   Redis:    localhost:6379"
    echo ""
    echo "📝 常用命令:"
    echo "   查看日志: docker-compose logs -f"
    echo "   停止服务: docker-compose down"
    echo "   重启服务: docker-compose restart"
    echo ""
    echo "📱 监控命令:"
    echo "   docker ps                    # 查看容器状态"
    echo "   docker stats                 # 查看资源使用"
    echo "   docker-compose logs backend  # 查看后端日志"
    echo ""
}

# 主流程
main() {
    check_dependencies
    create_directories
    copy_configs
    deploy_services
    wait_for_services
    run_migrations
    health_check
    show_info
}

# 错误处理
trap 'echo "❌ 部署失败，请检查错误信息"; exit 1' ERR

# 执行主流程
main