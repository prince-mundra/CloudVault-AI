// frontend/src/lib/api.ts

export interface AuthTokens {
  token: string;
  salt: string;
  userId?: string;
  isAnonymous?: boolean;
}

export interface User {
  id: string;
  name: string;
  email: string;
  role: string;
  is_anonymous: boolean;
}

export interface FileItem {
  id: string;
  name: string;
  size: number;
  content_type: string;
  sha256?: string;
  owner_id: string;
  folder_id?: string | null;
  created_at: string;
  updated_at: string;
}

export interface FolderItem {
  id: string;
  name: string;
  owner_id: string;
  parent_id?: string | null;
  created_at: string;
  updated_at: string;
}

export interface FileTypeUsage {
  content_type: string;
  file_count: number;
  size_bytes: number;
}

export interface FileActivity {
  id: number;
  actor_id?: string;
  file_id?: string;
  file_name?: string;
  action: string;
  size_bytes: number;
  created_at: string;
}

export interface StorageAnalytics {
  total_files: number;
  used_bytes: number;
  reserved_bytes: number;
  quota_bytes: number;
  remaining_bytes: number;
  file_types: FileTypeUsage[];
  recent_uploads: FileItem[];
  recent_activity: FileActivity[];
}

export interface ApiResponse<T = any> {
  data?: T;
  error?: string;
  message?: string;
  status?: number;
}

const STORAGE_KEY_TOKEN = 'cloudvault_auth_token';
const STORAGE_KEY_SALT = 'cloudvault_auth_salt';
const STORAGE_KEY_USER = 'cloudvault_auth_user';

export const getStoredTokens = (): AuthTokens | null => {
  const token = localStorage.getItem(STORAGE_KEY_TOKEN);
  const salt = localStorage.getItem(STORAGE_KEY_SALT);
  if (!token || !salt) return null;
  return { token, salt };
};

export const setStoredTokens = (token: string, salt: string) => {
  localStorage.setItem(STORAGE_KEY_TOKEN, token);
  localStorage.setItem(STORAGE_KEY_SALT, salt);
};

export const clearStoredTokens = () => {
  localStorage.removeItem(STORAGE_KEY_TOKEN);
  localStorage.removeItem(STORAGE_KEY_SALT);
  localStorage.removeItem(STORAGE_KEY_USER);
};

export const getStoredUser = (): User | null => {
  const raw = localStorage.getItem(STORAGE_KEY_USER);
  if (!raw) return null;
  try {
    return JSON.parse(raw);
  } catch {
    return null;
  }
};

export const setStoredUser = (user: User) => {
  localStorage.setItem(STORAGE_KEY_USER, JSON.stringify(user));
};

export interface ApiRequestOptions extends Omit<RequestInit, 'body'> {
  body?: any;
}

// Base Fetch Wrapper
async function request<T = any>(
  endpoint: string,
  options: ApiRequestOptions = {}
): Promise<T> {
  const tokens = getStoredTokens();
  const headers = new Headers(options.headers || {});

  let reqBody = options.body;
  if (reqBody && typeof reqBody === 'object' && !(reqBody instanceof FormData) && !(reqBody instanceof Blob)) {
    reqBody = JSON.stringify(reqBody);
  }

  if (!headers.has('Content-Type') && !(reqBody instanceof FormData)) {
    headers.set('Content-Type', 'application/json');
  }

  if (tokens?.token && tokens?.salt) {
    headers.set('Authorization', `Bearer ${tokens.token}`);
    headers.set('X-Salt', tokens.salt);
  }

  // Prepend /api/v1 if not starting with /api or /ai
  const url = endpoint.startsWith('/') ? endpoint : `/api/v1/${endpoint}`;

  const response = await fetch(url, {
    ...options,
    body: reqBody,
    headers,
  });

  if (response.status === 401) {
    // If unauthorized, clear tokens if we are not logging in or registering
    if (!endpoint.includes('/auth/oauth/login') && !endpoint.includes('/auth/anonymous/register')) {
      clearStoredTokens();
      window.dispatchEvent(new Event('cloudvault-unauthorized'));
    }
  }

  const contentType = response.headers.get('content-type');
  if (contentType && contentType.includes('application/json')) {
    const data = await response.json();
    if (!response.ok) {
      let errorMsg = `Request failed with status ${response.status}`;
      if (typeof data.error === 'string') {
        errorMsg = data.error;
      } else if (data.error && typeof data.error === 'object' && data.error.message) {
        errorMsg = data.error.message;
      } else if (data.message) {
        errorMsg = data.message;
      } else if (data.detail) {
        errorMsg = typeof data.detail === 'string' ? data.detail : JSON.stringify(data.detail);
      }
      throw new Error(errorMsg);
    }
    return data;
  }

  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || `Request failed with status ${response.status}`);
  }

  return response as unknown as T;
}

