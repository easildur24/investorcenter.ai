import {
  getMyTasks,
  getMyTask,
  updateMyTaskStatus,
  getMyTaskUpdates,
  postTaskUpdate,
  sendHeartbeat,
} from '../worker-api';

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

const wrap = (data: any) => ({ success: true, data });

beforeEach(() => {
  jest.clearAllMocks();
});

describe('getMyTasks', () => {
  it('calls GET /worker/tasks with no status', async () => {
    mockGet.mockResolvedValueOnce(wrap([]));
    const result = await getMyTasks();
    expect(mockGet).toHaveBeenCalledWith('/worker/tasks');
    expect(result).toEqual([]);
  });

  it('appends status filter', async () => {
    mockGet.mockResolvedValueOnce(wrap([]));
    await getMyTasks('pending');
    expect(mockGet).toHaveBeenCalledWith('/worker/tasks?status=pending');
  });
});

describe('getMyTask', () => {
  it('calls GET /worker/tasks/:id', async () => {
    mockGet.mockResolvedValueOnce(wrap({ id: 't-1' }));
    const result = await getMyTask('t-1');
    expect(mockGet).toHaveBeenCalledWith('/worker/tasks/t-1');
    expect(result).toEqual({ id: 't-1' });
  });
});

describe('updateMyTaskStatus', () => {
  it('calls PUT /worker/tasks/:id/status', async () => {
    mockPut.mockResolvedValueOnce(wrap({ id: 't-1', status: 'completed' }));
    const result = await updateMyTaskStatus('t-1', 'completed');
    expect(mockPut).toHaveBeenCalledWith('/worker/tasks/t-1/status', { status: 'completed' });
    expect(result.status).toBe('completed');
  });
});

describe('getMyTaskUpdates', () => {
  it('calls GET /worker/tasks/:taskId/updates', async () => {
    mockGet.mockResolvedValueOnce(wrap([]));
    await getMyTaskUpdates('t-1');
    expect(mockGet).toHaveBeenCalledWith('/worker/tasks/t-1/updates');
  });
});

describe('postTaskUpdate', () => {
  it('calls POST /worker/tasks/:taskId/updates', async () => {
    mockPost.mockResolvedValueOnce(wrap({ id: 'u-1' }));
    const result = await postTaskUpdate('t-1', 'Done with step 1');
    expect(mockPost).toHaveBeenCalledWith('/worker/tasks/t-1/updates', {
      content: 'Done with step 1',
    });
    expect(result).toEqual({ id: 'u-1' });
  });
});

describe('sendHeartbeat', () => {
  it('calls POST /worker/heartbeat', async () => {
    mockPost.mockResolvedValueOnce(wrap(null));
    await sendHeartbeat();
    expect(mockPost).toHaveBeenCalledWith('/worker/heartbeat', {});
  });
});
