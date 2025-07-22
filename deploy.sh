#!/bin/bash

# TaskC Deployment Script
# This script automates the deployment of TaskC with Docker Compose

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo -e "${BLUE}[TASKC]${NC} $1"
}

# Check if Docker and Docker Compose are installed
check_requirements() {
    print_header "Checking requirements..."
    
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed. Please install Docker first."
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        print_error "Docker Compose is not installed. Please install Docker Compose first."
        exit 1
    fi
    
    # Check Docker daemon is running
    if ! docker info &> /dev/null; then
        print_error "Docker daemon is not running. Please start Docker first."
        exit 1
    fi
    
    print_status "Requirements check passed ✓"
}

# Create necessary directories
setup_directories() {
    print_header "Setting up directories..."
    
    mkdir -p logs data config
    chmod 755 logs data config
    
    # Create log subdirectories
    mkdir -p logs/backend logs/nginx logs/mysql logs/redis
    chmod 755 logs/backend logs/nginx logs/mysql logs/redis
    
    print_status "Directories created ✓"
}

# Generate secure passwords and secrets
generate_secrets() {
    print_header "Generating secure secrets..."
    
    # Generate random passwords
    MYSQL_ROOT_PASSWORD=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-25)
    MYSQL_PASSWORD=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-25)
    JWT_SECRET=$(openssl rand -base64 64 | tr -d "=+/" | cut -c1-50)
    GRAFANA_PASSWORD=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-25)
    
    # Save secrets to file
    cat > .env << EOF
# Generated secrets for TaskC deployment
# Generated on: $(date)
MYSQL_ROOT_PASSWORD=${MYSQL_ROOT_PASSWORD}
MYSQL_PASSWORD=${MYSQL_PASSWORD}
JWT_SECRET=${JWT_SECRET}
GRAFANA_PASSWORD=${GRAFANA_PASSWORD}

# Database configuration
DB_HOST=mysql
DB_PORT=3306
DB_NAME=taskc
DB_USER=taskc_user

# Redis configuration
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
EOF

    chmod 600 .env
    print_status "Secrets generated and saved to .env ✓"
    print_warning "Please save these credentials securely:"
    print_warning "MySQL Root Password: ${MYSQL_ROOT_PASSWORD}"
    print_warning "MySQL User Password: ${MYSQL_PASSWORD}"
    print_warning "Grafana Password: ${GRAFANA_PASSWORD}"
}

# Pull Docker images
pull_images() {
    print_header "Pulling Docker images..."
    
    docker-compose pull
    
    print_status "Docker images pulled ✓"
}

# Start services
start_services() {
    print_header "Starting TaskC services..."
    
    # Start database and redis first
    print_status "Starting database and Redis..."
    docker-compose up -d mysql redis
    
    # Wait for database to be ready
    print_status "Waiting for database to be ready..."
    for i in {1..30}; do
        if docker-compose exec -T mysql mysqladmin ping -h localhost --silent; then
            break
        fi
        echo -n "."
        sleep 2
    done
    echo ""
    
    # Wait for Redis to be ready
    print_status "Waiting for Redis to be ready..."
    for i in {1..15}; do
        if docker-compose exec -T redis redis-cli ping > /dev/null 2>&1; then
            break
        fi
        echo -n "."
        sleep 1
    done
    echo ""
    
    # Start remaining services
    print_status "Starting application services..."
    docker-compose up -d
    
    print_status "All services started ✓"
}

# Wait for services to be healthy
wait_for_health() {
    print_header "Waiting for services to be healthy..."
    
    services=("backend" "frontend" "nginx")
    
    for service in "${services[@]}"; do
        print_status "Waiting for ${service} to be healthy..."
        for i in {1..30}; do
            if [ "$(docker-compose ps -q ${service} | xargs docker inspect --format='{{.State.Health.Status}}')" = "healthy" ]; then
                print_status "${service} is healthy ✓"
                break
            fi
            echo -n "."
            sleep 2
        done
        echo ""
    done
}

