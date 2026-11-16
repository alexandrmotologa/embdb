<p align="center">
  <img src="docs/images/logo.png?raw=true" alt="embdb Logo" width="130" style="border-radius: 24px;" />
</p>

<h1 align="center">embdb</h1>

<p align="center">
  <strong>Embedded Hybrid Vector & Semantic Database in Go</strong><br />
  Single-binary zero-CGO engine with HNSW dense vectors, BM25 lexical indexing, mmap storage, and native Model Context Protocol.
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.27+-00ADD8?style=flat&logo=go" alt="Go Version" />
  <img src="https://img.shields.io/badge/Binary_Size-10.5_MB-6366f1?style=flat" alt="Binary Size" />
  <img src="https://img.shields.io/badge/Cold_Start-%3C_3ms-10b981?style=flat" alt="Cold Start" />
  <img src="https://img.shields.io/badge/Protocol-MCP_Native-818cf8?style=flat" alt="MCP" />
  <img src="https://img.shields.io/badge/License-MIT-gray?style=flat" alt="License" />
</p>

---

## Overview

`embdb` is an in-process, zero-network hybrid vector search database designed for autonomous AI agents, CLI tools, and edge environments. It stores dense vector similarity graphs, sparse lexical keyword indices, relational metadata trees, and entity knowledge graphs in a single container file (`.embdb`).

<p align="center">
  <img src="docs/images/embdb_demo.gif?raw=true" alt="embdb Vector Studio Interactive Demo" width="90%" style="border-radius: 12px; border: 1px solid #27272a;" />
</p>

### Key Capabilities

* **Single-File Container (`.embdb`)**: 4KB slotted pages with CRC32 integrity checks and zero-copy memory mapping (`mmap`).
* **HNSW Dense Vector Index**: Multi-layer proximity graph with SIMD distance calculations and SQ8 scalar quantization (75% memory reduction).
* **BM25 Lexical Engine**: Okapi BM25 ranking with Unicode tokenization, stemming, and 64-bit word bitsets.
* **Reciprocal Rank Fusion (RRF)**: Merges dense and sparse rankings with automatic intent detection.
* **Relational B-Tree & Predicates**: Fast filtering supporting `$eq`, `$ne`, `$gt`, `$gte`, `$lt`, `$lte`, `$in`, and `$contains`.
* **Entity Knowledge Graph**: Adjacency index with multi-hop breadth-first traversal for Graph-Augmented Generation (RAG).
* **Native Model Context Protocol (MCP)**: JSON-RPC 2.0 interface over stdio and HTTP for Claude Code, Antigravity, and Cursor.
* **Embedded Vector Studio**: Bundled React 19 single page application served on port 8383 directly from the executable.
* **Interactive CLI Shell**: Terminal REPL with SQL query execution and ASCII table rendering.

---

## Studio Interface & Screenshots

<p align="center">
  <img src="docs/images/embdb_studio_dashboard.png?raw=true" alt="Hybrid Search Bench Dashboard" width="48%" style="border-radius: 8px; border: 1px solid #27272a;" />
  <img src="docs/images/embdb_studio_3d.png?raw=true" alt="3D Vector Space Explorer" width="48%" style="border-radius: 8px; border: 1px solid #27272a;" />
</p>

<p align="center">
  <img src="docs/images/embdb_studio_graph.png?raw=true" alt="Knowledge Graph Explorer" width="48%" style="border-radius: 8px; border: 1px solid #27272a;" />
  <img src="docs/images/embdb_mobile.png?raw=true" alt="Responsive Mobile Viewport" width="24%" style="border-radius: 8px; border: 1px solid #27272a;" />
</p>

---

## Performance Benchmarks

Measured on an AMD Ryzen 9 7950X (64-bit, single core query execution):

| Metric | Result | Context |
| :--- | :--- | :--- |
| **Binary Footprint** | `10.5 MB` | Standalone executable including React 19 Studio |
| **Cold Start Time** | `< 2.5 ms` | Container open, mmap initialization, HNSW reload |
| **Idle Memory** | `< 15 MB` | Zero background network threads |
| **Ingestion Rate** | `2,254 docs/s` | Batch insertion with vector normalization & page packing |
| **Hybrid Query Latency** | `0.38 ms` | Dense HNSW + BM25 sparse + RRF fusion (2,622 QPS) |

---

## Quick Start

### Installation

```bash
go install github.com/alexandrmotologa/embdb/cmd/embdb@latest
```

Or build from source:

```bash
git clone https://github.com/alexandrmotologa/embdb.git
cd embdb
go build -o bin/embdb ./cmd/embdb
```

### 1. Launch Vector Studio

```bash
embdb serve --db data.embdb --port 8383
```

Open `http://localhost:8383` in your browser.

### 2. Interactive Terminal Shell

```bash
embdb shell --db data.embdb
```

```sql
embdb> SELECT id, key, text FROM documents WHERE category = 'legal' ORDER BY hybrid('contract breach') LIMIT 5;
+----+----------------+------------------------------------------+
| ID | KEY            | TEXT                                     |
+----+----------------+------------------------------------------+
| 1  | legal_tos_01   | Master Services Agreement: Terms rega... |
| 2  | legal_nda_02   | Mutual Non-Disclosure Agreement cover... |
+----+----------------+------------------------------------------+
2 rows in set (0.001s)
```

### 3. Connect AI Agents via MCP

Add `embdb` to your agent configuration:

```json
{
  "mcpServers": {
    "embdb": {
      "command": "embdb",
      "args": ["mcp", "--db", "agent_memory.embdb"]
    }
  }
}
```

### 4. Bulk Ingestion & Compaction

```bash
# Bulk load JSONL dataset
embdb index --db data.embdb --file examples/datasets/enterprise_faq.jsonl

# Defragment and shrink container file
embdb compact --db data.embdb

# Export to portable JSONL
embdb dump --db data.embdb --out backup.jsonl

# Restore into fresh container
embdb restore --db new.embdb --in backup.jsonl
```

---

## Embedded Go Library Usage

```go
package main

import (
	"context"
	"fmt"

	"github.com/alexandrmotologa/embdb/internal/embeddings"
	"github.com/alexandrmotologa/embdb/internal/engine"
	"github.com/alexandrmotologa/embdb/internal/storage"
)

func main() {
	db, err := engine.OpenDB("app.embdb", 128, storage.MetricCosine, storage.QuantizeSQ8)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	embedder := embeddings.NewDeterministicAdapter(128)
	vec, _ := embedder.Embed(context.Background(), "contract liability clauses")

	_ = db.Insert(&engine.Document{
		Key:    "doc_01",
		Text:   "Non-disclosure agreement and confidential terms.",
		Vector: vec,
	})

	results, _ := db.SearchHybrid("contract liability", vec, 5, nil, 0.5, 0.5)
	for _, r := range results {
		fmt.Printf("%s (Score: %.4f)\n", r.Key, r.Score)
	}
}
```

---

## Documentation

* [Architecture Specification](docs/architecture.md)
* [Model Context Protocol Guide](docs/mcp-guide.md)
* [REST API Reference](docs/api-reference.md)
* [Performance Benchmarks](docs/benchmarks.md)
* [Full Go Library Example](examples/basic_usage.go)

## License

MIT License. See [LICENSE](LICENSE) for details.
