import { apiClient } from './client';

// Types
export type TaskStatus = 'pending' | 'in_progress' | 'completed' | 'failed';
export type TaskPriority = 'low' | 'medium' | 'high' | 'urgent';

export const TASK_STATUSES: TaskStatus[] = ['pending', 'in_progress', 'completed', 'failed'];
export const TASK_PRIORITIES: TaskPriority[] = ['low', 'medium', 'high', 'urgent'];

export const STATUS_LABELS: Record<TaskStatus, string> = {
  pending: 'Pending',
  in_progress: 'In Progress',
  completed: 'Completed',
  failed: 'Failed',
};

export const PRIORITY_LABELS: Record<TaskPriority, string> = {
  low: 'Low',
  medium: 'Medium',
  high: 'High',
  urgent: 'Urgent',
};

export const STATUS_COLORS: Record<TaskStatus, string> = {
  pending: 'bg-gray-100 text-gray-700',
  in_progress: 'bg-blue-100 text-blue-700',
  completed: 'bg-green-100 text-green-700',
  failed: 'bg-red-100 text-red-700',
};

export const PRIORITY_COLORS: Record<TaskPriority, string> = {
  low: 'bg-gray-100 text-gray-600',
  medium: 'bg-yellow-100 text-yellow-700',
  high: 'bg-orange-100 text-orange-700',
  urgent: 'bg-red-100 text-red-700',
};

export interface TaskType {
  id: number;
  name: string;
  label: string;
  sop?: string;
  param_schema?: Record<string, string> | null;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface Worker {
  id: string;
  email: string;
  full_name: string;
  last_login_at: string | null;
  last_activity_at: string | null;
  created_at: string;
  task_count: number;
  is_online: boolean;
}

export interface WorkerTask {
  id: string;
  title: string;
  description: string;
  assigned_to: string | null;
  assigned_to_name?: string | null;
  status: TaskStatus;
  priority: TaskPriority;
  task_type_id: number | null;
  task_type?: { id: number; name: string; label: string; sop?: string } | null;
  params: Record<string, unknown> | null;
  result: Record<string, unknown> | null;
  retry_count: number;
  created_by: string | null;
  created_by_name?: string | null;
  created_at: string;
  updated_at: string;
  started_at: string | null;
  completed_at: string | null;
}

export interface TaskUpdate {
  id: string;
  task_id: string;
  content: string;
  created_by: string | null;
  created_by_name?: string | null;
  created_at: string;
}

interface ApiResponse<T> {
  success: boolean;
  data: T;
  message?: string;
}

const BASE = '/admin/workers';

// Workers
export async function listWorkers(): Promise<Worker[]> {
  const res = await apiClient.get<ApiResponse<Worker[]>>(BASE);
  return res.data;
}

export async function registerWorker(email: string, password: string, fullName: string): Promise<Worker> {
  const res = await apiClient.post<ApiResponse<Worker>>(BASE, { email, password, full_name: fullName });
  return res.data;
}

export async function removeWorker(id: string): Promise<void> {
  await apiClient.delete(`${BASE}/${id}`);
}

// Task Types
export async function listTaskTypes(): Promise<TaskType[]> {
  const res = await apiClient.get<ApiResponse<TaskType[]>>(`${BASE}/task-types`);
  return res.data;
}

export async function createTaskType(data: {
  name: string;
  label: string;
  sop?: string;
  param_schema?: Record<string, string> | null;
}): Promise<TaskType> {
  const res = await apiClient.post<ApiResponse<TaskType>>(`${BASE}/task-types`, data);
  return res.data;
}

export async function updateTaskType(id: number, data: {
  label?: string;
  sop?: string;
  param_schema?: Record<string, string> | null;
}): Promise<TaskType> {
  const res = await apiClient.put<ApiResponse<TaskType>>(`${BASE}/task-types/${id}`, data);
  return res.data;
}

export async function deleteTaskType(id: number): Promise<void> {
  await apiClient.delete(`${BASE}/task-types/${id}`);
}

// Tasks
export async function listTasks(params?: { status?: TaskStatus; assigned_to?: string; task_type?: string }): Promise<WorkerTask[]> {
  const query = new URLSearchParams();
  if (params?.status) query.set('status', params.status);
  if (params?.assigned_to) query.set('assigned_to', params.assigned_to);
  if (params?.task_type) query.set('task_type', params.task_type);
  const qs = query.toString();
  const res = await apiClient.get<ApiResponse<WorkerTask[]>>(`${BASE}/tasks${qs ? '?' + qs : ''}`);
  return res.data;
}

export async function getTask(id: string): Promise<WorkerTask> {
  const res = await apiClient.get<ApiResponse<WorkerTask>>(`${BASE}/tasks/${id}`);
  return res.data;
}

export async function createTask(data: {
  title: string;
  description?: string;
  assigned_to?: string;
  priority?: TaskPriority;
  task_type_id?: number;
  params?: Record<string, unknown>;
}): Promise<WorkerTask> {
  const res = await apiClient.post<ApiResponse<WorkerTask>>(`${BASE}/tasks`, data);
  return res.data;
}

export async function updateTask(id: string, data: {
  title?: string;
  description?: string;
  assigned_to?: string;
  status?: TaskStatus;
  priority?: TaskPriority;
  task_type_id?: number;
  params?: Record<string, unknown>;
}): Promise<WorkerTask> {
  const res = await apiClient.put<ApiResponse<WorkerTask>>(`${BASE}/tasks/${id}`, data);
  return res.data;
}

export async function deleteTask(id: string): Promise<void> {
  await apiClient.delete(`${BASE}/tasks/${id}`);
}

// Task Updates
export async function listTaskUpdates(taskId: string): Promise<TaskUpdate[]> {
  const res = await apiClient.get<ApiResponse<TaskUpdate[]>>(`${BASE}/tasks/${taskId}/updates`);
  return res.data;
}

export async function createTaskUpdate(taskId: string, content: string): Promise<TaskUpdate> {
  const res = await apiClient.post<ApiResponse<TaskUpdate>>(`${BASE}/tasks/${taskId}/updates`, { content });
  return res.data;
}

// Task Data (S3-backed)
export interface TaskDataFile {
  key: string;
  data_type: string;
  item_count: number;
  size: number;
  uploaded_at: string;
  uploaded_by?: string;
}

export interface TaskDataFilesResponse {
  files: TaskDataFile[];
  total: number;
  limit: number;
  offset: number;
}

export interface TaskDataBatch {
  task_id: string;
  data_type: string;
  uploaded_by: string;
  uploaded_at: string;
  item_count: number;
  items: Array<{
    ticker: string | null;
    external_id: string | null;
    collected_at: string | null;
    data: Record<string, unknown>;
  }>;
}

export async function getTaskData(
  taskId: string,
  params?: { data_type?: string; limit?: number; offset?: number }
): Promise<TaskDataFilesResponse> {
  const query = new URLSearchParams();
  if (params?.data_type) query.set('data_type', params.data_type);
  if (params?.limit) query.set('limit', String(params.limit));
  if (params?.offset) query.set('offset', String(params.offset));
  const qs = query.toString();
  const res = await apiClient.get<ApiResponse<TaskDataFilesResponse>>(`${BASE}/tasks/${taskId}/data${qs ? '?' + qs : ''}`);
  return res.data;
}

export async function getTaskDataFile(taskId: string, key: string): Promise<TaskDataBatch> {
  const query = new URLSearchParams({ key });
  const res = await apiClient.get<ApiResponse<TaskDataBatch>>(`${BASE}/tasks/${taskId}/data/file?${query.toString()}`);
  return res.data;
}
