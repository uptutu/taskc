# TaskC Deployment Guide

This guide will help you deploy TaskC using Docker and Docker Compose.

## Prerequisites

- Docker 20.10+ and Docker Compose 2.0+
- At least 4GB RAM and 2 CPU cores
- 20GB+ available disk space

## Quick Start

1. **Clone and prepare the repository:**
   ```bash
   git clone <your-repo-url>
   cd taskc
   
   # Create required directories
   mkdir -p logs data config
   chmod 755 logs data config
   ```

2. **Configure environment (optional):**
   ```bash
   # Copy and modify configuration if needed
   cp config/config.yaml config/config.prod.yaml
   
   # Update passwords and secrets in docker-compose.yml
   # Change the following values:
   # - MYSQL_ROOT_PASSWORD
   # - MYSQL_PASSWORD  
   # - JWT_SECRET
   # - GF_SECURITY_ADMIN_PASSWORD
   ```

3. **Start the services:**
   ```bash
   # Start all services
   docker-compose up -d
   
   # View logs
   docker-compose logs -f
   
   # Check service status
   docker-compose ps
   ```

4. **Access the applications:**
   - **TaskC Web UI**: http://localhost (via Nginx)
   - **TaskC API**: http://localhost/api/v1
   - **Grafana Dashboard**: http://localhost:3001
     - Username: `admin`
     - Password: `taskc_grafana_2024`
   - **Prometheus**: http://localhost:9090

## Service Architecture

```
Internet → Nginx (Port 80) → Frontend (Port 3000)
                          → Backend (Port 8080) → MySQL (Port 3306)
                                               → Redis (Port 6379)
                                               
Monitoring: Prometheus (Port 9090) → Grafana (Port 3001)
```

## Configuration

### Environment Variables

Key environment variables in `docker-compose.yml`:

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | mysql | Database host |
| `DB_NAME` | taskc | Database name |
| `DB_USER` | taskc_user | Database user |
| `DB_PASSWORD` | taskc_password_2024 | Database password |
| `REDIS_HOST` | redis | Redis host |
| `JWT_SECRET` | taskc_jwt_secret_... | JWT signing secret |
| `LOG_LEVEL` | info | Logging level |
| `HEARTBEAT_TIMEOUT` | 30s | Heartbeat timeout |
| `LOG_RETENTION_DAYS` | 30 | Log retention period |

### Database Configuration

The MySQL database is automatically initialized with:
- Database: `taskc`
- User: `taskc_user`
- Password: `taskc_password_2024`
- Character set: `utf8mb4`

### Redis Configuration

Redis is configured with:
- No authentication (internal network only)
- 256MB memory limit
- AOF persistence enabled
- Optimized for TaskC workloads

## Health Checks

All services include health checks:

```bash
# Check all service health
docker-compose ps

# Check specific service logs
docker-compose logs backend
docker-compose logs mysql
docker-compose logs redis
```

Health check endpoints:
- Backend: `GET /health`
- Frontend: `GET /health`
- Database: TCP connection check
- Redis: `ping` command

## Monitoring

### Prometheus Metrics

TaskC exposes metrics at `/metrics`:
- HTTP request metrics
- Database connection pool metrics
- Redis connection metrics
- Custom business metrics

### Grafana Dashboards

Pre-configured dashboards include:
- TaskC Overview
- System Health
- Database Performance
- Redis Performance

## Scaling

### Horizontal Scaling

To run multiple backend instances:

```yaml
# In docker-compose.yml
backend:
  # ... existing config
  deploy:
    replicas: 3
```

### Resource Limits

Configure resource limits:

```yaml
services:
  backend:
    deploy:
      resources:
        limits:
          memory: 512M
          cpus: '0.5'
        reservations:
          memory: 256M
          cpus: '0.25'
```

## Security

### Production Security Checklist

- [ ] Change all default passwords
- [ ] Update JWT secret
- [ ] Configure SSL/TLS certificates
- [ ] Set up firewall rules
- [ ] Enable security headers
- [ ] Configure rate limiting
- [ ] Set up log monitoring
- [ ] Enable audit logging

### SSL/TLS Setup

1. Place certificates in `deployments/ssl/`:
   ```
   deployments/ssl/
   ├── cert.pem
   └── key.pem
   ```

2. Update nginx configuration:
   ```nginx
   server {
       listen 443 ssl http2;
       ssl_certificate /etc/nginx/ssl/cert.pem;
       ssl_certificate_key /etc/nginx/ssl/key.pem;
   }
   ```

## Backup and Recovery

### Database Backup

```bash
# Create backup
docker-compose exec mysql mysqldump -u root -p taskc > backup.sql

# Restore backup
docker-compose exec -T mysql mysql -u root -p taskc < backup.sql
```

### Volume Backup

```bash
# Backup all volumes
docker run --rm -v taskc_mysql_data:/data -v $(pwd):/backup alpine tar czf /backup/mysql-backup.tar.gz -C /data .
docker run --rm -v taskc_redis_data:/data -v $(pwd):/backup alpine tar czf /backup/redis-backup.tar.gz -C /data .
```

## Troubleshooting

### Common Issues

1. **Database connection failed**
   ```bash
   # Check database status
   docker-compose logs mysql
   
   # Restart database
   docker-compose restart mysql
   ```

2. **Redis connection failed**
   ```bash
   # Check Redis status
   docker-compose exec redis redis-cli ping
   
   # Restart Redis
   docker-compose restart redis
   ```

3. **Backend service not starting**
   ```bash
   # Check backend logs
   docker-compose logs backend
   
   # Check configuration
   docker-compose exec backend cat /app/config/config.yaml
   ```

### Logs

```bash
# View all logs
docker-compose logs

# Follow specific service logs
docker-compose logs -f backend

# View last 100 lines
docker-compose logs --tail=100
```

### Performance Tuning

1. **Database Performance**
   - Increase `innodb_buffer_pool_size`
   - Optimize queries with `EXPLAIN`
   - Monitor slow query log

2. **Redis Performance**
   - Increase `maxmemory` if needed
   - Monitor memory usage
   - Optimize data structures

3. **Application Performance**
   - Increase worker processes
   - Tune connection pools
   - Enable HTTP/2

## Maintenance

### Updates

```bash
# Pull latest images
docker-compose pull

# Restart with new images
docker-compose up -d

# Clean up old images
docker image prune
```

### Log Rotation

Logs are automatically rotated by the application, but you can also configure Docker log rotation:

```yaml
services:
  backend:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

## Support

For issues and support:
1. Check logs: `docker-compose logs`
2. Verify configuration: Check `config/config.yaml`
3. Check resource usage: `docker stats`
4. Review health checks: `docker-compose ps`