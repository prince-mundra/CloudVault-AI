import React, { useEffect, useState, useCallback } from 'react';
import { Upload, FolderPlus, Sparkles, FileText, ArrowUpRight, Clock, FileCode, FileSpreadsheet, Image as ImageIcon, HardDrive } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Progress } from '@/components/ui/progress';
import { useAuth } from '@/context/AuthContext';
import { useNavigate } from 'react-router-dom';
import { api, StorageAnalytics, FileItem } from '@/lib/api';
import { formatBytes } from '@/lib/utils';

function getFileIcon(filename: string, contentType?: string) {
  const ext = filename.split('.').pop()?.toLowerCase() || '';
  if (ext === 'pdf' || contentType?.includes('pdf')) {
    return <FileText className="w-4 h-4 text-[#7667F5]" />;
  }
  if (['xls', 'xlsx', 'csv'].includes(ext) || contentType?.includes('spreadsheet') || contentType?.includes('excel')) {
    return <FileSpreadsheet className="w-4 h-4 text-[#22C55E]" />;
  }
  if (['png', 'jpg', 'jpeg', 'webp', 'svg', 'gif'].includes(ext) || contentType?.includes('image')) {
    return <ImageIcon className="w-4 h-4 text-[#EAB308]" />;
  }
  if (['js', 'ts', 'jsx', 'tsx', 'py', 'go', 'json', 'html', 'css'].includes(ext)) {
    return <FileCode className="w-4 h-4 text-[#36C8D8]" />;
  }
  return <FileText className="w-4 h-4 text-[#9296A3]" />;
}

