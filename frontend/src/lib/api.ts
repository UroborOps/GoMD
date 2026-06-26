const BASE = '';

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${url}`, {
    headers: { 'Content-Type': 'application/json', ...options?.headers },
    ...options,
  });
  if (!res.ok) throw new Error(`API error: ${res.status} ${res.statusText}`);
  return res.json();
}

export interface TreeNode {
  name: string;
  path: string;
  type: 'folder' | 'file';
  size?: number;
  lines?: number;
  chars?: number;
  modTime?: string;
  children?: Record<string, TreeNode>;
}

export const api = {
  // Tree
  getTree: () => request<TreeNode>('/api/tree'),

  // File CRUD
  listFiles: () => request<{ files: string[]; folders: string[]; count: number }>('/api/files'),
  getFile: (path: string) =>
    request<{ path: string; content: string; frontmatter: Record<string, unknown>; title: string; tags: string[] }>(
      `/api/files/${encodeURIComponent(path)}`
    ),
  createFile: (path: string, content: string) =>
    request<{ path: string }>('/api/files', {
      method: 'POST',
      body: JSON.stringify({ path, content }),
    }),
  updateFile: (path: string, content: string) =>
    request<{ path: string; updated: string }>(`/api/files/${encodeURIComponent(path)}`, {
      method: 'PUT',
      body: JSON.stringify({ content }),
    }),
  deleteFile: (path: string) =>
    request<{ path: string; deleted: string }>(`/api/files/${encodeURIComponent(path)}`, {
      method: 'DELETE',
    }),

  // Directories
  listFolders: (dir: string) =>
    request<{ directories: string[]; files: string[] }>(`/api/folders/${encodeURIComponent(dir)}`),
  createFolder: (path: string) =>
    request<{ path: string }>('/api/folders', {
      method: 'POST',
      body: JSON.stringify({ path }),
    }),
  deleteFolder: (path: string) =>
    request<{ path: string; deleted: string }>(`/api/folders/${encodeURIComponent(path)}`, {
      method: 'DELETE',
    }),
  
  // Rename
  renameNode: (oldPath: string, newPath: string) =>
    request<{ oldPath: string; newPath: string; renamed: string }>('/api/rename', {
      method: 'POST',
      body: JSON.stringify({ oldPath, newPath }),
    }),

  // Transfer
  upload: (pathPrefix: string, files: FileList | File[]) => {
    const formData = new FormData();
    formData.append('path', pathPrefix);
    Array.from(files).forEach(f => {
      formData.append('files', f);
      formData.append('paths', (f as any).webkitRelativePath || f.name);
    });
    return fetch('/api/upload', {
      method: 'POST',
      body: formData,
    }).then(res => {
      if (!res.ok) throw new Error('Upload failed');
      return res.json();
    });
  },

  // Search
  search: (q: string, type: 'lexical' | 'semantic' = 'lexical', limit = 20) =>
    request<{ query: string; results: { path: string; title: string; content: string }[]; count: number }>(
      `/api/search?q=${encodeURIComponent(q)}&limit=${limit}&type=${type}`
    ),

  // Backlinks
  getBacklinks: (path: string) =>
    request<{ path: string; backlinks: { file: string; heading?: string; alias?: string }[]; count: number }>(
      `/api/backlinks/${encodeURIComponent(path)}`
    ),

  // Graph
  getGraph: () => request<{ nodes: { id: string; label: string; depth: number }[]; edges: { source: string; target: string }[] }>('/api/graph'),

  // Config
  getConfig: () =>
    request<{ vault_path: string; port: number; host: string; theme: string; rag_enabled: boolean; qdrant_url: string; git_enabled: boolean; git_remote: string; s3_backup_enabled: boolean; s3_endpoint: string }>('/api/config'),

  // Locks
  getLocks: () => request<Record<string, boolean>>('/api/locks'),
  setLock: (path: string, locked: boolean) => request<void>('/api/locks', {
    method: 'POST',
    body: JSON.stringify({ path, locked }),
  }),
};
