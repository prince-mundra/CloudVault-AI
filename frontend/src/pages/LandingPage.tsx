import React from 'react';
import { useNavigate } from 'react-router-dom';
import {
  BrainCircuit,
  Sparkles,
  Search,
  ShieldCheck,
  ArrowRight,
  UserCheck,
  CheckCircle2,
  FileText,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Logo } from '@/components/brand/Logo';
import { useAuth } from '@/context/AuthContext';

export const LandingPage: React.FC = () => {
  const navigate = useNavigate();
  const { loginAsGuest, isAuthenticated } = useAuth();

  const handleGuestAccess = async () => {
    try {
      await loginAsGuest();
      navigate('/dashboard');
    } catch (err) {
      console.error('Guest access failed:', err);
      navigate('/register');
    }
  };

  return (
    <div className="min-h-screen bg-[#101114] text-[#F3F4F6] selection:bg-[#7667F5]/30 selection:text-[#F3F4F6]">
      {/* Top Sticky Navbar */}
      <header className="sticky top-0 z-50 h-16 px-6 md:px-12 bg-[#101114]/80 backdrop-blur-md border-b border-[#242731] flex items-center justify-between">
        <Logo size="md" />

        {/* Center Anchor Links */}
        <nav className="hidden md:flex items-center gap-8 text-sm text-[#9296A3]">
          <a href="#features" className="hover:text-[#F3F4F6] transition-colors">Features</a>
          <a href="#intelligence" className="hover:text-[#F3F4F6] transition-colors">Intelligence</a>
          <a href="#security" className="hover:text-[#F3F4F6] transition-colors">Security</a>
          <a href="#pricing" className="hover:text-[#F3F4F6] transition-colors">Pricing</a>
        </nav>

        {/* Right Action CTAs */}
        <div className="flex items-center gap-3">
          {isAuthenticated ? (
            <Button onClick={() => navigate('/dashboard')} className="bg-[#7667F5] hover:bg-[#8679F7] rounded-full px-5 text-sm">
              Open Vault <ArrowRight className="w-4 h-4 ml-1.5" />
            </Button>
          ) : (
            <>
              <Button
                variant="ghost"
                onClick={() => navigate('/login')}
                className="text-sm text-[#9296A3] hover:text-[#F3F4F6]"
              >
                Sign In
              </Button>
              <Button
                onClick={() => navigate('/register')}
                className="bg-[#7667F5] hover:bg-[#8679F7] text-white rounded-full px-5 text-sm font-medium shadow-md shadow-[#7667F5]/20"
              >
                Get Started
              </Button>
            </>
          )}
        </div>
      </header>

      {/* Hero Section */}
      <section className="relative px-6 pt-24 pb-20 md:pt-32 md:pb-28 max-w-5xl mx-auto text-center">
        {/* Subtle Ambient Radial Glow */}
        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[600px] h-[350px] bg-gradient-to-tr from-[#7667F5]/20 via-[#36C8D8]/15 to-transparent blur-[120px] rounded-full pointer-events-none -z-10" />

        {/* Hero Badge */}
        <div className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full bg-[#191B22] border border-[#242731] text-xs font-medium text-[#9296A3] mb-8">
          <Sparkles className="w-3.5 h-3.5 text-[#36C8D8]" />
          <span>Intelligent Cloud Storage. Powered by AI.</span>
        </div>

        {/* Headline */}
        <h1 className="text-4xl sm:text-5xl md:text-6xl font-bold tracking-tight text-[#F3F4F6] max-w-3xl mx-auto leading-[1.15]">
          Your Files. <span className="bg-gradient-to-r from-[#F3F4F6] via-[#7667F5] to-[#36C8D8] bg-clip-text text-transparent">Your Intelligence.</span>
        </h1>

        {/* Subtitle */}
        <p className="mt-6 text-base sm:text-lg text-[#9296A3] max-w-2xl mx-auto leading-relaxed">
          AI-powered cloud storage that organizes, summarizes, and answers questions from your documents with verifiable, source-grounded citations.
        </p>

        {/* Action Buttons */}
        <div className="mt-10 flex flex-wrap items-center justify-center gap-4">
          <Button
            size="lg"
            onClick={() => navigate('/register')}
            className="bg-[#7667F5] hover:bg-[#8679F7] text-white rounded-full px-7 font-medium shadow-lg shadow-[#7667F5]/25"
          >
            Get Started Free
          </Button>

          <Button
            size="lg"
            variant="secondary"
            onClick={handleGuestAccess}
            className="bg-[#191B22] text-[#F3F4F6] border border-[#242731] hover:bg-[#1F222B] hover:border-[#323644] rounded-full px-6 flex items-center gap-2"
          >
            <UserCheck className="w-4 h-4 text-[#36C8D8]" />
            Try as Guest (Instant)
          </Button>

          <a href="#features">
            <Button
              size="lg"
              variant="ghost"
              className="text-[#9296A3] hover:text-[#F3F4F6] rounded-full px-6"
            >
              Explore Features
            </Button>
          </a>
        </div>

        {/* Trust Badges */}
        <div className="mt-12 flex flex-wrap items-center justify-center gap-6 text-xs text-[#9296A3]">
          <span className="flex items-center gap-1.5"><CheckCircle2 className="w-4 h-4 text-[#22C55E]" /> Local pgvector Indexing</span>
          <span className="flex items-center gap-1.5"><CheckCircle2 className="w-4 h-4 text-[#22C55E]" /> Zero-Data Retention Models</span>
          <span className="flex items-center gap-1.5"><CheckCircle2 className="w-4 h-4 text-[#22C55E]" /> Server-Side AES-256-GCM</span>
        </div>

        {/* Workspace Preview Window Mockup */}
        <div className="mt-16 relative rounded-2xl p-1.5 bg-gradient-to-b from-[#242731] to-[#15171D] shadow-2xl border border-[#242731]">
          <div className="rounded-xl bg-[#15171D] border border-[#242731] overflow-hidden text-left">
            {/* Window Title Bar */}
            <div className="h-10 px-4 bg-[#191B22] border-b border-[#242731] flex items-center justify-between">
              <div className="flex items-center gap-2">
                <span className="w-3 h-3 rounded-full bg-[#EF4444]/60" />
                <span className="w-3 h-3 rounded-full bg-[#F59E0B]/60" />
                <span className="w-3 h-3 rounded-full bg-[#22C55E]/60" />
              </div>
              <span className="text-xs font-mono text-[#9296A3]">app.cloudvault.ai/dashboard</span>
              <div className="w-12" />
            </div>

            {/* Mock Dashboard Body */}
            <div className="p-6 grid grid-cols-1 md:grid-cols-12 gap-6 bg-[#101114]">
              {/* Left Column: Recent Files List */}
              <div className="md:col-span-7 space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-semibold text-[#F3F4F6] uppercase tracking-wider">Recent Documents</span>
                  <span className="text-xs text-[#36C8D8]">4 Active</span>
                </div>

                {[
                  { name: 'Annual_Business_Strategy_2026.pdf', size: '8.4 MB', time: '10m ago', indexed: true },
                  { name: 'Financial_Analysis_Q4.xlsx', size: '2.8 MB', time: '1h ago', indexed: true },
                  { name: 'Product_Roadmap_2027.docx', size: '1.9 MB', time: 'Yesterday', indexed: true },
                ].map((doc, idx) => (
                  <div key={idx} className="p-3 rounded-lg bg-[#191B22] border border-[#242731] flex items-center justify-between hover:border-[#323644] transition-colors">
                    <div className="flex items-center gap-3">
                      <FileText className="w-4 h-4 text-[#7667F5]" />
                      <div>
                        <p className="text-xs font-medium text-[#F3F4F6] truncate max-w-[200px]">{doc.name}</p>
                        <p className="text-[10px] text-[#9296A3]">{doc.size} · {doc.time}</p>
                      </div>
                    </div>
                    <span className="text-[10px] px-2 py-0.5 rounded-full bg-[#36C8D8]/15 text-[#36C8D8] border border-[#36C8D8]/30 font-medium">
                      Indexed
                    </span>
                  </div>
                ))}
              </div>

              {/* Right Column: AI Intelligence Preview */}
              <div className="md:col-span-5 p-4 rounded-xl bg-[#191B22] border border-[#242731] flex flex-col justify-between">
                <div>
                  <div className="flex items-center gap-2 text-xs font-semibold text-[#36C8D8] mb-2">
                    <Sparkles className="w-3.5 h-3.5" /> AI Executive Summary
                  </div>
                  <p className="text-xs text-[#9296A3] leading-relaxed">
                    The uploaded strategy document outlines a <strong className="text-[#F3F4F6]">24% YoY ARR target</strong> with APAC regional expansion commencing Q2 2026.
                  </p>
                </div>
                <div className="mt-4 pt-3 border-t border-[#242731] flex items-center justify-between text-[11px] text-[#9296A3]">
                  <span>Model: llama3.2:3b</span>
                  <span className="text-[#7667F5] font-medium">Grounded [p.12]</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Bento Grid Feature Section */}
      <section id="features" className="py-24 px-6 md:px-12 max-w-6xl mx-auto border-t border-[#242731]">
        <div className="text-center max-w-2xl mx-auto mb-16">
          <h2 className="text-3xl font-bold tracking-tight text-[#F3F4F6]">Architected for Intelligence.</h2>
          <p className="mt-3 text-sm text-[#9296A3]">
            Every uploaded document is automatically parsed, vectorized, and accessible via conversational AI.
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          {/* Card 1 */}
          <div className="p-6 rounded-xl bg-[#191B22] border border-[#242731] hover:border-[#323644] transition-all flex flex-col justify-between group">
            <div>
              <div className="w-10 h-10 rounded-lg bg-[#7667F5]/10 border border-[#7667F5]/20 flex items-center justify-center text-[#7667F5] mb-5 group-hover:scale-105 transition-transform">
                <BrainCircuit className="w-5 h-5" />
              </div>
              <h3 className="text-base font-semibold text-[#F3F4F6]">AI Document Intelligence</h3>
              <p className="mt-2 text-xs text-[#9296A3] leading-relaxed">
                Automated document extraction, structured executive summaries, and key takeaway extraction with local llama3.2 models.
              </p>
            </div>
            <span className="mt-6 text-[11px] font-mono text-[#7667F5]">Ollama · llama3.2:3b</span>
          </div>

          {/* Card 2 */}
          <div className="p-6 rounded-xl bg-[#191B22] border border-[#242731] hover:border-[#323644] transition-all flex flex-col justify-between group">
            <div>
              <div className="w-10 h-10 rounded-lg bg-[#36C8D8]/10 border border-[#36C8D8]/20 flex items-center justify-center text-[#36C8D8] mb-5 group-hover:scale-105 transition-transform">
                <Sparkles className="w-5 h-5" />
              </div>
              <h3 className="text-base font-semibold text-[#F3F4F6]">RAG Document Chat</h3>
              <p className="mt-2 text-xs text-[#9296A3] leading-relaxed">
                Conversational Q&A across multi-document contexts with clickable page-level citations and direct source verification.
              </p>
            </div>
            <span className="mt-6 text-[11px] font-mono text-[#36C8D8]">Verifiable Citations</span>
          </div>

          {/* Card 3 */}
          <div className="p-6 rounded-xl bg-[#191B22] border border-[#242731] hover:border-[#323644] transition-all flex flex-col justify-between group">
            <div>
              <div className="w-10 h-10 rounded-lg bg-[#7667F5]/10 border border-[#7667F5]/20 flex items-center justify-center text-[#7667F5] mb-5 group-hover:scale-105 transition-transform">
                <Search className="w-5 h-5" />
              </div>
              <h3 className="text-base font-semibold text-[#F3F4F6]">Smart Semantic Search</h3>
              <p className="mt-2 text-xs text-[#9296A3] leading-relaxed">
                Meaning-based vector discovery across your entire vault powered by pgvector cosine similarity and nomic-embed-text.
              </p>
            </div>
            <span className="mt-6 text-[11px] font-mono text-[#7667F5]">pgvector · 768-dim</span>
          </div>

          {/* Card 4 */}
          <div className="p-6 rounded-xl bg-[#191B22] border border-[#242731] hover:border-[#323644] transition-all flex flex-col justify-between group">
            <div>
              <div className="w-10 h-10 rounded-lg bg-[#22C55E]/10 border border-[#22C55E]/20 flex items-center justify-center text-[#22C55E] mb-5 group-hover:scale-105 transition-transform">
                <ShieldCheck className="w-5 h-5" />
              </div>
              <h3 className="text-base font-semibold text-[#F3F4F6]">Server-Side Encrypted Storage</h3>
              <p className="mt-2 text-xs text-[#9296A3] leading-relaxed">
                Authenticated AES-GCM encrypted storage with managed keys, role-based sharing permissions, and password-protected auto-expiring links.
              </p>
            </div>
            <span className="mt-6 text-[11px] font-mono text-[#22C55E]">AES-256-GCM</span>
          </div>
        </div>
      </section>

      {/* Minimal Footer */}
      <footer className="py-12 px-6 md:px-12 border-t border-[#242731] flex flex-col sm:flex-row items-center justify-between text-xs text-[#9296A3] max-w-6xl mx-auto gap-4">
        <div className="flex items-center gap-3">
          <Logo size="sm" iconOnly />
          <span>© 2026 CloudVault AI. All rights reserved.</span>
        </div>
        <div className="flex items-center gap-6">
          <a href="#privacy" className="hover:text-[#F3F4F6]">Privacy</a>
          <a href="#terms" className="hover:text-[#F3F4F6]">Terms</a>
          <a href="#security" className="hover:text-[#F3F4F6]">Security</a>
          <a href="https://github.com" target="_blank" rel="noreferrer" className="hover:text-[#F3F4F6]">GitHub</a>
        </div>
      </footer>
    </div>
  );
};
