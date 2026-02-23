import { useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import { publicApi } from '@/lib/api';
import type { FileItem } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Download, Lock, FileIcon } from 'lucide-react';

export default function ShareView() {
  const { code } = useParams<{ code: string }>();
  const [title, setTitle] = useState('');
  const [needPassword, setNeedPassword] = useState(false);
  const [password, setPassword] = useState('');
  const [files, setFiles] = useState<FileItem[]>([]);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  const [verifying, setVerifying] = useState(false);
  const [verified, setVerified] = useState(false);

  useEffect(() => {
    if (!code) return;
    publicApi
      .getInfo(code)
      .then((res) => {
        setTitle(res.data.title || '');
        setNeedPassword(res.data.has_password);
        if (!res.data.has_password && res.data.files) {
          setFiles(res.data.files);
          setVerified(true);
        }
      })
      .catch((err) => {
        if (err.response?.status === 410) {
          setError('此分享已过期');
        } else {
          setError('分享不存在');
        }
      })
      .finally(() => setLoading(false));
  }, [code]);

  const handleVerify = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!code) return;
    setVerifying(true);
    setError('');
    try {
      const res = await publicApi.verify(code, password);
      setFiles(res.data.files || []);
      setVerified(true);
    } catch {
      setError('密码错误');
    } finally {
      setVerifying(false);
    }
  };

  const handleDownload = (fileId: string) => {
    if (!code) return;
    const url = publicApi.downloadUrl(code, fileId, needPassword ? password : undefined);
    window.open(url, '_blank');
  };

  const formatSize = (bytes: number) => {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    if (bytes < 1024 * 1024 * 1024) return (bytes / 1024 / 1024).toFixed(1) + ' MB';
    return (bytes / 1024 / 1024 / 1024).toFixed(1) + ' GB';
  };

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-muted/40">
        <p className="text-muted-foreground">加载中...</p>
      </div>
    );
  }

  if (error && !verified) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-muted/40">
        <Card className="w-full max-w-sm">
          <CardContent className="pt-6 text-center text-destructive">{error}</CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-muted/40 p-4">
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <CardTitle>{title || '文件分享'}</CardTitle>
          {!verified && <CardDescription>请输入密码以访问文件</CardDescription>}
        </CardHeader>
        <CardContent>
          {!verified && needPassword ? (
            <form onSubmit={handleVerify} className="space-y-4">
              {error && (
                <div className="text-sm text-destructive text-center">{error}</div>
              )}
              <div className="flex items-center gap-2">
                <Lock className="size-4 text-muted-foreground" />
                <Input
                  type="password"
                  placeholder="请输入提取密码"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                />
              </div>
              <Button type="submit" className="w-full" disabled={verifying}>
                {verifying ? '验证中...' : '提取文件'}
              </Button>
            </form>
          ) : (
            <div className="space-y-2">
              {files.map((f) => (
                <div
                  key={f.id}
                  className="flex items-center justify-between p-3 rounded-lg border bg-background"
                >
                  <div className="flex items-center gap-2 min-w-0">
                    <FileIcon className="size-4 text-muted-foreground shrink-0" />
                    <div className="min-w-0">
                      <p className="text-sm font-medium truncate">{f.name}</p>
                      <p className="text-xs text-muted-foreground">{formatSize(f.size)}</p>
                    </div>
                  </div>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleDownload(f.id)}
                  >
                    <Download className="size-4" />
                  </Button>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
