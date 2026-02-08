'use client';

import { useState, useEffect, useCallback } from 'react';
import {
  listWorkers,
  registerWorker,
  removeWorker,
  listTasks,
  createTask,
  updateTask,
  deleteTask,
  listTaskUpdates,
  Worker,
  WorkerTask,
  TaskUpdate,
  TaskStatus,
  TaskPriority,
  TASK_STATUSES,
  TASK_PRIORITIES,
  STATUS_LABELS,
  PRIORITY_LABELS,
  STATUS_COLORS,
  PRIORITY_COLORS,
} from '@/lib/api/workers';
import {
  Bot,
  Plus,
  Trash2,
  Clock,
  MessageSquare,
  ArrowLeft,
  Loader2,
  ListTodo,
  Users,
  Circle,
} from 'lucide-react';

type View = 'workers' | 'tasks';
type RightPanel = 'none' | 'worker' | 'task' | 'create-worker' | 'create-task';

export default function WorkersPage() {
  const [view, setView] = useState<View>('workers');
  const [workers, setWorkers] = useState<Worker[]>([]);
  const [tasks, setTasks] = useState<WorkerTask[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Selection
  const [rightPanel, setRightPanel] = useState<RightPanel>('none');
  const [selectedWorker, setSelectedWorker] = useState<Worker | null>(null);
  const [selectedTask, setSelectedTask] = useState<WorkerTask | null>(null);
  const [workerTasks, setWorkerTasks] = useState<WorkerTask[]>([]);
  const [taskUpdates, setTaskUpdates] = useState<TaskUpdate[]>([]);

  // Create worker form
  const [newWorkerEmail, setNewWorkerEmail] = useState('');
  const [newWorkerPassword, setNewWorkerPassword] = useState('');
  const [newWorkerName, setNewWorkerName] = useState('');
  const [creating, setCreating] = useState(false);

  // Create task form
  const [newTaskTitle, setNewTaskTitle] = useState('');
  const [newTaskDesc, setNewTaskDesc] = useState('');
  const [newTaskAssignee, setNewTaskAssignee] = useState('');
  const [newTaskPriority, setNewTaskPriority] = useState<TaskPriority>('medium');

  // Task filter
  const [taskFilter, setTaskFilter] = useState<TaskStatus | ''>('');

  // Fetch data
  const fetchWorkers = useCallback(async () => {
    try {
      const data = await listWorkers();
      setWorkers(data || []);
    } catch (err: any) {
      setError(err.message);
    }
  }, []);

  const fetchTasks = useCallback(async () => {
    try {
      const params: { status?: TaskStatus } = {};
      if (taskFilter) params.status = taskFilter;
      const data = await listTasks(params);
      setTasks(data || []);
    } catch (err: any) {
      setError(err.message);
    }
  }, [taskFilter]);

  const fetchAll = useCallback(async () => {
    setLoading(true);
    await Promise.all([fetchWorkers(), fetchTasks()]);
    setLoading(false);
  }, [fetchWorkers, fetchTasks]);

  useEffect(() => {
    fetchAll();
  }, [fetchAll]);

  // Fetch tasks for a worker
  const fetchWorkerTasks = async (workerId: string) => {
    try {
      const data = await listTasks({ assigned_to: workerId });
      setWorkerTasks(data || []);
    } catch {
      setWorkerTasks([]);
    }
  };

  // Fetch updates for a task
  const fetchTaskUpdates = async (taskId: string) => {
    try {
      const data = await listTaskUpdates(taskId);
      setTaskUpdates(data || []);
    } catch {
      setTaskUpdates([]);
    }
  };

  // Select worker
  const selectWorker = (worker: Worker) => {
    setSelectedWorker(worker);
    setSelectedTask(null);
    setRightPanel('worker');
    fetchWorkerTasks(worker.id);
  };

  // Select task
  const selectTask = (task: WorkerTask) => {
    setSelectedTask(task);
    setRightPanel('task');
    fetchTaskUpdates(task.id);
  };

  // Create worker
  const handleCreateWorker = async () => {
    if (!newWorkerEmail || !newWorkerPassword || !newWorkerName) return;
    setCreating(true);
    try {
      await registerWorker(newWorkerEmail, newWorkerPassword, newWorkerName);
      setNewWorkerEmail('');
      setNewWorkerPassword('');
      setNewWorkerName('');
      setRightPanel('none');
      fetchWorkers();
    } catch (err: any) {
      alert('Failed to create worker: ' + err.message);
    } finally {
      setCreating(false);
    }
  };

  // Remove worker
  const handleRemoveWorker = async (id: string) => {
    if (!confirm('Remove this worker? Their user account will remain but they will no longer be flagged as a worker.')) return;
    try {
      await removeWorker(id);
      if (selectedWorker?.id === id) {
        setSelectedWorker(null);
        setRightPanel('none');
      }
      fetchWorkers();
    } catch (err: any) {
      alert('Failed to remove worker: ' + err.message);
    }
  };

  // Create task
  const handleCreateTask = async () => {
    if (!newTaskTitle) return;
    setCreating(true);
    try {
      await createTask({
        title: newTaskTitle,
        description: newTaskDesc,
        assigned_to: newTaskAssignee || undefined,
        priority: newTaskPriority,
      });
      setNewTaskTitle('');
      setNewTaskDesc('');
      setNewTaskAssignee('');
      setNewTaskPriority('medium');
      setRightPanel('none');
      fetchTasks();
      if (selectedWorker) fetchWorkerTasks(selectedWorker.id);
    } catch (err: any) {
      alert('Failed to create task: ' + err.message);
    } finally {
      setCreating(false);
    }
  };

  // Update task status
  const handleUpdateTaskStatus = async (taskId: string, status: TaskStatus) => {
    try {
      const updated = await updateTask(taskId, { status });
      setSelectedTask(updated);
      fetchTasks();
      if (selectedWorker) fetchWorkerTasks(selectedWorker.id);
    } catch (err: any) {
      alert('Failed to update status: ' + err.message);
    }
  };

  // Delete task
  const handleDeleteTask = async (taskId: string) => {
    if (!confirm('Delete this task and all its updates?')) return;
    try {
      await deleteTask(taskId);
      if (selectedTask?.id === taskId) {
        setSelectedTask(null);
        setRightPanel(selectedWorker ? 'worker' : 'none');
      }
      fetchTasks();
      if (selectedWorker) fetchWorkerTasks(selectedWorker.id);
    } catch (err: any) {
      alert('Failed to delete task: ' + err.message);
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-ic-bg-primary flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-ic-bg-primary">
      {/* Header */}
      <div className="bg-ic-surface border-b border-ic-border">
        <div className="max-w-[1600px] mx-auto px-6 py-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Bot className="w-6 h-6 text-purple-500" />
            <h1 className="text-xl font-bold text-ic-text-primary">Workers & Tasks</h1>
          </div>
          <div className="flex items-center gap-2 text-sm">
            <span className="text-ic-text-secondary">{workers.length} workers</span>
            <span className="text-ic-text-secondary">|</span>
            <span className="text-ic-text-secondary">{tasks.length} tasks</span>
          </div>
        </div>
      </div>

      <div className="max-w-[1600px] mx-auto flex" style={{ height: 'calc(100vh - 65px)' }}>
        {/* ===== LEFT SIDEBAR ===== */}
        <div className="w-80 flex-shrink-0 bg-ic-surface border-r border-ic-border overflow-y-auto">
          {/* View toggle */}
          <div className="flex border-b border-ic-border">
            <button
              onClick={() => setView('workers')}
              className={`flex-1 flex items-center justify-center gap-2 px-4 py-3 text-sm font-medium transition ${
                view === 'workers'
                  ? 'text-blue-600 border-b-2 border-blue-500'
                  : 'text-ic-text-secondary hover:text-ic-text-primary'
              }`}
            >
              <Users className="w-4 h-4" />
              Workers
            </button>
            <button
              onClick={() => setView('tasks')}
              className={`flex-1 flex items-center justify-center gap-2 px-4 py-3 text-sm font-medium transition ${
                view === 'tasks'
                  ? 'text-blue-600 border-b-2 border-blue-500'
                  : 'text-ic-text-secondary hover:text-ic-text-primary'
              }`}
            >
              <ListTodo className="w-4 h-4" />
              All Tasks
            </button>
          </div>

          <div className="p-4">
            {/* Workers View */}
            {view === 'workers' && (
              <>
                <button
                  onClick={() => {
                    setRightPanel('create-worker');
                    setSelectedWorker(null);
                    setSelectedTask(null);
                  }}
                  className="w-full flex items-center gap-2 px-3 py-2 text-sm text-blue-600 bg-blue-50 hover:bg-blue-100 rounded-lg transition mb-4"
                >
                  <Plus className="w-4 h-4" />
                  Register Worker
                </button>

                {workers.length === 0 ? (
                  <p className="text-sm text-ic-text-secondary text-center py-8">No workers registered yet.</p>
                ) : (
                  <div className="space-y-1">
                    {workers.map((worker) => (
                      <div
                        key={worker.id}
                        className={`flex items-center gap-3 px-3 py-2.5 rounded-lg cursor-pointer group transition ${
                          selectedWorker?.id === worker.id && rightPanel === 'worker'
                            ? 'bg-blue-50 text-blue-700'
                            : 'hover:bg-ic-bg-secondary text-ic-text-primary'
                        }`}
                        onClick={() => selectWorker(worker)}
                      >
                        <Circle
                          className={`w-2.5 h-2.5 flex-shrink-0 ${
                            worker.is_online ? 'text-green-500 fill-green-500' : 'text-gray-300 fill-gray-300'
                          }`}
                        />
                        <div className="flex-1 min-w-0">
                          <p className="text-sm font-medium truncate">{worker.full_name}</p>
                          <p className="text-xs text-ic-text-secondary truncate">{worker.email}</p>
                        </div>
                        {worker.task_count > 0 && (
                          <span className="text-xs bg-ic-bg-secondary text-ic-text-secondary px-1.5 py-0.5 rounded-full">
                            {worker.task_count}
                          </span>
                        )}
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            handleRemoveWorker(worker.id);
                          }}
                          className="p-0.5 opacity-0 group-hover:opacity-100 text-ic-text-secondary hover:text-red-600 transition"
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      </div>
                    ))}
                  </div>
                )}
              </>
            )}

            {/* Tasks View */}
            {view === 'tasks' && (
              <>
                <div className="flex items-center gap-2 mb-4">
                  <button
                    onClick={() => {
                      setRightPanel('create-task');
                      setSelectedTask(null);
                    }}
                    className="flex items-center gap-2 px-3 py-2 text-sm text-blue-600 bg-blue-50 hover:bg-blue-100 rounded-lg transition"
                  >
                    <Plus className="w-4 h-4" />
                    New Task
                  </button>
                  <select
                    value={taskFilter}
                    onChange={(e) => setTaskFilter(e.target.value as TaskStatus | '')}
                    className="flex-1 text-sm px-2 py-2 border border-ic-border rounded-lg bg-ic-bg-primary text-ic-text-primary"
                  >
                    <option value="">All statuses</option>
                    {TASK_STATUSES.map((s) => (
                      <option key={s} value={s}>{STATUS_LABELS[s]}</option>
                    ))}
                  </select>
                </div>

                {tasks.length === 0 ? (
                  <p className="text-sm text-ic-text-secondary text-center py-8">No tasks found.</p>
                ) : (
                  <div className="space-y-1">
                    {tasks.map((task) => (
                      <div
                        key={task.id}
                        className={`px-3 py-2.5 rounded-lg cursor-pointer group transition ${
                          selectedTask?.id === task.id
                            ? 'bg-blue-50 text-blue-700'
                            : 'hover:bg-ic-bg-secondary text-ic-text-primary'
                        }`}
                        onClick={() => selectTask(task)}
                      >
                        <div className="flex items-center gap-2 mb-1">
                          <span className={`text-xs px-1.5 py-0.5 rounded font-medium ${STATUS_COLORS[task.status]}`}>
                            {STATUS_LABELS[task.status]}
                          </span>
                          <span className={`text-xs px-1.5 py-0.5 rounded ${PRIORITY_COLORS[task.priority]}`}>
                            {PRIORITY_LABELS[task.priority]}
                          </span>
                        </div>
                        <p className="text-sm font-medium truncate">{task.title}</p>
                        {task.assigned_to_name && (
                          <p className="text-xs text-ic-text-secondary mt-0.5">{task.assigned_to_name}</p>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </>
            )}
          </div>
        </div>

        {/* ===== RIGHT PANEL ===== */}
        <div className="flex-1 overflow-y-auto">
          {/* Empty state */}
          {rightPanel === 'none' && (
            <div className="flex items-center justify-center h-full text-ic-text-secondary">
              <div className="text-center">
                <Bot className="w-12 h-12 mx-auto mb-4 opacity-30" />
                <p className="text-lg">Select a worker or task</p>
                <p className="text-sm mt-1">Or register a new worker to get started</p>
              </div>
            </div>
          )}

          {/* Create Worker Form */}
          {rightPanel === 'create-worker' && (
            <div className="p-8 max-w-lg">
              <h2 className="text-lg font-semibold text-ic-text-primary mb-6 flex items-center gap-2">
                <Bot className="w-5 h-5 text-purple-500" />
                Register New Worker
              </h2>
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-ic-text-secondary mb-1">Name</label>
                  <input
                    value={newWorkerName}
                    onChange={(e) => setNewWorkerName(e.target.value)}
                    placeholder="Worker name"
                    className="w-full px-3 py-2 border border-ic-border rounded-lg bg-ic-bg-primary text-ic-text-primary focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-ic-text-secondary mb-1">Email</label>
                  <input
                    type="email"
                    value={newWorkerEmail}
                    onChange={(e) => setNewWorkerEmail(e.target.value)}
                    placeholder="worker@example.com"
                    className="w-full px-3 py-2 border border-ic-border rounded-lg bg-ic-bg-primary text-ic-text-primary focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-ic-text-secondary mb-1">Password</label>
                  <input
                    type="password"
                    value={newWorkerPassword}
                    onChange={(e) => setNewWorkerPassword(e.target.value)}
                    placeholder="Min 8 characters"
                    className="w-full px-3 py-2 border border-ic-border rounded-lg bg-ic-bg-primary text-ic-text-primary focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  />
                </div>
                <div className="flex gap-2">
                  <button
                    onClick={handleCreateWorker}
                    disabled={creating || !newWorkerEmail || !newWorkerPassword || !newWorkerName}
                    className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition disabled:opacity-50"
                  >
                    {creating ? <Loader2 className="w-4 h-4 animate-spin" /> : <Plus className="w-4 h-4" />}
                    Register
                  </button>
                  <button
                    onClick={() => setRightPanel('none')}
                    className="px-4 py-2 text-ic-text-secondary border border-ic-border rounded-lg hover:bg-ic-bg-secondary transition"
                  >
                    Cancel
                  </button>
                </div>
              </div>
            </div>
          )}

          {/* Create Task Form */}
          {rightPanel === 'create-task' && (
            <div className="p-8 max-w-lg">
              <h2 className="text-lg font-semibold text-ic-text-primary mb-6 flex items-center gap-2">
                <ListTodo className="w-5 h-5 text-blue-500" />
                Create New Task
              </h2>
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-ic-text-secondary mb-1">Title</label>
                  <input
                    value={newTaskTitle}
                    onChange={(e) => setNewTaskTitle(e.target.value)}
                    placeholder="Task title"
                    className="w-full px-3 py-2 border border-ic-border rounded-lg bg-ic-bg-primary text-ic-text-primary focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-ic-text-secondary mb-1">Description</label>
                  <textarea
                    value={newTaskDesc}
                    onChange={(e) => setNewTaskDesc(e.target.value)}
                    rows={4}
                    placeholder="Task description..."
                    className="w-full px-3 py-2 border border-ic-border rounded-lg bg-ic-bg-primary text-ic-text-primary focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-y"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-ic-text-secondary mb-1">Assign To</label>
                  <select
                    value={newTaskAssignee}
                    onChange={(e) => setNewTaskAssignee(e.target.value)}
                    className="w-full px-3 py-2 border border-ic-border rounded-lg bg-ic-bg-primary text-ic-text-primary"
                  >
                    <option value="">Unassigned</option>
                    {workers.map((w) => (
                      <option key={w.id} value={w.id}>{w.full_name} ({w.email})</option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium text-ic-text-secondary mb-1">Priority</label>
                  <select
                    value={newTaskPriority}
                    onChange={(e) => setNewTaskPriority(e.target.value as TaskPriority)}
                    className="w-full px-3 py-2 border border-ic-border rounded-lg bg-ic-bg-primary text-ic-text-primary"
                  >
                    {TASK_PRIORITIES.map((p) => (
                      <option key={p} value={p}>{PRIORITY_LABELS[p]}</option>
                    ))}
                  </select>
                </div>
                <div className="flex gap-2">
                  <button
                    onClick={handleCreateTask}
                    disabled={creating || !newTaskTitle}
                    className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition disabled:opacity-50"
                  >
                    {creating ? <Loader2 className="w-4 h-4 animate-spin" /> : <Plus className="w-4 h-4" />}
                    Create Task
                  </button>
                  <button
                    onClick={() => setRightPanel('none')}
                    className="px-4 py-2 text-ic-text-secondary border border-ic-border rounded-lg hover:bg-ic-bg-secondary transition"
                  >
                    Cancel
                  </button>
                </div>
              </div>
            </div>
          )}

          {/* Worker Detail */}
          {rightPanel === 'worker' && selectedWorker && (
            <div className="p-8">
              {/* Worker info card */}
              <div className="bg-ic-surface border border-ic-border rounded-lg p-6 mb-6 max-w-3xl">
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-4">
                    <div className={`w-3 h-3 rounded-full ${selectedWorker.is_online ? 'bg-green-500' : 'bg-gray-300'}`} />
                    <div>
                      <h2 className="text-lg font-semibold text-ic-text-primary">{selectedWorker.full_name}</h2>
                      <p className="text-sm text-ic-text-secondary">{selectedWorker.email}</p>
                    </div>
                  </div>
                  <span className={`text-xs font-medium px-2 py-1 rounded ${selectedWorker.is_online ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-600'}`}>
                    {selectedWorker.is_online ? 'Online' : 'Offline'}
                  </span>
                </div>
                <div className="mt-4 grid grid-cols-3 gap-4 text-sm">
                  <div>
                    <p className="text-ic-text-secondary">Last Login</p>
                    <p className="text-ic-text-primary">
                      {selectedWorker.last_login_at ? new Date(selectedWorker.last_login_at).toLocaleString() : 'Never'}
                    </p>
                  </div>
                  <div>
                    <p className="text-ic-text-secondary">Last Activity</p>
                    <p className="text-ic-text-primary">
                      {selectedWorker.last_activity_at ? new Date(selectedWorker.last_activity_at).toLocaleString() : 'Never'}
                    </p>
                  </div>
                  <div>
                    <p className="text-ic-text-secondary">Tasks</p>
                    <p className="text-ic-text-primary">{selectedWorker.task_count}</p>
                  </div>
                </div>
              </div>

              {/* Worker's tasks */}
              <div className="max-w-3xl">
                <div className="flex items-center justify-between mb-4">
                  <h3 className="text-sm font-medium text-ic-text-secondary">Assigned Tasks</h3>
                  <button
                    onClick={() => {
                      setNewTaskAssignee(selectedWorker.id);
                      setRightPanel('create-task');
                    }}
                    className="flex items-center gap-1 px-3 py-1.5 text-sm text-blue-600 bg-blue-50 hover:bg-blue-100 rounded-lg transition"
                  >
                    <Plus className="w-3.5 h-3.5" />
                    Assign Task
                  </button>
                </div>

                {workerTasks.length === 0 ? (
                  <p className="text-sm text-ic-text-secondary text-center py-6">No tasks assigned to this worker.</p>
                ) : (
                  <div className="space-y-2">
                    {workerTasks.map((task) => (
                      <div
                        key={task.id}
                        className="flex items-center gap-3 px-4 py-3 bg-ic-surface border border-ic-border rounded-lg cursor-pointer hover:border-blue-300 transition group"
                        onClick={() => selectTask(task)}
                      >
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 mb-1">
                            <span className={`text-xs px-1.5 py-0.5 rounded font-medium ${STATUS_COLORS[task.status]}`}>
                              {STATUS_LABELS[task.status]}
                            </span>
                            <span className={`text-xs px-1.5 py-0.5 rounded ${PRIORITY_COLORS[task.priority]}`}>
                              {PRIORITY_LABELS[task.priority]}
                            </span>
                          </div>
                          <p className="text-sm font-medium text-ic-text-primary truncate">{task.title}</p>
                        </div>
                        <span className="text-xs text-ic-text-secondary">{new Date(task.updated_at).toLocaleDateString()}</span>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Task Detail */}
          {rightPanel === 'task' && selectedTask && (
            <div className="p-8 max-w-3xl">
              <button
                onClick={() => {
                  setSelectedTask(null);
                  setRightPanel(selectedWorker ? 'worker' : 'none');
                }}
                className="flex items-center gap-1 text-sm text-ic-text-secondary hover:text-ic-text-primary mb-4 transition"
              >
                <ArrowLeft className="w-4 h-4" />
                Back
              </button>

              {/* Task header */}
              <div className="mb-6">
                <h2 className="text-xl font-semibold text-ic-text-primary mb-2">{selectedTask.title}</h2>
                {selectedTask.description && (
                  <p className="text-sm text-ic-text-secondary whitespace-pre-wrap">{selectedTask.description}</p>
                )}
              </div>

              {/* Task meta */}
              <div className="flex flex-wrap gap-3 mb-6">
                <div className="flex items-center gap-2">
                  <span className="text-xs text-ic-text-secondary">Status:</span>
                  <select
                    value={selectedTask.status}
                    onChange={(e) => handleUpdateTaskStatus(selectedTask.id, e.target.value as TaskStatus)}
                    className={`text-xs px-2 py-1 rounded font-medium border-0 ${STATUS_COLORS[selectedTask.status]}`}
                  >
                    {TASK_STATUSES.map((s) => (
                      <option key={s} value={s}>{STATUS_LABELS[s]}</option>
                    ))}
                  </select>
                </div>
                <span className={`text-xs px-2 py-1 rounded ${PRIORITY_COLORS[selectedTask.priority]}`}>
                  {PRIORITY_LABELS[selectedTask.priority]}
                </span>
                {selectedTask.assigned_to_name && (
                  <span className="text-xs text-ic-text-secondary flex items-center gap-1">
                    <Bot className="w-3 h-3" />
                    {selectedTask.assigned_to_name}
                  </span>
                )}
                <span className="text-xs text-ic-text-secondary flex items-center gap-1">
                  <Clock className="w-3 h-3" />
                  {new Date(selectedTask.created_at).toLocaleString()}
                </span>
              </div>

              {/* Delete task */}
              <button
                onClick={() => handleDeleteTask(selectedTask.id)}
                className="flex items-center gap-1 text-xs text-red-500 hover:text-red-700 mb-6 transition"
              >
                <Trash2 className="w-3 h-3" />
                Delete task
              </button>

              {/* Updates timeline */}
              <div className="border-t border-ic-border pt-6">
                <h3 className="text-sm font-medium text-ic-text-secondary mb-4 flex items-center gap-2">
                  <MessageSquare className="w-4 h-4" />
                  Updates ({taskUpdates.length})
                </h3>

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
                        <p className="text-sm text-ic-text-primary whitespace-pre-wrap">{update.content}</p>
                      </div>
                    ))}
                  </div>
                )}

                {taskUpdates.length === 0 && (
                  <p className="text-sm text-ic-text-secondary text-center py-4">No updates yet. Workers will post updates here.</p>
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
