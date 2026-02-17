import {
  listWorkers,
  registerWorker,
  removeWorker,
  listTaskTypes,
  createTaskType,
  updateTaskType,
  deleteTaskType,
  listTasks,
  getTask,
  createTask,
  updateTask,
  deleteTask,
  listTaskUpdates,
  createTaskUpdate,
  getTaskData,
  getTaskFiles,
  getTaskFileDownloadUrl,
} from '../workers';

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
  },
}));

import { apiClient } from '../client';

const mockGet = apiClient.get as jest.Mock;
const mockPost = apiClient.post as jest.Mock;
const mockPut = apiClient.put as jest.Mock;
const mockDelete = apiClient.delete as jest.Mock;

beforeEach(() => {
  jest.clearAllMocks();
});

const wrap = (data: any) => ({ success: true, data });

describe('Workers', () => {
  describe('listWorkers', () => {
    it('calls GET /admin/workers and unwraps data', async () => {
      mockGet.mockResolvedValueOnce(wrap([{ id: '1' }]));
      const result = await listWorkers();
      expect(mockGet).toHaveBeenCalledWith('/admin/workers');
      expect(result).toEqual([{ id: '1' }]);
    });
  });

  describe('registerWorker', () => {
    it('calls POST /admin/workers with payload', async () => {
      mockPost.mockResolvedValueOnce(wrap({ id: '1' }));
      const result = await registerWorker('a@b.com', 'pass', 'Name');
      expect(mockPost).toHaveBeenCalledWith('/admin/workers', {
        email: 'a@b.com',
        password: 'pass',
        full_name: 'Name',
      });
      expect(result).toEqual({ id: '1' });
    });
  });

  describe('removeWorker', () => {
    it('calls DELETE /admin/workers/:id', async () => {
      mockDelete.mockResolvedValueOnce(undefined);
      await removeWorker('w-1');
      expect(mockDelete).toHaveBeenCalledWith('/admin/workers/w-1');
    });
  });
});

describe('Task Types', () => {
  describe('listTaskTypes', () => {
    it('calls GET /admin/workers/task-types', async () => {
      mockGet.mockResolvedValueOnce(wrap([]));
      await listTaskTypes();
      expect(mockGet).toHaveBeenCalledWith('/admin/workers/task-types');
    });
  });

  describe('createTaskType', () => {
    it('calls POST /admin/workers/task-types', async () => {
      const data = { name: 'test', label: 'Test' };
      mockPost.mockResolvedValueOnce(wrap({ id: 1 }));
      await createTaskType(data);
      expect(mockPost).toHaveBeenCalledWith('/admin/workers/task-types', data);
    });
  });

  describe('updateTaskType', () => {
    it('calls PUT /admin/workers/task-types/:id', async () => {
      const data = { label: 'Updated' };
      mockPut.mockResolvedValueOnce(wrap({ id: 1 }));
      await updateTaskType(1, data);
      expect(mockPut).toHaveBeenCalledWith('/admin/workers/task-types/1', data);
    });
  });

  describe('deleteTaskType', () => {
    it('calls DELETE /admin/workers/task-types/:id', async () => {
      mockDelete.mockResolvedValueOnce(undefined);
      await deleteTaskType(5);
      expect(mockDelete).toHaveBeenCalledWith('/admin/workers/task-types/5');
    });
  });
});

