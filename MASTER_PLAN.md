# embdb: Architecture and Engineering Master Plan
> Single-Binary Embedded Hybrid Vector & Semantic Database with HNSW, BM25, and Model Context Protocol (MCP)
> Target: Zero-CGO Single-Binary Go Engine (< 25MB RAM, < 10ms cold start) with Single-File Storage (`.embdb`) & Embedded React 19 Vector Studio

---

## 1. Problem Statement and Motivation

### 1.1 The Operational Bottleneck of Modern Vector Databases
The rapid expansion of Retrieval-Augmented Generation (RAG), autonomous coding agents, and enterprise semantic search has created an explosion in vector database adoption. However, current options present severe architectural trade-offs:
1. **The Heavyweight Cloud Paradigm (Pinecone, Weaviate, Qdrant)**:
   - Operating a remote vector database cluster introduces network hop latencies (30ms to 120ms per query), vendor lock-in, and unpredictable SaaS billing (\$100s to \$1000s/month).
   - In private, on-premise, or local-first environments, maintaining distributed multi-node clusters adds massive DevOps complexity.
2. **The Python Framework Dilemma (txtai, Chroma before refactors)**:
   - Python-based embedding databases require massive runtime dependencies: PyTorch, Transformers, ONNX, and Faiss easily exceed 3GB to 5GB in container images.
   - High memory consumption (600MB to 1.5GB RAM at idle) and sluggish cold start times (4 to 8 seconds) render them non-viable in serverless runtimes (AWS Lambda), low-resource VPS nodes, edge devices, and desktop CLI tools.
3. **The Separate-Index Fragment Problem**:
   - Modern enterprise search requires **Hybrid Search**: dense semantic vectors combined with exact sparse keyword matching (BM25), strict metadata filtering (SQL), and entity relationship traversal (Knowledge Graphs).
   - Stitching together Elasticsearch + Qdrant + SQLite requires complex data synchronization pipelines that frequently suffer from stale reads and distributed transaction failures.

### 1.2 The embdb Solution
`embdb` is the "SQLite / DuckDB of Vector & Semantic Search"—a high-performance, single-binary, embedded hybrid vector database written in Go:
- **Zero-Dependency Single Binary**: Compiles to an autonomous executable (< 25MB) with zero external CGO dependencies and an idle memory footprint under 20MB RAM.
- **Unified Single-File Storage (`.embdb`)**: Encapsulates HNSW dense vector graphs, inverted BM25 keyword indices, relational metadata B-Trees, and entity graph linkages into a single, portable binary container file with memory-mapped (`mmap`) zero-copy reads.
- **Native Reciprocal Rank Fusion (RRF)**: Merges dense vector cosine similarities with sparse lexical BM25 rankings inside the query engine, eliminating the need for external re-ranking services.
- **Multi-Modal Model Context Protocol (MCP) Native**: Functions directly as an MCP server over stdio and SSE, allowing autonomous agents (Claude Code, Antigravity, Cursor) to mount local semantic memory with zero configuration.
- **Flexible Embedding Runtime**: Supports zero-overhead external API adapters (OpenAI, Voyage AI, Cohere, Ollama) and optional local ONNX Runtime INT8 quantization for 100% offline air-gapped deployments.
- **Embedded React 19 Vector Studio**: Bundles an interactive web workbench to visualize 2D/3D vector spaces, inspect hybrid query rank fusions, and traverse knowledge graphs.

---

## 2. High-Level System Architecture

