import { useState, useEffect } from 'react';
import { fileApi } from '@/lib/api';
import type { FileItem } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Trash2, FileIcon } from 'lucide-react';

interface FileListProps {
  onRefresh: () => void;
  selectedFiles: FileItem[];
  onSelectionChange: (files: FileItem[]) => void;
}

export default function FileList({ onRefresh, selectedFiles, onSelectionChange }: FileListProps) {
  const [files, setFiles] = useState<FileItem[]>([]);
  const [loading, setLoading] = useState(true);

  const loadFiles = async () => {
    setLoading(true);
    try {
      const res = await fileApi.list();
      setFiles(res.data.files || []);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadFiles();
  }, []);

  const handleDelete = async (id: string) => {
    if (!confirm('确定删除此文件？')) return;
    await fileApi.delete(id);
    onSelectionChange(selectedFiles.filter((f) => f.id !== id));
    onRefresh();
    loadFiles();
  };

  const toggleSelect = (file: FileItem) => {
    const exists = selectedFiles.find((f) => f.id === file.id);
    if (exists) {
      onSelectionChange(selectedFiles.filter((f) => f.id !== file.id));
    } else {
      onSelectionChange([...selectedFiles, file]);
    }
  };

  const formatSize = (bytes: number) => {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    if (bytes < 1024 * 1024 * 1024) return (bytes / 1024 / 1024).toFixed(1) + ' MB';
    return (bytes / 1024 / 1024 / 1024).toFixed(1) + ' GB';
  };

  if (loading) return <p className="text-sm text-muted-foreground">加载文件中...</p>;
  if (files.length === 0) return <p className="text-sm text-muted-foreground">暂无已上传的文件</p>;

  return (
    <div className="space-y-1">
      {files.map((f) => {
        const selected = !!selectedFiles.find((s) => s.id === f.id);
        return (
          <div
            key={f.id}
            className={`flex items-center gap-3 p-3 rounded-lg border transition-colors cursor-pointer ${
              selected ? 'bg-primary/5 border-primary/30' : 'bg-background hover:bg-muted/50'
            }`}
            onClick={() => toggleSelect(f)}
          >
            <Checkbox checked={selected} onCheckedChange={() => toggleSelect(f)} />
            <FileIcon className="size-4 text-muted-foreground shrink-0" />
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium truncate">{f.name}</p>
              <p className="text-xs text-muted-foreground">
                {formatSize(f.size)} &middot; {new Date(f.created_at).toLocaleString('zh-CN')}
              </p>
            </div>
            <Button
              variant="ghost"
              size="icon-xs"
              onClick={(e) => {
                e.stopPropagation();
                handleDelete(f.id);
              }}
            >
              <Trash2 className="size-3.5 text-destructive" />
            </Button>
          </div>
        );
      })}
    </div>
  );
}
