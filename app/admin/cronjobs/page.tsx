'use client';

import { useState, useEffect } from 'react';
import {
  getCronjobOverview,
  getCronjobHistory,
  getCronjobMetrics,
  CronjobOverviewResponse,
  CronjobStatusWithInfo,
  CronjobHistoryResponse,
  CronjobMetricsResponse,
  getStatusColor,
  getHealthStatusColor,
  getHealthStatusIcon,
  formatDuration,
  formatTimeAgo,
} from '@/lib/api/cronjobs';
import { Clock, CheckCircle, XCircle, AlertCircle, TrendingUp, Activity, X } from 'lucide-react';

export default function CronjobsMonitoringPage() {
  const [overview, setOverview] = useState<CronjobOverviewResponse | null>(null);
  const [metrics, setMetrics] = useState<CronjobMetricsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<'all' | 'core_pipeline' | 'ic_score_pipeline' | 'failed'>('all');

  // Modal state for job details
  const [selectedJob, setSelectedJob] = useState<string | null>(null);
  const [jobHistory, setJobHistory] = useState<CronjobHistoryResponse | null>(null);
  const [loadingHistory, setLoadingHistory] = useState(false);

  useEffect(() => {
    fetchData();
  }, []);

  async function fetchData() {
    try {
      setLoading(true);
      const [overviewData, metricsData] = await Promise.all([
        getCronjobOverview(),
        getCronjobMetrics(7),
      ]);
      setOverview(overviewData);
      setMetrics(metricsData);
    } catch (error) {
      console.error('Error fetching cronjob data:', error);
    } finally {
      setLoading(false);
    }
  }

  async function handleViewJobDetails(jobName: string) {
    setSelectedJob(jobName);
    setLoadingHistory(true);
    try {
      const history = await getCronjobHistory(jobName, { limit: 10, offset: 0 });
      setJobHistory(history);
    } catch (error) {
      console.error('Error fetching job history:', error);
    } finally {
      setLoadingHistory(false);
    }
  }

  function closeModal() {
    setSelectedJob(null);
    setJobHistory(null);
  }

  const filteredJobs = overview?.jobs.filter((job) => {
    if (filter === 'all') return true;
    if (filter === 'failed') return job.health_status === 'critical';
    return job.job_category === filter;
  }) || [];

  if (loading || !overview) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
          <p className="mt-4 text-gray-600">Loading cronjob data...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-white shadow-sm border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          <h1 className="text-3xl font-bold text-gray-900">Cronjob Monitoring Dashboard</h1>
          <p className="text-gray-600 mt-1">Monitor and track all scheduled cronjobs</p>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
          {/* Total Jobs */}
          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-gray-600">Total Cronjobs</p>
                <p className="text-3xl font-bold text-gray-900 mt-2">{overview.summary.total_jobs}</p>
                <p className="text-sm text-gray-500 mt-1">All configured</p>
              </div>
              <Activity className="h-12 w-12 text-blue-500" />
            </div>
          </div>

          {/* Active Jobs */}
          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-gray-600">Active Jobs</p>
                <p className="text-3xl font-bold text-green-600 mt-2">{overview.summary.active_jobs}</p>
                <p className="text-sm text-gray-500 mt-1">Currently enabled</p>
              </div>
              <CheckCircle className="h-12 w-12 text-green-500" />
            </div>
          </div>

          {/* Last 24h Success Rate */}
          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-gray-600">Last 24h Success</p>
                <p className="text-3xl font-bold text-blue-600 mt-2">
                  {overview.summary.last_24h.success_rate.toFixed(1)}%
                </p>
                <p className="text-sm text-gray-500 mt-1">
                  {overview.summary.last_24h.successful}/{overview.summary.last_24h.total_executions} succeeded
                </p>
              </div>
              <TrendingUp className="h-12 w-12 text-blue-500" />
            </div>
          </div>

          {/* Critical Alerts */}
          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-gray-600">Critical Alerts</p>
                <p className="text-3xl font-bold text-red-600 mt-2">
                  {overview.jobs.filter((j) => j.health_status === 'critical').length}
                </p>
                <p className="text-sm text-gray-500 mt-1">Needs attention</p>
              </div>
              <AlertCircle className="h-12 w-12 text-red-500" />
            </div>
          </div>
        </div>

        {/* Filters */}
        <div className="bg-white rounded-lg shadow p-4 mt-6">
          <div className="flex items-center space-x-4">
            <span className="text-sm font-medium text-gray-700">Filter:</span>
            <button
              onClick={() => setFilter('all')}
              className={`px-4 py-2 rounded-md text-sm font-medium ${
                filter === 'all'
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
              }`}
            >
              All Jobs
            </button>
            <button
              onClick={() => setFilter('core_pipeline')}
              className={`px-4 py-2 rounded-md text-sm font-medium ${
                filter === 'core_pipeline'
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
              }`}
            >
              Core Pipeline
            </button>
            <button
              onClick={() => setFilter('ic_score_pipeline')}
              className={`px-4 py-2 rounded-md text-sm font-medium ${
                filter === 'ic_score_pipeline'
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
              }`}
            >
              IC Score Pipeline
            </button>
            <button
              onClick={() => setFilter('failed')}
              className={`px-4 py-2 rounded-md text-sm font-medium ${
                filter === 'failed'
                  ? 'bg-red-600 text-white'
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
              }`}
            >
              Failed Only
            </button>
          </div>
        </div>

        {/* Job Status Table */}
        <div className="bg-white rounded-lg shadow mt-6 overflow-hidden">
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Job Name
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Status
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Last Run
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Duration
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Success Rate (7d)
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {filteredJobs.map((job) => (
                  <tr key={job.job_name} className="hover:bg-gray-50">
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="flex items-center">
                        <span className="mr-2">{getHealthStatusIcon(job.health_status)}</span>
                        <div>
                          <div className="text-sm font-medium text-gray-900">{job.job_name}</div>
                          <div className="text-xs text-gray-500">{job.schedule_description}</div>
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      {job.last_run ? (
                        <span
                          className={`px-2 py-1 inline-flex text-xs leading-5 font-semibold rounded-full ${getStatusColor(
                            job.last_run.status
                          )}`}
                        >
                          {job.last_run.status}
                        </span>
                      ) : (
                        <span className="text-sm text-gray-500">No runs</span>
                      )}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {job.last_run ? formatTimeAgo(job.last_run.started_at) : '--'}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {job.last_run?.duration_seconds ? formatDuration(job.last_run.duration_seconds) : '--'}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      {job.success_rate_7d !== null && job.success_rate_7d !== undefined ? (
                        <div className="flex items-center">
                          <div className="w-16 bg-gray-200 rounded-full h-2 mr-2">
                            <div
                              className={`h-2 rounded-full ${
                                job.success_rate_7d >= 95
                                  ? 'bg-green-500'
                                  : job.success_rate_7d >= 80
                                  ? 'bg-yellow-500'
                                  : 'bg-red-500'
                              }`}
                              style={{ width: `${job.success_rate_7d}%` }}
                            ></div>
                          </div>
                          <span className="text-sm text-gray-700">{job.success_rate_7d.toFixed(1)}%</span>
                        </div>
                      ) : (
                        <span className="text-sm text-gray-500">--</span>
                      )}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      <button
                        onClick={() => handleViewJobDetails(job.job_name)}
                        className="text-blue-600 hover:text-blue-800 font-medium"
                      >
                        View Details
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        {/* Daily Success Rate Chart */}
        {metrics && metrics.daily_success_rate.length > 0 && (
          <div className="bg-white rounded-lg shadow p-6 mt-6">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Daily Success Rate (Last 7 Days)</h3>
            <div className="h-64 flex items-end justify-between space-x-2">
              {metrics.daily_success_rate.slice(0, 7).reverse().map((day) => (
                <div key={day.date} className="flex-1 flex flex-col items-center">
                  <div className="w-full bg-gray-200 rounded-t" style={{ height: '200px', position: 'relative' }}>
                    <div
                      className={`absolute bottom-0 w-full rounded-t ${
                        day.rate >= 95 ? 'bg-green-500' : day.rate >= 80 ? 'bg-yellow-500' : 'bg-red-500'
                      }`}
                      style={{ height: `${day.rate}%` }}
                    ></div>
                  </div>
                  <div className="text-center mt-2">
                    <p className="text-xs font-medium text-gray-900">{day.rate.toFixed(0)}%</p>
                    <p className="text-xs text-gray-500">{new Date(day.date).toLocaleDateString('en-US', { month: 'short', day: 'numeric' })}</p>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* Job Details Modal */}
      {selectedJob && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-lg shadow-xl max-w-4xl w-full max-h-[90vh] overflow-hidden">
            <div className="flex items-center justify-between p-6 border-b border-gray-200">
              <h2 className="text-2xl font-bold text-gray-900">{selectedJob}</h2>
              <button onClick={closeModal} className="text-gray-400 hover:text-gray-600">
                <X className="h-6 w-6" />
              </button>
            </div>

            <div className="p-6 overflow-y-auto max-h-[calc(90vh-80px)]">
              {loadingHistory ? (
                <div className="flex items-center justify-center py-12">
                  <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
                </div>
              ) : jobHistory ? (
                <>
                  {/* Job Statistics */}
                  <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
                    <div className="bg-gray-50 rounded-lg p-4">
                      <p className="text-sm font-medium text-gray-600">Success Rate</p>
                      <p className="text-2xl font-bold text-gray-900 mt-1">
                        {jobHistory.metrics.success_rate?.toFixed(1) || '--'}%
                      </p>
                    </div>
                    <div className="bg-gray-50 rounded-lg p-4">
                      <p className="text-sm font-medium text-gray-600">Avg Duration</p>
                      <p className="text-2xl font-bold text-gray-900 mt-1">
                        {formatDuration(jobHistory.metrics.avg_duration)}
                      </p>
                    </div>
                    <div className="bg-gray-50 rounded-lg p-4">
                      <p className="text-sm font-medium text-gray-600">P50 Duration</p>
                      <p className="text-2xl font-bold text-gray-900 mt-1">
                        {formatDuration(jobHistory.metrics.p50_duration)}
                      </p>
                    </div>
                    <div className="bg-gray-50 rounded-lg p-4">
                      <p className="text-sm font-medium text-gray-600">P95 Duration</p>
                      <p className="text-2xl font-bold text-gray-900 mt-1">
                        {formatDuration(jobHistory.metrics.p95_duration)}
                      </p>
                    </div>
                  </div>

                  {/* Recent Executions */}
                  <h3 className="text-lg font-semibold text-gray-900 mb-4">Recent Executions (Last 10)</h3>
                  <div className="space-y-3">
                    {jobHistory.executions.map((execution) => (
                      <div key={execution.id} className="border border-gray-200 rounded-lg p-4">
                        <div className="flex items-center justify-between mb-2">
                          <div className="flex items-center space-x-3">
                            <span
                              className={`px-2 py-1 text-xs font-semibold rounded-full ${getStatusColor(
                                execution.status
                              )}`}
                            >
                              {execution.status}
                            </span>
                            <span className="text-sm text-gray-500">
                              {new Date(execution.started_at).toLocaleString()}
                            </span>
                          </div>
                          <span className="text-sm font-medium text-gray-700">
                            {formatDuration(execution.duration_seconds)}
                          </span>
                        </div>
                        <div className="grid grid-cols-3 gap-4 text-sm">
                          <div>
                            <span className="text-gray-600">Processed:</span>{' '}
                            <span className="font-medium text-gray-900">{execution.records_processed}</span>
                          </div>
                          <div>
                            <span className="text-gray-600">Updated:</span>{' '}
                            <span className="font-medium text-gray-900">{execution.records_updated}</span>
                          </div>
                          <div>
                            <span className="text-gray-600">Failed:</span>{' '}
                            <span className="font-medium text-gray-900">{execution.records_failed}</span>
                          </div>
                        </div>
                        {execution.error_message && (
                          <div className="mt-2 p-2 bg-red-50 rounded text-sm text-red-700">
                            <span className="font-medium">Error:</span> {execution.error_message}
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                </>
              ) : (
                <p className="text-center text-gray-500 py-12">No execution history available</p>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
