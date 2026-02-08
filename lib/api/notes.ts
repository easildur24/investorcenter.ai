import { apiClient } from './client';

// Types
export type Section = 'ui' | 'backend' | 'data' | 'infra';

export const SECTIONS: Section[] = ['ui', 'backend', 'data', 'infra'];

export const SECTION_LABELS: Record<Section, string> = {
  ui: 'UI',
  backend: 'Backend',
  data: 'Data',
  infra: 'Infra',
};

export interface FeatureGroup {
  id: string;
  name: string;
  notes: string;
  sort_order: number;
  created_at: string;
  updated_at: string;
}

export interface Feature {
  id: string;
  group_id: string;
  name: string;
  notes: string;
  sort_order: number;
  created_at: string;
  updated_at: string;
}

export interface FeatureNote {
  id: string;
  feature_id: string;
  section: Section;
  title: string;
  content: string;
  sort_order: number;
  created_at: string;
  updated_at: string;
}

export interface FeatureWithCounts extends Feature {
  note_counts: Record<Section, number>;
}

export interface GroupWithFeatures extends FeatureGroup {
  features: FeatureWithCounts[];
}

interface ApiResponse<T> {
  success: boolean;
  data: T;
  message?: string;
}

// API functions

const BASE = '/admin/notes';

// Tree
export async function getNotesTree(): Promise<GroupWithFeatures[]> {
  const res = await apiClient.get<ApiResponse<GroupWithFeatures[]>>(`${BASE}/tree`);
  return res.data;
}

// Groups
export async function listGroups(): Promise<FeatureGroup[]> {
  const res = await apiClient.get<ApiResponse<FeatureGroup[]>>(`${BASE}/groups`);
  return res.data;
}

export async function createGroup(name: string, notes: string = ''): Promise<FeatureGroup> {
  const res = await apiClient.post<ApiResponse<FeatureGroup>>(`${BASE}/groups`, { name, notes });
  return res.data;
}

export async function updateGroup(id: string, data: { name?: string; notes?: string; sort_order?: number }): Promise<FeatureGroup> {
  const res = await apiClient.put<ApiResponse<FeatureGroup>>(`${BASE}/groups/${id}`, data);
  return res.data;
}

export async function deleteGroup(id: string): Promise<void> {
  await apiClient.delete(`${BASE}/groups/${id}`);
}

// Features
export async function listFeatures(groupId: string): Promise<Feature[]> {
  const res = await apiClient.get<ApiResponse<Feature[]>>(`${BASE}/groups/${groupId}/features`);
  return res.data;
}

export async function createFeature(groupId: string, name: string, notes: string = ''): Promise<Feature> {
  const res = await apiClient.post<ApiResponse<Feature>>(`${BASE}/groups/${groupId}/features`, { name, notes });
  return res.data;
}

export async function updateFeature(id: string, data: { name?: string; notes?: string; sort_order?: number }): Promise<Feature> {
  const res = await apiClient.put<ApiResponse<Feature>>(`${BASE}/features/${id}`, data);
  return res.data;
}

export async function deleteFeature(id: string): Promise<void> {
  await apiClient.delete(`${BASE}/features/${id}`);
}

// Notes
export async function listNotes(featureId: string, section?: Section): Promise<FeatureNote[]> {
  const query = section ? `?section=${section}` : '';
  const res = await apiClient.get<ApiResponse<FeatureNote[]>>(`${BASE}/features/${featureId}/notes${query}`);
  return res.data;
}

export async function createNote(featureId: string, section: Section, title: string = 'Untitled', content: string = ''): Promise<FeatureNote> {
  const res = await apiClient.post<ApiResponse<FeatureNote>>(`${BASE}/features/${featureId}/notes`, { section, title, content });
  return res.data;
}

export async function updateNote(id: string, data: { title?: string; content?: string; sort_order?: number }): Promise<FeatureNote> {
  const res = await apiClient.put<ApiResponse<FeatureNote>>(`${BASE}/notes/${id}`, data);
  return res.data;
}

export async function deleteNote(id: string): Promise<void> {
  await apiClient.delete(`${BASE}/notes/${id}`);
}