```
       +-------------------------------------------------------------+
       |               AI Agents / Applications / CLI                |
       |     (Claude Code, Antigravity, LangChain, Custom Go/TS App)  |
       +-------------------------------------------------------------+
                                      |
         Access Interfaces: Native Go API | MCP (stdio/SSE) | HTTP REST
                                      v
       +-------------------------------------------------------------+
       |                      embdb Core Engine                      |
       |                    (High-Performance Go)                    |
       |                                                             |
       |  +---------------------+        +------------------------+  |
       |  | SQL / Hybrid Query  |------->| Reciprocal Rank Fusion |  |
       |  | Planner & Parser    |        | (RRF Hybrid Optimizer) |  |
       |  +---------------------+        +------------------------+  |
       |              |                               |              |
       |              +---------------+---------------+              |
       |                              |                              |
       |                              v                              |
       |  +-------------------------------------------------------+  |
       |  |                 Unified Storage Engine                |  |
       |  |                                                       |  |
       |  |  +-------------------+        +--------------------+  |  |
       |  |  | Dense Vector HNSW |        | Sparse BM25 Lexical|  |  |
       |  |  | (SIMD AVX2/NEON)  |        | (Roaring Bitmaps)  |  |  |
       |  |  +-------------------+        +--------------------+  |  |
       |  |            |                            |             |  |
       |  |  +-------------------+        +--------------------+  |  |
       |  |  | Relational B-Tree |        | Entity Knowledge   |  |  |
       |  |  | (Metadata Filter) |        | Graph (Adjacency)  |  |  |
       |  |  +-------------------+        +--------------------+  |  |
       |  +-------------------------------------------------------+  |
       |                              |                              |
       |                              v                              |
       |  +-------------------------------------------------------+  |
       |  |         mmap Zero-Copy Pager & WAL Journal            |  |
       |  +-------------------------------------------------------+  |
       |                              |                              |
       +------------------------------|------------------------------+
                                      |
                                      v
       +-------------------------------------------------------------+
       |                 Single `.embdb` File on Disk                |
       |  [File Header] [HNSW Graph] [BM25 Inverted] [Meta] [Graph]  |
       +-------------------------------------------------------------+
                                      |
                                      v
       +-------------------------------------------------------------+
       |                 Embedded Studio & Governance                |
       |  1. Local HTTP Web Studio (:8383) in React 19               |
       |  2. 2D/3D UMAP Vector Space Dimensionality Visualizer       |
       |  3. Real-Time MCP Tool Endpoints for Agent Memory           |
       +-------------------------------------------------------------+
```

---

## 3. Mathematical Formulations: HNSW, BM25 & Rank Fusion

### 3.1 Cosine Distance with SIMD Optimization
For query vector $\mathbf{q} \in \mathbb{R}^d$ and indexed document vector $\mathbf{v} \in \mathbb{R}^d$:

$$\text{CosDist}(\mathbf{q}, \mathbf{v}) = 1 - \frac{\mathbf{q} \cdot \mathbf{v}}{\|\mathbf{q}\|_2 \|\mathbf{v}\|_2} = 1 - \frac{\sum_{i=1}^d q_i v_i}{\sqrt{\sum_{i=1}^d q_i^2} \sqrt{\sum_{i=1}^d v_i^2}}$$

When vectors are normalized upon insertion ($\|\mathbf{v}\|_2 = 1$), the calculation reduces to a single SIMD-accelerated dot product ($1 - \mathbf{q} \cdot \mathbf{v}$).

### 3.2 BM25 Lexical Score
For query terms $Q = \{t_1, t_2, \dots, t_n\}$ and document $D$ with length $|D|$ and average corpus length $\text{avgdl}$:

$$\text{BM25}(D, Q) = \sum_{i=1}^n \text{IDF}(t_i) \cdot \frac{f(t_i, D) \cdot (k_1 + 1)}{f(t_i, D) + k_1 \cdot \left(1 - b + b \cdot \frac{|D|}{\text{avgdl}}\right)}$$

Where $k_1 = 1.2$, $b = 0.75$, and $\text{IDF}(t_i) = \ln\left( \frac{N - n(t_i) + 0.5}{n(t_i) + 0.5} + 1 \right)$.

### 3.3 Reciprocal Rank Fusion (RRF)
Given the dense vector ranking $\text{rank}_{\text{dense}}(d)$ and the sparse BM25 ranking $\text{rank}_{\text{bm25}}(d)$, with smoothing constant $k = 60$:

$$\text{RRF\_Score}(d) = \frac{w_{\text{dense}}}{60 + \text{rank}_{\text{dense}}(d)} + \frac{w_{\text{bm25}}}{60 + \text{rank}_{\text{bm25}}(d)}$$

---

