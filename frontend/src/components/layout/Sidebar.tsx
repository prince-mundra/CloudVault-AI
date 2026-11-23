import React, { useState, useEffect } from 'react';
import { NavLink, useLocation } from 'react-router-dom';
import {
  LayoutDashboard,
  Folder,
  Users,
  Sparkles,
  Search,
  BarChart2,
  Settings,
  HardDrive,
  LogOut,
} from 'lucide-react';
import { Logo } from '@/components/brand/Logo';
import { Progress } from '@/components/ui/progress';
import { useAuth } from '@/context/AuthContext';
import { api, StorageAnalytics } from '@/lib/api';
import { formatBytes } from '@/lib/utils';

interface SidebarProps {
  onCloseMobile?: () => void;
}

export const Sidebar: React.FC<SidebarProps> = ({ onCloseMobile }) => {
  const location = useLocation();
  const { user, isAnonymous, logout } = useAuth();
  const [stats, setStats] = useState<StorageAnalytics | null>(null);

  const fetchStorage = async () => {
    try {
      const res = await api.getStorageAnalytics();
      setStats(res.data);
    } catch {
      // Gracefully silence if unauthenticated or network failure
    }
  };

  useEffect(() => {
    fetchStorage();
    const handleUpdate = () => fetchStorage();
    window.addEventListener('cloudvault-storage-update', handleUpdate);
    return () => window.removeEventListener('cloudvault-storage-update', handleUpdate);
  }, []);

  const navItems = [
    { label: 'Overview', path: '/dashboard', icon: LayoutDashboard },
    { label: 'My Files', path: '/files', icon: Folder },
    { label: 'Shared Files', path: '/shared', icon: Users },
    { label: 'AI Assistant', path: '/assistant', icon: Sparkles },
    { label: 'Smart Search', path: '/search', icon: Search },
    { label: 'Analytics', path: '/analytics', icon: BarChart2 },
    { label: 'Settings', path: '/settings', icon: Settings },
  ];

  // Real storage metrics with 5 GB configurable quota default
  const usedBytes = stats?.used_bytes ?? 0;
  const quotaBytes = stats?.quota_bytes ?? 5368709120; // 5 GB
  const usedStr = formatBytes(usedBytes);
  const quotaStr = formatBytes(quotaBytes);
  const storagePercent = quotaBytes > 0 ? Math.min(Math.round((usedBytes / quotaBytes) * 100), 100) : 0;

  return (
    <aside className="w-[240px] shrink-0 h-screen bg-[#15171D] border-r border-[#242731] flex flex-col justify-between select-none">
      {/* Top Header & Brand */}
      <div>
        <div className="h-16 px-5 flex items-center border-b border-[#242731]">
          <NavLink to="/dashboard" onClick={onCloseMobile}>
            <Logo size="md" />
          </NavLink>
        </div>

        {/* Navigation Links */}
        <nav className="p-3 space-y-1">
          {navItems.map((item) => {
            const Icon = item.icon;
            const isActive = location.pathname.startsWith(item.path);

            return (
              <NavLink
                key={item.path}
                to={item.path}
                onClick={onCloseMobile}
                className={`relative flex items-center gap-3 px-3.5 py-2.5 rounded-lg text-sm font-medium transition-all group ${
                  isActive
                    ? 'bg-[#191B22] text-[#F3F4F6] border border-[#242731] shadow-sm'
                    : 'text-[#9296A3] hover:text-[#F3F4F6] hover:bg-[#191B22]/50'
                }`}
              >
                {/* Active Indicator Bar */}
                {isActive && (
                  <span className="absolute left-0 top-2 bottom-2 w-[3px] bg-[#7667F5] rounded-r" />
                )}

                <Icon
                  className={`w-4 h-4 transition-colors ${
                    isActive ? 'text-[#7667F5]' : 'text-[#9296A3] group-hover:text-[#F3F4F6]'
                  }`}
                />
                <span>{item.label}</span>

                {item.label === 'AI Assistant' && (
                  <span className="ml-auto w-1.5 h-1.5 rounded-full bg-[#36C8D8] animate-pulse" />
                )}
              </NavLink>
            );
          })}
        </nav>
      </div>

      {/* Bottom Storage & User Status */}
      <div className="p-4 border-t border-[#242731] space-y-4">
        {/* Storage Meter Card */}
        <div className="p-3 rounded-lg bg-[#191B22] border border-[#242731] space-y-2.5">
          <div className="flex items-center justify-between text-xs text-[#9296A3]">
            <span className="flex items-center gap-1.5 text-[#F3F4F6] font-medium">
              <HardDrive className="w-3.5 h-3.5 text-[#7667F5]" /> Storage
            </span>
            <span className="tabular-nums font-medium text-[#F3F4F6]">{usedStr} / {quotaStr}</span>
          </div>
          <Progress value={storagePercent} className="h-1.5 bg-[#242731]" />
          <div className="flex items-center justify-between text-[11px] text-[#9296A3]">
            <span>{storagePercent}% Used</span>
            {isAnonymous ? (
              <span className="text-[#F59E0B] font-medium">Guest Vault</span>
            ) : (
              <span className="text-[#36C8D8] font-medium">Encrypted</span>
            )}
          </div>
        </div>

        {/* Quick User Status */}
        <div className="flex items-center justify-between px-1 text-xs text-[#9296A3]">
          <div className="truncate max-w-[140px]">
            <p className="text-[#F3F4F6] font-medium truncate">{user?.name || 'Alex Morgan'}</p>
            <p className="text-[11px] text-[#9296A3] truncate">{user?.email || 'alex@cloudvault.io'}</p>
          </div>
          <button
            onClick={() => logout()}
            title="Sign Out"
            className="p-1.5 rounded-md hover:bg-[#191B22] text-[#9296A3] hover:text-[#EF4444] transition-colors"
          >
            <LogOut className="w-4 h-4" />
          </button>
        </div>
      </div>
    </aside>
  );
};
