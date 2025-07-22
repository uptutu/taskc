import api from './client';
import { Task, TaskHealth, TaskMetrics, Alert, Heartbeat, ProbeConfig, PaginatedResponse } from '@/types';

export const taskApi = {
  // 任务管理
  getTasks: (page = 1, limit = 20, status?: string) =>
    api.get<PaginatedResponse<Task>>('/tasks', { params: { page, limit, status } }),

  getTask: (taskId: string) =>
    api.get<Task>(`/tasks/${taskId}`),

  createTask: (data: { task_id: string; name: string; description?: string }) =>
    api.post<Task>('/tasks', data),

  updateTaskStatus: (taskId: string, status: string) =>
    api.put(`/tasks/${taskId}/status`, { status }),

  deleteTask: (taskId: string) =>
    api.delete(`/tasks/${taskId}`),

  // 健康状态
  getTaskHealth: (taskId: string) =>
    api.get<TaskHealth>(`/tasks/${taskId}/health`),

  getTaskMetrics: (taskId: string, hours = 24) =>
    api.get<TaskMetrics>(`/tasks/${taskId}/metrics`, { params: { hours } }),

  // 心跳管理
  sendHeartbeat: (taskId: string, data: { timestamp: number; metadata: any }) =>
    api.post(`/heartbeat/${taskId}`, data),

  // 探测管理
  triggerProbe: (data: { task_id: string; type: string; config: any }) =>
    api.post('/probe', data),

  // 告警管理
  getAlerts: (page = 1, limit = 20, level?: string) =>
    api.get<PaginatedResponse<Alert>>('/alerts', { params: { page, limit, level } }),

  getTaskAlerts: (taskId: string, limit = 10) =>
    api.get<Alert[]>(`/tasks/${taskId}/alerts`, { params: { limit } }),
};

export const dashboardApi = {
  getOverview: () =>
    api.get('/dashboard/overview'),

  getSystemStats: () =>
    api.get('/dashboard/stats'),
};