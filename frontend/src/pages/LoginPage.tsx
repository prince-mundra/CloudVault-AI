import React, { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { Eye, EyeOff, Lock, Mail, AlertCircle, Loader2, UserCheck } from 'lucide-react';
import { Logo } from '@/components/brand/Logo';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { useAuth } from '@/context/AuthContext';

export const LoginPage: React.FC = () => {
  const navigate = useNavigate();
  const { login, loginAsGuest } = useAuth();

  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [isGuestLoading, setIsGuestLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!email.trim() || !password.trim()) {
      setError('Please provide both email and password.');
      return;
    }

    setError(null);
    setIsLoading(true);

    try {
      await login(email.trim(), password);
      navigate('/dashboard');
    } catch (err: any) {
      setError(err.message || 'Invalid email or password.');
    } finally {
      setIsLoading(false);
    }
  };

  const handleGuestLogin = async () => {
    setError(null);
    setIsGuestLoading(true);
    try {
      await loginAsGuest();
      navigate('/dashboard');
    } catch (err: any) {
      setError(err.message || 'Failed to initialize guest session.');
    } finally {
      setIsGuestLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-[#101114] flex flex-col justify-center items-center px-4 relative selection:bg-[#7667F5]/30">
      {/* Subtle Ambient Radial Glow */}
      <div className="absolute top-1/3 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[450px] h-[300px] bg-gradient-to-tr from-[#7667F5]/15 to-[#36C8D8]/10 blur-[100px] rounded-full pointer-events-none -z-10" />

      {/* Main Login Card */}
      <div className="w-full max-w-[420px] bg-[#191B22] border border-[#242731] rounded-2xl p-8 shadow-2xl relative">
        {/* Header */}
        <div className="flex flex-col items-center text-center mb-8">
          <Link to="/" className="mb-4">
            <Logo size="lg" />
          </Link>
          <h1 className="text-xl font-semibold text-[#F3F4F6]">Welcome back</h1>
          <p className="text-xs text-[#9296A3] mt-1">Sign in to your encrypted workspace</p>
        </div>

        {/* Error Alert Banner */}
        {error && (
          <div className="mb-6 p-3 rounded-lg bg-[#EF4444]/10 border border-[#EF4444]/20 flex items-start gap-2.5 text-xs text-[#EF4444]">
            <AlertCircle className="w-4 h-4 shrink-0 mt-0.5" />
            <span>{error}</span>
          </div>
        )}

        {/* Login Form */}
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-[#F3F4F6]">Email Address</label>
            <Input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="alex@cloudvault.io"
              icon={<Mail className="w-4 h-4" />}
              required
              disabled={isLoading || isGuestLoading}
            />
          </div>

          <div className="space-y-1.5">
            <div className="flex items-center justify-between">
              <label className="text-xs font-medium text-[#F3F4F6]">Password</label>
              <a href="#forgot" className="text-[11px] text-[#7667F5] hover:underline">
                Forgot password?
              </a>
            </div>
            <div className="relative">
              <Input
                type={showPassword ? 'text' : 'password'}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="••••••••"
                icon={<Lock className="w-4 h-4" />}
                required
                disabled={isLoading || isGuestLoading}
              />
              <button
                type="button"
                onClick={() => setShowPassword(!showPassword)}
                className="absolute right-3 top-2.5 text-[#9296A3] hover:text-[#F3F4F6] transition-colors"
                tabIndex={-1}
              >
                {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
              </button>
            </div>
          </div>

          <Button
            type="submit"
            className="w-full bg-[#7667F5] hover:bg-[#8679F7] text-white font-medium h-10 mt-2"
            disabled={isLoading || isGuestLoading}
          >
            {isLoading ? (
              <>
                <Loader2 className="w-4 h-4 mr-2 animate-spin" /> Signing in...
              </>
            ) : (
              'Sign In'
            )}
          </Button>
        </form>

        {/* Divider */}
        <div className="relative my-6 text-center text-xs text-[#9296A3]">
          <div className="absolute inset-0 flex items-center">
            <div className="w-full border-t border-[#242731]" />
          </div>
          <span className="relative bg-[#191B22] px-3">or continue with</span>
        </div>

        {/* 1-Click Anonymous Guest Access */}
        <Button
          type="button"
          variant="secondary"
          onClick={handleGuestLogin}
          className="w-full bg-[#15171D] border-[#36C8D8]/30 hover:border-[#36C8D8]/60 hover:bg-[#1F222B] text-xs h-10 flex items-center justify-center gap-2 group"
          disabled={isLoading || isGuestLoading}
        >
          {isGuestLoading ? (
            <Loader2 className="w-4 h-4 animate-spin text-[#36C8D8]" />
          ) : (
            <UserCheck className="w-4 h-4 text-[#36C8D8]" />
          )}
          <span>Instant Anonymous Guest Vault</span>
          <span className="text-[10px] text-[#36C8D8] font-mono bg-[#36C8D8]/10 px-1.5 py-0.5 rounded ml-1">
            Recommended
          </span>
        </Button>

        {/* Footer Toggle */}
        <p className="mt-8 text-center text-xs text-[#9296A3]">
          Don't have an account?{' '}
          <Link to="/register" className="text-[#7667F5] font-medium hover:underline">
            Create account
          </Link>
        </p>
      </div>
    </div>
  );
};
