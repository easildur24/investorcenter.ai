import { apiClient } from './client';
import { admin } from './routes';

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
  const queryParams = new URLSearchParams();
  if (params?.limit) queryParams.append('limit', params.limit.toString());
  if (params?.offset) queryParams.append('offset', params.offset.toString());
  if (params?.search) queryParams.append('search', params.search);
  if (params?.sort) queryParams.append('sort', params.sort);
  if (params?.order) queryParams.append('order', params.order);

  const query = queryParams.toString();
  return apiClient.get(`${admin.stocks}${query ? `?${query}` : ''}`);
}

// Fetch users (admin)
export async function getAdminUsers(params?: {
  limit?: number;
  offset?: number;
  search?: string;
}): Promise<AdminDataResponse<any>> {
  const queryParams = new URLSearchParams();
  if (params?.limit) queryParams.append('limit', params.limit.toString());
  if (params?.offset) queryParams.append('offset', params.offset.toString());
  if (params?.search) queryParams.append('search', params.search);

  const query = queryParams.toString();
  return apiClient.get(`${admin.users}${query ? `?${query}` : ''}`);
}

// Fetch news articles (admin)
export async function getAdminNews(params?: {
  limit?: number;
  offset?: number;
  search?: string;
}): Promise<AdminDataResponse<any>> {
  const queryParams = new URLSearchParams();
  if (params?.limit) queryParams.append('limit', params.limit.toString());
  if (params?.offset) queryParams.append('offset', params.offset.toString());
  if (params?.search) queryParams.append('search', params.search);

  const query = queryParams.toString();
  return apiClient.get(`${admin.news}${query ? `?${query}` : ''}`);
}

// Fetch fundamentals (admin)
export async function getAdminFundamentals(params?: {
  limit?: number;
  offset?: number;
  search?: string;
}): Promise<AdminDataResponse<any>> {
  const queryParams = new URLSearchParams();
  if (params?.limit) queryParams.append('limit', params.limit.toString());
  if (params?.offset) queryParams.append('offset', params.offset.toString());
  if (params?.search) queryParams.append('search', params.search);

  const query = queryParams.toString();
  return apiClient.get(`${admin.fundamentals}${query ? `?${query}` : ''}`);
}

// Fetch alerts (admin)
export async function getAdminAlerts(params?: {
  limit?: number;
  offset?: number;
}): Promise<AdminDataResponse<any>> {
  const queryParams = new URLSearchParams();
  if (params?.limit) queryParams.append('limit', params.limit.toString());
  if (params?.offset) queryParams.append('offset', params.offset.toString());

  const query = queryParams.toString();
  return apiClient.get(`${admin.alerts}${query ? `?${query}` : ''}`);
}

// Fetch watch lists (admin)
export async function getAdminWatchLists(params?: {
  limit?: number;
  offset?: number;
}): Promise<AdminDataResponse<any>> {
  const queryParams = new URLSearchParams();
  if (params?.limit) queryParams.append('limit', params.limit.toString());
  if (params?.offset) queryParams.append('offset', params.offset.toString());

  const query = queryParams.toString();
  return apiClient.get(`${admin.watchlists}${query ? `?${query}` : ''}`);
}

// Fetch database statistics (admin)
export async function getAdminDatabaseStats(): Promise<{ stats: DatabaseStats }> {
  return apiClient.get(admin.stats);
}

// Fetch SEC financials (admin)
export async function getAdminSECFinancials(params?: {
  limit?: number;
  offset?: number;
  search?: string;
}): Promise<AdminDataResponse<any>> {
  const queryParams = new URLSearchParams();
  if (params?.limit) queryParams.append('limit', params.limit.toString());
  if (params?.offset) queryParams.append('offset', params.offset.toString());
  if (params?.search) queryParams.append('search', params.search);

  const query = queryParams.toString();
  return apiClient.get(`${admin.secFinancials}${query ? `?${query}` : ''}`);
}

// Fetch TTM financials (admin)
export async function getAdminTTMFinancials(params?: {
  limit?: number;
  offset?: number;
  search?: string;
}): Promise<AdminDataResponse<any>> {
  const queryParams = new URLSearchParams();
  if (params?.limit) queryParams.append('limit', params.limit.toString());
  if (params?.offset) queryParams.append('offset', params.offset.toString());
  if (params?.search) queryParams.append('search', params.search);

  const query = queryParams.toString();
  return apiClient.get(`${admin.ttmFinancials}${query ? `?${query}` : ''}`);
}

