import React, { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { Eye, EyeOff, Lock, Mail, User, AlertCircle, Loader2, UserCheck, Info } from 'lucide-react';
import { Logo } from '@/components/brand/Logo';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { useAuth } from '@/context/AuthContext';

export const RegisterPage: React.FC = () => {
  const navigate = useNavigate();
  const { register, loginAsGuest } = useAuth();

  const [fullName, setFullName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [isGuestLoading, setIsGuestLoading] = useState(false);

  // Simple password strength calculation
  const getPasswordStrength = (pass: string) => {
    if (!pass) return { score: 0, label: '', color: 'bg-transparent' };
    let score = 0;
    if (pass.length >= 8) score += 1;
    if (/[A-Z]/.test(pass)) score += 1;
    if (/[0-9]/.test(pass)) score += 1;
    if (/[^A-Za-z0-9]/.test(pass)) score += 1;

    if (score <= 1) return { score: 25, label: 'Weak', color: 'bg-[#EF4444]' };
    if (score === 2) return { score: 50, label: 'Fair', color: 'bg-[#F59E0B]' };
    if (score === 3) return { score: 75, label: 'Good', color: 'bg-[#7667F5]' };
    return { score: 100, label: 'Strong', color: 'bg-[#22C55E]' };
  };

  const strength = getPasswordStrength(password);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!fullName.trim() || !email.trim() || !password.trim()) {
      setError('Please fill in all required fields.');
      return;
    }
    if (password.length < 8) {
      setError('Password must be at least 8 characters.');
      return;
    }

    setError(null);
    setIsLoading(true);

    try {
      // Derive a clean, alphanumeric username from the email prefix or full name
      const username = (email.trim().split('@')[0] || fullName.trim()).replace(/[^a-zA-Z0-9_-]/g, '_');
      await register(username, email.trim(), password);
      navigate('/dashboard');
    } catch (err: any) {
      const msg = err.message || '';
      if (msg.includes('too many requests') || msg.includes('429')) {
        setError('Server registration rate limit reached (max 5 per hour). Please launch an Instant Guest Vault to continue without waiting.');
      } else if (msg.includes('already exists') || msg.includes('duplicate') || msg.includes('registration failed')) {
        setError('Registration failed: An account with this email or username may already exist, or registration is restricted. Try signing in or use Instant Guest Vault.');
      } else {
        setError(msg || 'Registration failed. Please try again.');
      }
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
      <div className="absolute top-1/3 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[480px] h-[320px] bg-gradient-to-tr from-[#7667F5]/15 via-[#36C8D8]/10 to-transparent blur-[110px] rounded-full pointer-events-none -z-10" />

      {/* Main Register Card */}
      <div className="w-full max-w-[420px] bg-[#191B22] border border-[#242731] rounded-2xl p-8 shadow-2xl relative">
        {/* Header */}
        <div className="flex flex-col items-center text-center mb-6">
          <Link to="/" className="mb-4">
            <Logo size="lg" />
          </Link>
          <h1 className="text-xl font-semibold text-[#F3F4F6]">Create your vault</h1>
          <p className="text-xs text-[#9296A3] mt-1">Get 100 GB of intelligent, encrypted cloud storage</p>
        </div>

        {/* Architecture Notice */}
        <div className="mb-6 p-3 rounded-xl bg-[#7667F5]/10 border border-[#7667F5]/20 text-xs text-[#9296A3] space-y-1">
          <div className="flex items-center gap-1.5 text-[#F3F4F6] font-medium">
            <Info className="w-3.5 h-3.5 text-[#36C8D8]" /> Standard Registration Notice
          </div>
          <p className="leading-relaxed">
            Standard email registration is in preview with strict backend rate-limiting (5/hour). For instant access to all AI features without restrictions, use the <strong className="text-[#36C8D8]">Instant Guest Vault</strong>.
          </p>
        </div>

        {/* Error Alert Banner */}
        {error && (
          <div className="mb-6 p-3 rounded-lg bg-[#EF4444]/10 border border-[#EF4444]/20 flex items-start gap-2.5 text-xs text-[#EF4444]">
            <AlertCircle className="w-4 h-4 shrink-0 mt-0.5" />
            <span>{error}</span>
          </div>
        )}

        {/* Form */}
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-[#F3F4F6]">Full Name</label>
            <Input
              type="text"
              value={fullName}
              onChange={(e) => setFullName(e.target.value)}
              placeholder="Alex Morgan"
              icon={<User className="w-4 h-4" />}
              required
              disabled={isLoading || isGuestLoading}
            />
          </div>

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
            <label className="text-xs font-medium text-[#F3F4F6]">Password</label>
            <div className="relative">
              <Input
                type={showPassword ? 'text' : 'password'}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="At least 8 characters"
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

            {/* Password Strength Indicator */}
            {password && (
              <div className="pt-1.5 space-y-1">
                <div className="h-1 w-full bg-[#242731] rounded-full overflow-hidden">
                  <div
                    className={`h-full transition-all duration-300 ${strength.color}`}
                    style={{ width: `${strength.score}%` }}
                  />
                </div>
                <div className="flex justify-between text-[10px] text-[#9296A3]">
                  <span>Strength: <strong className="text-[#F3F4F6]">{strength.label}</strong></span>
                  <span>Min. 8 chars</span>
                </div>
              </div>
            )}
          </div>

          <Button
            type="submit"
            className="w-full bg-[#7667F5] hover:bg-[#8679F7] text-white font-medium h-10 mt-2"
            disabled={isLoading || isGuestLoading}
          >
            {isLoading ? (
              <>
                <Loader2 className="w-4 h-4 mr-2 animate-spin" /> Creating account...
              </>
            ) : (
              'Create Account'
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
          Already have an account?{' '}
          <Link to="/login" className="text-[#7667F5] font-medium hover:underline">
            Sign in
          </Link>
        </p>
      </div>
    </div>
  );
};
