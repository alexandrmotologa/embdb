package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexandrmotologa/embdb/internal/embeddings"
	"github.com/alexandrmotologa/embdb/internal/engine"
	"github.com/alexandrmotologa/embdb/internal/mcp"
	"github.com/alexandrmotologa/embdb/internal/server"
	"github.com/alexandrmotologa/embdb/internal/storage"
)

const banner = `
  ___ _ __ ___  | |__   __| | |__  
 / _ \ '_ ` + "`" + ` _ \ | '_ \ / _` + "`" + ` | '_ \ 
|  __/ | | | | || |_) | (_| | |_) |
 \___|_| |_| |_||_.__/ \__,_|_.__/  v1.0.0
 High-Performance Embedded Hybrid Vector & Semantic Database
`

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	subcommand := os.Args[1]
	switch subcommand {
	case "serve":
		cmdServe(os.Args[2:])
	case "mcp":
		cmdMCP(os.Args[2:])
	case "query":
		cmdQuery(os.Args[2:])
	case "index":
		cmdIndex(os.Args[2:])
	case "stats":
		cmdStats(os.Args[2:])
	case "bench":
		cmdBench(os.Args[2:])
	default:
		fmt.Printf("Unknown command: %s\n", subcommand)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(banner)
	fmt.Println("Usage: embdb <command> [options]")
	fmt.Println("\nCommands:")
	fmt.Println("  serve    Start embedded Vector Studio & REST API server")
	fmt.Println("  mcp      Run Model Context Protocol (MCP) server over stdio")
	fmt.Println("  query    Execute vector, keyword, or hybrid search from CLI")
	fmt.Println("  index    Bulk ingest documents from JSONL or single text")
	fmt.Println("  stats    Inspect database page count, vector dim, and health")
	fmt.Println("  bench    Run local latency and throughput benchmarks")
}

func cmdServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	dbPath := fs.String("db", "data.embdb", "Path to .embdb container file")
	port := fs.Int("port", 8383, "HTTP port for studio and API")
	dim := fs.Int("dim", 384, "Vector dimension")
	metricStr := fs.String("metric", "cosine", "Distance metric (cosine, l2, dot)")
	quantStr := fs.String("quant", "none", "Quantization mode (none, sq8)")
	fs.Parse(args)

	fmt.Print(banner)

	metric := storage.MetricCosine
	if *metricStr == "l2" {
		metric = storage.MetricL2
	} else if *metricStr == "dot" {
		metric = storage.MetricDotProduct
	}

	quant := storage.QuantizeNone
	if *quantStr == "sq8" {
		quant = storage.QuantizeSQ8
	}

	start := time.Now()
	db, err := engine.OpenDB(*dbPath, *dim, metric, quant)
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()
	elapsed := time.Since(start)

	embedder := embeddings.NewDeterministicAdapter(*dim)
	distFS, err := server.GetEmbeddedFS()
	if err != nil {
		fmt.Printf("Warning: failed to load embedded studio assets: %v\n", err)
	}

	srv := server.NewServer(db, embedder, distFS)
	addr := fmt.Sprintf(":%d", *port)

	stats := db.Stats()
	fmt.Printf("\n🚀 embdb engine booted in %s\n", elapsed)
	fmt.Printf("  • Database File:   %s\n", *dbPath)
	fmt.Printf("  • Vector Dimension:%d\n", stats.VectorDim)
	fmt.Printf("  • Documents:       %d\n", stats.TotalDocuments)
	fmt.Printf("  • Web Studio:      http://localhost:%d\n", *port)
	fmt.Printf("  • MCP Endpoint:    http://localhost:%d/mcp\n", *port)
	fmt.Printf("  • SSE Stream:      http://localhost:%d/events\n\n", *port)

	if err := http.ListenAndServe(addr, srv.Routes()); err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}
}