// Fetch valuation ratios (admin)
export async function getAdminValuationRatios(params?: {
  limit?: number;
  offset?: number;
  search?: string;
}): Promise<AdminDataResponse<any>> {
  const queryParams = new URLSearchParams();
  if (params?.limit) queryParams.append('limit', params.limit.toString());
  if (params?.offset) queryParams.append('offset', params.offset.toString());
  if (params?.search) queryParams.append('search', params.search);

  const query = queryParams.toString();
  return apiClient.get(`${admin.valuationRatios}${query ? `?${query}` : ''}`);
}

// Fetch analyst ratings (admin)
export async function getAdminAnalystRatings(params?: {
  limit?: number;
  offset?: number;
  search?: string;
}): Promise<AdminDataResponse<any>> {
  const queryParams = new URLSearchParams();
  if (params?.limit) queryParams.append('limit', params.limit.toString());
  if (params?.offset) queryParams.append('offset', params.offset.toString());
  if (params?.search) queryParams.append('search', params.search);

  const query = queryParams.toString();
  return apiClient.get(`${admin.analystRatings}${query ? `?${query}` : ''}`);
}

// Fetch insider trades (admin)
export async function getAdminInsiderTrades(params?: {
  limit?: number;
  offset?: number;
  search?: string;
}): Promise<AdminDataResponse<any>> {
  const queryParams = new URLSearchParams();
  if (params?.limit) queryParams.append('limit', params.limit.toString());
  if (params?.offset) queryParams.append('offset', params.offset.toString());
  if (params?.search) queryParams.append('search', params.search);

  const query = queryParams.toString();
  return apiClient.get(`${admin.insiderTrades}${query ? `?${query}` : ''}`);
}

// Fetch institutional holdings (admin)
export async function getAdminInstitutionalHoldings(params?: {
  limit?: number;
  offset?: number;
  search?: string;
}): Promise<AdminDataResponse<any>> {
  const queryParams = new URLSearchParams();
  if (params?.limit) queryParams.append('limit', params.limit.toString());
  if (params?.offset) queryParams.append('offset', params.offset.toString());
  if (params?.search) queryParams.append('search', params.search);

  const query = queryParams.toString();
  return apiClient.get(`${admin.institutionalHoldings}${query ? `?${query}` : ''}`);
}

// Fetch technical indicators (admin)
export async function getAdminTechnicalIndicators(params?: {
  limit?: number;
  offset?: number;
  search?: string;
}): Promise<AdminDataResponse<any>> {
  const queryParams = new URLSearchParams();
  if (params?.limit) queryParams.append('limit', params.limit.toString());
  if (params?.offset) queryParams.append('offset', params.offset.toString());
  if (params?.search) queryParams.append('search', params.search);

  const query = queryParams.toString();
  return apiClient.get(`${admin.technicalIndicators}${query ? `?${query}` : ''}`);
}

// Fetch companies (admin)
export async function getAdminCompanies(params?: {
  limit?: number;
  offset?: number;
  search?: string;
}): Promise<AdminDataResponse<any>> {
  const queryParams = new URLSearchParams();
  if (params?.limit) queryParams.append('limit', params.limit.toString());
  if (params?.offset) queryParams.append('offset', params.offset.toString());
  if (params?.search) queryParams.append('search', params.search);

  const query = queryParams.toString();
  return apiClient.get(`${admin.companies}${query ? `?${query}` : ''}`);
}

// Fetch risk metrics (admin)
export async function getAdminRiskMetrics(params?: {
  limit?: number;
  offset?: number;
  search?: string;
}): Promise<AdminDataResponse<any>> {
  const queryParams = new URLSearchParams();
  if (params?.limit) queryParams.append('limit', params.limit.toString());
  if (params?.offset) queryParams.append('offset', params.offset.toString());
  if (params?.search) queryParams.append('search', params.search);

  const query = queryParams.toString();
  return apiClient.get(`${admin.riskMetrics}${query ? `?${query}` : ''}`);
}
