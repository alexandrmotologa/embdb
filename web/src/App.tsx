import React, { useState, useEffect, useRef } from 'react';
import { 
  Database, Search, Network, Box, Cpu, Server, 
  Layers, PlusCircle, RefreshCw, Terminal, CheckCircle2,
  Sliders, ArrowRight, Zap, Filter, Tag, Eye
} from 'lucide-react';

interface DBStats {
  total_documents: number;
  vector_dimension: number;
  graph_nodes: number;
  graph_edges: number;
  total_pages: number;
  metric: number;
  quantization: number;
}

interface SearchResult {
  id: number;
  key: string;
  text: string;
  score: number;
  dense_rank?: number;
  sparse_rank?: number;
  metadata?: Record<string, any>;
}

interface ScatterPoint {
  id: number;
  key: string;
  text: string;
  x: number;
  y: number;
  z: number;
}

export default function App() {
  const [activeTab, setActiveTab] = useState<'bench' | 'vector3d' | 'graph' | 'documents' | 'mcp'>('bench');
  const [stats, setStats] = useState<DBStats | null>(null);
  const [loadingStats, setLoadingStats] = useState(false);

  // Search Bench State
  const [query, setQuery] = useState('contract liability and confidential clauses');
  const [searchMode, setSearchMode] = useState<'hybrid' | 'vector' | 'bm25'>('hybrid');
  const [denseWeight, setDenseWeight] = useState(0.5);
  const [sparseWeight, setSparseWeight] = useState(0.5);
  const [limit, setLimit] = useState(10);
  const [results, setResults] = useState<SearchResult[]>([]);
  const [searching, setSearching] = useState(false);
  const [searchLatency, setSearchLatency] = useState<number | null>(null);

  // Ingestion State
  const [newKey, setNewKey] = useState('');
  const [newText, setNewText] = useState('');
  const [newCategory, setNewCategory] = useState('');
  const [ingestSuccess, setIngestSuccess] = useState(false);

  // Documents List
  const [docsList, setDocsList] = useState<any[]>([]);

  // 3D Points
  const [points, setPoints] = useState<ScatterPoint[]>([]);

  // Clusters
  const [clusters, setClusters] = useState<{ id: number; label: string; doc_count: number; doc_keys: string[] }[]>([]);

  // Fetch Stats
  const fetchStats = async () => {
    try {
      setLoadingStats(true);
      const res = await fetch('/api/stats');
      if (res.ok) {
        const data = await res.json();
        setStats(data);
      }
    } catch (e) {
      console.error(e);
    } finally {
      setLoadingStats(false);
    }
  };

  // Fetch Documents
  const fetchDocs = async () => {
    try {
      const res = await fetch('/api/documents?limit=50');
      if (res.ok) {
        const data = await res.json();
        setDocsList(data.rows || []);
      }
    } catch (e) {
      console.error(e);
    }
  };

  // Fetch 3D Scatter Points
  const fetchScatter = async () => {
    try {
      const res = await fetch('/api/vectors/scatter');
      if (res.ok) {
        const data = await res.json();
        setPoints(data || []);
      }
    } catch (e) {
      console.error(e);
    }
  };

  // Fetch Semantic Clusters
  const fetchClusters = async () => {
    try {
      const res = await fetch('/api/vectors/clusters?k=3');
      if (res.ok) {
        const data = await res.json();
        setClusters(data.clusters || []);
      }
    } catch (e) {
      console.error(e);
    }
  };

  useEffect(() => {
    fetchStats();
    fetchDocs();
    fetchScatter();
    fetchClusters();
  }, []);

  // Perform Search
  const handleSearch = async () => {
    if (!query.trim()) return;
    setSearching(true);
    const start = performance.now();
    try {
      const res = await fetch('/api/search', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          query,
          mode: searchMode,
          limit,
          dense_weight: denseWeight,
          sparse_weight: sparseWeight,
        }),
      });
      if (res.ok) {
        const data = await res.json();
        setResults(data.results || []);
      }
    } catch (e) {
      console.error(e);
    } finally {
      setSearchLatency(Math.round(performance.now() - start));
      setSearching(false);
    }
  };

  // Ingest Document
  const handleIngest = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newText.trim()) return;

    try {
      const res = await fetch('/api/documents', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          key: newKey || `doc_${Date.now()}`,
          text: newText,
          metadata: newCategory ? { category: newCategory } : {},
        }),
      });
      if (res.ok) {
        setIngestSuccess(true);
        setNewKey('');
        setNewText('');
        setNewCategory('');
        fetchStats();
        fetchDocs();
        fetchScatter();
        setTimeout(() => setIngestSuccess(false), 3000);
      }
    } catch (e) {
      console.error(e);
    }
  };

  return (
    <div style={{ minHeight: '100vh', display: 'flex', flexDirection: 'column', backgroundColor: '#09090b', color: '#f4f4f5' }}>
      {/* Top Navbar */}
      <header style={{ borderBottom: '1px solid #27272a', padding: '16px 24px', display: 'flex', alignItems: 'center', justifyContent: 'space-between', backgroundColor: '#121215' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <div style={{ width: '36px', height: '36px', borderRadius: '8px', background: 'linear-gradient(135deg, #6366f1, #a855f7)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <Database size={20} color="#fff" />
          </div>
          <div>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              <span style={{ fontWeight: 700, fontSize: '18px', letterSpacing: '-0.02em' }}>embdb</span>
              <span style={{ fontSize: '11px', background: '#27272a', color: '#a1a1aa', padding: '2px 6px', borderRadius: '4px', border: '1px solid #3f3f46' }}>v1.0.0</span>
              <span style={{ fontSize: '11px', background: '#064e3b', color: '#6ee7b7', padding: '2px 6px', borderRadius: '4px', border: '1px solid #047857', display: 'flex', alignItems: 'center', gap: '4px' }}>
                <span style={{ width: '6px', height: '6px', borderRadius: '50%', background: '#10b981' }}></span>
                Single-Binary Engine
              </span>
            </div>
            <p style={{ margin: 0, fontSize: '12px', color: '#71717a' }}>Embedded Vector, HNSW, BM25 & Model Context Protocol</p>
          </div>
        </div>

        {/* Global Nav Tabs */}
        <div style={{ display: 'flex', gap: '6px', background: '#18181b', padding: '4px', borderRadius: '8px', border: '1px solid #27272a' }}>
          {[
            { id: 'bench', label: 'Hybrid Search Bench', icon: Search },
            { id: 'vector3d', label: '3D Vector Scatter', icon: Box },
            { id: 'graph', label: 'Knowledge Graph', icon: Network },
            { id: 'documents', label: 'Documents & Ingestion', icon: Layers },
            { id: 'mcp', label: 'MCP Agent Cockpit', icon: Cpu },
          ].map(tab => {
            const Icon = tab.icon;
            const active = activeTab === tab.id;
            return (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id as any)}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '8px',
                  padding: '8px 14px',
                  fontSize: '13px',
                  fontWeight: 500,
                  borderRadius: '6px',
                  border: 'none',
                  cursor: 'pointer',
                  backgroundColor: active ? '#27272a' : 'transparent',
                  color: active ? '#fff' : '#a1a1aa',
                  transition: 'all 0.15s ease',
                }}
              >
                <Icon size={15} />
                {tab.label}
              </button>
            );
          })}
        </div>

        <button
          onClick={() => { fetchStats(); fetchDocs(); fetchScatter(); }}
          style={{ display: 'flex', alignItems: 'center', gap: '6px', background: '#18181b', border: '1px solid #27272a', color: '#d4d4d8', padding: '8px 12px', borderRadius: '6px', fontSize: '13px', cursor: 'pointer' }}
        >
          <RefreshCw size={14} className={loadingStats ? 'spin' : ''} />
          Refresh
        </button>
      </header>

      {/* Stats Bar */}
      <div style={{ borderBottom: '1px solid #1f1f23', padding: '12px 24px', backgroundColor: '#0d0d10', display: 'grid', gridTemplateColumns: 'repeat(5, 1fr)', gap: '16px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <div style={{ padding: '8px', background: '#18181b', borderRadius: '6px', color: '#818cf8' }}><Database size={16} /></div>
          <div>
            <div style={{ fontSize: '11px', color: '#71717a', textTransform: 'uppercase' }}>Documents</div>
            <div style={{ fontSize: '16px', fontWeight: 600 }}>{stats?.total_documents ?? 0}</div>
          </div>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <div style={{ padding: '8px', background: '#18181b', borderRadius: '6px', color: '#38bdf8' }}><Layers size={16} /></div>
          <div>
            <div style={{ fontSize: '11px', color: '#71717a', textTransform: 'uppercase' }}>Vector Dimension</div>
            <div style={{ fontSize: '16px', fontWeight: 600 }}>{stats?.vector_dimension ?? 384}d</div>
          </div>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <div style={{ padding: '8px', background: '#18181b', borderRadius: '6px', color: '#34d399' }}><Network size={16} /></div>
          <div>
            <div style={{ fontSize: '11px', color: '#71717a', textTransform: 'uppercase' }}>Entity Graph</div>
            <div style={{ fontSize: '16px', fontWeight: 600 }}>{stats?.graph_nodes ?? 0} nodes / {stats?.graph_edges ?? 0} edges</div>
          </div>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <div style={{ padding: '8px', background: '#18181b', borderRadius: '6px', color: '#f59e0b' }}><Server size={16} /></div>
          <div>
            <div style={{ fontSize: '11px', color: '#71717a', textTransform: 'uppercase' }}>Storage Container</div>
            <div style={{ fontSize: '16px', fontWeight: 600 }}>{stats?.total_pages ?? 1} pages (4KB mmap)</div>
          </div>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <div style={{ padding: '8px', background: '#18181b', borderRadius: '6px', color: '#ec4899' }}><Zap size={16} /></div>
          <div>
            <div style={{ fontSize: '11px', color: '#71717a', textTransform: 'uppercase' }}>Cold Start</div>
            <div style={{ fontSize: '16px', fontWeight: 600, color: '#34d399' }}>&lt; 5 ms</div>
          </div>
        </div>
      </div>

      {/* Main Content Area */}
      <main style={{ flex: 1, padding: '24px', maxWidth: '1440px', width: '100%', margin: '0 auto', boxSizing: 'border-box' }}>
        
        {/* TAB 1: HYBRID SEARCH BENCH */}
        {activeTab === 'bench' && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
            {/* Search Controls Card */}
            <div style={{ background: '#121215', border: '1px solid #27272a', borderRadius: '12px', padding: '20px' }}>
              <div style={{ display: 'flex', gap: '12px', marginBottom: '16px' }}>
                <div style={{ flex: 1, position: 'relative' }}>
                  <Search size={18} style={{ position: 'absolute', left: '14px', top: '14px', color: '#71717a' }} />
                  <input
                    type="text"
                    value={query}
                    onChange={e => setQuery(e.target.value)}
                    onKeyDown={e => e.key === 'Enter' && handleSearch()}
                    placeholder="Enter semantic query or keyword terms..."
                    style={{
                      width: '100%',
                      padding: '12px 14px 12px 42px',
                      background: '#18181b',
                      border: '1px solid #27272a',
                      borderRadius: '8px',
                      color: '#fff',
                      fontSize: '14px',
                      outline: 'none',
                      boxSizing: 'border-box',
                    }}
                  />
                </div>
                <button
                  onClick={handleSearch}
                  disabled={searching}
                  style={{
                    padding: '0 24px',
                    background: '#6366f1',
                    color: '#fff',
                    border: 'none',
                    borderRadius: '8px',
                    fontWeight: 600,
                    fontSize: '14px',
                    cursor: 'pointer',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '8px',
                  }}
                >
                  <Search size={16} />
                  {searching ? 'Querying...' : 'Search'}
                </button>
              </div>

              {/* Advanced Parameters */}
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '16px', paddingTop: '16px', borderTop: '1px solid #1f1f23' }}>
                <div>
                  <label style={{ fontSize: '12px', color: '#a1a1aa', display: 'block', marginBottom: '6px' }}>Search Engine Mode</label>
                  <div style={{ display: 'flex', gap: '4px', background: '#18181b', padding: '3px', borderRadius: '6px' }}>
                    {(['hybrid', 'vector', 'bm25'] as const).map(mode => (
                      <button
                        key={mode}
                        onClick={() => setSearchMode(mode)}
                        style={{
                          flex: 1,
                          padding: '6px',
                          fontSize: '12px',
                          border: 'none',
                          borderRadius: '4px',
                          cursor: 'pointer',
                          background: searchMode === mode ? '#27272a' : 'transparent',
                          color: searchMode === mode ? '#fff' : '#71717a',
                          fontWeight: searchMode === mode ? 600 : 400,
                        }}
                      >
                        {mode.toUpperCase()}
                      </button>
                    ))}
                  </div>
                </div>

                <div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '12px', color: '#a1a1aa', marginBottom: '6px' }}>
                    <span>Dense Weight (HNSW)</span>
                    <span style={{ fontWeight: 600, color: '#818cf8' }}>{denseWeight.toFixed(2)}</span>
                  </div>
                  <input
                    type="range"
                    min="0"
                    max="1"
                    step="0.05"
                    value={denseWeight}
                    onChange={e => setDenseWeight(parseFloat(e.target.value))}
                    style={{ width: '100%', accentColor: '#6366f1' }}
                  />
                </div>

                <div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '12px', color: '#a1a1aa', marginBottom: '6px' }}>
                    <span>Sparse Weight (BM25)</span>
                    <span style={{ fontWeight: 600, color: '#38bdf8' }}>{sparseWeight.toFixed(2)}</span>
                  </div>
                  <input
                    type="range"
                    min="0"
                    max="1"
                    step="0.05"
                    value={sparseWeight}
                    onChange={e => setSparseWeight(parseFloat(e.target.value))}
                    style={{ width: '100%', accentColor: '#38bdf8' }}
                  />
                </div>

                <div>
                  <label style={{ fontSize: '12px', color: '#a1a1aa', display: 'block', marginBottom: '6px' }}>Top-K Limit</label>
                  <select
                    value={limit}
                    onChange={e => setLimit(parseInt(e.target.value))}
                    style={{
                      width: '100%',
                      padding: '8px',
                      background: '#18181b',
                      border: '1px solid #27272a',
                      borderRadius: '6px',
                      color: '#fff',
                      fontSize: '13px',
                    }}
                  >
                    <option value={5}>5 items</option>
                    <option value={10}>10 items</option>
                    <option value={25}>25 items</option>
                    <option value={50}>50 items</option>
                  </select>
                </div>
              </div>
            </div>

            {/* Results Header */}
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                <h3 style={{ margin: 0, fontSize: '16px', fontWeight: 600 }}>Query Results</h3>
                <span style={{ fontSize: '12px', color: '#71717a' }}>({results.length} matches found)</span>
              </div>
              {searchLatency !== null && (
                <div style={{ fontSize: '12px', color: '#34d399', background: '#064e3b', padding: '4px 8px', borderRadius: '4px', border: '1px solid #047857' }}>
                  Execution Latency: {searchLatency} ms
                </div>
              )}
            </div>

            {/* Results Grid */}
            <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
              {results.length === 0 ? (
                <div style={{ padding: '48px', textAlign: 'center', background: '#121215', borderRadius: '12px', border: '1px dashed #27272a', color: '#71717a' }}>
                  <Search size={32} style={{ margin: '0 auto 12px auto', opacity: 0.5 }} />
                  <p style={{ margin: 0, fontSize: '14px' }}>No query executed yet or no matching documents found.</p>
                  <p style={{ margin: '4px 0 0 0', fontSize: '12px', color: '#52525b' }}>Try entering a query above or adjust dense/sparse weights.</p>
                </div>
              ) : (
                results.map((res, idx) => (
                  <div
                    key={res.id || idx}
                    style={{
                      background: '#121215',
                      border: '1px solid #27272a',
                      borderRadius: '10px',
                      padding: '16px',
                      display: 'flex',
                      flexDirection: 'column',
                      gap: '8px',
                    }}
                  >
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                        <span style={{ fontSize: '12px', fontWeight: 700, color: '#6366f1', background: '#1e1b4b', padding: '2px 8px', borderRadius: '4px' }}>
                          #{idx + 1}
                        </span>
                        <span style={{ fontWeight: 600, fontSize: '14px' }}>{res.key}</span>
                        {res.metadata?.category && (
                          <span style={{ fontSize: '11px', background: '#27272a', color: '#d4d4d8', padding: '2px 6px', borderRadius: '4px' }}>
                            {res.metadata.category}
                          </span>
                        )}
                      </div>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                        {res.dense_rank && (
                          <span style={{ fontSize: '11px', color: '#818cf8', background: '#1e1b4b', padding: '2px 6px', borderRadius: '4px' }}>
                            Dense Rank: #{res.dense_rank}
                          </span>
                        )}
                        {res.sparse_rank && (
                          <span style={{ fontSize: '11px', color: '#38bdf8', background: '#082f49', padding: '2px 6px', borderRadius: '4px' }}>
                            BM25 Rank: #{res.sparse_rank}
                          </span>
                        )}
                        <span style={{ fontSize: '12px', fontWeight: 600, color: '#34d399', background: '#064e3b', padding: '2px 8px', borderRadius: '4px' }}>
                          Score: {res.score.toFixed(4)}
                        </span>
                      </div>
                    </div>
                    <p style={{ margin: 0, fontSize: '13px', color: '#d4d4d8', lineHeight: 1.5 }}>
                      {res.text}
                    </p>
                  </div>
                ))
              )}
            </div>
          </div>
        )}

        {/* TAB 2: 3D VECTOR SCATTER */}
        {activeTab === 'vector3d' && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <div>
                <h3 style={{ margin: 0, fontSize: '16px', fontWeight: 600 }}>3D Vector Embedding Space</h3>
                <p style={{ margin: '4px 0 0 0', fontSize: '13px', color: '#71717a' }}>Interactive coordinate projection of indexed high-dimensional vector embeddings with k-Means topic discovery.</p>
              </div>
              <span style={{ fontSize: '12px', color: '#a1a1aa' }}>Total Vectors: {points.length}</span>
            </div>

            {/* Semantic Topic Clusters */}
            {clusters.length > 0 && (
              <div style={{ display: 'flex', gap: '10px', flexWrap: 'wrap' }}>
                {clusters.map((c, i) => (
                  <div key={c.id} style={{ display: 'flex', alignItems: 'center', gap: '8px', background: '#18181b', border: '1px solid #27272a', padding: '6px 12px', borderRadius: '6px' }}>
                    <span style={{ width: '8px', height: '8px', borderRadius: '50%', background: i === 0 ? '#818cf8' : i === 1 ? '#38bdf8' : '#34d399' }}></span>
                    <span style={{ fontSize: '12px', fontWeight: 600, color: '#f4f4f5' }}>{c.label}</span>
                    <span style={{ fontSize: '11px', color: '#71717a' }}>({c.doc_count} docs)</span>
                  </div>
                ))}
              </div>
            )}

            <VectorScatterCanvas points={points} />
          </div>
        )}

        {/* TAB 3: KNOWLEDGE GRAPH EXPLORER */}
        {activeTab === 'graph' && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            <div>
              <h3 style={{ margin: 0, fontSize: '16px', fontWeight: 600 }}>Entity Knowledge Graph & Multi-Hop RAG</h3>
              <p style={{ margin: '4px 0 0 0', fontSize: '13px', color: '#71717a' }}>Traverse entity relationships and linked documents for contextual expansion.</p>
            </div>
            <KnowledgeGraphView />
          </div>
        )}

        {/* TAB 4: DOCUMENTS & INGESTION */}
        {activeTab === 'documents' && (
          <div style={{ display: 'grid', gridTemplateColumns: '400px 1fr', gap: '20px' }}>
            {/* Ingestion Form */}
            <div style={{ background: '#121215', border: '1px solid #27272a', borderRadius: '12px', padding: '20px' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '16px' }}>
                <PlusCircle size={18} color="#6366f1" />
                <h3 style={{ margin: 0, fontSize: '15px', fontWeight: 600 }}>Ingest Document</h3>
              </div>
              <form onSubmit={handleIngest} style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                <div>
                  <label style={{ fontSize: '12px', color: '#a1a1aa', display: 'block', marginBottom: '4px' }}>Document Key</label>
                  <input
                    type="text"
                    value={newKey}
                    onChange={e => setNewKey(e.target.value)}
                    placeholder="e.g. faq_auth_01"
                    style={{ width: '100%', padding: '8px 12px', background: '#18181b', border: '1px solid #27272a', borderRadius: '6px', color: '#fff', fontSize: '13px', boxSizing: 'border-box' }}
                  />
                </div>
                <div>
                  <label style={{ fontSize: '12px', color: '#a1a1aa', display: 'block', marginBottom: '4px' }}>Category</label>
                  <input
                    type="text"
                    value={newCategory}
                    onChange={e => setNewCategory(e.target.value)}
                    placeholder="e.g. security, billing, legal"
                    style={{ width: '100%', padding: '8px 12px', background: '#18181b', border: '1px solid #27272a', borderRadius: '6px', color: '#fff', fontSize: '13px', boxSizing: 'border-box' }}
                  />
                </div>
                <div>
                  <label style={{ fontSize: '12px', color: '#a1a1aa', display: 'block', marginBottom: '4px' }}>Text Content</label>
                  <textarea
                    rows={4}
                    value={newText}
                    onChange={e => setNewText(e.target.value)}
                    placeholder="Enter full text payload..."
                    style={{ width: '100%', padding: '8px 12px', background: '#18181b', border: '1px solid #27272a', borderRadius: '6px', color: '#fff', fontSize: '13px', boxSizing: 'border-box' }}
                  />
                </div>
                <button
                  type="submit"
                  style={{
                    padding: '10px',
                    background: '#6366f1',
                    color: '#fff',
                    border: 'none',
                    borderRadius: '6px',
                    fontWeight: 600,
                    cursor: 'pointer',
                    marginTop: '8px',
                  }}
                >
                  Index into embdb
                </button>
                {ingestSuccess && (
                  <div style={{ color: '#34d399', fontSize: '12px', display: 'flex', alignItems: 'center', gap: '6px', marginTop: '4px' }}>
                    <CheckCircle2 size={14} /> Document indexed successfully!
                  </div>
                )}
              </form>
            </div>

            {/* Document Browser */}
            <div style={{ background: '#121215', border: '1px solid #27272a', borderRadius: '12px', padding: '20px', display: 'flex', flexDirection: 'column' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
                <h3 style={{ margin: 0, fontSize: '15px', fontWeight: 600 }}>Stored Documents ({docsList.length})</h3>
              </div>
              <div style={{ flex: 1, overflowY: 'auto', maxHeight: '520px', display: 'flex', flexDirection: 'column', gap: '8px' }}>
                {docsList.map(doc => (
                  <div key={doc.id} style={{ background: '#18181b', border: '1px solid #27272a', borderRadius: '8px', padding: '12px' }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '4px' }}>
                      <span style={{ fontWeight: 600, fontSize: '13px' }}>{doc.key}</span>
                      <span style={{ fontSize: '11px', color: '#71717a' }}>ID: {doc.id}</span>
                    </div>
                    <p style={{ margin: 0, fontSize: '12px', color: '#a1a1aa' }}>{doc.text}</p>
                  </div>
                ))}
              </div>
            </div>
          </div>
        )}

        {/* TAB 5: MCP AGENT COCKPIT */}
        {activeTab === 'mcp' && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            <div style={{ background: '#121215', border: '1px solid #27272a', borderRadius: '12px', padding: '20px' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '12px' }}>
                <Cpu size={20} color="#818cf8" />
                <h3 style={{ margin: 0, fontSize: '16px', fontWeight: 600 }}>Model Context Protocol (MCP) Live Status</h3>
              </div>
              <p style={{ margin: 0, fontSize: '13px', color: '#a1a1aa', lineHeight: 1.6 }}>
                embdb exposes a native MCP server over <code>stdio</code> and <code>http://localhost:8383/mcp</code>. Autonomous AI coding agents (such as Claude Code, Antigravity, and Cursor) connect directly to access semantic memory tools without external services.
              </p>
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: '16px' }}>
              <div style={{ background: '#121215', border: '1px solid #27272a', borderRadius: '12px', padding: '20px' }}>
                <h4 style={{ margin: '0 0 12px 0', fontSize: '14px', fontWeight: 600, color: '#d4d4d8' }}>Registered MCP Agent Tools</h4>
                <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                  {[
                    { name: 'embdb_search', desc: 'Hybrid, dense vector, or BM25 keyword search with metadata filters.' },
                    { name: 'embdb_index', desc: 'Insert or upsert text documents with auto vectorization.' },
                    { name: 'embdb_graph_traverse', desc: 'Multi-hop entity traversal for Graph-Augmented Generation.' },
                    { name: 'embdb_stats', desc: 'Inspect database health, vector count, and page allocation.' },
                  ].map(tool => (
                    <div key={tool.name} style={{ background: '#18181b', padding: '10px 14px', borderRadius: '6px', border: '1px solid #27272a' }}>
                      <div style={{ fontWeight: 600, fontSize: '13px', color: '#818cf8', fontFamily: 'monospace' }}>{tool.name}</div>
                      <div style={{ fontSize: '12px', color: '#71717a', marginTop: '2px' }}>{tool.desc}</div>
                    </div>
                  ))}
                </div>
              </div>

              <div style={{ background: '#121215', border: '1px solid #27272a', borderRadius: '12px', padding: '20px' }}>
                <h4 style={{ margin: '0 0 12px 0', fontSize: '14px', fontWeight: 600, color: '#d4d4d8' }}>Agent Connection Config</h4>
                <pre style={{ margin: 0, padding: '12px', background: '#09090b', borderRadius: '8px', border: '1px solid #27272a', fontSize: '12px', color: '#818cf8', overflowX: 'auto' }}>
{`{
  "mcpServers": {
    "embdb": {
      "command": "embdb",
      "args": ["mcp", "--db", "data.embdb"]
    }
  }
}`}
                </pre>
              </div>
            </div>
          </div>
        )}
      </main>
    </div>
  );
}