func cmdMCP(args []string) {
	fs := flag.NewFlagSet("mcp", flag.ExitOnError)
	dbPath := fs.String("db", "data.embdb", "Path to .embdb container file")
	dim := fs.Int("dim", 384, "Vector dimension")
	fs.Parse(args)

	db, err := engine.OpenDB(*dbPath, *dim, storage.MetricCosine, storage.QuantizeNone)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening db: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	embedder := embeddings.NewDeterministicAdapter(*dim)
	srv := mcp.NewServer(db, embedder)

	if err := srv.ServeStdio(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "MCP stdio loop error: %v\n", err)
		os.Exit(1)
	}
}

func cmdQuery(args []string) {
	fs := flag.NewFlagSet("query", flag.ExitOnError)
	dbPath := fs.String("db", "data.embdb", "Path to .embdb container file")
	queryText := fs.String("query", "", "Search query text")
	mode := fs.String("mode", "hybrid", "Search mode (hybrid, vector, bm25)")
	limit := fs.Int("limit", 5, "Number of results to return")
	denseWeight := fs.Float64("dense-weight", 0.5, "Weight for dense vector")
	sparseWeight := fs.Float64("sparse-weight", 0.5, "Weight for BM25 sparse")
	fs.Parse(args)

	if *queryText == "" {
		fmt.Println("Error: --query is required")
		os.Exit(1)
	}

	db, err := engine.OpenDB(*dbPath, 384, storage.MetricCosine, storage.QuantizeNone)
	if err != nil {
		fmt.Printf("Error opening db: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	embedder := embeddings.NewDeterministicAdapter(db.Stats().VectorDim)
	vec, _ := embedder.Embed(context.Background(), *queryText)

	start := time.Now()
	var results []engine.SearchResult
	switch *mode {
	case "vector":
		results, err = db.SearchVector(vec, *limit, nil)
	case "bm25":
		results, err = db.SearchBM25(*queryText, *limit, nil)
	default:
		results, err = db.SearchHybrid(*queryText, vec, *limit, nil, float32(*denseWeight), float32(*sparseWeight))
	}
	elapsed := time.Since(start)

	if err != nil {
		fmt.Printf("Search failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nSearch results for '%s' (%s mode, %s):\n", *queryText, *mode, elapsed)
	for i, r := range results {
		fmt.Printf("[%d] %s (Score: %.4f)\n", i+1, r.Key, r.Score)
		fmt.Printf("    %s\n", r.Text)
	}
}

func cmdIndex(args []string) {
	fs := flag.NewFlagSet("index", flag.ExitOnError)
	dbPath := fs.String("db", "data.embdb", "Path to .embdb container file")
	filePath := fs.String("file", "", "Path to JSONL dataset file")
	key := fs.String("key", "", "Document key")
	text := fs.String("text", "", "Document text")
	category := fs.String("category", "", "Document category")
	fs.Parse(args)

	db, err := engine.OpenDB(*dbPath, 384, storage.MetricCosine, storage.QuantizeNone)
	if err != nil {
		fmt.Printf("Error opening db: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	embedder := embeddings.NewDeterministicAdapter(db.Stats().VectorDim)

	if *filePath != "" {
		file, err := os.Open(*filePath)
		if err != nil {
			fmt.Printf("Failed to open file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		count := 0
		start := time.Now()
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			var item struct {
				Key      string         `json:"key"`
				Text     string         `json:"text"`
				Category string         `json:"category"`
				Metadata map[string]any `json:"metadata"`
			}
			if err := json.Unmarshal([]byte(line), &item); err != nil {
				continue
			}

			if item.Key == "" {
				item.Key = fmt.Sprintf("doc_%d", count+1)
			}
			if item.Metadata == nil {
				item.Metadata = make(map[string]any)
			}
			if item.Category != "" {
				item.Metadata["category"] = item.Category
			}

			vec, _ := embedder.Embed(context.Background(), item.Text)
			doc := &engine.Document{
				Key:      item.Key,
				Text:     item.Text,
				Vector:   vec,
				Metadata: item.Metadata,
			}
			if err := db.Insert(doc); err != nil {
				fmt.Printf("Insert error on item %s: %v\n", item.Key, err)
			} else {
				count++
			}
		}
		_ = db.Sync()
		fmt.Printf("Successfully indexed %d documents in %s\n", count, time.Since(start))
		return
	}

	if *text != "" {
		if *key == "" {
			*key = fmt.Sprintf("doc_%d", time.Now().Unix())
		}
		meta := make(map[string]any)
		if *category != "" {
			meta["category"] = *category
		}

		vec, _ := embedder.Embed(context.Background(), *text)
		doc := &engine.Document{
			Key:      *key,
			Text:     *text,
			Vector:   vec,
			Metadata: meta,
		}
		if err := db.Insert(doc); err != nil {
			fmt.Printf("Index error: %v\n", err)
			os.Exit(1)
		}
		_ = db.Sync()
		fmt.Printf("Indexed document '%s' successfully\n", *key)
		return
	}

	fmt.Println("Error: provide either --file <dataset.jsonl> or --text <content>")
}

func cmdStats(args []string) {
	fs := flag.NewFlagSet("stats", flag.ExitOnError)
	dbPath := fs.String("db", "data.embdb", "Path to .embdb container file")
	fs.Parse(args)

	db, err := engine.OpenDB(*dbPath, 384, storage.MetricCosine, storage.QuantizeNone)
	if err != nil {
		fmt.Printf("Error opening db: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	stats := db.Stats()
	fi, _ := os.Stat(*dbPath)
	var fileSize int64
	if fi != nil {
		fileSize = fi.Size()
	}

	fmt.Printf("\n=== embdb Database Statistics ===\n")
	fmt.Printf("File Path:         %s\n", *dbPath)
	fmt.Printf("File Size on Disk: %d bytes (%.2f KB)\n", fileSize, float64(fileSize)/1024.0)
	fmt.Printf("Total Documents:   %d\n", stats.TotalDocuments)
	fmt.Printf("Vector Dimension:  %d\n", stats.VectorDim)
	fmt.Printf("Allocated Pages:   %d (4KB per page)\n", stats.TotalPages)
	fmt.Printf("Knowledge Nodes:   %d\n", stats.GraphNodes)
	fmt.Printf("Knowledge Edges:   %d\n", stats.GraphEdges)
}

func cmdBench(args []string) {
	fs := flag.NewFlagSet("bench", flag.ExitOnError)
	numDocs := fs.Int("num-docs", 5000, "Number of benchmark documents")
	dim := fs.Int("dim", 128, "Vector dimension")
	fs.Parse(args)

	tempDir, err := os.MkdirTemp("", "embdb-bench-*")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "bench.embdb")
	db, err := engine.OpenDB(dbPath, *dim, storage.MetricCosine, storage.QuantizeSQ8)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()

	embedder := embeddings.NewDeterministicAdapter(*dim)

	fmt.Printf("Running embdb benchmark with %d vectors (dimension: %d, SQ8 quantization)...\n", *numDocs, *dim)

	// Insertion Benchmark
	startInsert := time.Now()
	for i := 0; i < *numDocs; i++ {
		text := fmt.Sprintf("Technical document chunk %d discussing vector indexing and BM25 search efficiency", i)
		vec, _ := embedder.Embed(context.Background(), text)
		_ = db.Insert(&engine.Document{
			Key:    fmt.Sprintf("doc_%d", i),
			Text:   text,
			Vector: vec,
		})
	}
	insertDuration := time.Since(startInsert)
	insertThroughput := float64(*numDocs) / insertDuration.Seconds()
	fmt.Printf("Insertion: %d documents in %s (%.1f docs/sec)\n", *numDocs, insertDuration, insertThroughput)

	// Hybrid Search Latency Benchmark
	numQueries := 100
	queryText := "vector indexing efficiency"
	queryVec, _ := embedder.Embed(context.Background(), queryText)

	startQuery := time.Now()
	for i := 0; i < numQueries; i++ {
		_, _ = db.SearchHybrid(queryText, queryVec, 10, nil, 0.5, 0.5)
	}
	queryDuration := time.Since(startQuery)
	avgLatency := float64(queryDuration.Milliseconds()) / float64(numQueries)
	qps := float64(numQueries) / queryDuration.Seconds()

	fmt.Printf("Hybrid Search: %d queries in %s (Average Latency: %.2f ms, %.1f QPS)\n", numQueries, queryDuration, avgLatency, qps)
}
