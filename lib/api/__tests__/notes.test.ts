import {
  getNotesTree,
  listGroups,
  createGroup,
  updateGroup,
  deleteGroup,
  listFeatures,
  createFeature,
  updateFeature,
  deleteFeature,
  listNotes,
  createNote,
  updateNote,
  deleteNote,
  SECTIONS,
  SECTION_LABELS,
} from '../notes';

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

const wrap = (data: any) => ({ success: true, data });

beforeEach(() => {
  jest.clearAllMocks();
});

describe('Constants', () => {
  it('SECTIONS has 4 entries', () => {
    expect(SECTIONS).toEqual(['ui', 'backend', 'data', 'infra']);
  });

  it('SECTION_LABELS maps correctly', () => {
    expect(SECTION_LABELS.ui).toBe('UI');
    expect(SECTION_LABELS.backend).toBe('Backend');
    expect(SECTION_LABELS.data).toBe('Data');
    expect(SECTION_LABELS.infra).toBe('Infra');
  });
});

describe('Tree', () => {
  describe('getNotesTree', () => {
    it('calls GET /admin/notes/tree and unwraps data', async () => {
      mockGet.mockResolvedValueOnce(wrap([{ id: 'g-1' }]));
      const result = await getNotesTree();
      expect(mockGet).toHaveBeenCalledWith('/admin/notes/tree');
      expect(result).toEqual([{ id: 'g-1' }]);
    });
  });
});

describe('Groups', () => {
  describe('listGroups', () => {
    it('calls GET /admin/notes/groups', async () => {
      mockGet.mockResolvedValueOnce(wrap([]));
      await listGroups();
      expect(mockGet).toHaveBeenCalledWith('/admin/notes/groups');
    });
  });

  describe('createGroup', () => {
    it('calls POST /admin/notes/groups with name and notes', async () => {
      mockPost.mockResolvedValueOnce(wrap({ id: 'g-1' }));
      await createGroup('Auth', 'Auth related');
      expect(mockPost).toHaveBeenCalledWith('/admin/notes/groups', {
        name: 'Auth',
        notes: 'Auth related',
      });
    });

    it('defaults notes to empty string', async () => {
      mockPost.mockResolvedValueOnce(wrap({ id: 'g-1' }));
      await createGroup('Auth');
      expect(mockPost).toHaveBeenCalledWith('/admin/notes/groups', {
        name: 'Auth',
        notes: '',
      });
    });
  });

  describe('updateGroup', () => {
    it('calls PUT /admin/notes/groups/:id', async () => {
      mockPut.mockResolvedValueOnce(wrap({ id: 'g-1' }));
      await updateGroup('g-1', { name: 'Updated' });
      expect(mockPut).toHaveBeenCalledWith('/admin/notes/groups/g-1', { name: 'Updated' });
    });
  });

  describe('deleteGroup', () => {
    it('calls DELETE /admin/notes/groups/:id', async () => {
      mockDelete.mockResolvedValueOnce(undefined);
      await deleteGroup('g-1');
      expect(mockDelete).toHaveBeenCalledWith('/admin/notes/groups/g-1');
    });
  });
});

describe('Features', () => {
  describe('listFeatures', () => {
    it('calls GET /admin/notes/groups/:groupId/features', async () => {
      mockGet.mockResolvedValueOnce(wrap([]));
      await listFeatures('g-1');
      expect(mockGet).toHaveBeenCalledWith('/admin/notes/groups/g-1/features');
    });
  });

  describe('createFeature', () => {
    it('calls POST /admin/notes/groups/:groupId/features', async () => {
      mockPost.mockResolvedValueOnce(wrap({ id: 'f-1' }));
      await createFeature('g-1', 'Login', 'Login feature');
      expect(mockPost).toHaveBeenCalledWith('/admin/notes/groups/g-1/features', {
        name: 'Login',
        notes: 'Login feature',
      });
    });

    it('defaults notes to empty string', async () => {
      mockPost.mockResolvedValueOnce(wrap({ id: 'f-1' }));
      await createFeature('g-1', 'Login');
      expect(mockPost).toHaveBeenCalledWith('/admin/notes/groups/g-1/features', {
        name: 'Login',
        notes: '',
      });
    });
  });

  describe('updateFeature', () => {
    it('calls PUT /admin/notes/features/:id', async () => {
      mockPut.mockResolvedValueOnce(wrap({ id: 'f-1' }));
      await updateFeature('f-1', { name: 'Updated' });
      expect(mockPut).toHaveBeenCalledWith('/admin/notes/features/f-1', { name: 'Updated' });
    });
  });

  describe('deleteFeature', () => {
    it('calls DELETE /admin/notes/features/:id', async () => {
      mockDelete.mockResolvedValueOnce(undefined);
      await deleteFeature('f-1');
      expect(mockDelete).toHaveBeenCalledWith('/admin/notes/features/f-1');
    });
  });
});

describe('Notes', () => {
  describe('listNotes', () => {
    it('calls GET /admin/notes/features/:featureId/notes', async () => {
      mockGet.mockResolvedValueOnce(wrap([]));
      await listNotes('f-1');
      expect(mockGet).toHaveBeenCalledWith('/admin/notes/features/f-1/notes');
    });

    it('appends section query param', async () => {
      mockGet.mockResolvedValueOnce(wrap([]));
      await listNotes('f-1', 'ui');
      expect(mockGet).toHaveBeenCalledWith('/admin/notes/features/f-1/notes?section=ui');
    });
  });

  describe('createNote', () => {
    it('calls POST /admin/notes/features/:featureId/notes', async () => {
      mockPost.mockResolvedValueOnce(wrap({ id: 'n-1' }));
      await createNote('f-1', 'ui', 'My Title', 'Content here');
      expect(mockPost).toHaveBeenCalledWith('/admin/notes/features/f-1/notes', {
        section: 'ui',
        title: 'My Title',
        content: 'Content here',
      });
    });

    it('defaults title and content', async () => {
      mockPost.mockResolvedValueOnce(wrap({ id: 'n-1' }));
      await createNote('f-1', 'backend');
      expect(mockPost).toHaveBeenCalledWith('/admin/notes/features/f-1/notes', {
        section: 'backend',
        title: 'Untitled',
        content: '',
      });
    });
  });

  describe('updateNote', () => {
    it('calls PUT /admin/notes/notes/:id', async () => {
      mockPut.mockResolvedValueOnce(wrap({ id: 'n-1' }));
      await updateNote('n-1', { title: 'Updated' });
      expect(mockPut).toHaveBeenCalledWith('/admin/notes/notes/n-1', { title: 'Updated' });
    });
  });

  describe('deleteNote', () => {
    it('calls DELETE /admin/notes/notes/:id', async () => {
      mockDelete.mockResolvedValueOnce(undefined);
      await deleteNote('n-1');
      expect(mockDelete).toHaveBeenCalledWith('/admin/notes/notes/n-1');
    });
  });
});
