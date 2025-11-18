import { apiClient } from './client';

export interface PaginationMeta {
  total: number;
  limit: number;
  offset: number;
}

export interface AdminDataResponse<T> {
  data: T[];
  meta: PaginationMeta;
}

export interface DatabaseStats {
  [tableName: string]: number;
}

// Fetch stocks (admin)
export async function getAdminStocks(params?: {
  limit?: number;
  offset?: number;
  search?: string;
  sort?: string;
  order?: 'asc' | 'desc';
}): Promise<AdminDataResponse<any>> {
  const response = await apiClient.get('/admin/stocks', { params });
  return response.data;
}

// Fetch users (admin)
export async function getAdminUsers(params?: {
  limit?: number;
  offset?: number;
  search?: string;
}): Promise<AdminDataResponse<any>> {
  const response = await apiClient.get('/admin/users', { params });
  return response.data;
}

// Fetch news articles (admin)
export async function getAdminNews(params?: {
  limit?: number;
  offset?: number;
  search?: string;
}): Promise<AdminDataResponse<any>> {
  const response = await apiClient.get('/admin/news', { params });
  return response.data;
}

// Fetch fundamentals (admin)
export async function getAdminFundamentals(params?: {
  limit?: number;
  offset?: number;
  search?: string;
}): Promise<AdminDataResponse<any>> {
  const response = await apiClient.get('/admin/fundamentals', { params });
  return response.data;
}

// Fetch alerts (admin)
export async function getAdminAlerts(params?: {
  limit?: number;
  offset?: number;
}): Promise<AdminDataResponse<any>> {
  const response = await apiClient.get('/admin/alerts', { params });
  return response.data;
}

// Fetch watch lists (admin)
export async function getAdminWatchLists(params?: {
  limit?: number;
  offset?: number;
}): Promise<AdminDataResponse<any>> {
  const response = await apiClient.get('/admin/watchlists', { params });
  return response.data;
}

// Fetch database statistics (admin)
export async function getAdminDatabaseStats(): Promise<{ stats: DatabaseStats }> {
  const response = await apiClient.get('/admin/stats');
  return response.data;
}
