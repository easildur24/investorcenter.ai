'use client';

import { useState, useEffect } from 'react';
import {
  getAdminStocks,
  getAdminUsers,
  getAdminNews,
  getAdminFundamentals,
  getAdminAlerts,
  getAdminWatchLists,
  getAdminDatabaseStats,
  AdminDataResponse
} from '@/lib/api/admin';
import {
  Search,
  ChevronLeft,
  ChevronRight,
  Database,
  Users,
  FileText,
  TrendingUp,
  Bell,
  BookmarkIcon,
  BarChart3,
  Activity
} from 'lucide-react';

type TabType = 'stats' | 'stocks' | 'users' | 'news' | 'fundamentals' | 'alerts' | 'watchlists';

export default function AdminDashboardPage() {
  const [activeTab, setActiveTab] = useState<TabType>('stats');
  const [data, setData] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [meta, setMeta] = useState({
    total: 0,
    limit: 50,
    offset: 0,
  });
  const [search, setSearch] = useState('');
  const [stats, setStats] = useState<any>(null);

  const currentPage = Math.floor(meta.offset / meta.limit) + 1;
  const totalPages = Math.ceil(meta.total / meta.limit);

  const tabs = [
    { id: 'stats' as TabType, name: 'Database Stats', icon: Database },
    { id: 'stocks' as TabType, name: 'Stocks', icon: TrendingUp },
    { id: 'users' as TabType, name: 'Users', icon: Users },
    { id: 'news' as TabType, name: 'News', icon: FileText },
    { id: 'fundamentals' as TabType, name: 'Fundamentals', icon: BarChart3 },
    { id: 'alerts' as TabType, name: 'Alerts', icon: Bell },
    { id: 'watchlists' as TabType, name: 'Watch Lists', icon: BookmarkIcon },
  ];

  useEffect(() => {
    fetchData();
  }, [activeTab, meta.offset, meta.limit]);

  async function fetchData() {
    setLoading(true);
    try {
      const params = {
        limit: meta.limit,
        offset: meta.offset,
        search: search || undefined,
      };

      let result: AdminDataResponse<any> | any;

      switch (activeTab) {
        case 'stats':
          result = await getAdminDatabaseStats();
          setStats(result.stats);
          setData([]);
          break;
        case 'stocks':
          result = await getAdminStocks(params);
          setData(result.data || []);
          setMeta(result.meta);
          break;
        case 'users':
          result = await getAdminUsers(params);
          setData(result.data || []);
          setMeta(result.meta);
          break;
        case 'news':
          result = await getAdminNews(params);
          setData(result.data || []);
          setMeta(result.meta);
          break;
        case 'fundamentals':
          result = await getAdminFundamentals(params);
          setData(result.data || []);
          setMeta(result.meta);
          break;
        case 'alerts':
          result = await getAdminAlerts(params);
          setData(result.data || []);
          setMeta(result.meta);
          break;
        case 'watchlists':
          result = await getAdminWatchLists(params);
          setData(result.data || []);
          setMeta(result.meta);
          break;
      }
    } catch (error) {
      console.error('Error fetching admin data:', error);
    } finally {
      setLoading(false);
    }
  }

  function handleSearchSubmit(e: React.FormEvent) {
    e.preventDefault();
    setMeta({ ...meta, offset: 0 });
    fetchData();
  }

  function handlePageChange(newPage: number) {
    setMeta({
      ...meta,
      offset: (newPage - 1) * meta.limit,
    });
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-white shadow-sm border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          <div className="flex items-center gap-3">
            <Activity className="w-8 h-8 text-blue-600" />
            <div>
              <h1 className="text-3xl font-bold text-gray-900">Admin Dashboard</h1>
              <p className="text-gray-600 mt-1">Query and manage all data in the system</p>
            </div>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="bg-white border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex gap-1 overflow-x-auto">
            {tabs.map((tab) => {
              const Icon = tab.icon;
              return (
                <button
                  key={tab.id}
                  onClick={() => {
                    setActiveTab(tab.id);
                    setSearch('');
                    setMeta({ ...meta, offset: 0 });
                  }}
                  className={`flex items-center gap-2 px-4 py-3 border-b-2 font-medium text-sm whitespace-nowrap transition-colors ${
                    activeTab === tab.id
                      ? 'border-blue-600 text-blue-600'
                      : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                  }`}
                >
                  <Icon className="w-4 h-4" />
                  {tab.name}
                </button>
              );
            })}
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {activeTab !== 'stats' && (
          <div className="bg-white rounded-lg shadow border border-gray-200 p-6 mb-6">
            <form onSubmit={handleSearchSubmit} className="flex gap-4">
              <div className="flex-1 relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-5 h-5" />
                <input
                  type="text"
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  placeholder={`Search ${activeTab}...`}
                  className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 text-gray-900"
                />
              </div>
              <button
                type="submit"
                className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors font-medium"
              >
                Search
              </button>
              {search && (
                <button
                  type="button"
                  onClick={() => {
                    setSearch('');
                    setMeta({ ...meta, offset: 0 });
                  }}
                  className="px-4 py-2 text-gray-600 hover:text-gray-900 transition-colors"
                >
                  Clear
                </button>
              )}
            </form>
          </div>
        )}

        {/* Content */}
        <div className="bg-white rounded-lg shadow border border-gray-200 overflow-hidden">
          {loading ? (
            <div className="p-12 text-center">
              <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
              <p className="text-gray-600 mt-4">Loading data...</p>
            </div>
          ) : activeTab === 'stats' ? (
            <StatsView stats={stats} />
          ) : (
            <DataTable
              data={data}
              type={activeTab}
              meta={meta}
              currentPage={currentPage}
              totalPages={totalPages}
              onPageChange={handlePageChange}
            />
          )}
        </div>
      </div>
    </div>
  );
}

