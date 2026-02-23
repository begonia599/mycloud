import { useState, useRef } from 'react';
import { fileApi } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Upload } from 'lucide-react';

interface FileUploadProps {
  onUploaded: () => void;
}

export default function FileUpload({ onUploaded }: FileUploadProps) {
  const [uploading, setUploading] = useState(false);
  const [dragActive, setDragActive] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  const handleUpload = async (fileList: FileList | null) => {
    if (!fileList || fileList.length === 0) return;
    setUploading(true);
    try {
      await fileApi.upload(Array.from(fileList));
      onUploaded();
    } catch {
      alert('上传失败');
    } finally {
      setUploading(false);
      if (inputRef.current) inputRef.current.value = '';
    }
  };

  return (
    <Card
      className={`border-2 border-dashed transition-colors ${
        dragActive ? 'border-primary bg-primary/5' : 'border-muted-foreground/25'
      }`}
      onDragOver={(e) => { e.preventDefault(); setDragActive(true); }}
      onDragLeave={() => setDragActive(false)}
      onDrop={(e) => {
        e.preventDefault();
        setDragActive(false);
        handleUpload(e.dataTransfer.files);
      }}
    >
      <CardContent className="flex flex-col items-center justify-center py-8 gap-3">
        <Upload className="size-8 text-muted-foreground" />
        <p className="text-sm text-muted-foreground">
          拖拽文件到此处，或点击选择文件
        </p>
        <Button
          variant="secondary"
          size="sm"
          disabled={uploading}
          onClick={() => inputRef.current?.click()}
        >
          {uploading ? '上传中...' : '选择文件'}
        </Button>
        <input
          ref={inputRef}
          type="file"
          multiple
          className="hidden"
          onChange={(e) => handleUpload(e.target.files)}
        />
      </CardContent>
    </Card>
  );
}