## 4. Single-File Binary Format Specification (`.embdb`)

```
+---------------------------------------------------------------------------------+
| Offset      | Size        | Content Description                                 |
+---------------------------------------------------------------------------------+
| 0x00000000  | 8 Bytes     | Magic Number: `EMBDB_V1`                            |
| 0x00000008  | 4 Bytes     | Flags: Vector Dimension, Distance Metric (Cosine/L2)|
| 0x0000000C  | 4 Bytes     | Quantization Mode: FP32 (0), FP16 (1), SQ8 (2)      |
| 0x00000010  | 8 Bytes     | Total Document Count ($N$)                          |
| 0x00000018  | 8 Bytes     | Pointer: HNSW Graph Index Block                     |
| 0x00000020  | 8 Bytes     | Pointer: BM25 Inverted Term Index Block             |
| 0x00000028  | 8 Bytes     | Pointer: Relational Metadata B-Tree Root            |
| 0x00000030  | 8 Bytes     | Pointer: Knowledge Graph Adjacency Block            |
| 0x00000038  | ...         | Data Pages (4096-byte aligned, managed via mmap)    |
+---------------------------------------------------------------------------------+
```

---

## 5. Repository Structure

```
embdb/
├── cmd/
│   ├── embdb/
│   │   └── main.go                 # CLI entry point, subcommands: serve, index, query, mcp, studio
│   └── mcp-server/
│       └── main.go                 # Dedicated standalone Model Context Protocol server
├── internal/
│   ├── config/
│   │   ├── config.go               # YAML configuration loader and environment overrides
│   │   └── schema.go               # Collection schemas, distance metrics, and cache budgets
│   ├── storage/
│   │   ├── file.go                 # Single-file .embdb container reader/writer
│   │   ├── mmap.go                 # Zero-copy memory-mapped I/O paging layer
│   │   ├── page.go                 # 4KB block pager and buffer pool
│   │   └── wal.go                  # Write-Ahead Log for ACID durability and crash recovery
│   ├── hnsw/
│   │   ├── graph.go                # Hierarchical Navigable Small World index implementation
│   │   ├── distance.go             # SIMD-accelerated AVX2/NEON vector distance calculations
│   │   ├── quantize.go             # Scalar Quantization (SQ8) compressing vectors by 75%
│   │   └── search.go               # Multi-layer nearest neighbor beam search
│   ├── bm25/
│   │   ├── inverted.go             # Sparse inverted index with Roaring Bitmaps postings
│   │   ├── tokenizer.go            # High-speed Unicode text tokenizer and stemmer
│   │   └── scorer.go               # BM25 scoring algorithm with length normalization
│   ├── metadata/
│   │   ├── btree.go                # Pure Go B-Tree for relational metadata indexing
│   │   ├── filter.go               # Predicate evaluation engine (AND, OR, NOT, IN, range)
│   │   └── json_store.go           # Fast binary JSON payload storage
│   ├── graph/
│   │   ├── adjacency.go            # Entity knowledge graph node-link adjacency index
│   │   └── traversal.go            # Multi-hop graph neighborhood expansion for RAG
│   ├── engine/
│   │   ├── db.go                   # Primary Database interface and collection manager
│   │   ├── hybrid.go               # Reciprocal Rank Fusion (RRF) query optimizer
│   │   └── sql.go                  # Lightweight SQL dialect parser for semantic queries
│   ├── embeddings/
│   │   ├── adapter.go              # Interface for external API providers (OpenAI, Voyage, Ollama)
│   │   └── onnx_local.go           # Optional local ONNX Runtime INT8 embedding runner
│   ├── mcp/
│   │   ├── protocol.go             # Model Context Protocol stdio & SSE JSON-RPC 2.0 handler
│   │   └── tools.go                # Tools: embdb_search, embdb_index, embdb_graph_traverse
│   └── server/
│       ├── http.go                 # REST & GraphQL API server
│       ├── embed.go                # Go 1.16+ embed filesystem for React 19 studio
│       └── sse.go                  # Real-time event streaming for vector updates
├── web/
│   ├── src/
│   │   ├── App.tsx                 # React 19 root application
│   │   ├── components/
│   │   │   ├── VectorScatter3D.tsx # WebGL / Three.js 3D UMAP vector space projection
│   │   │   ├── HybridQueryBench.tsx# Interactive query test bench with RRF score sliders
│   │   │   ├── KnowledgeGraph.tsx  # Force-directed entity node-link graph explorer
│   │   │   ├── CollectionStats.tsx # Memory, disk footprint, and vector count gauges
│   │   │   └── McpStatus.tsx       # Live status of connected AI agent sessions
│   │   └── pages/
│   │       └── Dashboard.tsx       # Main Vector Studio cockpit
│   ├── package.json
│   ├── tsconfig.json
│   └── vite.config.ts
├── examples/
│   ├── basic_usage.go              # Native Go embedded library example
│   ├── mcp_config.json             # Configuration snippet for Claude Desktop / Antigravity
│   └── datasets/
│       └── enterprise_faq.jsonl    # Realistic evaluation dataset with text & metadata
├── Makefile
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```

