import React from 'react';
import { User, Shield, Sparkles, HardDrive } from 'lucide-react';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { useAuth } from '@/context/AuthContext';

export const SettingsPage: React.FC = () => {
  const { user, isAnonymous } = useAuth();

  return (
    <div className="space-y-6 max-w-4xl mx-auto animate-fade-in">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight text-[#F3F4F6]">Settings</h1>
        <p className="text-sm text-[#9296A3] mt-0.5">Manage your profile, security, and AI workspace preferences.</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-12 gap-6">
        {/* Left Section Navigation */}
        <div className="md:col-span-4 space-y-1">
          <Button variant="secondary" className="w-full justify-start text-xs bg-[#191B22] text-[#F3F4F6] border-[#242731]">
            <User className="w-4 h-4 mr-2 text-[#7667F5]" /> Profile
          </Button>
          <Button variant="ghost" className="w-full justify-start text-xs text-[#9296A3] hover:text-[#F3F4F6]">
            <Shield className="w-4 h-4 mr-2" /> Security & Sessions
          </Button>
          <Button variant="ghost" className="w-full justify-start text-xs text-[#9296A3] hover:text-[#F3F4F6]">
            <Sparkles className="w-4 h-4 mr-2 text-[#36C8D8]" /> AI Preferences
          </Button>
          <Button variant="ghost" className="w-full justify-start text-xs text-[#9296A3] hover:text-[#F3F4F6]">
            <HardDrive className="w-4 h-4 mr-2" /> Storage & Plan
          </Button>
        </div>

        {/* Right Settings Content Card */}
        <Card className="md:col-span-8 p-6 space-y-6">
          <div>
            <h2 className="text-base font-semibold text-[#F3F4F6]">Profile Details</h2>
            <p className="text-xs text-[#9296A3] mt-1">Update your account credentials and personal information.</p>
          </div>

          <div className="space-y-4">
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-[#F3F4F6]">Full Name</label>
              <Input defaultValue={user?.name || 'Alex Morgan'} className="h-9" />
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-medium text-[#F3F4F6]">Email Address</label>
              <Input defaultValue={user?.email || 'alex@cloudvault.io'} disabled className="h-9 opacity-75" />
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-medium text-[#F3F4F6]">Account Type</label>
              <p className="text-xs text-[#36C8D8] font-mono">
                {isAnonymous ? 'Anonymous Guest Vault (24h Token Expiry)' : 'Standard User Account (JWT + Salt)'}
              </p>
            </div>
          </div>

          <div className="pt-4 border-t border-[#242731] flex justify-end">
            <Button size="sm" className="bg-[#7667F5] hover:bg-[#8679F7] text-white">
              Save Changes
            </Button>
          </div>
        </Card>
      </div>
    </div>
  );
};
