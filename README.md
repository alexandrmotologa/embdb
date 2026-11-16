# embdb

Embedded hybrid vector and semantic database in Go with single-file storage, HNSW indexing, BM25 retrieval, and native Model Context Protocol support.

## Overview

embdb is an in-process search database designed for local AI agents, CLI tools, and low-resource environments. It combines dense vector similarity, sparse lexical keyword matching, metadata filtering, and entity relationship graphs into a single portable binary and a single container file (`.embdb`).

### Key Capabilities

* **Single-File Container Format (`.embdb`)**: Uses 4KB slotted pages and memory-mapped reads for low latency.
* **Dense Vector Indexing**: Hierarchical Navigable Small World (HNSW) graph with AVX2 SIMD acceleration and SQ8 scalar quantization.
* **Lexical Text Retrieval**: Okapi BM25 engine with Unicode tokenization, stemming, and 64-bit word bitsets.
* **Hybrid Ranking Fusion**: Merges dense and sparse result lists using Reciprocal Rank Fusion (RRF).
* **Metadata Filtering**: Relational B-Tree index supporting exact, range, and compound predicate evaluation.
* **Knowledge Graph Traversal**: Directed entity adjacency index for multi-hop graph-augmented generation.
* **Native Model Context Protocol (MCP)**: Embedded JSON-RPC 2.0 server for Claude Code, Antigravity, and Cursor.
* **Embedded Vector Studio**: Bundled React 19 interface compiled directly into the Go binary on port 8383.

## Quick Start

### Installation

```bash
go install github.com/alexandrmotologa/embdb/cmd/embdb@latest
```

Or clone and build locally:

```bash
git clone https://github.com/alexandrmotologa/embdb.git
cd embdb
go build -o bin/embdb ./cmd/embdb
```

### Start the Embedded Studio & API

```bash
embdb serve --db data.embdb --port 8383
```

Open `http://localhost:8383` in your browser to view the interactive 3D vector scatter plot, hybrid search tester, and knowledge graph viewer.

### Connect AI Agents via MCP

Add the following to your agent configuration:

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

### CLI Search

```bash
embdb query --db data.embdb --query "contract liability" --mode hybrid --limit 5
```

### Bulk Ingestion

```bash
embdb index --db data.embdb --file examples/datasets/enterprise_faq.jsonl
```

## Performance Highlights

* **Binary Size**: 10.2 MB (single binary including full React 19 Studio)
* **Cold Start**: < 3 ms
* **Insertion Rate**: 2,250+ documents / second
* **Search Latency**: 0.38 ms average per hybrid query (2,600+ QPS)

See [docs/benchmarks.md](docs/benchmarks.md) for full benchmark methodology.

## Documentation

* [Architecture Specification](docs/architecture.md)
* [Model Context Protocol Guide](docs/mcp-guide.md)
* [REST API Reference](docs/api-reference.md)
* [Performance Benchmarks](docs/benchmarks.md)
* [Embedded Library Example](examples/basic_usage.go)

## License

MIT License. See LICENSE for details.
