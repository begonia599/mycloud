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

// Chunked upload types and API
export interface UploadProgress {
  phase: 'uploading' | 'merging' | 'done' | 'error';
  percent: number; // 0-100
  uploadedChunks: number;
  totalChunks: number;
  error?: string;
}

export const chunkedUploadApi = {
  init: (fileName: string, fileSize: number, chunkSize?: number) =>
    api.post<{ upload_id: string; chunk_size: number; total_chunks: number }>(
      '/files/upload/init',
      { file_name: fileName, file_size: fileSize, chunk_size: chunkSize }
    ),
  uploadChunk: (
    uploadId: string,
    chunkIndex: number,
    chunk: Blob,
    onProgress?: (loaded: number, total: number) => void
  ) => {
    const form = new FormData();
    form.append('upload_id', uploadId);
    form.append('chunk_index', String(chunkIndex));
    form.append('chunk', chunk);
    return api.post<{ chunk_index: number }>('/files/upload/chunk', form, {
      onUploadProgress: (e) => {
        if (onProgress && e.total) onProgress(e.loaded, e.total);
      },
    });
  },
  getStatus: (uploadId: string) =>
    api.get<{
      upload_id: string;
      file_name: string;
      file_size: number;
      chunk_size: number;
      total_chunks: number;
      uploaded: number[];
    }>('/files/upload/status', { params: { upload_id: uploadId } }),
  complete: (uploadId: string) =>
    api.post<{ file: FileItem }>('/files/upload/complete', { upload_id: uploadId }),
};

const CHUNK_SIZE = 4 * 1024 * 1024; // 4MB

export async function uploadFileInChunks(
  file: File,
  onProgress?: (progress: UploadProgress) => void,
  signal?: AbortSignal,
  resumeUploadId?: string
): Promise<FileItem> {
  let uploadId: string;
  let totalChunks: number;
  let chunkSize: number;
  let uploadedSet: Set<number>;

  if (resumeUploadId) {
    // Resume existing upload
    const { data } = await chunkedUploadApi.getStatus(resumeUploadId);
    uploadId = data.upload_id;
    totalChunks = data.total_chunks;
    chunkSize = data.chunk_size;
    uploadedSet = new Set(data.uploaded || []);
  } else {
    // Start new upload
    const { data } = await chunkedUploadApi.init(file.name, file.size, CHUNK_SIZE);
    uploadId = data.upload_id;
    totalChunks = data.total_chunks;
    chunkSize = data.chunk_size;
    uploadedSet = new Set<number>();
  }

  // Upload missing chunks sequentially
  for (let i = 0; i < totalChunks; i++) {
    if (signal?.aborted) throw new DOMException('Upload aborted', 'AbortError');
    if (uploadedSet.has(i)) continue;

    const start = i * chunkSize;
    const end = Math.min(start + chunkSize, file.size);
    const chunk = file.slice(start, end);

    await chunkedUploadApi.uploadChunk(uploadId, i, chunk, (_loaded, _total) => {
      // Calculate overall progress: completed chunks + fraction of current chunk
      const completedChunks = [...uploadedSet].filter((idx) => idx < i).length + i - [...uploadedSet].filter((idx) => idx < i).length;
      const chunkFraction = _loaded / _total;
      const percent = Math.round(((completedChunks + chunkFraction) / totalChunks) * 100);
      onProgress?.({
        phase: 'uploading',
        percent: Math.min(percent, 99),
        uploadedChunks: completedChunks,
        totalChunks,
      });
    });

    uploadedSet.add(i);
    const doneCount = uploadedSet.size;
    onProgress?.({
      phase: 'uploading',
      percent: Math.min(Math.round((doneCount / totalChunks) * 100), 99),
      uploadedChunks: doneCount,
      totalChunks,
    });
  }

  // Merge phase
  onProgress?.({ phase: 'merging', percent: 99, uploadedChunks: totalChunks, totalChunks });
  const { data: result } = await chunkedUploadApi.complete(uploadId);
  onProgress?.({ phase: 'done', percent: 100, uploadedChunks: totalChunks, totalChunks });
  return result.file;
}

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
