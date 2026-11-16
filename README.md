# embdb

Embedded hybrid vector and semantic database written in Go with single-file storage, HNSW dense indexing, BM25 sparse retrieval, and native Model Context Protocol support.

## Overview

embdb is an in-process, zero-network vector search engine designed for local AI agents, CLI utilities, and serverless environments. It packages dense vector similarity, sparse lexical search, relational metadata filtering, and entity relationship graphs into a single portable binary and a single container file.

### Key Capabilities

* Single-file container format (`.embdb`) using memory-mapped pages for zero-copy queries
* Dense vector indexing via Hierarchical Navigable Small World (HNSW) graphs with AVX2 SIMD acceleration and SQ8 scalar quantization
* Lexical text retrieval via Okapi BM25 with Roaring Bitmaps for inverted index operations
* Hybrid ranking fusion using Reciprocal Rank Fusion (RRF) to merge semantic and keyword results
* Relational metadata filtering with B-Tree indices supporting range and equality predicates
* Entity knowledge graph traversal for multi-hop graph-augmented generation
* Built-in Model Context Protocol (MCP) server supporting stdio and Server-Sent Events (SSE)
* Embedded web-based Vector Studio running on local port 8383

## Storage Architecture

embdb encapsulates all indices and document payloads into one structured binary file:

```
+-------------------------------------------------------------------+
| Header (Magic, Dimensions, Metric, Quantization Mode, Counters)   |
+-------------------------------------------------------------------+
| HNSW Graph Nodes, Layers, and Vector Vectors Storage              |
+-------------------------------------------------------------------+
| BM25 Lexical Inverted Postings Lists and Term Dictionary          |
+-------------------------------------------------------------------+
| Relational Metadata B-Tree Root and Secondary Index Pages         |
+-------------------------------------------------------------------+
| Entity Relationship Graph Adjacency Records                       |
+-------------------------------------------------------------------+
| 4KB Memory-Mapped Slotted Document Pages                          |
+-------------------------------------------------------------------+
```

A Write-Ahead Log (WAL) ensures durability across unexpected restarts, using CRC32 checksums on all written records.

## Quick Start

### Installation

```bash
go get github.com/alexandrmotologa/embdb
```

### CLI Usage

Start the embedded studio and HTTP API:

```bash
embdb serve --db data.embdb --port 8383
```

Run as an MCP server for AI agents:

```bash
embdb mcp --db data.embdb
```

Query via command line:

```bash
embdb query --db data.embdb --query "hybrid search query" --limit 10
```

## License

MIT License. See LICENSE file for details.