---

## 6. Implementation Roadmap

### Phase 1: Go Engine Foundation & Binary Container Storage
- Initialize Go module with zero-CGO dependencies.
- Build memory-mapped file reader and pager in `internal/storage/file.go` and `mmap.go`.
- Implement binary container header (`EMBDB_V1`) and Write-Ahead Log (WAL) for transactional writes.

### Phase 2: HNSW Dense Vector Engine & SIMD Acceleration
- Build HNSW graph indexing in `internal/hnsw/graph.go` supporting configurable $M$ and $efConstruction$.
- Implement SIMD-accelerated Cosine and L2 distance routines using Go assembly / compiler auto-vectorization.
- Implement SQ8 scalar quantization compressing 1536-dimensional float32 vectors to int8 bytes, reducing memory usage by 75%.

### Phase 3: BM25 Lexical Inverted Index
- Implement fast Unicode tokenizer and stemmer in `internal/bm25/tokenizer.go`.
- Build inverted postings lists using Roaring Bitmaps for high-speed intersection and union filtering.
- Implement BM25 scoring with exact document frequency and average corpus length tracking.

### Phase 4: Relational Metadata Filtering & Knowledge Graph Layer
- Implement pure Go B-Tree in `internal/metadata/btree.go` supporting structured types (strings, numbers, timestamps, tags).
- Build predicate evaluation engine allowing pre-filtering and post-filtering during vector traversals.
- Implement entity adjacency graph for multi-hop semantic relationship traversal.

### Phase 5: Hybrid Engine & SQL Query Planner
- Implement Reciprocal Rank Fusion (RRF) in `internal/engine/hybrid.go` combining dense vector scores and sparse BM25 scores.
- Build SQL-like parser:
  `SELECT id, text, score FROM documents WHERE category = 'legal' ORDER BY hybrid(vector, 'breach of contract') LIMIT 10`.

### Phase 6: Model Context Protocol (MCP) Server & External Adapters
- Build native MCP server in `internal/mcp` supporting stdio and SSE transports.
- Expose agent tools: `embdb_search`, `embdb_upsert`, and `embdb_explore_graph`.
- Implement external embedding adapters for OpenAI, Voyage AI, Cohere, and local Ollama endpoints.

### Phase 7: Embedded React 19 Vector Studio
- Develop interactive web workbench in `web/` using React 19, Tailwind CSS, and Lucide icons.
- Build WebGL 3D vector scatter plot, interactive hybrid query testing bench, and force-directed knowledge graph viewer.
- Embed frontend build artifacts into the Go binary via `//go:embed` and serve locally on `:8383`.

### Phase 8: Verification, Benchmarking, and Documentation
- Run benchmarks comparing `embdb` against Faiss and Qdrant, verifying $< 10\text{ms}$ cold start, $< 20\text{MB}$ idle RAM, and sub-5ms query latency on 100,000 vectors.
- Write unit tests for concurrent reads/writes, WAL crash recovery, and index persistence.
- Author humanized README with quick-start examples, embedded Go snippets, MCP configuration guides, and architecture diagrams.
