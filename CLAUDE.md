# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Backend Development
```bash
# Navigate to backend directory
cd backend

# Install Go dependencies
go mod download

# Run development server
go run cmd/server/main.go

# Build for production
go build -o bin/server cmd/server/main.go
```

### Frontend Development
```bash
# Navigate to frontend directory
cd frontend

# Install dependencies
npm install

# Start development server
npm run dev

# Build for production
npm run build

# Run linting
npm run lint

# Format code
npm run format
```

### Full Stack Development with Docker
```bash
# Deploy entire stack (recommended for development)
./scripts/deploy.sh

# Deploy for production
./scripts/deploy.sh prod

# Start only database services for local development
cd docker && docker-compose up -d mysql redis

# View logs
docker-compose logs -f

# Stop all services
docker-compose down
```

## Architecture Overview

This is a distributed task scheduling and monitoring platform built with Go backend and React frontend.

### Core Components

**Backend (Go + Fiber)**
- `cmd/server/main.go`: Application entry point
- `internal/api/`: REST API handlers and routes
- `internal/service/`: Business logic layer
- `internal/repository/`: Database access layer
- `internal/model/`: Data models and structures
- `pkg/`: Reusable packages (logger, redis, utils)

**Frontend (React + TypeScript)**
- `src/pages/`: Main application screens (Dashboard, TaskList, AlertList, etc.)
- `src/components/`: Reusable UI components
- `src/store/`: State management using Zustand
- `src/api/`: API client and HTTP utilities
- `src/types/`: TypeScript type definitions

**Key Architectural Patterns:**
- Clean Architecture with distinct layers (handler -> service -> repository)
- State management via Zustand stores for frontend
- WebSocket real-time communication for live updates
- Redis for caching and message queuing
- MySQL for persistent data storage

### Service Dependencies
- **MySQL 8.0**: Primary database for task metadata, alerts, logs
- **Redis 7.0**: Caching layer and message streaming for heartbeats
- **Nginx**: Reverse proxy and load balancer

## Core Functionality

**Task Health Monitoring:**
- Heartbeat system with <5s detection latency
- State machine: HEALTHY → SUSPECTED → FAILED
- Active probing via HTTP/TCP/Ping protocols
- Resource monitoring (CPU, memory, queue metrics)

**Alert Management:**
- Multi-channel alerts: SMS, Email, Slack
- Rate limiting to prevent alert storms
- Configurable severity levels and escalation

**Log Management:**
- Intelligent retention policies (default 90 days)
- Automatic cleanup based on disk thresholds
- Audit log permanent retention

## Configuration

Backend configuration is in `backend/configs/config.yaml` with environment-specific overrides.

Key configuration sections:
- `server`: HTTP server settings
- `database`: MySQL connection parameters
- `heartbeat`: Health check intervals and timeouts
- `alert`: Notification channel settings
- `log`: Retention and cleanup policies

## API Endpoints

**Core Health APIs:**
- `POST /api/v1/heartbeat/{task_id}`: Submit heartbeat data
- `GET /api/v1/tasks/{task_id}/health`: Query health status
- `POST /api/v1/probe`: Trigger active probe
- `GET /api/v1/health`: Service health check

**Frontend URLs:**
- Dashboard: `http://localhost:3000`
- Backend API: `http://localhost:8080`

## Database Schema

The system uses MySQL with GORM for ORM. Key entities:
- Tasks: Core task definitions and metadata
- Heartbeats: Health check data streams
- Alerts: Notification history and configuration
- Probes: Active health check results
- Logs: System and application logs

Migrations are handled in `backend/migrations/`.

## Development Workflow

1. Start dependencies: `cd docker && docker-compose up -d mysql redis`
2. Backend: `cd backend && go run cmd/server/main.go`
3. Frontend: `cd frontend && npm run dev`
4. Access frontend at `http://localhost:3000`

For production-like testing, use `./scripts/deploy.sh` which orchestrates the entire stack with Docker Compose.