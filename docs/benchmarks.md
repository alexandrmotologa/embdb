# Performance Benchmarks

Benchmarks were performed locally on an AMD Ryzen 9 7950X / Windows 64-bit environment with standard compiler optimizations (`go build`).

## Summary Metrics

* **Binary Size**: 10.2 MB (Single autonomous executable with embedded React 19 Studio)
* **Cold Start Time**: 2.2 ms to initialize storage pager and HNSW graph
* **Idle Memory**: < 15 MB RAM
* **Insertion Throughput**: 2,254 documents / second
* **Hybrid Search Latency**: 0.38 ms average per query (2,622 QPS on single core)

## Test Configurations

### Test 1: Ingestion Throughput (1,000 vectors, 64 dimensions, SQ8)

```
Running embdb benchmark with 1000 vectors (dimension: 64, SQ8 quantization)...
Insertion: 1000 documents in 443.65ms (2254.0 docs/sec)
```

### Test 2: Hybrid Search Latency (100 queries, Top-10)

```
Hybrid Search: 100 queries in 38.12ms (Average Latency: 0.38 ms, 2622.6 QPS)
```

### Test 3: Cold Reopen Persistence (20 enterprise documents)

```
Reopen Time: 2.19 ms
Pages Allocated: 3 (4KB per page, 12 KB total file size)
```
