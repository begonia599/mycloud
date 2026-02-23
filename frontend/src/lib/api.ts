import axios from 'axios';

const api = axios.create({
  baseURL: '/api',
});

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

api.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response?.status === 401 && window.location.pathname !== '/login') {
      localStorage.removeItem('token');
      window.location.href = '/login';
    }
    return Promise.reject(err);
  }
);

export interface FileItem {
  id: string;
  name: string;
  stored_name: string;
  size: number;
  mime_type: string;
  created_at: string;
}

export interface ShareItem {
  id: string;
  code: string;
  title: string;
  has_password: boolean;
  expires_at: string | null;
  created_at: string;
  files: FileItem[];
}

export const authApi = {
  login: (username: string, password: string) =>
    api.post<{ token: string }>('/auth/login', { username, password }),
};

export const fileApi = {
  upload: (files: File[]) => {
    const form = new FormData();
    files.forEach((f) => form.append('files', f));
    return api.post<{ files: FileItem[] }>('/files/upload', form);
  },
  list: () => api.get<{ files: FileItem[] }>('/files'),
  delete: (id: string) => api.delete(`/files/${id}`),
};

export const shareApi = {
  create: (data: {
    title?: string;
    password?: string;
    file_ids: string[];
    expires_in?: number;
  }) => api.post<{ share: ShareItem }>('/shares', data),
  list: () => api.get<{ shares: ShareItem[] }>('/shares'),
  delete: (id: string) => api.delete(`/shares/${id}`),
};

export const publicApi = {
  getInfo: (code: string) =>
    api.get<{ title: string; has_password: boolean; files?: FileItem[] }>(
      `/s/${code}`
    ),
  verify: (code: string, password: string) =>
    api.post<{ files: FileItem[] }>(`/s/${code}/verify`, { password }),
  downloadUrl: (code: string, fileId: string, pwd?: string) =>
    `/api/s/${code}/download/${fileId}${pwd ? `?pwd=${encodeURIComponent(pwd)}` : ''}`,
};

export default api;
