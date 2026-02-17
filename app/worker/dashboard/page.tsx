'use client';

import { useState, useEffect, useCallback, useRef } from 'react';
import { useAuth } from '@/lib/auth/AuthContext';
import { useRouter } from 'next/navigation';
import {
  getMyTasks,
  getMyTaskUpdates,
  updateMyTaskStatus,
  postTaskUpdate,
  sendHeartbeat,
} from '@/lib/api/worker-api';
import type { WorkerTask, TaskUpdate, TaskStatus } from '@/lib/api/workers';
import {
  STATUS_LABELS,
  PRIORITY_LABELS,
  STATUS_COLORS,
  PRIORITY_COLORS,
  TASK_STATUSES,
} from '@/lib/api/workers';
import {
  Bot,
  Clock,
  MessageSquare,
  Send,
  ArrowLeft,
  Loader2,
  ListTodo,
  RefreshCw,
} from 'lucide-react';

export default function WorkerDashboardPage() {
  const { user, loading: authLoading } = useAuth();
  const router = useRouter();

  const [tasks, setTasks] = useState<WorkerTask[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Selected task
  const [selectedTask, setSelectedTask] = useState<WorkerTask | null>(null);
  const [taskUpdates, setTaskUpdates] = useState<TaskUpdate[]>([]);
  const [loadingUpdates, setLoadingUpdates] = useState(false);

  // Post update
  const [newUpdateContent, setNewUpdateContent] = useState('');
  const [sendingUpdate, setSendingUpdate] = useState(false);

  // Filter
  const [statusFilter, setStatusFilter] = useState<TaskStatus | ''>('');

  // Heartbeat ref
  const heartbeatRef = useRef<NodeJS.Timeout | null>(null);

  // Auth guard — redirect non-workers
  useEffect(() => {
    if (!authLoading && (!user || !user.is_worker)) {
      router.push('/');
    }
  }, [user, authLoading, router]);

  // Heartbeat every 2 minutes
  useEffect(() => {
    if (!user?.is_worker) return;

    sendHeartbeat().catch(() => {});
    heartbeatRef.current = setInterval(
      () => {
        sendHeartbeat().catch(() => {});
      },
      2 * 60 * 1000
    );

    return () => {
      if (heartbeatRef.current) clearInterval(heartbeatRef.current);
    };
  }, [user]);

  // Fetch tasks
  const fetchTasks = useCallback(async () => {
    try {
      setError(null);
      const data = await getMyTasks(statusFilter || undefined);
      setTasks(data || []);
    } catch (err: any) {
      setError(err.message);
    }
  }, [statusFilter]);

  useEffect(() => {
    if (!user?.is_worker) return;
    setLoading(true);
    fetchTasks().finally(() => setLoading(false));
  }, [fetchTasks, user]);

  // Fetch updates for selected task
  const fetchUpdates = async (taskId: string) => {
    setLoadingUpdates(true);
    try {
      const data = await getMyTaskUpdates(taskId);
      setTaskUpdates(data || []);
    } catch {
      setTaskUpdates([]);
    } finally {
      setLoadingUpdates(false);
    }
  };

  // Select a task
  const handleSelectTask = (task: WorkerTask) => {
    setSelectedTask(task);
    setNewUpdateContent('');
    fetchUpdates(task.id);
  };

  // Update status
  const handleStatusChange = async (taskId: string, status: TaskStatus) => {
    try {
      const updated = await updateMyTaskStatus(taskId, status);
      setSelectedTask(updated);
      // Refresh the task list
      fetchTasks();
    } catch (err: any) {
      alert('Failed to update status: ' + err.message);
    }
  };

  // Post update
  const handlePostUpdate = async () => {
    if (!newUpdateContent.trim() || !selectedTask) return;
    setSendingUpdate(true);
    try {
      await postTaskUpdate(selectedTask.id, newUpdateContent.trim());
      setNewUpdateContent('');
      fetchUpdates(selectedTask.id);
    } catch (err: any) {
      alert('Failed to post update: ' + err.message);
    } finally {
      setSendingUpdate(false);
    }
  };

  if (authLoading || !user?.is_worker) {
    return (
      <div className="min-h-screen bg-ic-bg-primary flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-purple-500"></div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-ic-bg-primary">
      {/* Header */}
      <div className="bg-ic-surface border-b border-ic-border">
        <div className="max-w-5xl mx-auto px-6 py-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Bot className="w-6 h-6 text-purple-500" />
            <div>
              <h1 className="text-lg font-bold text-ic-text-primary">Worker Dashboard</h1>
              <p className="text-xs text-ic-text-secondary">
                {user.full_name} &middot; {user.email}
              </p>
            </div>
          </div>
          <button
            onClick={() => {
              setLoading(true);
              fetchTasks().finally(() => setLoading(false));
            }}
            className="flex items-center gap-1.5 px-3 py-1.5 text-sm text-ic-text-secondary hover:text-ic-text-primary border border-ic-border rounded-lg hover:bg-ic-bg-secondary transition"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
            Refresh
          </button>
        </div>
      </div>

      <div className="max-w-5xl mx-auto px-6 py-6">
        {error && (
          <div className="mb-4 px-4 py-3 bg-red-50 border border-red-200 text-red-700 rounded-lg text-sm">
            {error}
          </div>
        )}

        {!selectedTask ? (
          /* ===== TASK LIST VIEW ===== */
          <>
            {/* Filter bar */}
            <div className="flex items-center justify-between mb-6">
              <div className="flex items-center gap-2">
                <ListTodo className="w-5 h-5 text-ic-text-secondary" />
                <h2 className="text-sm font-medium text-ic-text-secondary">
                  My Tasks ({tasks.length})
                </h2>
              </div>
              <select
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value as TaskStatus | '')}
                className="text-sm px-3 py-1.5 border border-ic-border rounded-lg bg-ic-bg-primary text-ic-text-primary"
              >
                <option value="">All statuses</option>
                {TASK_STATUSES.map((s) => (
                  <option key={s} value={s}>
                    {STATUS_LABELS[s]}
                  </option>
                ))}
              </select>
            </div>

            {loading ? (
              <div className="flex items-center justify-center py-16">
                <Loader2 className="w-6 h-6 animate-spin text-purple-500" />
              </div>
            ) : tasks.length === 0 ? (
              <div className="text-center py-16">
                <ListTodo className="w-10 h-10 mx-auto mb-3 text-ic-text-secondary opacity-30" />
                <p className="text-ic-text-secondary">
                  No tasks assigned to you
                  {statusFilter ? ` with status "${STATUS_LABELS[statusFilter]}"` : ''}.
                </p>
              </div>
            ) : (
              <div className="space-y-2">
                {tasks.map((task) => (
                  <div
                    key={task.id}
                    onClick={() => handleSelectTask(task)}
                    className="bg-ic-surface border border-ic-border rounded-lg px-5 py-4 cursor-pointer hover:border-purple-300 transition group"
                  >
                    <div className="flex items-start justify-between">
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-2">
                          <span
                            className={`text-xs px-2 py-0.5 rounded font-medium ${STATUS_COLORS[task.status]}`}
                          >
                            {STATUS_LABELS[task.status]}
                          </span>
                          <span
                            className={`text-xs px-2 py-0.5 rounded ${PRIORITY_COLORS[task.priority]}`}
                          >
                            {PRIORITY_LABELS[task.priority]}
                          </span>
                        </div>
                        <p className="text-sm font-semibold text-ic-text-primary group-hover:text-purple-600 transition">
                          {task.title}
                        </p>
                        {task.description && (
                          <p className="text-xs text-ic-text-secondary mt-1 line-clamp-2">
                            {task.description}
                          </p>
                        )}
                      </div>
                      <span className="text-xs text-ic-text-secondary ml-4 flex-shrink-0 flex items-center gap-1">
                        <Clock className="w-3 h-3" />
                        {new Date(task.updated_at).toLocaleDateString()}
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </>
        ) : (
          /* ===== TASK DETAIL VIEW ===== */
          <>
            <button
              onClick={() => {
                setSelectedTask(null);
                setTaskUpdates([]);
              }}
              className="flex items-center gap-1 text-sm text-ic-text-secondary hover:text-ic-text-primary mb-4 transition"
            >
              <ArrowLeft className="w-4 h-4" />
              Back to tasks
            </button>

            {/* Task header */}
            <div className="bg-ic-surface border border-ic-border rounded-lg p-6 mb-6">
              <h2 className="text-xl font-semibold text-ic-text-primary mb-2">
                {selectedTask.title}
              </h2>
              {selectedTask.description && (
                <p className="text-sm text-ic-text-secondary whitespace-pre-wrap mb-4">
                  {selectedTask.description}
                </p>
              )}

              <div className="flex flex-wrap items-center gap-3">
                <div className="flex items-center gap-2">
                  <span className="text-xs text-ic-text-secondary">Status:</span>
                  <select
                    value={selectedTask.status}
                    onChange={(e) =>
                      handleStatusChange(selectedTask.id, e.target.value as TaskStatus)
                    }
                    className={`text-xs px-2 py-1 rounded font-medium border-0 cursor-pointer ${STATUS_COLORS[selectedTask.status]}`}
                  >
                    <option value="in_progress">In Progress</option>
                    <option value="completed">Completed</option>
                    <option value="failed">Failed</option>
                  </select>
                </div>
                <span
                  className={`text-xs px-2 py-1 rounded ${PRIORITY_COLORS[selectedTask.priority]}`}
                >
                  {PRIORITY_LABELS[selectedTask.priority]}
                </span>
                <span className="text-xs text-ic-text-secondary flex items-center gap-1">
                  <Clock className="w-3 h-3" />
                  Created {new Date(selectedTask.created_at).toLocaleString()}
                </span>
              </div>
            </div>

            {/* Updates */}
            <div className="bg-ic-surface border border-ic-border rounded-lg p-6">
              <h3 className="text-sm font-medium text-ic-text-secondary mb-4 flex items-center gap-2">
                <MessageSquare className="w-4 h-4" />
                Updates ({taskUpdates.length})
              </h3>

              {loadingUpdates ? (
                <div className="flex items-center justify-center py-8">
                  <Loader2 className="w-5 h-5 animate-spin text-purple-500" />
                </div>
              ) : (
                <>
                  {taskUpdates.length > 0 && (
                    <div className="space-y-3 mb-6">
                      {taskUpdates.map((update) => (
                        <div key={update.id} className="bg-ic-bg-secondary rounded-lg p-3">
                          <div className="flex items-center justify-between mb-1">
                            <span className="text-xs font-medium text-ic-text-primary">
                              {update.created_by_name || 'System'}
                            </span>
                            <span className="text-xs text-ic-text-secondary">
                              {new Date(update.created_at).toLocaleString()}
                            </span>
                          </div>
                          <p className="text-sm text-ic-text-primary whitespace-pre-wrap">
                            {update.content}
                          </p>
                        </div>
                      ))}
                    </div>
                  )}

                  {taskUpdates.length === 0 && (
                    <p className="text-sm text-ic-text-secondary text-center py-4 mb-4">
                      No updates yet. Post the first one below.
                    </p>
                  )}

                  {/* Post update input */}
                  <div className="flex gap-2">
                    <textarea
                      value={newUpdateContent}
                      onChange={(e) => setNewUpdateContent(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' && !e.shiftKey) {
                          e.preventDefault();
                          handlePostUpdate();
                        }
                      }}
                      placeholder="Write an update... (Enter to send, Shift+Enter for new line)"
                      rows={2}
                      className="flex-1 px-3 py-2 border border-ic-border rounded-lg bg-ic-bg-primary text-ic-text-primary text-sm focus:ring-2 focus:ring-purple-500 focus:border-transparent resize-y"
                    />
                    <button
                      onClick={handlePostUpdate}
                      disabled={sendingUpdate || !newUpdateContent.trim()}
                      className="flex items-center gap-1 px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700 transition disabled:opacity-50 self-end"
                    >
                      {sendingUpdate ? (
                        <Loader2 className="w-4 h-4 animate-spin" />
                      ) : (
                        <Send className="w-4 h-4" />
                      )}
                    </button>
                  </div>
                </>
              )}
            </div>
          </>
        )}
      </div>
    </div>
  );
}
