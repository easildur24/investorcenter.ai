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
  createTaskUpdate,
  listTaskTypes,
  createTaskType,
  deleteTaskType,
  Worker,
  WorkerTask,
  TaskUpdate,
  TaskType,
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
  FileText,
  Send,
  ChevronDown,
  ChevronRight,
  Braces,
  CheckCircle2,
  XCircle,
  PlayCircle,
} from 'lucide-react';

type View = 'workers' | 'tasks' | 'task-types';
type RightPanel = 'none' | 'worker' | 'task' | 'create-worker' | 'create-task' | 'task-type' | 'create-task-type';

export default function WorkersPage() {
  const [view, setView] = useState<View>('workers');
  const [workers, setWorkers] = useState<Worker[]>([]);
  const [tasks, setTasks] = useState<WorkerTask[]>([]);
  const [taskTypes, setTaskTypes] = useState<TaskType[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Selection
  const [rightPanel, setRightPanel] = useState<RightPanel>('none');
  const [selectedWorker, setSelectedWorker] = useState<Worker | null>(null);
  const [selectedTask, setSelectedTask] = useState<WorkerTask | null>(null);
  const [selectedTaskType, setSelectedTaskType] = useState<TaskType | null>(null);
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
  const [newTaskTypeId, setNewTaskTypeId] = useState<number | ''>('');
  const [newTaskParams, setNewTaskParams] = useState<Record<string, string>>({});

  // Create task type form
  const [newTTName, setNewTTName] = useState('');
  const [newTTLabel, setNewTTLabel] = useState('');
  const [newTTSop, setNewTTSop] = useState('');

  // Task update form
  const [newUpdateContent, setNewUpdateContent] = useState('');
  const [sendingUpdate, setSendingUpdate] = useState(false);

  // Task filter
  const [taskFilter, setTaskFilter] = useState<TaskStatus | ''>('');

  // Collapsible sections
  const [showSop, setShowSop] = useState(false);
  const [showResult, setShowResult] = useState(false);
  const [showParams, setShowParams] = useState(true);

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

  const fetchTaskTypes = useCallback(async () => {
    try {
      const data = await listTaskTypes();
      setTaskTypes(data || []);
    } catch (err: any) {
      setError(err.message);
    }
  }, []);

  const fetchAll = useCallback(async () => {
    setLoading(true);
    await Promise.all([fetchWorkers(), fetchTasks(), fetchTaskTypes()]);
    setLoading(false);
  }, [fetchWorkers, fetchTasks, fetchTaskTypes]);

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
    setSelectedTaskType(null);
    setRightPanel('worker');
    fetchWorkerTasks(worker.id);
  };

  // Select task
  const selectTask = (task: WorkerTask) => {
    setSelectedTask(task);
    setSelectedTaskType(null);
    setRightPanel('task');
    setShowSop(false);
    setShowResult(false);
    setShowParams(true);
    fetchTaskUpdates(task.id);
  };

  // Select task type
  const selectTaskType = (tt: TaskType) => {
    setSelectedTaskType(tt);
    setSelectedTask(null);
    setSelectedWorker(null);
    setRightPanel('task-type');
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

  // Get selected task type's param schema
  const getSelectedTaskTypeSchema = (): Record<string, string> | null => {
    if (!newTaskTypeId) return null;
    const tt = taskTypes.find((t) => t.id === newTaskTypeId);
    return tt?.param_schema || null;
  };

  // Handle task type change in create task form
  const handleTaskTypeChange = (id: number | '') => {
    setNewTaskTypeId(id);
    if (id) {
      const tt = taskTypes.find((t) => t.id === id);
      if (tt?.param_schema) {
        const defaults: Record<string, string> = {};
        for (const key of Object.keys(tt.param_schema)) {
          defaults[key] = '';
        }
        setNewTaskParams(defaults);
      } else {
        setNewTaskParams({});
      }
    } else {
      setNewTaskParams({});
    }
  };

  // Create task
  const handleCreateTask = async () => {
    if (!newTaskTitle) return;
    setCreating(true);
    try {
      const params: Record<string, unknown> = {};
      const schema = getSelectedTaskTypeSchema();
      for (const [key, val] of Object.entries(newTaskParams)) {
        if (val === '') continue;
        const typeHint = schema?.[key];
        if (typeHint === 'number') {
          const num = Number(val);
          params[key] = !isNaN(num) ? num : val;
        } else if (typeHint?.endsWith('[]')) {
          // Array types — parse as JSON array
          try {
            params[key] = JSON.parse(val);
          } catch {
            // Fallback: split by comma
            params[key] = val.split(',').map((s) => s.trim()).filter(Boolean);
          }
        } else {
          // String and unknown types — keep as string
          params[key] = val;
        }
      }

      await createTask({
        title: newTaskTitle,
        description: newTaskDesc,
        assigned_to: newTaskAssignee || undefined,
        priority: newTaskPriority,
        task_type_id: newTaskTypeId || undefined,
        params: Object.keys(params).length > 0 ? params : undefined,
      });
      setNewTaskTitle('');
      setNewTaskDesc('');
      setNewTaskAssignee('');
      setNewTaskPriority('medium');
      setNewTaskTypeId('');
      setNewTaskParams({});
      setRightPanel('none');
      fetchTasks();
      if (selectedWorker) fetchWorkerTasks(selectedWorker.id);
    } catch (err: any) {
      alert('Failed to create task: ' + err.message);
    } finally {
      setCreating(false);
    }
  };

  // Create task type
  const handleCreateTaskType = async () => {
    if (!newTTName || !newTTLabel) return;
    setCreating(true);
    try {
      await createTaskType({
        name: newTTName,
        label: newTTLabel,
        sop: newTTSop || undefined,
      });
      setNewTTName('');
      setNewTTLabel('');
      setNewTTSop('');
      setRightPanel('none');
      fetchTaskTypes();
    } catch (err: any) {
      alert('Failed to create task type: ' + err.message);
    } finally {
      setCreating(false);
    }
  };

  // Delete task type
  const handleDeleteTaskType = async (id: number) => {
    if (!confirm('Delete this task type? Existing tasks using it will keep their reference.')) return;
    try {
      await deleteTaskType(id);
      if (selectedTaskType?.id === id) {
        setSelectedTaskType(null);
        setRightPanel('none');
      }
      fetchTaskTypes();
    } catch (err: any) {
      alert('Failed to delete task type: ' + err.message);
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

  // Send task update
  const handleSendUpdate = async () => {
    if (!selectedTask || !newUpdateContent.trim()) return;
    setSendingUpdate(true);
    try {
      await createTaskUpdate(selectedTask.id, newUpdateContent.trim());
      setNewUpdateContent('');
      fetchTaskUpdates(selectedTask.id);
    } catch (err: any) {
      alert('Failed to send update: ' + err.message);
    } finally {
      setSendingUpdate(false);
    }
  };

  // Format timestamp
  const formatTime = (ts: string | null) => {
    if (!ts) return '--';
    return new Date(ts).toLocaleString();
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
            <span className="text-ic-text-secondary">|</span>
            <span className="text-ic-text-secondary">{taskTypes.length} types</span>
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
              className={`flex-1 flex items-center justify-center gap-1.5 px-3 py-3 text-sm font-medium transition ${
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
              className={`flex-1 flex items-center justify-center gap-1.5 px-3 py-3 text-sm font-medium transition ${
                view === 'tasks'
                  ? 'text-blue-600 border-b-2 border-blue-500'
                  : 'text-ic-text-secondary hover:text-ic-text-primary'
              }`}
            >
              <ListTodo className="w-4 h-4" />
              Tasks
            </button>
            <button
              onClick={() => setView('task-types')}
              className={`flex-1 flex items-center justify-center gap-1.5 px-3 py-3 text-sm font-medium transition ${
                view === 'task-types'
                  ? 'text-blue-600 border-b-2 border-blue-500'
                  : 'text-ic-text-secondary hover:text-ic-text-primary'
              }`}
            >
              <FileText className="w-4 h-4" />
              Types
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
                    setSelectedTaskType(null);
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
                      setSelectedTaskType(null);
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
                          {task.task_type && (
                            <span className="text-xs px-1.5 py-0.5 rounded bg-purple-50 text-purple-700">
                              {task.task_type.label}
                            </span>
                          )}
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

            {/* Task Types View */}
            {view === 'task-types' && (
              <>
                <button
                  onClick={() => {
                    setRightPanel('create-task-type');
                    setSelectedTaskType(null);
                    setSelectedTask(null);
                    setSelectedWorker(null);
                  }}
                  className="w-full flex items-center gap-2 px-3 py-2 text-sm text-purple-600 bg-purple-50 hover:bg-purple-100 rounded-lg transition mb-4"
                >
                  <Plus className="w-4 h-4" />
                  New Task Type
                </button>

                {taskTypes.length === 0 ? (
                  <p className="text-sm text-ic-text-secondary text-center py-8">No task types defined yet.</p>
                ) : (
                  <div className="space-y-1">
                    {taskTypes.map((tt) => (
                      <div
                        key={tt.id}
                        className={`flex items-center gap-3 px-3 py-2.5 rounded-lg cursor-pointer group transition ${
                          selectedTaskType?.id === tt.id
                            ? 'bg-purple-50 text-purple-700'
                            : 'hover:bg-ic-bg-secondary text-ic-text-primary'
                        }`}
                        onClick={() => selectTaskType(tt)}
                      >
                        <FileText className="w-4 h-4 flex-shrink-0 text-purple-400" />
                        <div className="flex-1 min-w-0">
                          <p className="text-sm font-medium truncate">{tt.label}</p>
                          <p className="text-xs text-ic-text-secondary truncate font-mono">{tt.name}</p>
                        </div>
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            handleDeleteTaskType(tt.id);
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
          </div>
        </div>

        {/* ===== RIGHT PANEL ===== */}
        <div className="flex-1 overflow-y-auto">
          {/* Empty state */}
          {rightPanel === 'none' && (
            <div className="flex items-center justify-center h-full text-ic-text-secondary">
              <div className="text-center">
                <Bot className="w-12 h-12 mx-auto mb-4 opacity-30" />
                <p className="text-lg">Select a worker, task, or task type</p>
                <p className="text-sm mt-1">Or create something new to get started</p>
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
                {/* Task Type */}
                <div>
                  <label className="block text-sm font-medium text-ic-text-secondary mb-1">Task Type</label>
                  <select
                    value={newTaskTypeId}
                    onChange={(e) => handleTaskTypeChange(e.target.value ? Number(e.target.value) : '')}
                    className="w-full px-3 py-2 border border-ic-border rounded-lg bg-ic-bg-primary text-ic-text-primary"
                  >
                    <option value="">No type (custom task)</option>
                    {taskTypes.map((tt) => (
                      <option key={tt.id} value={tt.id}>{tt.label}</option>
                    ))}
                  </select>
                </div>

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
                    rows={3}
                    placeholder="Task description..."
                    className="w-full px-3 py-2 border border-ic-border rounded-lg bg-ic-bg-primary text-ic-text-primary focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-y"
                  />
                </div>

                {/* Dynamic params from task type schema */}
                {(() => {
                  const schema = getSelectedTaskTypeSchema();
                  if (!schema) return null;
                  return (
                  <div className="border border-ic-border rounded-lg p-4 bg-ic-bg-secondary">
                    <p className="text-sm font-medium text-ic-text-secondary mb-3 flex items-center gap-1.5">
                      <Braces className="w-4 h-4" />
                      Parameters
                    </p>
                    <div className="space-y-3">
                      {Object.entries(schema).map(([key, type]) => (
                        <div key={key}>
                          <label className="block text-xs font-medium text-ic-text-secondary mb-1">
                            {key} <span className="text-ic-text-secondary font-normal">({type})</span>
                          </label>
                          <input
                            value={newTaskParams[key] || ''}
                            onChange={(e) =>
                              setNewTaskParams((prev) => ({ ...prev, [key]: e.target.value }))
                            }
                            placeholder={type === 'string[]' ? '["value1", "value2"]' : type === 'number' ? '0' : ''}
                            className="w-full px-3 py-1.5 text-sm border border-ic-border rounded-lg bg-ic-bg-primary text-ic-text-primary focus:ring-2 focus:ring-blue-500 focus:border-transparent font-mono"
                          />
                        </div>
                      ))}
                    </div>
                  </div>
                  );
                })()}

                <div>
                  <label className="block text-sm font-medium text-ic-text-secondary mb-1">Assign To</label>
                  <select
                    value={newTaskAssignee}
                    onChange={(e) => setNewTaskAssignee(e.target.value)}
                    className="w-full px-3 py-2 border border-ic-border rounded-lg bg-ic-bg-primary text-ic-text-primary"
                  >
                    <option value="">Unassigned</option>
                    {workers.map((w) => (
                      <option key={w.id} value={w.id}>
                        {w.full_name} ({w.email})
                        {w.is_online ? ' - Online' : ''}
                      </option>
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

          {/* Create Task Type Form */}
          {rightPanel === 'create-task-type' && (
            <div className="p-8 max-w-2xl">
              <h2 className="text-lg font-semibold text-ic-text-primary mb-6 flex items-center gap-2">
                <FileText className="w-5 h-5 text-purple-500" />
                Create New Task Type
              </h2>
              <div className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-ic-text-secondary mb-1">Name (slug)</label>
                    <input
                      value={newTTName}
                      onChange={(e) => setNewTTName(e.target.value.toLowerCase().replace(/[^a-z0-9_]/g, '_'))}
                      placeholder="reddit_crawl"
                      className="w-full px-3 py-2 border border-ic-border rounded-lg bg-ic-bg-primary text-ic-text-primary focus:ring-2 focus:ring-purple-500 focus:border-transparent font-mono"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-ic-text-secondary mb-1">Label</label>
                    <input
                      value={newTTLabel}
                      onChange={(e) => setNewTTLabel(e.target.value)}
                      placeholder="Reddit Crawl"
                      className="w-full px-3 py-2 border border-ic-border rounded-lg bg-ic-bg-primary text-ic-text-primary focus:ring-2 focus:ring-purple-500 focus:border-transparent"
                    />
                  </div>
                </div>
                <div>
                  <label className="block text-sm font-medium text-ic-text-secondary mb-1">SOP (Markdown)</label>
                  <textarea
                    value={newTTSop}
                    onChange={(e) => setNewTTSop(e.target.value)}
                    rows={12}
                    placeholder="## Standard Operating Procedure&#10;&#10;### What You're Doing&#10;...&#10;&#10;### How To Do It&#10;..."
                    className="w-full px-3 py-2 border border-ic-border rounded-lg bg-ic-bg-primary text-ic-text-primary focus:ring-2 focus:ring-purple-500 focus:border-transparent resize-y font-mono text-sm"
                  />
                </div>
                <div className="flex gap-2">
                  <button
                    onClick={handleCreateTaskType}
                    disabled={creating || !newTTName || !newTTLabel}
                    className="flex items-center gap-2 px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700 transition disabled:opacity-50"
                  >
                    {creating ? <Loader2 className="w-4 h-4 animate-spin" /> : <Plus className="w-4 h-4" />}
                    Create Task Type
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

          {/* Task Type Detail */}
          {rightPanel === 'task-type' && selectedTaskType && (
            <div className="p-8 max-w-3xl">
              <div className="mb-6">
                <div className="flex items-start justify-between">
                  <div>
                    <h2 className="text-xl font-semibold text-ic-text-primary flex items-center gap-2">
                      <FileText className="w-5 h-5 text-purple-500" />
                      {selectedTaskType.label}
                    </h2>
                    <p className="text-sm text-ic-text-secondary font-mono mt-1">{selectedTaskType.name}</p>
                  </div>
                  <button
                    onClick={() => handleDeleteTaskType(selectedTaskType.id)}
                    className="flex items-center gap-1 text-xs text-red-500 hover:text-red-700 transition"
                  >
                    <Trash2 className="w-3 h-3" />
                    Delete
                  </button>
                </div>
              </div>

              {/* Param Schema */}
              {selectedTaskType.param_schema && (
                <div className="mb-6">
                  <h3 className="text-sm font-medium text-ic-text-secondary mb-2 flex items-center gap-1.5">
                    <Braces className="w-4 h-4" />
                    Parameter Schema
                  </h3>
                  <div className="bg-ic-bg-secondary rounded-lg p-4">
                    <pre className="text-sm text-ic-text-primary font-mono whitespace-pre-wrap">
                      {JSON.stringify(selectedTaskType.param_schema, null, 2)}
                    </pre>
                  </div>
                </div>
              )}

              {/* SOP */}
              {selectedTaskType.sop && (
                <div>
                  <h3 className="text-sm font-medium text-ic-text-secondary mb-2">Standard Operating Procedure</h3>
                  <div className="bg-ic-bg-secondary rounded-lg p-4 max-h-[60vh] overflow-y-auto">
                    <pre className="text-sm text-ic-text-primary whitespace-pre-wrap font-mono leading-relaxed">
                      {selectedTaskType.sop}
                    </pre>
                  </div>
                </div>
              )}

              <div className="mt-4 text-xs text-ic-text-secondary">
                Created: {formatTime(selectedTaskType.created_at)} | Updated: {formatTime(selectedTaskType.updated_at)}
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
                            {task.task_type && (
                              <span className="text-xs px-1.5 py-0.5 rounded bg-purple-50 text-purple-700">
                                {task.task_type.label}
                              </span>
                            )}
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
              <div className="mb-4">
                <h2 className="text-xl font-semibold text-ic-text-primary mb-2">{selectedTask.title}</h2>
                {selectedTask.description && (
                  <p className="text-sm text-ic-text-secondary whitespace-pre-wrap">{selectedTask.description}</p>
                )}
              </div>

              {/* Task meta */}
              <div className="flex flex-wrap gap-3 mb-4">
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
                {selectedTask.task_type && (
                  <span className="text-xs px-2 py-1 rounded bg-purple-50 text-purple-700 flex items-center gap-1">
                    <FileText className="w-3 h-3" />
                    {selectedTask.task_type.label}
                  </span>
                )}
                {selectedTask.assigned_to_name && (
                  <span className="text-xs text-ic-text-secondary flex items-center gap-1">
                    <Bot className="w-3 h-3" />
                    {selectedTask.assigned_to_name}
                  </span>
                )}
              </div>

              {/* Timestamps */}
              <div className="flex flex-wrap gap-4 mb-4 text-xs text-ic-text-secondary">
                <span className="flex items-center gap-1">
                  <Clock className="w-3 h-3" />
                  Created: {formatTime(selectedTask.created_at)}
                </span>
                {selectedTask.started_at && (
                  <span className="flex items-center gap-1">
                    <PlayCircle className="w-3 h-3 text-blue-500" />
                    Started: {formatTime(selectedTask.started_at)}
                  </span>
                )}
                {selectedTask.completed_at && (
                  <span className="flex items-center gap-1">
                    {selectedTask.status === 'completed' ? (
                      <CheckCircle2 className="w-3 h-3 text-green-500" />
                    ) : (
                      <XCircle className="w-3 h-3 text-red-500" />
                    )}
                    {selectedTask.status === 'completed' ? 'Completed' : 'Failed'}: {formatTime(selectedTask.completed_at)}
                  </span>
                )}
              </div>

              {/* Params */}
              {selectedTask.params && Object.keys(selectedTask.params).length > 0 && (
                <div className="mb-4">
                  <button
                    onClick={() => setShowParams(!showParams)}
                    className="flex items-center gap-1 text-sm font-medium text-ic-text-secondary mb-2 hover:text-ic-text-primary transition"
                  >
                    {showParams ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />}
                    <Braces className="w-4 h-4" />
                    Parameters
                  </button>
                  {showParams && (
                    <div className="bg-ic-bg-secondary rounded-lg p-3">
                      <pre className="text-sm text-ic-text-primary font-mono whitespace-pre-wrap">
                        {JSON.stringify(selectedTask.params, null, 2)}
                      </pre>
                    </div>
                  )}
                </div>
              )}

              {/* Result */}
              {selectedTask.result && Object.keys(selectedTask.result).length > 0 && (
                <div className="mb-4">
                  <button
                    onClick={() => setShowResult(!showResult)}
                    className="flex items-center gap-1 text-sm font-medium text-ic-text-secondary mb-2 hover:text-ic-text-primary transition"
                  >
                    {showResult ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />}
                    <CheckCircle2 className="w-4 h-4" />
                    Result
                  </button>
                  {showResult && (
                    <div className="bg-green-50 border border-green-200 rounded-lg p-3">
                      <pre className="text-sm text-ic-text-primary font-mono whitespace-pre-wrap">
                        {JSON.stringify(selectedTask.result, null, 2)}
                      </pre>
                    </div>
                  )}
                </div>
              )}

              {/* SOP (from task type) */}
              {selectedTask.task_type?.sop && (
                <div className="mb-4">
                  <button
                    onClick={() => setShowSop(!showSop)}
                    className="flex items-center gap-1 text-sm font-medium text-ic-text-secondary mb-2 hover:text-ic-text-primary transition"
                  >
                    {showSop ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />}
                    <FileText className="w-4 h-4" />
                    SOP ({selectedTask.task_type.label})
                  </button>
                  {showSop && (
                    <div className="bg-purple-50 border border-purple-200 rounded-lg p-3 max-h-64 overflow-y-auto">
                      <pre className="text-sm text-ic-text-primary font-mono whitespace-pre-wrap leading-relaxed">
                        {selectedTask.task_type.sop}
                      </pre>
                    </div>
                  )}
                </div>
              )}

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
                  <div className="space-y-3 mb-4">
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
                  <p className="text-sm text-ic-text-secondary text-center py-4 mb-4">No updates yet.</p>
                )}

                {/* Post update form */}
                <div className="flex gap-2">
                  <input
                    value={newUpdateContent}
                    onChange={(e) => setNewUpdateContent(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter' && !e.shiftKey) {
                        e.preventDefault();
                        handleSendUpdate();
                      }
                    }}
                    placeholder="Post an update..."
                    className="flex-1 px-3 py-2 text-sm border border-ic-border rounded-lg bg-ic-bg-primary text-ic-text-primary focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  />
                  <button
                    onClick={handleSendUpdate}
                    disabled={sendingUpdate || !newUpdateContent.trim()}
                    className="flex items-center gap-1.5 px-3 py-2 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition disabled:opacity-50"
                  >
                    {sendingUpdate ? <Loader2 className="w-4 h-4 animate-spin" /> : <Send className="w-4 h-4" />}
                    Send
                  </button>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
