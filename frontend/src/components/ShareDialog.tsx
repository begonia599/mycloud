import { useState } from 'react';
import { shareApi } from '@/lib/api';
import type { FileItem } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Copy, Check } from 'lucide-react';

interface ShareDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  files: FileItem[];
  onCreated: () => void;
}

export default function ShareDialog({ open, onOpenChange, files, onCreated }: ShareDialogProps) {
  const [title, setTitle] = useState('');
  const [password, setPassword] = useState('');
  const [expiresIn, setExpiresIn] = useState('');
  const [creating, setCreating] = useState(false);
  const [shareUrl, setShareUrl] = useState('');
  const [copied, setCopied] = useState(false);

  const handleCreate = async () => {
    setCreating(true);
    try {
      const res = await shareApi.create({
        title: title || undefined,
        password: password || undefined,
        file_ids: files.map((f) => f.id),
        expires_in: expiresIn ? parseInt(expiresIn) : undefined,
      });
      const url = `${window.location.origin}/s/${res.data.share.code}`;
      setShareUrl(url);
    } catch {
      alert('创建分享失败');
    } finally {
      setCreating(false);
    }
  };

  const handleCopy = () => {
    navigator.clipboard.writeText(shareUrl);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleClose = (open: boolean) => {
    if (!open) {
      setTitle('');
      setPassword('');
      setExpiresIn('');
      setShareUrl('');
      setCopied(false);
      if (shareUrl) onCreated();
    }
    onOpenChange(open);
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>创建分享链接</DialogTitle>
          <DialogDescription>
            将 {files.length} 个文件通过链接分享
          </DialogDescription>
        </DialogHeader>

        {shareUrl ? (
          <div className="space-y-3">
            <Label>分享链接</Label>
            <div className="flex gap-2">
              <Input value={shareUrl} readOnly />
              <Button variant="secondary" size="sm" onClick={handleCopy}>
                {copied ? <Check className="size-4" /> : <Copy className="size-4" />}
              </Button>
            </div>
            <p className="text-xs text-muted-foreground">
              {password && '已设置密码保护。'}
              {expiresIn && `${expiresIn} 小时后过期。`}
            </p>
          </div>
        ) : (
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="share-title">标题（可选）</Label>
              <Input
                id="share-title"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="我的分享文件"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="share-password">密码（可选）</Label>
              <Input
                id="share-password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="留空则无需密码"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="share-expires">过期时间（小时，可选）</Label>
              <Input
                id="share-expires"
                type="number"
                min="1"
                value={expiresIn}
                onChange={(e) => setExpiresIn(e.target.value)}
                placeholder="例如 24"
              />
            </div>
            <div className="text-sm text-muted-foreground">
              已选文件：{files.map((f) => f.name).join('、')}
            </div>
          </div>
        )}

        {!shareUrl && (
          <DialogFooter>
            <Button variant="outline" onClick={() => handleClose(false)}>
              取消
            </Button>
            <Button onClick={handleCreate} disabled={creating}>
              {creating ? '创建中...' : '创建链接'}
            </Button>
          </DialogFooter>
        )}
      </DialogContent>
    </Dialog>
  );
}
