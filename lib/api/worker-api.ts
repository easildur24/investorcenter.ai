import { apiClient } from './client';
import { worker } from './routes';
import type { WorkerTask, TaskUpdate, TaskStatus } from './workers';

interface ApiResponse<T> {
  success: boolean;
  data: T;
  message?: string;
}

// Get my assigned tasks
export async function getMyTasks(status?: TaskStatus): Promise<WorkerTask[]> {
  const query = status ? `?status=${status}` : '';
  const res = await apiClient.get<ApiResponse<WorkerTask[]>>(`${worker.tasks}${query}`);
  return res.data;
}

// Get a specific task
export async function getMyTask(id: string): Promise<WorkerTask> {
  const res = await apiClient.get<ApiResponse<WorkerTask>>(worker.taskById(id));
  return res.data;
}

// Update task status
export async function updateMyTaskStatus(id: string, status: TaskStatus): Promise<WorkerTask> {
  const res = await apiClient.put<ApiResponse<WorkerTask>>(worker.taskStatus(id), {
    status,
  });
  return res.data;
}

// Get updates for a task
export async function getMyTaskUpdates(taskId: string): Promise<TaskUpdate[]> {
  const res = await apiClient.get<ApiResponse<TaskUpdate[]>>(worker.taskUpdates(taskId));
  return res.data;
}

// Post an update to a task
export async function postTaskUpdate(taskId: string, content: string): Promise<TaskUpdate> {
  const res = await apiClient.post<ApiResponse<TaskUpdate>>(worker.taskUpdates(taskId), {
    content,
  });
  return res.data;
}

// Send heartbeat
export async function sendHeartbeat(): Promise<void> {
  await apiClient.post<ApiResponse<null>>(worker.heartbeat, {});
}