export const api = {
  get: <T = any>(endpoint: string) => request<T>(endpoint, { method: 'GET' }),
  post: <T = any>(endpoint: string, body?: any) =>
    request<T>(endpoint, {
      method: 'POST',
      body: body instanceof FormData ? body : JSON.stringify(body),
    }),
  put: <T = any>(endpoint: string, body?: any) =>
    request<T>(endpoint, {
      method: 'PUT',
      body: body instanceof FormData ? body : JSON.stringify(body),
    }),
  patch: <T = any>(endpoint: string, body?: any) =>
    request<T>(endpoint, {
      method: 'PATCH',
      body: body instanceof FormData ? body : JSON.stringify(body),
    }),
  delete: <T = any>(endpoint: string) =>
    request<T>(endpoint, { method: 'DELETE' }),

  // Auth Endpoints
  login: async (email: string, password: string) => {
    const res = await request<{ user: User; token: string; salt: string }>(
      '/api/v1/auth/oauth/login',
      {
        method: 'POST',
        body: JSON.stringify({ email, password }),
      }
    );
    setStoredTokens(res.token, res.salt);
    setStoredUser(res.user);
    return res;
  },

  register: async (username: string, email: string, password: string) => {
    const res = await request<{ user: User; token: string; salt: string }>(
      '/api/v1/auth/oauth/register',
      {
        method: 'POST',
        body: JSON.stringify({ username, email, password }),
      }
    );
    setStoredTokens(res.token, res.salt);
    setStoredUser(res.user);
    return res;
  },

  registerAnonymous: async () => {
    const res = await request<{ anonymous_id: string; token: string; salt: string }>(
      '/api/v1/auth/anonymous/register',
      {
        method: 'POST',
      }
    );
    setStoredTokens(res.token, res.salt);
    const guestUser: User = {
      id: res.anonymous_id,
      name: 'Guest User',
      email: `anon_${res.anonymous_id.substring(0, 8)}@anonymous.cloudvault.internal`,
      role: 'anonymous',
      is_anonymous: true,
    };
    setStoredUser(guestUser);
    return { ...res, user: guestUser };
  },

  logout: async () => {
    try {
      await request('/api/v1/auth/logout', { method: 'POST' });
    } finally {
      clearStoredTokens();
    }
  },

  validateToken: async () => {
    return request<{ subject: string; valid: boolean }>('/api/v1/auth/validate', {
      method: 'POST',
    });
  },

  // Storage & Analytics Endpoints
  getStorageAnalytics: async () => {
    return request<{ data: StorageAnalytics }>('/api/v1/analytics/storage');
  },

  // File Management Endpoints
  listFiles: async (params?: {
    folder_id?: string;
    limit?: number;
    offset?: number;
    search?: string;
    sort?: string;
    order?: string;
  }) => {
    const q = new URLSearchParams();
    if (params?.folder_id !== undefined) q.set('folder_id', params.folder_id);
    if (params?.limit) q.set('limit', String(params.limit));
    if (params?.offset !== undefined) q.set('offset', String(params.offset));
    if (params?.search) q.set('search', params.search);
    if (params?.sort) q.set('sort', params.sort);
    if (params?.order) q.set('order', params.order);
    const qs = q.toString() ? `?${q.toString()}` : '';
    return request<{ data: FileItem[]; pagination: { limit: number; offset: number } }>(`/api/v1/files${qs}`);
  },

  getFile: async (id: string) => {
    return request<{ data: FileItem }>(`/api/v1/files/${id}`);
  },

  deleteFile: async (id: string) => {
    return request<{ message: string }>(`/api/v1/files/${id}`, { method: 'DELETE' });
  },

  renameFile: async (id: string, name: string) => {
    return request<{ data: FileItem }>(`/api/v1/files/${id}`, {
      method: 'PUT',
      body: { name },
    });
  },

  moveFile: async (id: string, folderId: string | null) => {
    return request<{ data: FileItem }>(`/api/v1/files/${id}/folder`, {
      method: 'PATCH',
      body: { folder_id: folderId },
    });
  },

  uploadFile: (
    file: File,
    folderId?: string | null,
    onProgress?: (percent: number) => void
  ): Promise<{ data: FileItem; deduplicated?: boolean }> => {
    return new Promise((resolve, reject) => {
      const tokens = getStoredTokens();
      const xhr = new XMLHttpRequest();
      xhr.open('POST', '/api/v1/files');

      if (tokens?.token && tokens?.salt) {
        xhr.setRequestHeader('Authorization', `Bearer ${tokens.token}`);
        xhr.setRequestHeader('X-Salt', tokens.salt);
      }

      if (xhr.upload && onProgress) {
        xhr.upload.onprogress = (e) => {
          if (e.lengthComputable) {
            const percent = Math.round((e.loaded / e.total) * 100);
            onProgress(percent);
          }
        };
      }

      xhr.onload = async () => {
        try {
          const contentType = xhr.getResponseHeader('content-type') || '';
          let data: any = null;
          if (contentType.includes('application/json')) {
            data = JSON.parse(xhr.responseText);
          }
          if (xhr.status >= 200 && xhr.status < 300) {
            const fileItem: FileItem = data?.data || data;
            // If uploaded into a specific folder, move it into that folder
            if (folderId && fileItem?.id) {
              try {
                const moved = await api.moveFile(fileItem.id, folderId);
                resolve({ data: moved.data, deduplicated: data?.deduplicated });
                return;
              } catch {
                // Return file even if folder move had an error
                resolve({ data: fileItem, deduplicated: data?.deduplicated });
                return;
              }
            }
            resolve({ data: fileItem, deduplicated: data?.deduplicated });
          } else {
            const errorMsg =
              data?.error?.message ||
              (typeof data?.error === 'string' ? data.error : null) ||
              data?.message ||
              data?.detail ||
              `Upload failed with status ${xhr.status}`;
            reject(new Error(errorMsg));
          }
        } catch (err: any) {
          reject(new Error(err.message || 'Failed to parse upload response'));
        }
      };

      xhr.onerror = () => {
        reject(new Error('Network error during file upload'));
      };

      const formData = new FormData();
      formData.append('file', file);
      xhr.send(formData);
    });
  },

  downloadFile: async (id: string, filename: string): Promise<void> => {
    const tokens = getStoredTokens();
    const headers = new Headers();
    if (tokens?.token && tokens?.salt) {
      headers.set('Authorization', `Bearer ${tokens.token}`);
      headers.set('X-Salt', tokens.salt);
    }

    const response = await fetch(`/api/v1/files/${id}/download`, {
      method: 'GET',
      headers,
    });

    if (!response.ok) {
      throw new Error(`Download failed with status ${response.status}`);
    }

    const blob = await response.blob();
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename || 'download';
    document.body.appendChild(a);
    a.click();
    window.URL.revokeObjectURL(url);
    document.body.removeChild(a);
  },

  // Folder Management Endpoints
  listFolders: async (parentId?: string | null) => {
    const qs = parentId ? `?parent_id=${encodeURIComponent(parentId)}` : '';
    return request<{ data: FolderItem[] }>(`/api/v1/folders${qs}`);
  },

  createFolder: async (name: string, parentId?: string | null) => {
    return request<{ data: FolderItem }>('/api/v1/folders', {
      method: 'POST',
      body: { name, parent_id: parentId || null },
    });
  },

  deleteFolder: async (id: string) => {
    return request<{ message?: string }>(`/api/v1/folders/${id}`, {
      method: 'DELETE',
    });
  },

  updateFolder: async (id: string, name: string) => {
    return request<{ data: FolderItem }>(`/api/v1/folders/${id}`, {
      method: 'PATCH',
      body: { name },
    });
  },
};
