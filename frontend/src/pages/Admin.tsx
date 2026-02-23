import { useState, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Button } from '@/components/ui/button';
import FileUpload from '@/components/FileUpload';
import FileList from '@/components/FileList';
import ShareDialog from '@/components/ShareDialog';
import type { FileItem, ShareItem } from '@/lib/api';
import { shareApi } from '@/lib/api';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { LogOut, Trash2, Link, Copy } from 'lucide-react';

export default function Admin() {
  const navigate = useNavigate();
  const [refreshKey, setRefreshKey] = useState(0);
  const [shareDialogOpen, setShareDialogOpen] = useState(false);
  const [selectedFiles, setSelectedFiles] = useState<FileItem[]>([]);
  const [shares, setShares] = useState<ShareItem[]>([]);
  const [sharesLoading, setSharesLoading] = useState(false);

  const refresh = useCallback(() => setRefreshKey((k) => k + 1), []);

  const loadShares = useCallback(async () => {
    setSharesLoading(true);
    try {
      const res = await shareApi.list();
      setShares(res.data.shares || []);
    } catch {
      // ignore
    } finally {
      setSharesLoading(false);
    }
  }, []);

  const handleDeleteShare = async (id: string) => {
    if (!confirm('确定删除此分享？')) return;
    await shareApi.delete(id);
    loadShares();
  };

  const copyLink = (code: string) => {
    const url = `${window.location.origin}/s/${code}`;
    navigator.clipboard.writeText(url);
  };

  const handleLogout = () => {
    localStorage.removeItem('token');
    navigate('/login');
  };

  const formatSize = (bytes: number) => {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    if (bytes < 1024 * 1024 * 1024) return (bytes / 1024 / 1024).toFixed(1) + ' MB';
    return (bytes / 1024 / 1024 / 1024).toFixed(1) + ' GB';
  };

  const formatDate = (d: string) => new Date(d).toLocaleString('zh-CN');

  return (
    <div className="min-h-screen bg-muted/40">
      <header className="border-b bg-background">
        <div className="container mx-auto flex items-center justify-between h-14 px-4">
          <h1 className="text-lg font-semibold">云盘管理</h1>
          <Button variant="ghost" size="sm" onClick={handleLogout}>
            <LogOut className="size-4 mr-1" /> 退出登录
          </Button>
        </div>
      </header>

      <main className="container mx-auto py-6 px-4">
        <Tabs defaultValue="files" onValueChange={(v) => v === 'shares' && loadShares()}>
          <TabsList>
            <TabsTrigger value="files">文件管理</TabsTrigger>
            <TabsTrigger value="shares">分享管理</TabsTrigger>
          </TabsList>

          <TabsContent value="files" className="space-y-4 mt-4">
            <FileUpload onUploaded={refresh} />
            <FileList
              key={refreshKey}
              onRefresh={refresh}
              selectedFiles={selectedFiles}
              onSelectionChange={setSelectedFiles}
            />
            {selectedFiles.length > 0 && (
              <div className="fixed bottom-6 left-1/2 -translate-x-1/2">
                <Button onClick={() => setShareDialogOpen(true)}>
                  <Link className="size-4 mr-1" />
                  分享 {selectedFiles.length} 个文件
                </Button>
              </div>
            )}
          </TabsContent>

          <TabsContent value="shares" className="mt-4 space-y-3">
            {sharesLoading ? (
              <p className="text-muted-foreground">加载中...</p>
            ) : shares.length === 0 ? (
              <p className="text-muted-foreground">暂无分享记录</p>
            ) : (
              shares.map((s) => (
                <Card key={s.id}>
                  <CardHeader className="pb-2">
                    <div className="flex items-center justify-between">
                      <CardTitle className="text-base">
                        {s.title || '未命名分享'}
                      </CardTitle>
                      <div className="flex items-center gap-2">
                        <Button
                          variant="ghost"
                          size="icon-xs"
                          onClick={() => copyLink(s.code)}
                          title="复制链接"
                        >
                          <Copy className="size-3.5" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon-xs"
                          onClick={() => handleDeleteShare(s.id)}
                        >
                          <Trash2 className="size-3.5 text-destructive" />
                        </Button>
                      </div>
                    </div>
                  </CardHeader>
                  <CardContent className="text-sm text-muted-foreground space-y-1">
                    <div className="flex gap-2 items-center">
                      <code className="bg-muted px-1.5 py-0.5 rounded text-xs">
                        /s/{s.code}
                      </code>
                      {s.has_password && <Badge variant="secondary">有密码</Badge>}
                      {s.expires_at && <Badge variant="outline">过期时间 {formatDate(s.expires_at)}</Badge>}
                    </div>
                    <div>
                      {s.files.length} 个文件 &middot;{' '}
                      共 {formatSize(s.files.reduce((a, f) => a + f.size, 0))} &middot;{' '}
                      创建于 {formatDate(s.created_at)}
                    </div>
                  </CardContent>
                </Card>
              ))
            )}
          </TabsContent>
        </Tabs>
      </main>

      <ShareDialog
        open={shareDialogOpen}
        onOpenChange={setShareDialogOpen}
        files={selectedFiles}
        onCreated={() => {
          setShareDialogOpen(false);
          setSelectedFiles([]);
          refresh();
        }}
      />
    </div>
  );
}