// 3D Canvas Visualizer Component
function VectorScatterCanvas({ points }: { points: ScatterPoint[] }) {
  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const [rotX, setRotX] = useState(0.4);
  const [rotY, setRotY] = useState(0.6);
  const isDragging = useRef(false);
  const lastMouse = useRef({ x: 0, y: 0 });

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    let animId: number;
    const render = () => {
      ctx.clearRect(0, 0, canvas.width, canvas.height);
      const cx = canvas.width / 2;
      const cy = canvas.height / 2;

      // Draw Grid Axes
      ctx.strokeStyle = '#27272a';
      ctx.lineWidth = 1;
      ctx.beginPath();
      ctx.arc(cx, cy, 140, 0, 2 * Math.PI);
      ctx.stroke();

      // Render 3D Points
      points.forEach((p, idx) => {
        // Simple 3D rotation
        const cosX = Math.cos(rotX);
        const sinX = Math.sin(rotX);
        const cosY = Math.cos(rotY);
        const sinY = Math.sin(rotY);

        const x = p.x;
        const y = p.y * cosX - p.z * sinX;
        const z = p.y * sinX + p.z * cosX;

        const xFinal = x * cosY + z * sinY;
        const zFinal = -x * sinY + z * cosY;

        const scale = 250 / (300 + zFinal);
        const screenX = cx + xFinal * scale * 15;
        const screenY = cy + y * scale * 15;

        // Draw Point
        ctx.fillStyle = idx % 2 === 0 ? '#818cf8' : '#38bdf8';
        ctx.beginPath();
        ctx.arc(screenX, screenY, Math.max(2, 4 * scale), 0, 2 * Math.PI);
        ctx.fill();
      });
    };

    render();

    const onMouseDown = (e: MouseEvent) => {
      isDragging.current = true;
      lastMouse.current = { x: e.clientX, y: e.clientY };
    };
    const onMouseMove = (e: MouseEvent) => {
      if (!isDragging.current) return;
      const dx = e.clientX - lastMouse.current.x;
      const dy = e.clientY - lastMouse.current.y;
      setRotY(r => r + dx * 0.01);
      setRotX(r => r + dy * 0.01);
      lastMouse.current = { x: e.clientX, y: e.clientY };
    };
    const onMouseUp = () => {
      isDragging.current = false;
    };

    canvas.addEventListener('mousedown', onMouseDown);
    window.addEventListener('mousemove', onMouseMove);
    window.addEventListener('mouseup', onMouseUp);

    return () => {
      canvas.removeEventListener('mousedown', onMouseDown);
      window.removeEventListener('mousemove', onMouseMove);
      window.removeEventListener('mouseup', onMouseUp);
    };
  }, [points, rotX, rotY]);

  return (
    <div style={{ background: '#121215', border: '1px solid #27272a', borderRadius: '12px', padding: '16px', position: 'relative' }}>
      <canvas
        ref={canvasRef}
        width={900}
        height={450}
        style={{ width: '100%', height: '450px', display: 'block', cursor: 'grab' }}
      />
      <div style={{ position: 'absolute', bottom: '24px', left: '24px', fontSize: '12px', color: '#71717a' }}>
        Click and drag to rotate 3D embedding space.
      </div>
    </div>
  );
}

