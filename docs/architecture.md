# embdb Architecture Specification

## Overview

embdb is an embedded, in-process hybrid vector and semantic search engine implemented in Go. It operates without network round-trips, without external database processes, and without heavy Python runtime frameworks.

## Storage Layer

embdb stores all data within a single binary container file with the `.embdb` extension, coupled with an append-only Write-Ahead Log (`.embdb-wal`).

### Slotted Page Structure

* Page size is fixed at 4096 bytes (4KB), aligning with operating system virtual memory pages.
* Slotted page design:
  * Page header at offset 0 to 18 bytes.
  * Slot directory starting after the header and growing forward.
  * Record payloads positioned at the bottom of the page and growing backward toward the header.
  * CRC32 checksum recorded on each page for corruption detection.

### Write-Ahead Log (WAL)

The WAL records transactional updates before they are committed to the data pages. Each WAL entry contains:
* Record length (4 bytes)
* Record type: Insert, Update, Delete, Commit, Checkpoint (1 byte)
* Transaction identifier (8 bytes)
* Target page identifier (4 bytes)
* Slot index (2 bytes)
* Record payload
* CRC32 checksum (4 bytes)

Crash recovery scans the WAL upon startup, verifies checksums, and replays committed transactions into the page cache.

## Dense Vector Engine (HNSW)

The vector index uses a Hierarchical Navigable Small World (HNSW) graph:
* Multi-layer proximity graph with logarithmic search scaling.
* Layer assignment uses a geometric distribution parameterized by $m_L = 1 / \ln(M)$.
* Layer 0 contains all vectors, while upper layers contain progressively sparser subsets.
* Distance metrics: Cosine Distance, Euclidean (L2) Distance, and Maximum Inner Product Search.
* Loop unrolling in 8-lane increments enables automatic SIMD vectorization by modern Go compiler releases.

### Scalar Quantization (SQ8)

Scalar Quantization compresses float32 values to 8-bit unsigned integers:
* Normalizes float components between local minimum and maximum values.
* Compresses memory requirements from 4 bytes per dimension down to 1 byte per dimension (75% memory reduction).
* Asymmetric distance calculation computes dot products between unquantized query vectors and quantized stored vectors.

## Sparse Lexical Engine (BM25)

* Custom Unicode tokenizer segments alphanumeric tokens, converts characters to lowercase, and strips punctuation.
* Common English stop words are filtered out.
* Suffix stemming strips common verb and plural endings.
* Postings lists map terms to document identifiers and term frequencies.
* Bitset operations enable bitwise intersection, union, and difference across document sets.
* Okapi BM25 scoring uses standard parameters ($k_1 = 1.2$, $b = 0.75$) with corpus length normalization.

## Reciprocal Rank Fusion (RRF)

Dense vector search and sparse BM25 search produce independent ranked result lists. The hybrid engine merges these rankings using Reciprocal Rank Fusion:

$$\text{RRF}(d) = \frac{w_{\text{dense}}}{k + \text{rank}_{\text{dense}}(d)} + \frac{w_{\text{sparse}}}{k + \text{rank}_{\text{sparse}}(d)}$$

where:
* $k$ defaults to 60.
* $w_{\text{dense}}$ and $w_{\text{sparse}}$ control relative emphasis between semantic meaning and exact keyword match.

## Relational Filtering and Knowledge Graph

* B-Tree index structure enables exact and range queries over scalar metadata fields.
* Predicate engine evaluates compound conditions (`$eq`, `$ne`, `$gt`, `$gte`, `$lt`, `$lte`, `$in`, `$contains`) with logical `$and`, `$or`, `$not`.
* Adjacency list graph maintains directed relationships between entities and document chunks.
* Breadth-first graph traversal expands context neighborhoods across multiple hops for graph-augmented generation.
