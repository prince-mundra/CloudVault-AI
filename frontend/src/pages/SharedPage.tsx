import React from 'react';
import { ShieldCheck, Plus, Copy } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';

export const SharedPage: React.FC = () => {
  return (
    <div className="space-y-8 animate-fade-in">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2">
            <h1 className="text-2xl font-semibold tracking-tight text-[#F3F4F6]">Secure File Sharing</h1>
            <span className="text-[11px] px-2.5 py-0.5 rounded-full bg-[#36C8D8]/10 text-[#36C8D8] border border-[#36C8D8]/20 font-medium flex items-center gap-1">
              <ShieldCheck className="w-3.5 h-3.5" /> AES-256-GCM Encrypted
            </span>
          </div>
          <p className="text-sm text-[#9296A3] mt-0.5">Manage permissions, link expirations, and active collaborators.</p>
        </div>

        <Button className="bg-[#7667F5] hover:bg-[#8679F7] text-white flex items-center gap-2 text-xs">
          <Plus className="w-4 h-4" /> Share File
        </Button>
      </div>

      {/* Metric Summary Strip */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-5">
        <Card className="p-4">
          <span className="text-xs text-[#9296A3]">Active Links</span>
          <p className="text-xl font-bold text-[#F3F4F6] mt-1 tabular-nums">18</p>
        </Card>
        <Card className="p-4">
          <span className="text-xs text-[#9296A3]">Recipients</span>
          <p className="text-xl font-bold text-[#F3F4F6] mt-1 tabular-nums">24</p>
        </Card>
        <Card className="p-4">
          <span className="text-xs text-[#9296A3]">Expiring Soon</span>
          <p className="text-xl font-bold text-[#F59E0B] mt-1 tabular-nums">3</p>
        </Card>
      </div>

      {/* Permissions Table Card */}
      <Card className="p-6 space-y-4">
        <h2 className="text-base font-semibold text-[#F3F4F6]">Shared Documents</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead className="text-[#9296A3] border-b border-[#242731]">
              <tr>
                <th className="pb-3 font-medium">Document</th>
                <th className="pb-3 font-medium">Recipient</th>
                <th className="pb-3 font-medium">Access</th>
                <th className="pb-3 font-medium">Expires</th>
                <th className="pb-3 font-medium text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[#242731] text-[#F3F4F6]">
              {[
                { doc: 'Annual_Business_Strategy_2026.pdf', email: 'sarah@acme.io', role: 'Editor', expires: 'Dec 15, 2026' },
                { doc: 'Financial_Analysis.xlsx', email: 'david@acme.io', role: 'Viewer', expires: 'Nov 30, 2026' },
                { doc: 'Project_Report.pdf', email: 'john@cloudvault.io', role: 'Editor', expires: 'Never' },
              ].map((row, idx) => (
                <tr key={idx} className="hover:bg-[#15171D] transition-colors">
                  <td className="py-3.5 font-medium">{row.doc}</td>
                  <td className="py-3.5 text-[#9296A3]">{row.email}</td>
                  <td className="py-3.5">
                    <span className="px-2 py-0.5 rounded text-[11px] bg-[#7667F5]/20 text-[#8679F7]">
                      {row.role}
                    </span>
                  </td>
                  <td className="py-3.5 text-[#9296A3]">{row.expires}</td>
                  <td className="py-3.5 text-right space-x-2">
                    <Button variant="ghost" size="sm" className="h-7 px-2 text-[#9296A3] hover:text-[#F3F4F6]">
                      <Copy className="w-3.5 h-3.5" />
                    </Button>
                    <Button variant="ghost" size="sm" className="h-7 px-2 text-[#EF4444] hover:bg-[#EF4444]/10">
                      Revoke
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>
    </div>
  );
};