# Run health checks
run_health_checks() {
    print_header "Running health checks..."
    
    # Check backend health
    if curl -f http://localhost/health > /dev/null 2>&1; then
        print_status "Backend health check passed ✓"
    else
        print_error "Backend health check failed"
    fi
    
    # Check frontend
    if curl -f http://localhost > /dev/null 2>&1; then
        print_status "Frontend health check passed ✓"
    else
        print_error "Frontend health check failed"
    fi
    
    # Check database connection
    if docker-compose exec -T mysql mysql -u root -p${MYSQL_ROOT_PASSWORD} -e "SELECT 1" > /dev/null 2>&1; then
        print_status "Database connection check passed ✓"
    else
        print_error "Database connection check failed"
    fi
    
    # Check Redis connection
    if docker-compose exec -T redis redis-cli ping > /dev/null 2>&1; then
        print_status "Redis connection check passed ✓"
    else
        print_error "Redis connection check failed"
    fi
}

# Show deployment summary
show_summary() {
    print_header "Deployment Summary"
    echo ""
    echo "🎉 TaskC has been successfully deployed!"
    echo ""
    echo "📊 Access URLs:"
    echo "  • TaskC Web UI:      http://localhost"
    echo "  • TaskC API:         http://localhost/api/v1"
    echo "  • Grafana Dashboard: http://localhost:3001"
    echo "  • Prometheus:        http://localhost:9090"
    echo ""
    echo "🔐 Login Credentials:"
    echo "  • Grafana Username:  admin"
    echo "  • Grafana Password:  ${GRAFANA_PASSWORD}"
    echo ""
    echo "📝 Management Commands:"
    echo "  • View logs:         docker-compose logs -f"
    echo "  • Check status:      docker-compose ps"
    echo "  • Stop services:     docker-compose down"
    echo "  • Update services:   docker-compose pull && docker-compose up -d"
    echo ""
    echo "📖 For more information, see DEPLOYMENT.md"
    echo ""
}

# Main deployment function
deploy() {
    print_header "Starting TaskC Deployment"
    echo ""
    
    check_requirements
    setup_directories
    
    # Check if .env exists, if not generate secrets
    if [ ! -f .env ]; then
        generate_secrets
    else
        print_status "Using existing .env file"
        source .env
    fi
    
    pull_images
    start_services
    wait_for_health
    run_health_checks
    show_summary
}

# Handle script arguments
case "${1:-deploy}" in
    "deploy")
        deploy
        ;;
    "start")
        print_header "Starting TaskC services..."
        docker-compose up -d
        print_status "Services started ✓"
        ;;
    "stop")
        print_header "Stopping TaskC services..."
        docker-compose down
        print_status "Services stopped ✓"
        ;;
    "restart")
        print_header "Restarting TaskC services..."
        docker-compose restart
        print_status "Services restarted ✓"
        ;;
    "logs")
        docker-compose logs -f
        ;;
    "status")
        docker-compose ps
        ;;
    "health")
        run_health_checks
        ;;
    "clean")
        print_header "Cleaning up TaskC deployment..."
        print_warning "This will remove all containers, volumes, and data!"
        read -p "Are you sure? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            docker-compose down -v --remove-orphans
            docker system prune -f
            print_status "Cleanup completed ✓"
        else
            print_status "Cleanup cancelled"
        fi
        ;;
    "help"|"-h"|"--help")
        echo "TaskC Deployment Script"
        echo ""
        echo "Usage: $0 [COMMAND]"
        echo ""
        echo "Commands:"
        echo "  deploy    Deploy TaskC (default)"
        echo "  start     Start services"
        echo "  stop      Stop services"
        echo "  restart   Restart services"
        echo "  logs      Show logs"
        echo "  status    Show service status"
        echo "  health    Run health checks"
        echo "  clean     Clean up deployment (removes all data)"
        echo "  help      Show this help"
        echo ""
        ;;
    *)
        print_error "Unknown command: $1"
        print_status "Use '$0 help' for available commands"
        exit 1
        ;;
esac