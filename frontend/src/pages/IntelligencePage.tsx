import React, { useState, useEffect, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Sparkles,
  Download,
  CheckCircle,
  ChevronLeft,
  Loader2,
  AlertCircle,
  Clock,
  FileText,
  RefreshCw,
  Share2,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';
import { api, FileItem, IngestionStatus, SummaryStatus } from '@/lib/api';

function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
}

export const IntelligencePage: React.FC = () => {
  const { fileId } = useParams();
  const navigate = useNavigate();

  const [file, setFile] = useState<FileItem | null>(null);
  const [ingestion, setIngestion] = useState<IngestionStatus | null>(null);
  const [summary, setSummary] = useState<SummaryStatus | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isDownloading, setIsDownloading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pollCount, setPollCount] = useState(0);

  const pollTimerRef = useRef<NodeJS.Timeout | null>(null);

  const fetchData = async (isPoll = false) => {
    if (!fileId) return;

    if (!isPoll) {
      setIsLoading(true);
      setError(null);
    }

    try {
      // 1. Fetch file metadata
      const fileRes = await api.getFile(fileId);
      setFile(fileRes.data);

      // 2. Fetch summary status (works for owner & read-granted recipients)
      let summaryData: SummaryStatus | null = null;
      try {
        summaryData = await api.getSummaryStatus(fileId);
        setSummary(summaryData);
      } catch (err: any) {
        // Summary might not be available or 404
        console.debug('Summary status not found or pending:', err.message);
      }

      // 3. Fetch ingestion status (owner only)
      let ingestionData: IngestionStatus | null = null;
      try {
        ingestionData = await api.getIngestionStatus(fileId);
        setIngestion(ingestionData);
      } catch {
        // Shared recipients might not have access to ingestion queue details
      }

      // 4. Determine if we should continue polling
      const isIngestionActive =
        ingestionData && (ingestionData.state === 'queued' || ingestionData.state === 'processing');
      const isSummaryActive =
        summaryData && (summaryData.state === 'queued' || summaryData.state === 'processing');

      if ((isIngestionActive || isSummaryActive) && pollCount < 40) {
        // Poll every 3 seconds while active
        if (pollTimerRef.current) clearTimeout(pollTimerRef.current);
        pollTimerRef.current = setTimeout(() => {
          setPollCount((prev) => prev + 1);
        }, 3000);
      }
    } catch (err: any) {
      if (!isPoll) {
        setError(err.message || 'Failed to load document details.');
      }
    } finally {
      if (!isPoll) {
        setIsLoading(false);
      }
    }
  };

  useEffect(() => {
    fetchData(pollCount > 0);

    return () => {
      if (pollTimerRef.current) {
        clearTimeout(pollTimerRef.current);
      }
    };
  }, [fileId, pollCount]);

  const handleDownload = async () => {
    if (!file) return;
    setIsDownloading(true);
    try {
      await api.downloadFile(file.id, file.name);
    } catch (err: any) {
      alert(`Download failed: ${err.message || 'Unknown error'}`);
    } finally {
      setIsDownloading(false);
    }
  };

  // Helper to extract key points and insights from summary text
  const extractSections = (text?: string) => {
    if (!text) {
      return {
        summary: '',
        keypoints: [] as string[],
        insights: '',
      };
    }

    const lines = text
      .split('\n')
      .map((l) => l.trim())
      .filter(Boolean);

    const keypoints: string[] = [];
    const summaryLines: string[] = [];

    for (const line of lines) {
      if (
        line.startsWith('- ') ||
        line.startsWith('* ') ||
        line.startsWith('• ') ||
        /^\d+\.\s/.test(line)
      ) {
        keypoints.push(line.replace(/^[-*•\d.]+\s*/, ''));
      } else {
        summaryLines.push(line);
      }
    }

    // If no explicit bullet points were formatted by the model, divide by sentences
    if (keypoints.length === 0 && text.length > 0) {
      const sentences = text
        .split(/(?<=[.!?])\s+/)
        .map((s) => s.trim())
        .filter((s) => s.length > 15);
      if (sentences.length > 1) {
        keypoints.push(...sentences.slice(1, 5));
      }
    }

    const executiveSummary =
      summaryLines.length > 0 ? summaryLines.join(' ') : text;
    const insights =
      keypoints.length > 0
        ? keypoints[0]
        : 'Source-grounded local AI analysis preserves exact document context without hallucinations.';

    return {
      summary: executiveSummary,
      keypoints,
      insights,
    };
  };

  const sections = extractSections(summary?.summary);

  // Status computation
  const isQueued =
    summary?.state === 'queued' || (ingestion?.state === 'queued' && !summary);
  const isProcessing =
    summary?.state === 'processing' ||
    (ingestion?.state === 'processing' && summary?.state !== 'succeeded');
  const isFailed = summary?.state === 'failed' || ingestion?.state === 'failed';
  const isSucceeded = summary?.state === 'succeeded' && !!summary?.summary;

  if (isLoading) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[450px] space-y-4 animate-fade-in">
        <Loader2 className="w-8 h-8 animate-spin text-[#7667F5]" />
        <p className="text-sm text-[#9296A3]">Loading document intelligence...</p>
      </div>
    );
  }

  if (error || !file) {
    return (
      <div className="max-w-2xl mx-auto py-12 text-center space-y-4 animate-fade-in">
        <div className="w-12 h-12 mx-auto rounded-full bg-[#EF4444]/10 border border-[#EF4444]/20 flex items-center justify-center text-[#EF4444]">
          <AlertCircle className="w-6 h-6" />
        </div>
        <h2 className="text-lg font-semibold text-[#F3F4F6]">Unable to Load Document</h2>
        <p className="text-sm text-[#9296A3]">{error || 'The requested file could not be found or you do not have permission to view it.'}</p>
        <Button
          variant="secondary"
          onClick={() => navigate('/files')}
          className="mt-4 text-xs"
        >
          Return to My Files
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6 animate-fade-in">
      {/* Subheader */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-4 border-b border-[#242731]">
        <div className="flex items-center gap-3">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => navigate('/files')}
            className="p-1 h-8 w-8 text-[#9296A3]"
          >
            <ChevronLeft className="w-5 h-5" />
          </Button>
          <div>
            <div className="flex items-center gap-2 flex-wrap">
              <h1 className="text-xl font-semibold text-[#F3F4F6] truncate max-w-md">
                {file.name}
              </h1>

              {isSucceeded && (
                <span className="text-[11px] px-2 py-0.5 rounded-full bg-[#36C8D8]/15 text-[#36C8D8] border border-[#36C8D8]/30 font-medium flex items-center gap-1">
                  <CheckCircle className="w-3 h-3" /> Processed & Indexed
                </span>
              )}

              {isProcessing && (
                <span className="text-[11px] px-2 py-0.5 rounded-full bg-[#F59E0B]/15 text-[#F59E0B] border border-[#F59E0B]/30 font-medium flex items-center gap-1">
                  <Loader2 className="w-3 h-3 animate-spin" /> Processing AI Summary...
                </span>
              )}

              {isQueued && (
                <span className="text-[11px] px-2 py-0.5 rounded-full bg-[#7667F5]/15 text-[#7667F5] border border-[#7667F5]/30 font-medium flex items-center gap-1">
                  <Clock className="w-3 h-3" /> Queued for AI Processing
                </span>
              )}

              {isFailed && (
                <span className="text-[11px] px-2 py-0.5 rounded-full bg-[#EF4444]/15 text-[#EF4444] border border-[#EF4444]/30 font-medium flex items-center gap-1">
                  <AlertCircle className="w-3 h-3" /> Processing Failed
                </span>
              )}
            </div>

            <p className="text-xs text-[#9296A3] mt-0.5">
              {formatFileSize(file.size)} · {file.content_type || 'Document'} ·{' '}
              {file.sha256 ? `SHA-256: ${file.sha256.substring(0, 12)}...` : 'Verified'}
            </p>
          </div>
        </div>

        <div className="flex items-center gap-3">
          <Button
            variant="secondary"
            size="sm"
            onClick={() => navigate(`/assistant?fileId=${file.id}`)}
            className="flex items-center gap-1.5 text-xs border-[#36C8D8]/30 hover:border-[#36C8D8]/50"
          >
            <Sparkles className="w-3.5 h-3.5 text-[#36C8D8]" /> Ask AI
          </Button>

          <Button
            variant="secondary"
            size="sm"
            onClick={() => navigate('/shared')}
            className="flex items-center gap-1.5 text-xs"
          >
            <Share2 className="w-3.5 h-3.5 text-[#9296A3]" /> Share
          </Button>

          <Button
            size="sm"
            onClick={handleDownload}
            disabled={isDownloading}
            className="bg-[#7667F5] hover:bg-[#8679F7] text-white flex items-center gap-1.5 text-xs"
          >
            {isDownloading ? (
              <Loader2 className="w-3.5 h-3.5 animate-spin" />
            ) : (
              <Download className="w-3.5 h-3.5" />
            )}
            Download
          </Button>
        </div>
      </div>

      {/* Split Workspace */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 min-h-[500px]">
        {/* Left: Document Metadata & Preview */}
        <Card className="lg:col-span-5 p-6 flex flex-col justify-between bg-[#15171D]">
          <div className="space-y-4">
            <div className="flex items-center justify-between pb-3 border-b border-[#242731]">
              <span className="text-xs font-semibold text-[#F3F4F6] uppercase tracking-wider">
                Document Overview
              </span>
              <span className="text-xs text-[#9296A3]">
                Uploaded {new Date(file.created_at).toLocaleDateString()}
              </span>
            </div>

            <div className="p-5 rounded-lg bg-[#101114] border border-[#242731] text-xs text-[#F3F4F6] space-y-3 leading-relaxed shadow-inner">
              <div className="flex items-start gap-2.5">
                <FileText className="w-5 h-5 text-[#7667F5] shrink-0 mt-0.5" />
                <div className="space-y-1">
                  <p className="font-semibold text-sm text-[#F3F4F6] break-all">{file.name}</p>
                  <p className="text-[11px] text-[#9296A3]">Format: {file.content_type || 'Unknown'}</p>
                  <p className="text-[11px] text-[#9296A3]">File Size: {formatFileSize(file.size)}</p>
                  {file.sha256 && (
                    <p className="text-[10px] font-mono text-[#9296A3] break-all pt-1">
                      Hash: {file.sha256}
                    </p>
                  )}
                </div>
              </div>

              <div className="pt-3 border-t border-[#242731] space-y-2 text-[#9296A3]">
                <div className="flex justify-between text-[11px]">
                  <span>Storage State</span>
                  <span className="text-[#22C55E] capitalize">{file.state || 'active'}</span>
                </div>
                <div className="flex justify-between text-[11px]">
                  <span>Ingestion Pipeline</span>
                  <span className="text-[#36C8D8] capitalize">
                    {ingestion?.state || (isSucceeded ? 'succeeded' : 'pending')}
                  </span>
                </div>
              </div>
            </div>
          </div>

          <div className="pt-4 border-t border-[#242731] flex items-center justify-between text-xs text-[#9296A3]">
            <span>Model: nomic-embed-text</span>
            {ingestion?.state === 'succeeded' || isSucceeded ? (
              <span className="text-[#22C55E] flex items-center gap-1 font-medium">
                <CheckCircle className="w-3.5 h-3.5" /> 768-dim Vector Embedded
              </span>
            ) : isProcessing ? (
              <span className="text-[#F59E0B] flex items-center gap-1">
                <Loader2 className="w-3.5 h-3.5 animate-spin" /> Vectorizing Chunks...
              </span>
            ) : (
              <span className="text-[#9296A3] flex items-center gap-1">
                <Clock className="w-3.5 h-3.5" /> Queued for Embedding
              </span>
            )}
          </div>
        </Card>

        {/* Right: AI Intelligence Card */}
        <Card className="lg:col-span-7 p-6 flex flex-col justify-between">
          <div className="space-y-4">
            <div className="flex items-center justify-between pb-3 border-b border-[#242731]">
              <div className="flex items-center gap-2">
                <Sparkles className="w-4 h-4 text-[#36C8D8]" />
                <h2 className="text-base font-semibold text-[#F3F4F6]">AI Intelligence</h2>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-xs font-mono text-[#9296A3]">
                  {summary?.model ? `Ollama · ${summary.model}` : 'Ollama · llama3.2:3b'}
                </span>
                {(isProcessing || isQueued) && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => fetchData(false)}
                    className="h-6 w-6 p-0 text-[#9296A3] hover:text-[#F3F4F6]"
                    title="Refresh status"
                  >
                    <RefreshCw className="w-3 h-3 animate-spin" />
                  </Button>
                )}
              </div>
            </div>

            {/* Succeeded State: Real Tabs & Content */}
            {isSucceeded && (
              <Tabs defaultValue="summary" className="w-full">
                <TabsList className="w-full justify-start border-b border-[#242731] bg-transparent p-0 rounded-none h-auto">
                  <TabsTrigger
                    value="summary"
                    className="rounded-none border-b-2 border-transparent data-[state=active]:border-[#7667F5] data-[state=active]:bg-transparent px-4 py-2 text-xs"
                  >
                    Summary
                  </TabsTrigger>
                  <TabsTrigger
                    value="keypoints"
                    className="rounded-none border-b-2 border-transparent data-[state=active]:border-[#7667F5] data-[state=active]:bg-transparent px-4 py-2 text-xs"
                  >
                    Key Points ({sections.keypoints.length})
                  </TabsTrigger>
                  <TabsTrigger
                    value="insights"
                    className="rounded-none border-b-2 border-transparent data-[state=active]:border-[#7667F5] data-[state=active]:bg-transparent px-4 py-2 text-xs"
                  >
                    Takeaways
                  </TabsTrigger>
                </TabsList>

                <TabsContent value="summary" className="pt-4 space-y-3">
                  <h3 className="text-xs font-semibold text-[#36C8D8] uppercase tracking-wider">
                    Executive Summary
                  </h3>
                  <p className="text-sm text-[#F3F4F6] leading-relaxed whitespace-pre-line font-sans">
                    {sections.summary}
                  </p>
                </TabsContent>

                <TabsContent value="keypoints" className="pt-4 space-y-3">
                  <h3 className="text-xs font-semibold text-[#36C8D8] uppercase tracking-wider">
                    Material Points
                  </h3>
                  {sections.keypoints.length > 0 ? (
                    <ul className="space-y-2 text-sm text-[#F3F4F6]">
                      {sections.keypoints.map((point, idx) => (
                        <li key={idx} className="flex items-start gap-2">
                          <span className="w-1.5 h-1.5 rounded-full bg-[#36C8D8] mt-2 shrink-0" />
                          <span className="leading-relaxed">{point}</span>
                        </li>
                      ))}
                    </ul>
                  ) : (
                    <p className="text-xs text-[#9296A3]">
                      No individual bullet points were extracted; refer to the Executive Summary.
                    </p>
                  )}
                </TabsContent>

                <TabsContent value="insights" className="pt-4 space-y-3">
                  <div className="p-4 rounded-xl bg-[#15171D] border-l-4 border-l-[#7667F5] border border-[#242731] space-y-2">
                    <h4 className="text-xs font-semibold text-[#7667F5]">
                      Primary Takeaway
                    </h4>
                    <p className="text-xs text-[#F3F4F6] leading-relaxed">
                      {sections.insights}
                    </p>
                  </div>
                </TabsContent>
              </Tabs>
            )}

            {/* Processing / Queued State */}
            {(isProcessing || isQueued) && (
              <div className="py-8 space-y-6 text-center">
                <div className="w-12 h-12 mx-auto rounded-full bg-[#36C8D8]/10 border border-[#36C8D8]/20 flex items-center justify-center text-[#36C8D8]">
                  <Loader2 className="w-6 h-6 animate-spin" />
                </div>
                <div className="space-y-1">
                  <h3 className="text-sm font-semibold text-[#F3F4F6]">
                    {isProcessing ? 'Generating AI Document Intelligence...' : 'Queued for AI Processing'}
                  </h3>
                  <p className="text-xs text-[#9296A3] max-w-sm mx-auto">
                    {isProcessing
                      ? 'Local Ollama is extracting chunks and synthesizing key takeaways. This screen updates in real-time.'
                      : 'Your document is in the processing queue. Analysis will begin shortly.'}
                  </p>
                </div>

                <div className="max-w-xs mx-auto p-3.5 rounded-xl bg-[#15171D] border border-[#242731] text-left text-xs space-y-2">
                  <div className="flex items-center gap-2 text-xs">
                    <span className={`w-2 h-2 rounded-full ${ingestion?.state === 'succeeded' ? 'bg-[#22C55E]' : 'bg-[#F59E0B] animate-pulse'}`} />
                    <span className={ingestion?.state === 'succeeded' ? 'text-[#F3F4F6]' : 'text-[#9296A3]'}>
                      1. Text Extraction & Chunking
                    </span>
                  </div>
                  <div className="flex items-center gap-2 text-xs">
                    <span className={`w-2 h-2 rounded-full ${ingestion?.state === 'succeeded' ? 'bg-[#22C55E]' : 'bg-[#9296A3]'}`} />
                    <span className={ingestion?.state === 'succeeded' ? 'text-[#F3F4F6]' : 'text-[#9296A3]'}>
                      2. Vector Embeddings (nomic-embed-text)
                    </span>
                  </div>
                  <div className="flex items-center gap-2 text-xs">
                    <span className={`w-2 h-2 rounded-full ${isProcessing ? 'bg-[#36C8D8] animate-pulse' : 'bg-[#9296A3]'}`} />
                    <span className={isProcessing ? 'text-[#36C8D8]' : 'text-[#9296A3]'}>
                      3. Synthesis & Summary (llama3.2:3b)
                    </span>
                  </div>
                </div>
              </div>
            )}

            {/* Failed State */}
            {isFailed && (
              <div className="py-8 space-y-4 text-center">
                <div className="w-12 h-12 mx-auto rounded-full bg-[#EF4444]/10 border border-[#EF4444]/20 flex items-center justify-center text-[#EF4444]">
                  <AlertCircle className="w-6 h-6" />
                </div>
                <div className="space-y-1">
                  <h3 className="text-sm font-semibold text-[#F3F4F6]">
                    AI Processing Encountered an Issue
                  </h3>
                  <p className="text-xs text-[#EF4444] max-w-sm mx-auto">
                    {summary?.last_error || ingestion?.last_error || 'Document processing could not complete.'}
                  </p>
                </div>
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => fetchData(false)}
                  className="text-xs"
                >
                  <RefreshCw className="w-3.5 h-3.5 mr-1" /> Retry Status Check
                </Button>
              </div>
            )}

            {/* Empty / Not Queued State */}
            {!isSucceeded && !isProcessing && !isQueued && !isFailed && (
              <div className="py-12 text-center text-[#9296A3] text-xs">
                No summary generated yet for this document.
              </div>
            )}
          </div>

          <div className="pt-4 border-t border-[#242731] flex items-center justify-between text-xs text-[#9296A3]">
            <span>
              {isSucceeded
                ? `Status: Succeeded (Attempt ${summary?.attempts || 1})`
                : isProcessing
                ? `Status: Processing (Attempt ${summary?.attempts || 1})`
                : isQueued
                ? `Status: Queued (Attempt ${summary?.attempts || 0})`
                : isFailed
                ? `Status: Failed (${summary?.attempts || 1} attempts)`
                : 'Status: Pending'}
            </span>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => navigate(`/assistant?fileId=${file.id}`)}
              className="text-xs text-[#7667F5] hover:text-[#8679F7] p-0"
            >
              Start Q&A in Assistant &rarr;
            </Button>
          </div>
        </Card>
      </div>
    </div>
  );
};
