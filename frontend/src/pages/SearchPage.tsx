import React, { useState } from 'react';
import { Search, Sparkles, FileText, Loader2, AlertCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { api } from '@/lib/api';
import { useNavigate } from 'react-router-dom';

export const SearchPage: React.FC = () => {
  const navigate = useNavigate();
  const [query, setQuery] = useState('cloud storage and artificial intelligence');
  const [results, setResults] = useState<any[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [hasSearched, setHasSearched] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSearch = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!query.trim()) return;

    setIsLoading(true);
    setError(null);
    setHasSearched(true);

    try {
      const res = await api.post('/api/v1/search/semantic', {
        query: query.trim(),
        limit: 5,
      });
      setResults(res.results || []);
    } catch (err: any) {
      setError(err.message || 'Semantic search request failed.');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="space-y-6 max-w-4xl mx-auto animate-fade-in">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight text-[#F3F4F6]">Smart Semantic Search</h1>
        <p className="text-sm text-[#9296A3] mt-1">
          Search by concept, topic, or question across your entire vault powered by pgvector.
        </p>
      </div>

      {/* Prominent Search Bar */}
      <form onSubmit={handleSearch} className="relative">
        <Input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Find documents discussing financial growth strategies..."
          className="h-14 pl-12 pr-32 bg-[#191B22] border-[#242731] text-base focus-visible:border-[#7667F5] rounded-xl shadow-lg"
          icon={<Search className="w-5 h-5 text-[#9296A3]" />}
        />
        <div className="absolute right-3 top-3 flex items-center gap-2">
          <span className="hidden sm:inline-flex items-center gap-1 px-2.5 py-1 rounded-full bg-[#36C8D8]/10 text-[#36C8D8] border border-[#36C8D8]/30 text-xs font-medium">
            <Sparkles className="w-3 h-3" /> Semantic
          </span>
          <Button
            type="submit"
            size="sm"
            className="bg-[#7667F5] hover:bg-[#8679F7] text-white h-8 px-4"
            disabled={isLoading}
          >
            {isLoading ? <Loader2 className="w-4 h-4 animate-spin" /> : 'Search'}
          </Button>
        </div>
      </form>

      {/* Error Banner */}
      {error && (
        <div className="p-4 rounded-xl bg-[#EF4444]/10 border border-[#EF4444]/20 flex items-center gap-3 text-xs text-[#EF4444]">
          <AlertCircle className="w-4 h-4 shrink-0" />
          <span>{error}</span>
        </div>
      )}

      {/* Results List */}
      <div className="space-y-4 pt-2">
        {isLoading && (
          <div className="text-center py-12 text-[#9296A3] text-sm flex items-center justify-center gap-2">
            <Loader2 className="w-5 h-5 animate-spin text-[#7667F5]" />
            <span>Computing embeddings and querying vector index...</span>
          </div>
        )}

        {!isLoading && hasSearched && results.length === 0 && !error && (
          <Card className="p-8 text-center text-[#9296A3] text-sm">
            No matching semantic passages found. Try broader keywords.
          </Card>
        )}

        {results.map((hit, idx) => (
          <Card
            key={idx}
            className="p-5 hover:border-[#323644] transition-colors cursor-pointer group"
            onClick={() => navigate(`/intelligence/${hit.file_id}`)}
          >
            <div className="flex items-start justify-between gap-4">
              <div className="flex items-center gap-3">
                <div className="w-9 h-9 rounded-lg bg-[#7667F5]/10 border border-[#7667F5]/20 flex items-center justify-center text-[#7667F5]">
                  <FileText className="w-4 h-4" />
                </div>
                <div>
                  <h3 className="text-sm font-semibold text-[#F3F4F6] group-hover:text-[#7667F5] transition-colors">
                    {hit.file_name}
                  </h3>
                  <p className="text-xs text-[#9296A3]">Page {hit.page_number || 1} · Chunk {hit.chunk_index + 1}</p>
                </div>
              </div>

              <span className="text-xs px-2.5 py-1 rounded-full bg-[#36C8D8]/15 text-[#36C8D8] border border-[#36C8D8]/30 font-semibold tabular-nums">
                {Math.round((hit.score || 0.85) * 100)}% Match
              </span>
            </div>

            <div className="mt-4 p-3 rounded-lg bg-[#15171D] border border-[#242731] text-xs text-[#9296A3] leading-relaxed font-sans">
              "{hit.text}"
            </div>
          </Card>
        ))}

        {!hasSearched && (
          <div className="pt-8 text-center text-xs text-[#9296A3]">
            Enter a natural-language query above to search through indexed PDF vectors.
          </div>
        )}
      </div>
    </div>
  );
};
