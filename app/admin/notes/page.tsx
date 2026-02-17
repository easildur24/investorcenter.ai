'use client';

import { useState, useEffect, useCallback, useRef } from 'react';
import {
  getNotesTree,
  createGroup,
  updateGroup,
  deleteGroup,
  createFeature,
  updateFeature,
  deleteFeature,
  listNotes,
  createNote,
  updateNote,
  deleteNote,
  GroupWithFeatures,
  FeatureWithCounts,
  FeatureNote,
  Section,
  SECTIONS,
  SECTION_LABELS,
} from '@/lib/api/notes';
import {
  Plus,
  ChevronRight,
  ChevronDown,
  Folder,
  FolderOpen,
  Layers,
  FileText,
  Trash2,
  Edit3,
  X,
  Check,
  Save,
  ArrowLeft,
  Monitor,
  Server,
  Database,
  Cloud,
} from 'lucide-react';

// Section icons
const SECTION_ICONS: Record<Section, React.ReactNode> = {
  ui: <Monitor className="w-4 h-4" />,
  backend: <Server className="w-4 h-4" />,
  data: <Database className="w-4 h-4" />,
  infra: <Cloud className="w-4 h-4" />,
};

type Selection =
  | { type: 'group'; id: string }
  | { type: 'feature'; id: string; groupId: string }
  | { type: 'note'; id: string; featureId: string; section: Section }
  | null;

