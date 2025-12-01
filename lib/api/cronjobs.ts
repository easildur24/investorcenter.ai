import { apiClient } from './client';

// Types
export interface CronjobSummary {
  total_jobs: number;
  active_jobs: number;
  last_24h: {
    total_executions: number;
    successful: number;
    failed: number;
    success_rate: number;
  };
}

export interface LastRunInfo {
  status: string;
  started_at: string;
  completed_at?: string;
  duration_seconds?: number;
  records_processed: number;
}

export interface CronjobStatusWithInfo {
  job_name: string;
  job_category: string;
  schedule: string;
  schedule_description: string;
  last_run?: LastRunInfo;
  health_status: string; // 'healthy', 'warning', 'critical', 'unknown'
  consecutive_failures: number;
  avg_duration_7d?: number;
  success_rate_7d?: number;
}

export interface CronjobOverviewResponse {
  summary: CronjobSummary;
  jobs: CronjobStatusWithInfo[];
}

export interface CronjobExecutionLog {
  id: number;
  job_name: string;
  job_category: string;
  execution_id: string;
  status: string;
  started_at: string;
  completed_at?: string;
  duration_seconds?: number;
  records_processed: number;
  records_updated: number;
  records_failed: number;
  error_message?: string;
  error_stack_trace?: string;
  k8s_pod_name?: string;
  k8s_namespace?: string;
  exit_code?: number;
  created_at: string;
}

export interface CronjobMetrics {
  avg_duration?: number;
  p50_duration?: number;
  p95_duration?: number;
  success_rate?: number;
}

export interface CronjobHistoryResponse {
  job_name: string;
  total_executions: number;
  executions: CronjobExecutionLog[];
  metrics: CronjobMetrics;
}

export interface CronjobDailySummary {
  date: string;
  total: number;
  successful: number;
  rate: number;
}

export interface CronjobPerformance {
  job_name: string;
  avg_duration: number;
  trend: string;
}

export interface CronjobMetricsResponse {
  daily_success_rate: CronjobDailySummary[];
  job_performance: CronjobPerformance[];
  failure_breakdown: Record<string, number>;
}

export interface CronjobSchedule {
  id: number;
  job_name: string;
  job_category: string;
  description: string;
  schedule_cron: string;
  schedule_description: string;
  is_active: boolean;
  expected_duration_seconds?: number;
  timeout_seconds?: number;
  last_success_at?: string;
  last_failure_at?: string;
  consecutive_failures: number;
  created_at: string;
  updated_at: string;
}

// API Functions
export async function getCronjobOverview(): Promise<CronjobOverviewResponse> {
  return apiClient.get<CronjobOverviewResponse>('/admin/cronjobs/overview');
}

export async function getCronjobHistory(
  jobName: string,
  params?: { limit?: number; offset?: number }
): Promise<CronjobHistoryResponse> {
  const queryParams = new URLSearchParams();
  if (params?.limit) queryParams.set('limit', params.limit.toString());
  if (params?.offset) queryParams.set('offset', params.offset.toString());

  const query = queryParams.toString();
  return apiClient.get<CronjobHistoryResponse>(
    `/admin/cronjobs/${jobName}/history${query ? `?${query}` : ''}`
  );
}

export async function getCronjobDetails(executionId: string): Promise<CronjobExecutionLog> {
  return apiClient.get<CronjobExecutionLog>(`/admin/cronjobs/details/${executionId}`);
}

export async function getCronjobMetrics(period: number = 7): Promise<CronjobMetricsResponse> {
  return apiClient.get<CronjobMetricsResponse>(`/admin/cronjobs/metrics?period=${period}`);
}

export async function getCronjobSchedules(): Promise<CronjobSchedule[]> {
  return apiClient.get<CronjobSchedule[]>('/admin/cronjobs/schedules');
}

// Helper functions
export function getStatusColor(status: string): string {
  switch (status) {
    case 'success':
      return 'text-green-600 bg-green-50';
    case 'failed':
    case 'timeout':
      return 'text-red-600 bg-red-50';
    case 'running':
      return 'text-yellow-600 bg-yellow-50';
    default:
      return 'text-ic-text-muted bg-ic-surface';
  }
}

export function getHealthStatusColor(healthStatus: string): string {
  switch (healthStatus) {
    case 'healthy':
      return 'text-green-600';
    case 'warning':
      return 'text-yellow-600';
    case 'critical':
      return 'text-red-600';
    default:
      return 'text-ic-text-muted';
  }
}

export function getHealthStatusIcon(healthStatus: string): string {
  switch (healthStatus) {
    case 'healthy':
      return '🟢';
    case 'warning':
      return '🟡';
    case 'critical':
      return '🔴';
    default:
      return '⚪';
  }
}

export function formatDuration(seconds?: number): string {
  if (!seconds) return '--';

  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const secs = seconds % 60;

  if (hours > 0) {
    return `${hours}h ${minutes}m ${secs}s`;
  } else if (minutes > 0) {
    return `${minutes}m ${secs}s`;
  } else {
    return `${secs}s`;
  }
}

export function formatTimeAgo(dateString: string): string {
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMins / 60);
  const diffDays = Math.floor(diffHours / 24);

  if (diffMins < 1) return 'just now';
  if (diffMins < 60) return `${diffMins} min${diffMins > 1 ? 's' : ''} ago`;
  if (diffHours < 24) return `${diffHours} hour${diffHours > 1 ? 's' : ''} ago`;
  if (diffDays < 7) return `${diffDays} day${diffDays > 1 ? 's' : ''} ago`;

  return date.toLocaleDateString();
}