interface StatsViewProps {
  stats: any;
}

function StatsView({ stats }: StatsViewProps) {
  if (!stats) {
    return (
      <div className="p-12 text-center text-gray-600">
        No statistics available
      </div>
    );
  }

  const statItems = Object.entries(stats).map(([key, value]) => ({
    name: key.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase()),
    value: (value as number).toLocaleString(),
    key,
  }));

  return (
    <div className="p-6">
      <h2 className="text-xl font-bold text-gray-900 mb-6">Database Statistics</h2>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {statItems.map((stat) => (
          <div key={stat.key} className="bg-gradient-to-br from-blue-50 to-blue-100 border border-blue-200 rounded-lg p-4">
            <div className="text-sm text-blue-600 font-medium">{stat.name}</div>
            <div className="text-3xl font-bold text-blue-900 mt-2">{stat.value}</div>
          </div>
        ))}
      </div>
    </div>
  );
}

interface DataTableProps {
  data: any[];
  type: TabType;
  meta: any;
  currentPage: number;
  totalPages: number;
  onPageChange: (page: number) => void;
}

function DataTable({ data, type, meta, currentPage, totalPages, onPageChange }: DataTableProps) {
  if (data.length === 0) {
    return (
      <div className="p-12 text-center text-gray-600">
        No data found
      </div>
    );
  }

  return (
    <>
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead className="bg-gray-50 border-b border-gray-200">
            <tr>
              {Object.keys(data[0]).map((key) => (
                <th key={key} className="px-6 py-3 text-left text-xs font-semibold text-gray-700 uppercase tracking-wider">
                  {key.replace(/_/g, ' ')}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {data.map((row, idx) => (
              <tr key={idx} className="hover:bg-gray-50">
                {Object.entries(row).map(([key, value]) => (
                  <td key={key} className="px-6 py-4 text-sm text-gray-900 align-top">
                    {formatValue(value, key)}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      <div className="px-6 py-4 bg-gray-50 border-t border-gray-200">
        <div className="flex items-center justify-between">
          <div className="text-sm text-gray-600">
            Showing {meta.offset + 1} to {Math.min(meta.offset + meta.limit, meta.total)} of {meta.total} results
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={() => onPageChange(currentPage - 1)}
              disabled={currentPage === 1}
              className="p-2 rounded hover:bg-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <ChevronLeft className="w-5 h-5" />
            </button>
            <span className="text-sm text-gray-700">
              Page {currentPage} of {totalPages}
            </span>
            <button
              onClick={() => onPageChange(currentPage + 1)}
              disabled={currentPage === totalPages}
              className="p-2 rounded hover:bg-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <ChevronRight className="w-5 h-5" />
            </button>
          </div>
        </div>
      </div>
    </>
  );
}

function ExpandableSummary({ text }: { text: string }) {
  const [isExpanded, setIsExpanded] = useState(false);
  const shortText = text.length > 100 ? text.substring(0, 100) + '...' : text;

  if (text.length <= 100) {
    return <span>{text}</span>;
  }

  return (
    <div className="max-w-md">
      <p className="text-sm">
        {isExpanded ? text : shortText}
      </p>
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className="mt-1 text-xs text-blue-600 hover:text-blue-800 font-medium"
      >
        {isExpanded ? 'Show less' : 'Show more'}
      </button>
    </div>
  );
}

function formatValue(value: any, key?: string): React.ReactNode {
  if (value === null || value === undefined) {
    return '-';
  }
  if (typeof value === 'boolean') {
    return value ? 'Yes' : 'No';
  }
  if (typeof value === 'object' && value instanceof Date) {
    return new Date(value).toLocaleString();
  }
  if (Array.isArray(value)) {
    // Handle arrays (like tickers)
    if (value.length === 0) return '-';
    if (value.length <= 3) {
      return (
        <div className="flex flex-wrap gap-1">
          {value.map((item, idx) => (
            <span key={idx} className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-800">
              {item}
            </span>
          ))}
        </div>
      );
    }
    return (
      <div className="flex flex-wrap gap-1">
        {value.slice(0, 2).map((item, idx) => (
          <span key={idx} className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-800">
            {item}
          </span>
        ))}
        <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-800">
          +{value.length - 2} more
        </span>
      </div>
    );
  }
  if (typeof value === 'object') {
    return JSON.stringify(value);
  }
  if (typeof value === 'number') {
    return value.toLocaleString();
  }
  // Truncate long text for certain columns
  const str = String(value);
  if (key === 'summary' && str.length > 0) {
    return <ExpandableSummary text={str} />;
  }
  if (key === 'url') {
    return (
      <a href={str} target="_blank" rel="noopener noreferrer" className="text-blue-600 hover:text-blue-800 underline truncate block max-w-xs">
        {str.length > 40 ? str.substring(0, 40) + '...' : str}
      </a>
    );
  }
  return str;
}