export default function NotesPage() {
  const [tree, setTree] = useState<GroupWithFeatures[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selection, setSelection] = useState<Selection>(null);
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(new Set());
  const [expandedFeatures, setExpandedFeatures] = useState<Set<string>>(new Set());

  // Inline creation state
  const [creatingGroup, setCreatingGroup] = useState(false);
  const [creatingFeatureGroupId, setCreatingFeatureGroupId] = useState<string | null>(null);
  const [newName, setNewName] = useState('');

  // Editor state
  const [editName, setEditName] = useState('');
  const [editNotes, setEditNotes] = useState('');
  const [activeSection, setActiveSection] = useState<Section>('ui');
  const [sectionNotes, setSectionNotes] = useState<FeatureNote[]>([]);
  const [loadingNotes, setLoadingNotes] = useState(false);
  const [selectedNote, setSelectedNote] = useState<FeatureNote | null>(null);
  const [editNoteTitle, setEditNoteTitle] = useState('');
  const [editNoteContent, setEditNoteContent] = useState('');
  const [saving, setSaving] = useState(false);

  // Auto-save timer
  const saveTimerRef = useRef<NodeJS.Timeout | null>(null);

  // Fetch tree
  const fetchTree = useCallback(async () => {
    try {
      setLoading(true);
      const data = await getNotesTree();
      setTree(data || []);
      setError(null);
    } catch (err: any) {
      setError(err.message || 'Failed to load notes');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchTree();
  }, [fetchTree]);

  // Fetch section notes when feature+section changes
  useEffect(() => {
    if (selection?.type === 'feature') {
      fetchSectionNotes(selection.id, activeSection);
    }
  }, [selection, activeSection]);

  const fetchSectionNotes = async (featureId: string, section: Section) => {
    try {
      setLoadingNotes(true);
      const notes = await listNotes(featureId, section);
      setSectionNotes(notes || []);
    } catch (err) {
      setSectionNotes([]);
    } finally {
      setLoadingNotes(false);
    }
  };

  // Toggle group expansion
  const toggleGroup = (groupId: string) => {
    setExpandedGroups((prev) => {
      const next = new Set(prev);
      if (next.has(groupId)) next.delete(groupId);
      else next.add(groupId);
      return next;
    });
  };

  // Toggle feature expansion (not used currently but could be useful)
  const toggleFeature = (featureId: string) => {
    setExpandedFeatures((prev) => {
      const next = new Set(prev);
      if (next.has(featureId)) next.delete(featureId);
      else next.add(featureId);
      return next;
    });
  };

  // Select group
  const selectGroup = (group: GroupWithFeatures) => {
    setSelection({ type: 'group', id: group.id });
    setEditName(group.name);
    setEditNotes(group.notes);
    setSelectedNote(null);
  };

  // Select feature
  const selectFeature = (feature: FeatureWithCounts, groupId: string) => {
    setSelection({ type: 'feature', id: feature.id, groupId });
    setEditName(feature.name);
    setEditNotes(feature.notes);
    setActiveSection('ui');
    setSelectedNote(null);
  };

  // Select note
  const selectNoteItem = (note: FeatureNote) => {
    setSelectedNote(note);
    setEditNoteTitle(note.title);
    setEditNoteContent(note.content);
  };

  // Back from note to feature
  const backToFeature = () => {
    setSelectedNote(null);
  };

  // Create group
  const handleCreateGroup = async () => {
    if (!newName.trim()) return;
    try {
      await createGroup(newName.trim());
      setCreatingGroup(false);
      setNewName('');
      fetchTree();
    } catch (err: any) {
      alert('Failed to create group: ' + err.message);
    }
  };

  // Create feature
  const handleCreateFeature = async (groupId: string) => {
    if (!newName.trim()) return;
    try {
      const feature = await createFeature(groupId, newName.trim());
      setCreatingFeatureGroupId(null);
      setNewName('');
      setExpandedGroups((prev) => {
        const next = new Set(prev);
        next.add(groupId);
        return next;
      });
      fetchTree();
    } catch (err: any) {
      alert('Failed to create feature: ' + err.message);
    }
  };

  // Save group/feature
  const handleSaveEntity = async () => {
    if (!selection) return;
    setSaving(true);
    try {
      if (selection.type === 'group') {
        await updateGroup(selection.id, { name: editName, notes: editNotes });
      } else if (selection.type === 'feature') {
        await updateFeature(selection.id, { name: editName, notes: editNotes });
      }
      fetchTree();
    } catch (err: any) {
      alert('Failed to save: ' + err.message);
    } finally {
      setSaving(false);
    }
  };

  // Delete group
  const handleDeleteGroup = async (groupId: string) => {
    if (!confirm('Delete this group and all its features and notes?')) return;
    try {
      await deleteGroup(groupId);
      if (selection?.type === 'group' && selection.id === groupId) setSelection(null);
      fetchTree();
    } catch (err: any) {
      alert('Failed to delete: ' + err.message);
    }
  };

  // Delete feature
  const handleDeleteFeature = async (featureId: string) => {
    if (!confirm('Delete this feature and all its notes?')) return;
    try {
      await deleteFeature(featureId);
      if (selection?.type === 'feature' && selection.id === featureId) setSelection(null);
      fetchTree();
    } catch (err: any) {
      alert('Failed to delete: ' + err.message);
    }
  };

  // Create note in section
  const handleCreateNote = async () => {
    if (selection?.type !== 'feature') return;
    try {
      const note = await createNote(selection.id, activeSection, 'Untitled', '');
      fetchSectionNotes(selection.id, activeSection);
      fetchTree();
      selectNoteItem(note);
    } catch (err: any) {
      alert('Failed to create note: ' + err.message);
    }
  };

  // Auto-save note (debounced)
  const autoSaveNote = useCallback(
    (title: string, content: string) => {
      if (!selectedNote) return;
      if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
      saveTimerRef.current = setTimeout(async () => {
        try {
          setSaving(true);
          await updateNote(selectedNote.id, { title, content });
          // Update local state
          setSectionNotes((prev) =>
            prev.map((n) => (n.id === selectedNote.id ? { ...n, title, content } : n))
          );
          fetchTree();
        } catch (err) {
          console.error('Auto-save failed', err);
        } finally {
          setSaving(false);
        }
      }, 1500);
    },
    [selectedNote]
  );

  // Handle note title change
  const handleNoteTitleChange = (value: string) => {
    setEditNoteTitle(value);
    autoSaveNote(value, editNoteContent);
  };

  // Handle note content change
  const handleNoteContentChange = (value: string) => {
    setEditNoteContent(value);
    autoSaveNote(editNoteTitle, value);
  };

  // Delete note
  const handleDeleteNote = async (noteId: string) => {
    if (!confirm('Delete this note?')) return;
    try {
      await deleteNote(noteId);
      if (selectedNote?.id === noteId) setSelectedNote(null);
      if (selection?.type === 'feature') {
        fetchSectionNotes(selection.id, activeSection);
      }
      fetchTree();
    } catch (err: any) {
      alert('Failed to delete note: ' + err.message);
    }
  };

  // Helper: find group/feature from tree by selection
  const getSelectedGroup = (): GroupWithFeatures | null => {
    if (selection?.type !== 'group') return null;
    return tree.find((g) => g.id === selection.id) || null;
  };

  const getSelectedFeature = (): FeatureWithCounts | null => {
    if (selection?.type !== 'feature') return null;
    for (const g of tree) {
      const f = g.features.find((f) => f.id === selection.id);
      if (f) return f;
    }
    return null;
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-ic-bg-primary flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="min-h-screen bg-ic-bg-primary flex items-center justify-center">
        <div className="bg-red-50 border border-red-200 rounded-lg p-6 max-w-md">
          <h2 className="text-red-800 font-semibold">Error</h2>
          <p className="text-red-600 mt-2">{error}</p>
          <button
            onClick={fetchTree}
            className="mt-4 px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700 transition"
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-ic-bg-primary">
      {/* Header */}
      <div className="bg-ic-surface border-b border-ic-border">
        <div className="max-w-[1600px] mx-auto px-6 py-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Layers className="w-6 h-6 text-blue-500" />
            <h1 className="text-xl font-bold text-ic-text-primary">Feature Notes</h1>
          </div>
          {saving && (
            <span className="text-sm text-ic-text-secondary flex items-center gap-1">
              <Save className="w-3 h-3 animate-pulse" /> Saving...
            </span>
          )}
        </div>
      </div>

      <div className="max-w-[1600px] mx-auto flex" style={{ height: 'calc(100vh - 65px)' }}>
        {/* ===== LEFT SIDEBAR ===== */}
        <div className="w-80 flex-shrink-0 bg-ic-surface border-r border-ic-border overflow-y-auto">
          <div className="p-4">
            {/* Add Group button */}
            <button
              onClick={() => {
                setCreatingGroup(true);
                setNewName('');
              }}
              className="w-full flex items-center gap-2 px-3 py-2 text-sm text-blue-600 bg-blue-50 hover:bg-blue-100 rounded-lg transition mb-4"
            >
              <Plus className="w-4 h-4" />
              New Group
            </button>

            {/* Inline group creation */}
            {creatingGroup && (
              <div className="flex items-center gap-1 mb-3 px-1">
                <input
                  autoFocus
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') handleCreateGroup();
                    if (e.key === 'Escape') setCreatingGroup(false);
                  }}
                  placeholder="Group name..."
                  className="flex-1 text-sm px-2 py-1 border border-ic-border rounded bg-ic-bg-primary text-ic-text-primary"
                />
                <button
                  onClick={handleCreateGroup}
                  className="p-1 text-green-600 hover:bg-green-50 rounded"
                >
                  <Check className="w-4 h-4" />
                </button>
                <button
                  onClick={() => setCreatingGroup(false)}
                  className="p-1 text-ic-text-secondary hover:bg-ic-bg-secondary rounded"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>
            )}

            {/* Tree */}
            {tree.length === 0 && !creatingGroup && (
              <p className="text-sm text-ic-text-secondary text-center py-8">
                No groups yet. Create one to get started.
              </p>
            )}

            {tree.map((group) => (
              <div key={group.id} className="mb-1">
                {/* Group row */}
                <div
                  className={`flex items-center gap-1 px-2 py-1.5 rounded-lg cursor-pointer group transition ${
                    selection?.type === 'group' && selection.id === group.id
                      ? 'bg-blue-50 text-blue-700'
                      : 'hover:bg-ic-bg-secondary text-ic-text-primary'
                  }`}
                >
                  <button onClick={() => toggleGroup(group.id)} className="p-0.5">
                    {expandedGroups.has(group.id) ? (
                      <ChevronDown className="w-4 h-4 text-ic-text-secondary" />
                    ) : (
                      <ChevronRight className="w-4 h-4 text-ic-text-secondary" />
                    )}
                  </button>
                  {expandedGroups.has(group.id) ? (
                    <FolderOpen className="w-4 h-4 text-yellow-500" />
                  ) : (
                    <Folder className="w-4 h-4 text-yellow-500" />
                  )}
                  <span
                    className="flex-1 text-sm font-medium truncate"
                    onClick={() => {
                      selectGroup(group);
                      if (!expandedGroups.has(group.id)) toggleGroup(group.id);
                    }}
                  >
                    {group.name}
                  </span>
                  <span className="text-xs text-ic-text-secondary opacity-0 group-hover:opacity-100">
                    {group.features.length}
                  </span>
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      setCreatingFeatureGroupId(group.id);
                      setNewName('');
                      if (!expandedGroups.has(group.id)) toggleGroup(group.id);
                    }}
                    className="p-0.5 opacity-0 group-hover:opacity-100 text-ic-text-secondary hover:text-blue-600"
                    title="Add feature"
                  >
                    <Plus className="w-3.5 h-3.5" />
                  </button>
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      handleDeleteGroup(group.id);
                    }}
                    className="p-0.5 opacity-0 group-hover:opacity-100 text-ic-text-secondary hover:text-red-600"
                    title="Delete group"
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                </div>

                {/* Expanded content */}
                {expandedGroups.has(group.id) && (
                  <div className="ml-5 mt-0.5">
                    {/* Inline feature creation */}
                    {creatingFeatureGroupId === group.id && (
                      <div className="flex items-center gap-1 mb-1 px-1">
                        <input
                          autoFocus
                          value={newName}
                          onChange={(e) => setNewName(e.target.value)}
                          onKeyDown={(e) => {
                            if (e.key === 'Enter') handleCreateFeature(group.id);
                            if (e.key === 'Escape') setCreatingFeatureGroupId(null);
                          }}
                          placeholder="Feature name..."
                          className="flex-1 text-sm px-2 py-1 border border-ic-border rounded bg-ic-bg-primary text-ic-text-primary"
                        />
                        <button
                          onClick={() => handleCreateFeature(group.id)}
                          className="p-1 text-green-600 hover:bg-green-50 rounded"
                        >
                          <Check className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => setCreatingFeatureGroupId(null)}
                          className="p-1 text-ic-text-secondary hover:bg-ic-bg-secondary rounded"
                        >
                          <X className="w-4 h-4" />
                        </button>
                      </div>
                    )}

                    {/* Features */}
                    {group.features.map((feature) => {
                      const totalNotes = Object.values(feature.note_counts).reduce(
                        (a, b) => a + b,
                        0
                      );
                      return (
                        <div
                          key={feature.id}
                          className={`flex items-center gap-2 px-2 py-1.5 rounded-lg cursor-pointer group transition ${
                            selection?.type === 'feature' && selection.id === feature.id
                              ? 'bg-blue-50 text-blue-700'
                              : 'hover:bg-ic-bg-secondary text-ic-text-primary'
                          }`}
                          onClick={() => selectFeature(feature, group.id)}
                        >
                          <Layers className="w-4 h-4 text-purple-500" />
                          <span className="flex-1 text-sm truncate">{feature.name}</span>
                          {totalNotes > 0 && (
                            <span className="text-xs text-ic-text-secondary bg-ic-bg-secondary px-1.5 py-0.5 rounded-full">
                              {totalNotes}
                            </span>
                          )}
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              handleDeleteFeature(feature.id);
                            }}
                            className="p-0.5 opacity-0 group-hover:opacity-100 text-ic-text-secondary hover:text-red-600"
                            title="Delete feature"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        </div>
                      );
                    })}

                    {group.features.length === 0 && !creatingFeatureGroupId && (
                      <p className="text-xs text-ic-text-secondary px-2 py-2">No features yet</p>
                    )}
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>

        {/* ===== RIGHT PANEL ===== */}
        <div className="flex-1 overflow-y-auto">
          {!selection && (
            <div className="flex items-center justify-center h-full text-ic-text-secondary">
              <div className="text-center">
                <Layers className="w-12 h-12 mx-auto mb-4 opacity-30" />
                <p className="text-lg">Select a group or feature to get started</p>
                <p className="text-sm mt-1">Or create a new group from the sidebar</p>
              </div>
            </div>
          )}

          {/* Group Editor */}
          {selection?.type === 'group' && (
            <div className="p-8 max-w-3xl">
              <div className="flex items-center gap-3 mb-6">
                <Folder className="w-6 h-6 text-yellow-500" />
                <h2 className="text-lg font-semibold text-ic-text-primary">Group</h2>
              </div>

              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-ic-text-secondary mb-1">
                    Name
                  </label>
                  <input
                    value={editName}
                    onChange={(e) => setEditName(e.target.value)}
                    className="w-full px-3 py-2 border border-ic-border rounded-lg bg-ic-bg-primary text-ic-text-primary focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-ic-text-secondary mb-1">
                    Notes
                  </label>
                  <textarea
                    value={editNotes}
                    onChange={(e) => setEditNotes(e.target.value)}
                    rows={8}
                    className="w-full px-3 py-2 border border-ic-border rounded-lg bg-ic-bg-primary text-ic-text-primary focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-y font-mono text-sm"
                    placeholder="Add notes about this group..."
                  />
                </div>
                <button
                  onClick={handleSaveEntity}
                  disabled={saving}
                  className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition disabled:opacity-50"
                >
                  <Save className="w-4 h-4" />
                  {saving ? 'Saving...' : 'Save'}
                </button>
              </div>
            </div>
          )}

          {/* Feature Editor */}
          {selection?.type === 'feature' && !selectedNote && (
            <div className="p-8">
              <div className="max-w-3xl">
                <div className="flex items-center gap-3 mb-6">
                  <Layers className="w-6 h-6 text-purple-500" />
                  <h2 className="text-lg font-semibold text-ic-text-primary">Feature</h2>
                </div>

                <div className="space-y-4 mb-8">
                  <div>
                    <label className="block text-sm font-medium text-ic-text-secondary mb-1">
                      Name
                    </label>
                    <input
                      value={editName}
                      onChange={(e) => setEditName(e.target.value)}
                      className="w-full px-3 py-2 border border-ic-border rounded-lg bg-ic-bg-primary text-ic-text-primary focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-ic-text-secondary mb-1">
                      Notes
                    </label>
                    <textarea
                      value={editNotes}
                      onChange={(e) => setEditNotes(e.target.value)}
                      rows={4}
                      className="w-full px-3 py-2 border border-ic-border rounded-lg bg-ic-bg-primary text-ic-text-primary focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-y font-mono text-sm"
                      placeholder="Add notes about this feature..."
                    />
                  </div>
                  <button
                    onClick={handleSaveEntity}
                    disabled={saving}
                    className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition disabled:opacity-50"
                  >
                    <Save className="w-4 h-4" />
                    {saving ? 'Saving...' : 'Save'}
                  </button>
                </div>
              </div>

              {/* Section Tabs */}
              <div className="border-b border-ic-border mb-6">
                <div className="flex gap-1">
                  {SECTIONS.map((sec) => {
                    const feature = getSelectedFeature();
                    const count = feature?.note_counts[sec] || 0;
                    return (
                      <button
                        key={sec}
                        onClick={() => setActiveSection(sec)}
                        className={`flex items-center gap-2 px-4 py-2.5 text-sm font-medium border-b-2 transition ${
                          activeSection === sec
                            ? 'border-blue-500 text-blue-600'
                            : 'border-transparent text-ic-text-secondary hover:text-ic-text-primary hover:border-ic-border'
                        }`}
                      >
                        {SECTION_ICONS[sec]}
                        {SECTION_LABELS[sec]}
                        {count > 0 && (
                          <span className="text-xs bg-ic-bg-secondary px-1.5 py-0.5 rounded-full">
                            {count}
                          </span>
                        )}
                      </button>
                    );
                  })}
                </div>
              </div>

              {/* Section Notes List */}
              <div className="max-w-3xl">
                <div className="flex items-center justify-between mb-4">
                  <h3 className="text-sm font-medium text-ic-text-secondary">
                    {SECTION_LABELS[activeSection]} Notes
                  </h3>
                  <button
                    onClick={handleCreateNote}
                    className="flex items-center gap-1 px-3 py-1.5 text-sm text-blue-600 bg-blue-50 hover:bg-blue-100 rounded-lg transition"
                  >
                    <Plus className="w-3.5 h-3.5" />
                    Add Note
                  </button>
                </div>

                {loadingNotes ? (
                  <div className="flex justify-center py-8">
                    <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-500"></div>
                  </div>
                ) : sectionNotes.length === 0 ? (
                  <div className="text-center py-8 text-ic-text-secondary">
                    <FileText className="w-8 h-8 mx-auto mb-2 opacity-30" />
                    <p className="text-sm">No notes in {SECTION_LABELS[activeSection]} yet</p>
                  </div>
                ) : (
                  <div className="space-y-2">
                    {sectionNotes.map((note) => (
                      <div
                        key={note.id}
                        className="flex items-center gap-3 px-4 py-3 bg-ic-surface border border-ic-border rounded-lg cursor-pointer hover:border-blue-300 transition group"
                        onClick={() => selectNoteItem(note)}
                      >
                        <FileText className="w-4 h-4 text-ic-text-secondary flex-shrink-0" />
                        <div className="flex-1 min-w-0">
                          <p className="text-sm font-medium text-ic-text-primary truncate">
                            {note.title || 'Untitled'}
                          </p>
                          {note.content && (
                            <p className="text-xs text-ic-text-secondary truncate mt-0.5">
                              {note.content.substring(0, 100)}
                            </p>
                          )}
                        </div>
                        <span className="text-xs text-ic-text-secondary flex-shrink-0">
                          {new Date(note.updated_at).toLocaleDateString()}
                        </span>
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            handleDeleteNote(note.id);
                          }}
                          className="p-1 opacity-0 group-hover:opacity-100 text-ic-text-secondary hover:text-red-600 transition"
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Note Editor */}
          {selection?.type === 'feature' && selectedNote && (
            <div className="p-8 max-w-3xl">
              <button
                onClick={backToFeature}
                className="flex items-center gap-1 text-sm text-ic-text-secondary hover:text-ic-text-primary mb-4 transition"
              >
                <ArrowLeft className="w-4 h-4" />
                Back to {SECTION_LABELS[activeSection]}
              </button>

              <div className="space-y-4">
                <input
                  value={editNoteTitle}
                  onChange={(e) => handleNoteTitleChange(e.target.value)}
                  className="w-full text-xl font-semibold px-0 py-2 bg-transparent text-ic-text-primary border-0 border-b-2 border-ic-border focus:border-blue-500 focus:outline-none focus:ring-0"
                  placeholder="Note title..."
                />
                <textarea
                  value={editNoteContent}
                  onChange={(e) => handleNoteContentChange(e.target.value)}
                  rows={20}
                  className="w-full px-3 py-3 border border-ic-border rounded-lg bg-ic-bg-primary text-ic-text-primary focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-y font-mono text-sm leading-relaxed"
                  placeholder="Write your notes here..."
                />
                <div className="flex items-center justify-between text-xs text-ic-text-secondary">
                  <span>Last updated: {new Date(selectedNote.updated_at).toLocaleString()}</span>
                  <span className="flex items-center gap-1">
                    {saving ? (
                      <>
                        <Save className="w-3 h-3 animate-pulse" /> Saving...
                      </>
                    ) : (
                      'Auto-saved'
                    )}
                  </span>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
