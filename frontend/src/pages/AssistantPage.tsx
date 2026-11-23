import React from 'react';
import { Plus, Send, Sparkles, FileText, CheckSquare, MessageSquare } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';

export const AssistantPage: React.FC = () => {
  return (
    <div className="h-[calc(100vh-120px)] flex flex-col md:flex-row gap-6 animate-fade-in">
      {/* Left Conversations Rail */}
      <Card className="w-full md:w-60 shrink-0 p-4 flex flex-col justify-between hidden sm:flex">
        <div className="space-y-4">
          <Button className="w-full bg-[#7667F5] hover:bg-[#8679F7] text-white flex items-center justify-center gap-2 text-xs">
            <Plus className="w-4 h-4" /> New Chat
          </Button>

          <div className="space-y-1">
            <span className="text-[11px] font-semibold text-[#9296A3] uppercase tracking-wider px-2">Conversations</span>
            <div className="p-2.5 rounded-lg bg-[#15171D] border border-[#242731] text-xs font-medium text-[#F3F4F6] cursor-pointer flex items-center gap-2">
              <MessageSquare className="w-3.5 h-3.5 text-[#36C8D8]" />
              <span className="truncate">Cloud Storage Test Analysis</span>
            </div>
            <div className="p-2.5 rounded-lg hover:bg-[#15171D] text-xs text-[#9296A3] cursor-pointer flex items-center gap-2 transition-colors">
              <MessageSquare className="w-3.5 h-3.5" />
              <span className="truncate">Q4 Financial Risk Analysis</span>
            </div>
          </div>
        </div>
        <span className="text-[10px] text-[#9296A3] text-center">Grounded with llama3.2</span>
      </Card>

      {/* Center Chat Area */}
      <Card className="flex-1 flex flex-col justify-between p-6 overflow-hidden">
        {/* Chat Title */}
        <div className="pb-3 border-b border-[#242731] flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Sparkles className="w-4 h-4 text-[#36C8D8]" />
            <h2 className="text-sm font-semibold text-[#F3F4F6]">Cloud Storage Test Analysis</h2>
          </div>
          <span className="text-[11px] px-2 py-0.5 rounded-full bg-[#22C55E]/10 text-[#22C55E] border border-[#22C55E]/20">
            1 Active Doc
          </span>
        </div>

        {/* Message Thread */}
        <div className="flex-1 overflow-y-auto py-6 space-y-6">
          {/* User Bubble */}
          <div className="flex justify-end">
            <div className="max-w-md p-3.5 rounded-xl rounded-tr-none bg-[#1F222B] border border-[#242731] text-sm text-[#F3F4F6]">
              What does the test document conclude about AI in cloud storage?
            </div>
          </div>

          {/* AI Response Card */}
          <div className="flex justify-start">
            <div className="max-w-xl p-5 rounded-xl rounded-tl-none bg-[#15171D] border border-[#242731] space-y-3">
              <div className="flex items-center gap-2 text-xs font-medium text-[#36C8D8]">
                <Sparkles className="w-3.5 h-3.5" /> CloudVault Assistant
              </div>
              <p className="text-sm text-[#F3F4F6] leading-relaxed">
                The document demonstrates that integrating artificial intelligence into cloud storage provides automated summarization and conversational question-answering capabilities directly from stored documents [1].
              </p>

              {/* Citations Footer */}
              <div className="pt-3 border-t border-[#242731] flex items-center gap-2 text-xs text-[#9296A3]">
                <span>Sources:</span>
                <span className="px-2 py-0.5 rounded bg-[#191B22] border border-[#242731] text-[11px] text-[#36C8D8] font-mono flex items-center gap-1">
                  <FileText className="w-3 h-3" /> cloudvault-test.pdf · Page 1 [1]
                </span>
              </div>
            </div>
          </div>
        </div>

        {/* Bottom Input */}
        <div className="pt-3 border-t border-[#242731] relative">
          <Input
            placeholder="Ask a question about your documents..."
            className="pr-12 h-11 bg-[#101114]"
          />
          <Button
            size="sm"
            className="absolute right-2 top-4.5 bg-[#7667F5] hover:bg-[#8679F7] text-white h-7 w-7 p-0"
          >
            <Send className="w-3.5 h-3.5" />
          </Button>
        </div>
      </Card>

      {/* Right Document Context Panel */}
      <Card className="w-full md:w-72 shrink-0 p-5 space-y-4 hidden lg:flex lg:flex-col">
        <div className="pb-3 border-b border-[#242731] flex items-center justify-between">
          <span className="text-xs font-semibold text-[#F3F4F6] uppercase tracking-wider">Document Context</span>
          <span className="text-xs text-[#36C8D8]">1 Selected</span>
        </div>

        <div className="space-y-2">
          <div className="p-3 rounded-lg bg-[#15171D] border border-[#7667F5]/40 flex items-center justify-between">
            <div className="flex items-center gap-2.5 truncate">
              <CheckSquare className="w-4 h-4 text-[#7667F5]" />
              <span className="text-xs font-medium text-[#F3F4F6] truncate">cloudvault-test.pdf</span>
            </div>
            <span className="text-[10px] text-[#9296A3]">775 B</span>
          </div>
        </div>

        {/* Excerpt Viewer */}
        <div className="mt-4 p-3 rounded-lg bg-[#101114] border border-[#242731] text-xs text-[#9296A3] space-y-1.5">
          <span className="text-[10px] uppercase font-semibold text-[#36C8D8]">Citation [1] Excerpt</span>
          <p className="italic text-[#F3F4F6]/90 leading-relaxed">
            "Artificial intelligence can summarize documents and answer questions."
          </p>
        </div>
      </Card>
    </div>
  );
};
