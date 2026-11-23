import React from 'react';
import { Card } from '@/components/ui/card';

import { Progress } from '@/components/ui/progress';

export const AnalyticsPage: React.FC = () => {
  return (
    <div className="space-y-8 animate-fade-in">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight text-[#F3F4F6]">Analytics Dashboard</h1>
          <p className="text-sm text-[#9296A3] mt-0.5">Real-time visibility into storage consumption and AI processing.</p>
        </div>
        <span className="text-xs text-[#9296A3] bg-[#191B22] border border-[#242731] px-3 py-1.5 rounded-lg">
          Last 30 Days
        </span>
      </div>

      {/* KPI Row (4 Cards) */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">
        <Card className="p-5 space-y-2">
          <span className="text-xs text-[#9296A3] font-medium">Storage Used</span>
          <p className="text-2xl font-bold text-[#F3F4F6] tabular-nums">64.2 GB</p>
          <Progress value={64.2} className="h-1.5" />
        </Card>

        <Card className="p-5 space-y-2">
          <span className="text-xs text-[#9296A3] font-medium">Available</span>
          <p className="text-2xl font-bold text-[#F3F4F6] tabular-nums">35.8 GB</p>
          <span className="text-xs text-[#22C55E]">Healthy headroom</span>
        </Card>

        <Card className="p-5 space-y-2">
          <span className="text-xs text-[#9296A3] font-medium">Total Files</span>
          <p className="text-2xl font-bold text-[#F3F4F6] tabular-nums">1,284</p>
          <span className="text-xs text-[#36C8D8]">100% Vectorized</span>
        </Card>

        <Card className="p-5 space-y-2">
          <span className="text-xs text-[#9296A3] font-medium">AI Summaries</span>
          <p className="text-2xl font-bold text-[#F3F4F6] tabular-nums">342</p>
          <span className="text-xs text-[#7667F5]">llama3.2:3b</span>
        </Card>
      </div>

      {/* Charts Preview Section */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        <Card className="lg:col-span-7 p-6 space-y-4">
          <h2 className="text-base font-semibold text-[#F3F4F6]">File Uploads Over Time</h2>
          <div className="h-48 rounded-lg bg-[#15171D] border border-[#242731] flex items-center justify-center text-xs text-[#9296A3]">
            [Interactive Recharts AreaChart &middot; Indigo to Cyan Gradient]
          </div>
        </Card>

        <Card className="lg:col-span-5 p-6 space-y-4">
          <h2 className="text-base font-semibold text-[#F3F4F6]">Storage by File Type</h2>
          <div className="h-48 rounded-lg bg-[#15171D] border border-[#242731] flex items-center justify-center text-xs text-[#9296A3]">
            [Interactive Donut Chart &middot; PDFs 48%, Spreadsheets 24%]
          </div>
        </Card>
      </div>
    </div>
  );
};
