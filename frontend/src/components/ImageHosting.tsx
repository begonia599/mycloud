import { useState, useEffect, useCallback, useRef } from 'react';
import { imageApi } from '@/lib/api';
import type { ImageItem } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import {
  Upload,
  Trash2,
  Copy,
  Eye,
  EyeOff,
  ImagePlus,
  Check,
  Loader2,
} from 'lucide-react';

export default function ImageHosting() {
  const [images, setImages] = useState<ImageItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [copiedId, setCopiedId] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const loadImages = useCallback(async () => {
    setLoading(true);
    try {
      const res = await imageApi.list();
      setImages(res.data.images || []);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadImages();
  }, [loadImages]);

  const handleUpload = async (fileList: FileList | File[]) => {
    const files = Array.from(fileList).filter((f) =>
      f.type.startsWith('image/')
    );
    if (files.length === 0) return;

    setUploading(true);
    try {
      await imageApi.upload(files);
      loadImages();
    } catch {
      // ignore
    } finally {
      setUploading(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('确定删除此图片？')) return;
    await imageApi.delete(id);
    loadImages();
  };

  const handleToggleVisibility = async (
    id: string,
    currentPublic: boolean
  ) => {
    await imageApi.toggleVisibility(id, !currentPublic);
    loadImages();
  };

  const copyMarkdownLink = (image: ImageItem) => {
    const url = `${window.location.origin}${imageApi.publicUrl(image.id)}`;
    const markdown = `![${image.name}](${url})`;
    navigator.clipboard.writeText(markdown);
    setCopiedId(image.id);
    setTimeout(() => setCopiedId(null), 2000);
  };

  const copyDirectLink = (image: ImageItem) => {
    const url = `${window.location.origin}${imageApi.publicUrl(image.id)}`;
    navigator.clipboard.writeText(url);
    setCopiedId(image.id);
    setTimeout(() => setCopiedId(null), 2000);
  };

  const formatSize = (bytes: number) => {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / 1024 / 1024).toFixed(1) + ' MB';
  };

  const formatDate = (d: string) => new Date(d).toLocaleString('zh-CN');

  // Drag & Drop handlers
  const [dragOver, setDragOver] = useState(false);

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(true);
  };
  const handleDragLeave = () => setDragOver(false);
  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(false);
    if (e.dataTransfer.files.length > 0) {
      handleUpload(e.dataTransfer.files);
    }
  };

  return (
    <div className="space-y-4">
      {/* 上传区域 */}
      <div
        className={`border-2 border-dashed rounded-lg p-8 text-center cursor-pointer transition-colors ${
          dragOver
            ? 'border-primary bg-primary/5'
            : 'border-muted-foreground/25 hover:border-primary/50'
        }`}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
        onClick={() => fileInputRef.current?.click()}
      >
        <input
          ref={fileInputRef}
          type="file"
          accept="image/*"
          multiple
          className="hidden"
          onChange={(e) => e.target.files && handleUpload(e.target.files)}
        />
        {uploading ? (
          <div className="flex flex-col items-center gap-2 text-muted-foreground">
            <Loader2 className="size-8 animate-spin" />
            <p>上传中...</p>
          </div>
        ) : (
          <div className="flex flex-col items-center gap-2 text-muted-foreground">
            <ImagePlus className="size-8" />
            <p>拖拽图片到此处，或点击选择图片</p>
            <p className="text-xs">支持 JPG、PNG、GIF、WebP、SVG，单张最大 10MB</p>
          </div>
        )}
      </div>

      {/* 图片列表 */}
      {loading ? (
        <p className="text-muted-foreground">加载中...</p>
      ) : images.length === 0 ? (
        <p className="text-muted-foreground">暂无图片</p>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {images.map((img) => (
            <Card key={img.id} className="overflow-hidden">
              <div className="aspect-video bg-muted relative overflow-hidden">
                <img
                  src={imageApi.publicUrl(img.id)}
                  alt={img.name}
                  className="w-full h-full object-cover"
                  loading="lazy"
                />
                <div className="absolute top-2 right-2">
                  <Badge variant={img.is_public ? 'default' : 'secondary'}>
                    {img.is_public ? '公开' : '私有'}
                  </Badge>
                </div>
              </div>
              <CardContent className="p-3 space-y-2">
                <p
                  className="text-sm font-medium truncate"
                  title={img.name}
                >
                  {img.name}
                </p>
                <p className="text-xs text-muted-foreground">
                  {formatSize(img.size)} · {formatDate(img.created_at)}
                </p>
                <div className="flex items-center gap-1">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => copyMarkdownLink(img)}
                    title="复制 Markdown 链接"
                  >
                    {copiedId === img.id ? (
                      <Check className="size-3.5 text-green-500" />
                    ) : (
                      <Copy className="size-3.5" />
                    )}
                    <span className="ml-1 text-xs">MD</span>
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => copyDirectLink(img)}
                    title="复制直链"
                  >
                    <Upload className="size-3.5" />
                    <span className="ml-1 text-xs">链接</span>
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() =>
                      handleToggleVisibility(img.id, img.is_public)
                    }
                    title={img.is_public ? '设为私有' : '设为公开'}
                  >
                    {img.is_public ? (
                      <EyeOff className="size-3.5" />
                    ) : (
                      <Eye className="size-3.5" />
                    )}
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleDelete(img.id)}
                    title="删除"
                  >
                    <Trash2 className="size-3.5 text-destructive" />
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