export const DashboardPage: React.FC = () => {
  const { user } = useAuth();
  const navigate = useNavigate();

  const [analytics, setAnalytics] = useState<StorageAnalytics | null>(null);
  const [loading, setLoading] = useState(true);

  const loadAnalytics = useCallback(async () => {
    try {
      const res = await api.getStorageAnalytics();
      if (res?.data) {
        setAnalytics(res.data);
      }
    } catch (err) {
      console.error('Failed to load storage analytics:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadAnalytics();

    const handleUpdate = () => {
      loadAnalytics();
    };
    window.addEventListener('cloudvault-storage-update', handleUpdate);
    return () => window.removeEventListener('cloudvault-storage-update', handleUpdate);
  }, [loadAnalytics]);

  const usedBytes = analytics?.used_bytes ?? 0;
  const quotaBytes = analytics?.quota_bytes ?? 5368709120; // 5 GB default
  const usedPercent = quotaBytes > 0 ? Math.min(100, Math.round((usedBytes / quotaBytes) * 100)) : 0;
  const totalFiles = analytics?.total_files ?? 0;
  const recentUploads: FileItem[] = analytics?.recent_uploads ?? [];
  const recentActivity = analytics?.recent_activity ?? [];

  return (
    <div className="space-y-8 animate-fade-in">
      {/* Greeting & Quick Action Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight text-[#F3F4F6]">
            Good {new Date().getHours() < 12 ? 'morning' : new Date().getHours() < 18 ? 'afternoon' : 'evening'}, {user?.name ? user.name.split(' ')[0] : 'Guest'}.
          </h1>
          <p className="text-sm text-[#9296A3] mt-0.5">Your workspace is indexed and AES-256-GCM encrypted.</p>
        </div>

        <div className="flex flex-wrap items-center gap-3">
          <Button
            onClick={() => navigate('/files')}
            className="bg-[#7667F5] hover:bg-[#8679F7] text-white flex items-center gap-2"
          >
            <Upload className="w-4 h-4" /> Upload File
          </Button>

          <Button
            variant="secondary"
            onClick={() => navigate('/files')}
            className="flex items-center gap-2"
          >
            <FolderPlus className="w-4 h-4 text-[#9296A3]" /> New Folder
          </Button>

          <Button
            variant="secondary"
            onClick={() => navigate('/assistant')}
            className="flex items-center gap-2 border-[#36C8D8]/30 hover:border-[#36C8D8]/50"
          >
            <Sparkles className="w-4 h-4 text-[#36C8D8]" /> Ask AI
          </Button>
        </div>
      </div>

      {/* KPI Cards Strip */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-5">
        {/* Storage Used */}
        <Card className="p-5 flex flex-col justify-between">
          <div className="space-y-2">
            <span className="text-xs text-[#9296A3] font-medium">Storage Used</span>
            <div className="flex items-baseline justify-between">
              <span className="text-2xl font-bold tracking-tight text-[#F3F4F6] tabular-nums">
                {formatBytes(usedBytes)}
              </span>
              <span className="text-xs text-[#9296A3]">of {formatBytes(quotaBytes)}</span>
            </div>
            <Progress value={usedPercent} className="h-1.5" />
          </div>
          <span className="text-xs text-[#9296A3] mt-3">
            {formatBytes(analytics?.remaining_bytes ?? Math.max(0, quotaBytes - usedBytes))} remaining
          </span>
        </Card>

        {/* Total Files */}
        <Card className="p-5 flex flex-col justify-between">
          <div className="space-y-1">
            <span className="text-xs text-[#9296A3] font-medium">Total Files</span>
            <div className="flex items-baseline justify-between">
              <span className="text-2xl font-bold tracking-tight text-[#F3F4F6] tabular-nums">
                {totalFiles.toLocaleString()}
              </span>
              <span className="text-xs text-[#22C55E] font-medium">
                {totalFiles > 0 ? 'Active vault' : 'Empty vault'}
              </span>
            </div>
          </div>
          <span className="text-xs text-[#9296A3] mt-3">AES-256-GCM encrypted & isolated</span>
        </Card>

        {/* Shared Links */}
        <Card className="p-5 flex flex-col justify-between">
          <div className="space-y-1">
            <span className="text-xs text-[#9296A3] font-medium">Shared Links</span>
            <div className="flex items-baseline justify-between">
              <span className="text-2xl font-bold tracking-tight text-[#F3F4F6] tabular-nums">0</span>
              <span className="text-xs text-[#36C8D8] font-medium">Active Links</span>
            </div>
          </div>
          <span className="text-xs text-[#9296A3] mt-3">Granular link sharing available</span>
        </Card>
      </div>

      {/* Main 2-Column Split: Recent Documents & AI Insights / Activity */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Left: Recent Documents Table */}
        <Card className="lg:col-span-7 p-6 space-y-4">
          <div className="flex items-center justify-between pb-2 border-b border-[#242731]">
            <h2 className="text-base font-semibold text-[#F3F4F6]">Recent Documents</h2>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => navigate('/files')}
              className="text-xs text-[#7667F5] hover:text-[#8679F7]"
            >
              View All <ArrowUpRight className="w-3.5 h-3.5 ml-1" />
            </Button>
          </div>

          {loading ? (
            <div className="py-10 text-center text-sm text-[#9296A3]">
              Loading recent documents...
            </div>
          ) : recentUploads.length > 0 ? (
            <div className="space-y-2">
              {recentUploads.slice(0, 5).map((file) => (
                <div
                  key={file.id}
                  onClick={() => navigate('/files')}
                  className="p-3 rounded-lg bg-[#15171D] border border-[#242731] hover:bg-[#1F222B] hover:border-[#323644] transition-colors cursor-pointer flex items-center justify-between"
                >
                  <div className="flex items-center gap-3 min-w-0">
                    <div className="w-8 h-8 rounded-lg bg-[#7667F5]/10 border border-[#7667F5]/20 flex items-center justify-center shrink-0">
                      {getFileIcon(file.name, file.content_type)}
                    </div>
                    <div className="min-w-0">
                      <p className="text-sm font-medium text-[#F3F4F6] hover:text-[#7667F5] transition-colors truncate">
                        {file.name}
                      </p>
                      <p className="text-xs text-[#9296A3]">
                        {formatBytes(file.size)} · {new Date(file.created_at).toLocaleDateString()}
                      </p>
                    </div>
                  </div>
                  <span className="text-[11px] px-2 py-0.5 rounded-full bg-[#36C8D8]/10 text-[#36C8D8] border border-[#36C8D8]/20 font-medium shrink-0 ml-2">
                    {file.name.toLowerCase().endsWith('.pdf') ? 'Indexed' : 'Stored'}
                  </span>
                </div>
              ))}
            </div>
          ) : (
            <div className="py-12 flex flex-col items-center justify-center text-center space-y-3">
              <div className="w-12 h-12 rounded-full bg-[#1F222B] border border-[#242731] flex items-center justify-center text-[#9296A3]">
                <HardDrive className="w-6 h-6" />
              </div>
              <div>
                <p className="text-sm font-medium text-[#F3F4F6]">No files uploaded yet</p>
                <p className="text-xs text-[#9296A3] mt-1 max-w-xs">
                  Upload your documents, spreadsheets, or images to begin indexing and storage.
                </p>
              </div>
              <Button
                size="sm"
                onClick={() => navigate('/files')}
                className="bg-[#7667F5] hover:bg-[#8679F7] text-white text-xs mt-2"
              >
                <Upload className="w-3.5 h-3.5 mr-1.5" /> Upload File
              </Button>
            </div>
          )}
        </Card>

        {/* Right Column: AI Insights & Recent Activity */}
        <div className="lg:col-span-5 space-y-6">
          {/* AI Insights Card */}
          <Card className="p-6 relative overflow-hidden border border-[#242731]">
            <div className="absolute top-0 right-0 w-32 h-32 bg-[#36C8D8]/10 blur-2xl pointer-events-none rounded-full" />
            <div className="flex items-center gap-2 mb-4">
              <Sparkles className="w-4 h-4 text-[#36C8D8]" />
              <h2 className="text-base font-semibold text-[#F3F4F6]">AI Insights</h2>
            </div>

            <div className="space-y-3">
              {recentUploads.length > 0 ? (
                <>
                  <div className="p-3.5 rounded-lg bg-[#15171D] border border-[#242731] text-xs text-[#9296A3] leading-relaxed">
                    <span className="text-[#36C8D8] font-semibold block mb-1">Knowledge Base Ready:</span>
                    {totalFiles} file{totalFiles === 1 ? '' : 's'} registered in your encrypted vault. Semantic vector embeddings are active for supported PDF documents.
                  </div>
                  <div className="p-3.5 rounded-lg bg-[#15171D] border border-[#242731] text-xs text-[#9296A3] leading-relaxed">
                    <span className="text-[#7667F5] font-semibold block mb-1">Next Step:</span>
                    Open the <strong className="text-[#F3F4F6] cursor-pointer hover:underline" onClick={() => navigate('/assistant')}>AI Assistant</strong> to ask questions across your documents.
                  </div>
                </>
              ) : (
                <div className="p-3.5 rounded-lg bg-[#15171D] border border-[#242731] text-xs text-[#9296A3] leading-relaxed">
                  <span className="text-[#36C8D8] font-semibold block mb-1">AI Ingestion Pipeline:</span>
                  Upload PDF documents to automatically extract summaries, generate vector embeddings, and search semantically.
                </div>
              )}
            </div>
          </Card>

          {/* Recent Activity Card */}
          <Card className="p-6">
            <h2 className="text-base font-semibold text-[#F3F4F6] mb-4">Recent Activity</h2>
            {recentActivity.length > 0 ? (
              <div className="space-y-3">
                {recentActivity.slice(0, 5).map((act, idx) => (
                  <div key={idx} className="flex items-start gap-3 text-xs">
                    <Clock className="w-4 h-4 text-[#9296A3] shrink-0 mt-0.5" />
                    <p className="text-[#9296A3] leading-snug">
                      <strong className="text-[#F3F4F6]">
                        {act.actor_id === user?.id ? 'You' : 'User'}
                      </strong>{' '}
                      {act.action}{' '}
                      <span className="text-[#7667F5]">{act.file_name || 'file'}</span> ·{' '}
                      {new Date(act.created_at).toLocaleDateString()}
                    </p>
                  </div>
                ))}
              </div>
            ) : (
              <div className="py-6 text-center text-xs text-[#9296A3]">
                No recent activity recorded yet.
              </div>
            )}
          </Card>
        </div>
      </div>
    </div>
  );
};