describe('Tasks', () => {
  describe('listTasks', () => {
    it('calls GET /admin/workers/tasks with no params', async () => {
      mockGet.mockResolvedValueOnce(wrap([]));
      await listTasks();
      expect(mockGet).toHaveBeenCalledWith('/admin/workers/tasks');
    });

    it('appends status filter', async () => {
      mockGet.mockResolvedValueOnce(wrap([]));
      await listTasks({ status: 'pending' });
      expect(mockGet).toHaveBeenCalledWith('/admin/workers/tasks?status=pending');
    });

    it('appends multiple filters', async () => {
      mockGet.mockResolvedValueOnce(wrap([]));
      await listTasks({ status: 'in_progress', assigned_to: 'user-1', task_type: 'review' });
      const url = mockGet.mock.calls[0][0];
      expect(url).toContain('status=in_progress');
      expect(url).toContain('assigned_to=user-1');
      expect(url).toContain('task_type=review');
    });
  });

  describe('getTask', () => {
    it('calls GET /admin/workers/tasks/:id', async () => {
      mockGet.mockResolvedValueOnce(wrap({ id: 't-1' }));
      const result = await getTask('t-1');
      expect(mockGet).toHaveBeenCalledWith('/admin/workers/tasks/t-1');
      expect(result).toEqual({ id: 't-1' });
    });
  });

  describe('createTask', () => {
    it('calls POST /admin/workers/tasks', async () => {
      const data = { title: 'New task' };
      mockPost.mockResolvedValueOnce(wrap({ id: 't-2' }));
      await createTask(data);
      expect(mockPost).toHaveBeenCalledWith('/admin/workers/tasks', data);
    });
  });

  describe('updateTask', () => {
    it('calls PUT /admin/workers/tasks/:id', async () => {
      const data = { status: 'completed' as const };
      mockPut.mockResolvedValueOnce(wrap({ id: 't-1' }));
      await updateTask('t-1', data);
      expect(mockPut).toHaveBeenCalledWith('/admin/workers/tasks/t-1', data);
    });
  });

  describe('deleteTask', () => {
    it('calls DELETE /admin/workers/tasks/:id', async () => {
      mockDelete.mockResolvedValueOnce(undefined);
      await deleteTask('t-1');
      expect(mockDelete).toHaveBeenCalledWith('/admin/workers/tasks/t-1');
    });
  });
});

describe('Task Updates', () => {
  describe('listTaskUpdates', () => {
    it('calls GET /admin/workers/tasks/:taskId/updates', async () => {
      mockGet.mockResolvedValueOnce(wrap([]));
      await listTaskUpdates('t-1');
      expect(mockGet).toHaveBeenCalledWith('/admin/workers/tasks/t-1/updates');
    });
  });

  describe('createTaskUpdate', () => {
    it('calls POST /admin/workers/tasks/:taskId/updates', async () => {
      mockPost.mockResolvedValueOnce(wrap({ id: 'u-1' }));
      await createTaskUpdate('t-1', 'Progress update');
      expect(mockPost).toHaveBeenCalledWith('/admin/workers/tasks/t-1/updates', {
        content: 'Progress update',
      });
    });
  });
});

describe('Task Data', () => {
  describe('getTaskData', () => {
    it('calls GET /admin/workers/tasks/:taskId/data', async () => {
      mockGet.mockResolvedValueOnce(wrap({ items: [], total: 0, limit: 50, offset: 0 }));
      await getTaskData('t-1');
      expect(mockGet).toHaveBeenCalledWith('/admin/workers/tasks/t-1/data');
    });

    it('appends query params', async () => {
      mockGet.mockResolvedValueOnce(wrap({ items: [], total: 0, limit: 10, offset: 0 }));
      await getTaskData('t-1', { data_type: 'screenshot', ticker: 'AAPL', limit: 10, offset: 5 });
      const url = mockGet.mock.calls[0][0];
      expect(url).toContain('data_type=screenshot');
      expect(url).toContain('ticker=AAPL');
      expect(url).toContain('limit=10');
      expect(url).toContain('offset=5');
    });
  });
});

describe('Task Files', () => {
  describe('getTaskFiles', () => {
    it('calls GET /admin/workers/tasks/:taskId/files', async () => {
      mockGet.mockResolvedValueOnce(wrap({ files: [], total: 0, limit: 50, offset: 0 }));
      await getTaskFiles('t-1');
      expect(mockGet).toHaveBeenCalledWith('/admin/workers/tasks/t-1/files');
    });

    it('appends limit and offset', async () => {
      mockGet.mockResolvedValueOnce(wrap({ files: [], total: 0, limit: 10, offset: 0 }));
      await getTaskFiles('t-1', { limit: 10, offset: 5 });
      const url = mockGet.mock.calls[0][0];
      expect(url).toContain('limit=10');
      expect(url).toContain('offset=5');
    });
  });

  describe('getTaskFileDownloadUrl', () => {
    it('returns correct URL', () => {
      const url = getTaskFileDownloadUrl('t-1', 42);
      expect(url).toBe('/api/v1/admin/workers/tasks/t-1/files/42/download');
    });
  });
});
