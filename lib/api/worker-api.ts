import { apiClient } from './client';
import type { WorkerTask, TaskUpdate, TaskStatus } from './workers';

interface ApiResponse<T> {
  success: boolean;
  data: T;
  message?: string;
}

const BASE = '/worker';

// Get my assigned tasks
export async function getMyTasks(status?: TaskStatus): Promise<WorkerTask[]> {
  const query = status ? `?status=${status}` : '';
  const res = await apiClient.get<ApiResponse<WorkerTask[]>>(`${BASE}/tasks${query}`);
  return res.data;
}

// Get a specific task
export async function getMyTask(id: string): Promise<WorkerTask> {
  const res = await apiClient.get<ApiResponse<WorkerTask>>(`${BASE}/tasks/${id}`);
  return res.data;
}

// Update task status
export async function updateMyTaskStatus(id: string, status: TaskStatus): Promise<WorkerTask> {
  const res = await apiClient.put<ApiResponse<WorkerTask>>(`${BASE}/tasks/${id}/status`, { status });
  return res.data;
}

// Get updates for a task
export async function getMyTaskUpdates(taskId: string): Promise<TaskUpdate[]> {
  const res = await apiClient.get<ApiResponse<TaskUpdate[]>>(`${BASE}/tasks/${taskId}/updates`);
  return res.data;
}

// Post an update to a task
export async function postTaskUpdate(taskId: string, content: string): Promise<TaskUpdate> {
  const res = await apiClient.post<ApiResponse<TaskUpdate>>(`${BASE}/tasks/${taskId}/updates`, { content });
  return res.data;
}

// Send heartbeat
export async function sendHeartbeat(): Promise<void> {
  await apiClient.post<ApiResponse<null>>(`${BASE}/heartbeat`, {});
}
