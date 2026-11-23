import React from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Sparkles, Share2, Download, CheckCircle, ChevronLeft } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';

export const IntelligencePage: React.FC = () => {
  const { fileId } = useParams();
  const navigate = useNavigate();

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
            <div className="flex items-center gap-2">
              <h1 className="text-xl font-semibold text-[#F3F4F6]">
                {fileId ? 'cloudvault-test.pdf' : 'Document Intelligence'}
              </h1>
              <span className="text-[11px] px-2 py-0.5 rounded-full bg-[#36C8D8]/15 text-[#36C8D8] border border-[#36C8D8]/30 font-medium">
                Processed & Indexed
              </span>
            </div>
            <p className="text-xs text-[#9296A3] mt-0.5">1 Page · 775 B · SHA-256 Verified</p>
          </div>
        </div>

        <div className="flex items-center gap-3">
          <Button
            variant="secondary"
            size="sm"
            onClick={() => navigate('/assistant')}
            className="flex items-center gap-1.5 text-xs border-[#36C8D8]/30 hover:border-[#36C8D8]/50"
          >
            <Sparkles className="w-3.5 h-3.5 text-[#36C8D8]" /> Ask AI
          </Button>

          <Button variant="secondary" size="sm" className="flex items-center gap-1.5 text-xs">
            <Share2 className="w-3.5 h-3.5 text-[#9296A3]" /> Share
          </Button>

          <Button size="sm" className="bg-[#7667F5] hover:bg-[#8679F7] text-white flex items-center gap-1.5 text-xs">
            <Download className="w-3.5 h-3.5" /> Download
          </Button>
        </div>
      </div>

      {/* Split Workspace */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 min-h-[500px]">
        {/* Left: Document Preview */}
        <Card className="lg:col-span-5 p-6 flex flex-col justify-between bg-[#15171D]">
          <div className="space-y-4">
            <div className="flex items-center justify-between pb-3 border-b border-[#242731]">
              <span className="text-xs font-semibold text-[#F3F4F6]">Document Preview</span>
              <span className="text-xs text-[#9296A3]">Page 1 of 1</span>
            </div>

            <div className="p-6 rounded-lg bg-[#101114] border border-[#242731] font-mono text-xs text-[#F3F4F6] space-y-3 leading-relaxed shadow-inner">
              <p className="font-bold text-[#7667F5]">CloudVault AI Integration Test</p>
              <p className="text-[#9296A3]">Cloud storage allows users to store and retrieve documents.</p>
              <p className="text-[#9296A3]">Artificial intelligence can summarize documents and answer questions.</p>
            </div>
          </div>

          <div className="pt-4 border-t border-[#242731] flex items-center justify-between text-xs text-[#9296A3]">
            <span>Model: nomic-embed-text</span>
            <span className="text-[#22C55E] flex items-center gap-1">
              <CheckCircle className="w-3.5 h-3.5" /> 768-dim Vector Embedded
            </span>
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
              <span className="text-xs font-mono text-[#9296A3]">Ollama · llama3.2:3b</span>
            </div>

            <Tabs defaultValue="summary" className="w-full">
              <TabsList className="w-full justify-start border-b border-[#242731] bg-transparent p-0 rounded-none h-auto">
                <TabsTrigger
                  value="summary"
                  className="rounded-none border-b-2 border-transparent data-[state=active]:border-[#7667F5] data-[state=active]:bg-transparent px-4 py-2"
                >
                  Summary
                </TabsTrigger>
                <TabsTrigger
                  value="keypoints"
                  className="rounded-none border-b-2 border-transparent data-[state=active]:border-[#7667F5] data-[state=active]:bg-transparent px-4 py-2"
                >
                  Key Points
                </TabsTrigger>
                <TabsTrigger
                  value="insights"
                  className="rounded-none border-b-2 border-transparent data-[state=active]:border-[#7667F5] data-[state=active]:bg-transparent px-4 py-2"
                >
                  Insights
                </TabsTrigger>
              </TabsList>

              <TabsContent value="summary" className="pt-4 space-y-3">
                <h3 className="text-xs font-semibold text-[#36C8D8] uppercase tracking-wider">Executive Summary</h3>
                <p className="text-sm text-[#F3F4F6] leading-relaxed">
                  The document discusses the integration of CloudVault AI into cloud storage infrastructure, highlighting the convergence of cloud storage capabilities with artificial intelligence to organize, summarize, and retrieve knowledge from documents.
                </p>
              </TabsContent>

              <TabsContent value="keypoints" className="pt-4 space-y-3">
                <h3 className="text-xs font-semibold text-[#36C8D8] uppercase tracking-wider">Material Points</h3>
                <ul className="space-y-2 text-sm text-[#9296A3]">
                  <li className="flex items-start gap-2">
                    <span className="w-1.5 h-1.5 rounded-full bg-[#36C8D8] mt-2 shrink-0" />
                    <span>Cloud storage provides persistent, secure storage and retrieval for documents.</span>
                  </li>
                  <li className="flex items-start gap-2">
                    <span className="w-1.5 h-1.5 rounded-full bg-[#36C8D8] mt-2 shrink-0" />
                    <span>Artificial intelligence can extract, summarize, and answer questions across document collections.</span>
                  </li>
                </ul>
              </TabsContent>

              <TabsContent value="insights" className="pt-4 space-y-3">
                <div className="p-4 rounded-xl bg-[#15171D] border-l-4 border-l-[#7667F5] border border-[#242731] space-y-2">
                  <h4 className="text-xs font-semibold text-[#7667F5]">Important Strategic Takeaway</h4>
                  <p className="text-xs text-[#F3F4F6] leading-relaxed">
                    Source-grounded RAG eliminates hallucinations by restricting query answers strictly to verified document chunks.
                  </p>
                </div>
              </TabsContent>
            </Tabs>
          </div>

          <div className="pt-4 border-t border-[#242731] flex items-center justify-between text-xs text-[#9296A3]">
            <span>Status: Succeeded (Attempt 1)</span>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => navigate('/assistant')}
              className="text-xs text-[#7667F5] hover:text-[#8679F7]"
            >
              Start Q&A in Assistant &rarr;
            </Button>
          </div>
        </Card>
      </div>
    </div>
  );
};
