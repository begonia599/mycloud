import { useState, useRef, useCallback } from 'react';
import { uploadFileInChunks, type UploadProgress } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Upload, CheckCircle2, XCircle, Loader2, RotateCcw } from 'lucide-react';

interface FileUploadProps {
  onUploaded: () => void;
}

interface FileTask {
  id: string;
  file: File;
  progress: UploadProgress;
  uploadId?: string;
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B';
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
  if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
  return (bytes / (1024 * 1024 * 1024)).toFixed(2) + ' GB';
}

export default function FileUpload({ onUploaded }: FileUploadProps) {
  const [tasks, setTasks] = useState<FileTask[]>([]);
  const [dragActive, setDragActive] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  const updateTask = useCallback((id: string, update: Partial<FileTask>) => {
    setTasks((prev) => prev.map((t) => (t.id === id ? { ...t, ...update } : t)));
  }, []);

  const startUpload = useCallback(
    async (task: FileTask) => {
      try {
        updateTask(task.id, {
          progress: { phase: 'uploading', percent: 0, uploadedChunks: 0, totalChunks: 0 },
        });
        await uploadFileInChunks(
          task.file,
          (progress) => updateTask(task.id, { progress }),
          undefined,
          task.uploadId
        );
        onUploaded();
      } catch (err) {
        const message = err instanceof Error ? err.message : '上传失败';
        updateTask(task.id, {
          progress: { phase: 'error', percent: 0, uploadedChunks: 0, totalChunks: 0, error: message },
        });
      }
    },
    [updateTask, onUploaded]
  );

  const handleFiles = useCallback(
    (fileList: FileList | null) => {
      if (!fileList || fileList.length === 0) return;
      const newTasks: FileTask[] = Array.from(fileList).map((file) => ({
        id: crypto.randomUUID(),
        file,
        progress: { phase: 'uploading' as const, percent: 0, uploadedChunks: 0, totalChunks: 0 },
      }));
      setTasks((prev) => [...newTasks, ...prev]);
      newTasks.forEach((task) => startUpload(task));
    },
    [startUpload]
  );

  const retryTask = useCallback(
    (task: FileTask) => {
      startUpload(task);
    },
    [startUpload]
  );

  const removeTask = useCallback((id: string) => {
    setTasks((prev) => prev.filter((t) => t.id !== id));
  }, []);

  const isUploading = tasks.some(
    (t) => t.progress.phase === 'uploading' || t.progress.phase === 'merging'
  );

  return (
    <div className="space-y-3">
      <Card
        className={`border-2 border-dashed transition-colors ${
          dragActive ? 'border-primary bg-primary/5' : 'border-muted-foreground/25'
        }`}
        onDragOver={(e) => {
          e.preventDefault();
          setDragActive(true);
        }}
        onDragLeave={() => setDragActive(false)}
        onDrop={(e) => {
          e.preventDefault();
          setDragActive(false);
          handleFiles(e.dataTransfer.files);
        }}
      >
        <CardContent className="flex flex-col items-center justify-center py-8 gap-3">
          <Upload className="size-8 text-muted-foreground" />
          <p className="text-sm text-muted-foreground">
            拖拽文件到此处，或点击选择文件（支持大文件分片上传）
          </p>
          <Button
            variant="secondary"
            size="sm"
            disabled={isUploading}
            onClick={() => inputRef.current?.click()}
          >
            {isUploading ? '上传中...' : '选择文件'}
          </Button>
          <input
            ref={inputRef}
            type="file"
            multiple
            className="hidden"
            onChange={(e) => {
              handleFiles(e.target.files);
              if (inputRef.current) inputRef.current.value = '';
            }}
          />
        </CardContent>
      </Card>

      {tasks.length > 0 && (
        <div className="space-y-2">
          {tasks.map((task) => (
            <div
              key={task.id}
              className="flex items-center gap-3 rounded-lg border p-3 text-sm"
            >
              {/* Status icon */}
              <div className="shrink-0">
                {task.progress.phase === 'done' && (
                  <CheckCircle2 className="size-5 text-green-500" />
                )}
                {task.progress.phase === 'error' && (
                  <XCircle className="size-5 text-red-500" />
                )}
                {(task.progress.phase === 'uploading' || task.progress.phase === 'merging') && (
                  <Loader2 className="size-5 animate-spin text-primary" />
                )}
              </div>

              {/* File info + progress */}
              <div className="flex-1 min-w-0">
                <div className="flex items-center justify-between gap-2">
                  <span className="truncate font-medium">{task.file.name}</span>
                  <span className="shrink-0 text-xs text-muted-foreground">
                    {formatSize(task.file.size)}
                  </span>
                </div>

                {(task.progress.phase === 'uploading' || task.progress.phase === 'merging') && (
                  <div className="mt-1.5">
                    <div className="h-1.5 w-full rounded-full bg-muted overflow-hidden">
                      <div
                        className="h-full rounded-full bg-primary transition-all duration-300"
                        style={{ width: `${task.progress.percent}%` }}
                      />
                    </div>
                    <div className="mt-0.5 text-xs text-muted-foreground">
                      {task.progress.phase === 'merging'
                        ? '合并中...'
                        : `${task.progress.percent}%`}
                      {task.progress.phase === 'uploading' &&
                        task.progress.totalChunks > 1 &&
                        ` (${task.progress.uploadedChunks}/${task.progress.totalChunks} 分片)`}
                    </div>
                  </div>
                )}

                {task.progress.phase === 'error' && (
                  <p className="mt-0.5 text-xs text-red-500">{task.progress.error}</p>
                )}
              </div>

              {/* Actions */}
              <div className="shrink-0 flex gap-1">
                {task.progress.phase === 'error' && (
                  <Button
                    variant="ghost"
                    size="icon"
                    className="size-7"
                    onClick={() => retryTask(task)}
                    title="重试"
                  >
                    <RotateCcw className="size-3.5" />
                  </Button>
                )}
                {(task.progress.phase === 'done' || task.progress.phase === 'error') && (
                  <Button
                    variant="ghost"
                    size="icon"
                    className="size-7"
                    onClick={() => removeTask(task.id)}
                    title="移除"
                  >
                    <XCircle className="size-3.5" />
                  </Button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