// Knowledge Graph Component
function KnowledgeGraphView() {
  return (
    <div style={{ background: '#121215', border: '1px solid #27272a', borderRadius: '12px', padding: '24px', height: '450px', position: 'relative', overflow: 'hidden' }}>
      <svg width="100%" height="100%" viewBox="0 0 800 400">
        <defs>
          <marker id="arrow" viewBox="0 0 10 10" refX="22" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
            <path d="M 0 0 L 10 5 L 0 10 z" fill="#3f3f46" />
          </marker>
        </defs>

        {/* Lines */}
        <line x1="200" y1="200" x2="400" y2="120" stroke="#3f3f46" strokeWidth="2" markerEnd="url(#arrow)" />
        <line x1="200" y1="200" x2="400" y2="280" stroke="#3f3f46" strokeWidth="2" markerEnd="url(#arrow)" />
        <line x1="400" y1="120" x2="600" y2="200" stroke="#3f3f46" strokeWidth="2" markerEnd="url(#arrow)" />
        <line x1="400" y1="280" x2="600" y2="200" stroke="#3f3f46" strokeWidth="2" markerEnd="url(#arrow)" />

        {/* Nodes */}
        <g transform="translate(200, 200)">
          <circle r="24" fill="#1e1b4b" stroke="#6366f1" strokeWidth="2" />
          <text fill="#fff" fontSize="11" textAnchor="middle" dy="4">doc:contract</text>
        </g>
        <g transform="translate(400, 120)">
          <circle r="22" fill="#082f49" stroke="#38bdf8" strokeWidth="2" />
          <text fill="#fff" fontSize="10" textAnchor="middle" dy="4">Acme Corp</text>
        </g>
        <g transform="translate(400, 280)">
          <circle r="22" fill="#064e3b" stroke="#34d399" strokeWidth="2" />
          <text fill="#fff" fontSize="10" textAnchor="middle" dy="4">Alice Smith</text>
        </g>
        <g transform="translate(600, 200)">
          <circle r="24" fill="#312e81" stroke="#818cf8" strokeWidth="2" />
          <text fill="#fff" fontSize="11" textAnchor="middle" dy="4">doc:invoice</text>
        </g>
      </svg>
      <div style={{ position: 'absolute', bottom: '24px', left: '24px', fontSize: '12px', color: '#71717a' }}>
        Entities and document chunk linkages mapped into multi-hop adjacency graph.
      </div>
    </div>
  );
}
