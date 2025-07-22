export interface Task {
  id: number;
  task_id: string;
  name: string;
  description: string;
  status: TaskStatus;
  created_at: string;
  updated_at: string;
}

export enum TaskStatus {
  HEALTHY = 'HEALTHY',
  SUSPECTED = 'SUSPECTED',
  FAILED = 'FAILED',
}

export interface Heartbeat {
  id: number;
  task_id: string;
  timestamp: string;
  metadata: string;
  created_at: string;
}

export interface HeartbeatMetadata {
  cpu_load: number;
  mem_used_mb: number;
  queue_size: number;
  custom_data?: Record<string, any>;
}

export interface Alert {
  id: number;
  task_id: string;
  level: AlertLevel;
  title: string;
  message: string;
  channels: string;
  sent: boolean;
  sent_at?: string;
  created_at: string;
}

export enum AlertLevel {
  CRITICAL = 'CRITICAL',
  WARNING = 'WARNING',
  INFO = 'INFO',
}

export interface ProbeConfig {
  id: number;
  task_id: string;
  type: ProbeType;
  config: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export enum ProbeType {
  HTTP_GET = 'HTTP_GET',
  TCP = 'TCP',
  PING = 'PING',
}

export interface ProbeResult {
  id: number;
  probe_config_id: number;
  result: string;
  latency_ms: number;
  error_message: string;
  timestamp: string;
}

export interface TaskHealth {
  status: TaskStatus;
  last_heartbeat?: string;
  probe_history: ProbeResult[];
  resource_usage: {
    avg_cpu: number;
    max_mem_mb: number;
  };
}

export interface TaskMetrics {
  heartbeat_count: number;
  alert_count: number;
  status_history: Array<{
    timestamp: string;
    status: TaskStatus;
  }>;
  uptime_ratio: number;
}

export interface ApiResponse<T> {
  data?: T;
  error?: string;
  message?: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  limit: number;
}