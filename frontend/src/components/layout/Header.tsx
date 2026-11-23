import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Search, Bell, Menu, User, Settings, LogOut } from 'lucide-react';
import { Input } from '@/components/ui/input';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { useAuth } from '@/context/AuthContext';

interface HeaderProps {
  onOpenMobileMenu?: () => void;
}

export const Header: React.FC<HeaderProps> = ({ onOpenMobileMenu }) => {
  const navigate = useNavigate();
  const { user, isAnonymous, logout } = useAuth();
  const [searchQuery, setSearchQuery] = useState('');

  const handleSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (searchQuery.trim()) {
      navigate(`/search?q=${encodeURIComponent(searchQuery.trim())}`);
    }
  };

  return (
    <header className="h-16 px-6 bg-[#15171D] border-b border-[#242731] flex items-center justify-between gap-4 sticky top-0 z-30 select-none">
      {/* Left: Mobile Toggle & Global Search Bar */}
      <div className="flex items-center gap-3 flex-1 max-w-[480px]">
        {/* Mobile Hamburger Button */}
        <button
          onClick={onOpenMobileMenu}
          className="md:hidden p-2 text-[#9296A3] hover:text-[#F3F4F6] hover:bg-[#191B22] rounded-lg transition-colors"
          aria-label="Toggle navigation"
        >
          <Menu className="w-5 h-5" />
        </button>

        {/* Global Command / Search Input */}
        <form onSubmit={handleSearchSubmit} className="relative w-full">
          <Input
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search or ask AI... ⌘K"
            icon={<Search className="w-4 h-4 text-[#9296A3]" />}
            className="h-9 bg-[#191B22] border-[#242731] text-xs placeholder:text-[#9296A3] focus-visible:border-[#7667F5] pr-12 rounded-lg"
          />
          <kbd className="absolute right-2.5 top-2 text-[10px] text-[#9296A3] bg-[#242731] px-1.5 py-0.5 rounded border border-[#323644] font-mono pointer-events-none">
            ⌘K
          </kbd>
        </form>
      </div>

      {/* Right Controls */}
      <div className="flex items-center gap-3">
        {/* Indexing Indicator Badge (Live status) */}
        <div className="hidden sm:flex items-center gap-2 px-2.5 py-1 rounded-full bg-[#36C8D8]/10 border border-[#36C8D8]/20 text-[11px] text-[#36C8D8]">
          <span className="w-2 h-2 rounded-full bg-[#36C8D8] animate-pulse" />
          <span className="font-medium">AI Vault Online</span>
        </div>

        {/* Notification Bell */}
        <div className="relative">
          <button
            className="p-2 rounded-lg text-[#9296A3] hover:text-[#F3F4F6] hover:bg-[#191B22] transition-colors relative"
            aria-label="Notifications"
          >
            <Bell className="w-4 h-4" />
            <span className="absolute top-1.5 right-1.5 w-2 h-2 rounded-full bg-[#7667F5]" />
          </button>
        </div>

        {/* User Profile Dropdown */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button className="flex items-center gap-2 p-1 pl-2 rounded-lg hover:bg-[#191B22] transition-colors focus:outline-none">
              <span className="hidden sm:inline text-xs font-medium text-[#F3F4F6]">
                {user?.name || (isAnonymous ? 'Guest' : 'Alex Morgan')}
              </span>
              <div className="w-8 h-8 rounded-full bg-gradient-to-tr from-[#7667F5] to-[#36C8D8] p-[1.5px] flex items-center justify-center">
                <div className="w-full h-full rounded-full bg-[#191B22] flex items-center justify-center text-xs font-semibold text-[#F3F4F6]">
                  {user?.name ? user.name[0].toUpperCase() : 'A'}
                </div>
              </div>
            </button>
          </DropdownMenuTrigger>

          <DropdownMenuContent align="end" className="w-56 bg-[#191B22] border-[#242731] p-1.5">
            <div className="px-2 py-1.5 border-b border-[#242731] mb-1">
              <p className="text-sm font-medium text-[#F3F4F6]">{user?.name || 'Alex Morgan'}</p>
              <p className="text-xs text-[#9296A3] truncate">{user?.email || 'alex@cloudvault.io'}</p>
              {isAnonymous && (
                <span className="mt-1 inline-block text-[10px] px-1.5 py-0.5 rounded bg-[#F59E0B]/20 text-[#F59E0B] font-medium">
                  Anonymous Guest
                </span>
              )}
            </div>

            <DropdownMenuItem onClick={() => navigate('/settings')} className="cursor-pointer">
              <User className="w-4 h-4 mr-2 text-[#9296A3]" /> Profile & Plan
            </DropdownMenuItem>

            <DropdownMenuItem onClick={() => navigate('/settings')} className="cursor-pointer">
              <Settings className="w-4 h-4 mr-2 text-[#9296A3]" /> AI Preferences
            </DropdownMenuItem>

            <DropdownMenuSeparator className="bg-[#242731]" />

            <DropdownMenuItem onClick={() => logout()} danger className="cursor-pointer text-[#EF4444]">
              <LogOut className="w-4 h-4 mr-2" /> Sign Out
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </header>
  );
};
