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
	case "shell":
		cmdShell(os.Args[2:])
	case "compact":
		cmdCompact(os.Args[2:])
	case "dump":
		cmdDump(os.Args[2:])
	case "restore":
		cmdRestore(os.Args[2:])
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
	fmt.Println("  shell    Launch interactive SQL & semantic query terminal REPL")
	fmt.Println("  mcp      Run Model Context Protocol (MCP) server over stdio")
	fmt.Println("  query    Execute vector, keyword, or hybrid search from CLI")
	fmt.Println("  index    Bulk ingest documents from JSONL or single text")
	fmt.Println("  compact  Defragment and shrink database file on disk")
	fmt.Println("  dump     Export all documents and graph data to portable JSONL")
	fmt.Println("  restore  Rebuild database container from JSONL dump")
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

func cmdCompact(args []string) {
	fs := flag.NewFlagSet("compact", flag.ExitOnError)
	dbPath := fs.String("db", "data.embdb", "Path to source .embdb file")
	outPath := fs.String("out", "", "Path to compacted file (defaults to in-place replacement)")
	fs.Parse(args)

	target := *outPath
	replace := false
	if target == "" {
		target = *dbPath + ".compact.tmp"
		replace = true
	}

	db, err := engine.OpenDB(*dbPath, 384, storage.MetricCosine, storage.QuantizeNone)
	if err != nil {
		fmt.Printf("Error opening db: %v\n", err)
		os.Exit(1)
	}

	start := time.Now()
	if err := db.Compact(target); err != nil {
		db.Close()
		fmt.Printf("Compaction failed: %v\n", err)
		os.Exit(1)
	}
	db.Close()

	if replace {
		_ = os.Remove(*dbPath)
		_ = os.Remove(*dbPath + "-wal")
		_ = os.Rename(target, *dbPath)
		fmt.Printf("Database successfully compacted in-place in %s\n", time.Since(start))
	} else {
		fmt.Printf("Database compacted to %s in %s\n", target, time.Since(start))
	}
}

func cmdDump(args []string) {
	fs := flag.NewFlagSet("dump", flag.ExitOnError)
	dbPath := fs.String("db", "data.embdb", "Path to source .embdb file")
	outFile := fs.String("out", "dump.jsonl", "Output dump file path")
	fs.Parse(args)

	db, err := engine.OpenDB(*dbPath, 384, storage.MetricCosine, storage.QuantizeNone)
	if err != nil {
		fmt.Printf("Error opening db: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	f, err := os.Create(*outFile)
	if err != nil {
		fmt.Printf("Failed to create dump file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	queryRes, err := db.ExecuteSQL("SELECT * FROM documents LIMIT 1000000", nil)
	if err != nil {
		fmt.Printf("Query error: %v\n", err)
		os.Exit(1)
	}

	count := 0
	for _, row := range queryRes.Rows {
		b, _ := json.Marshal(row)
		f.Write(b)
		f.WriteString("\n")
		count++
	}

	fmt.Printf("Dumped %d documents to %s\n", count, *outFile)
}

func cmdRestore(args []string) {
	fs := flag.NewFlagSet("restore", flag.ExitOnError)
	dbPath := fs.String("db", "restored.embdb", "Path to target .embdb file")
	inFile := fs.String("in", "dump.jsonl", "Input dump file path")
	dim := fs.Int("dim", 384, "Vector dimension")
	fs.Parse(args)

	f, err := os.Open(*inFile)
	if err != nil {
		fmt.Printf("Failed to open dump file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	db, err := engine.OpenDB(*dbPath, *dim, storage.MetricCosine, storage.QuantizeNone)
	if err != nil {
		fmt.Printf("Error creating db: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	embedder := embeddings.NewDeterministicAdapter(*dim)
	scanner := bufio.NewScanner(f)
	count := 0
	start := time.Now()

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var row map[string]any
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			continue
		}

		key, _ := row["key"].(string)
		text, _ := row["text"].(string)
		meta, _ := row["metadata"].(map[string]any)

		vec, _ := embedder.Embed(context.Background(), text)
		_ = db.Insert(&engine.Document{
			Key:      key,
			Text:     text,
			Vector:   vec,
			Metadata: meta,
		})
		count++
	}

	_ = db.Sync()
	fmt.Printf("Restored %d documents to %s in %s\n", count, *dbPath, time.Since(start))
}

func cmdShell(args []string) {
	fs := flag.NewFlagSet("shell", flag.ExitOnError)
	dbPath := fs.String("db", "data.embdb", "Path to .embdb file")
	dim := fs.Int("dim", 384, "Vector dimension")
	fs.Parse(args)

	db, err := engine.OpenDB(*dbPath, *dim, storage.MetricCosine, storage.QuantizeNone)
	if err != nil {
		fmt.Printf("Error opening db: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	embedder := embeddings.NewDeterministicAdapter(*dim)

	fmt.Print(banner)
	fmt.Printf("Connected to: %s\n", *dbPath)
	fmt.Println("Type SQL query (e.g. SELECT * FROM documents LIMIT 5;)")
	fmt.Println("Type .help for special commands, .exit to quit.")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("embdb> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, ".") {
			switch line {
			case ".help":
				fmt.Println("\nShell Commands:")
				fmt.Println("  .help        Show this help message")
				fmt.Println("  .stats       Show database statistics")
				fmt.Println("  .schema      Show columns in documents table")
				fmt.Println("  .exit, .quit Exit the shell")
			case ".stats":
				stats := db.Stats()
				fmt.Printf("Total Documents: %d | Pages: %d | Vector Dim: %d | Graph Nodes: %d\n",
					stats.TotalDocuments, stats.TotalPages, stats.VectorDim, stats.GraphNodes)
			case ".schema":
				fmt.Println("Table 'documents': [id uint64, key string, text string, vector []float32, metadata object, score float32]")
			case ".exit", ".quit":
				return
			default:
				fmt.Printf("Unknown command: %s. Type .help for assistance.\n", line)
			}
			continue
		}

		// Execute SQL Query
		cleanSQL := strings.TrimSuffix(line, ";")
		start := time.Now()
		qVec, _ := embedder.Embed(context.Background(), cleanSQL)
		res, err := db.ExecuteSQL(cleanSQL, qVec)
		elapsed := time.Since(start)

		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		if len(res.Rows) == 0 {
			fmt.Printf("Empty set (%s)\n", elapsed)
			continue
		}

		// Print ASCII Table
		renderTable(res)
		fmt.Printf("%d rows in set (%s)\n\n", len(res.Rows), elapsed)
	}
}

func renderTable(res *engine.QueryResult) {
	if len(res.Rows) == 0 {
		return
	}

	cols := res.Columns
	if len(cols) == 1 && cols[0] == "*" {
		cols = []string{"id", "key", "text", "score"}
	}

	// Calculate column widths
	widths := make([]int, len(cols))
	for i, c := range cols {
		widths[i] = len(c)
	}

	for _, r := range res.Rows {
		for i, c := range cols {
			valStr := fmt.Sprintf("%v", r[c])
			if len(valStr) > 40 {
				valStr = valStr[:37] + "..."
			}
			if len(valStr) > widths[i] {
				widths[i] = len(valStr)
			}
		}
	}

	// Print Border
	printSep := func() {
		fmt.Print("+")
		for _, w := range widths {
			fmt.Print(strings.Repeat("-", w+2) + "+")
		}
		fmt.Println()
	}

	printSep()
	// Print Header
	fmt.Print("|")
	for i, c := range cols {
		fmt.Printf(" %-*s |", widths[i], strings.ToUpper(c))
	}
	fmt.Println()
	printSep()

	// Print Rows
	for _, r := range res.Rows {
		fmt.Print("|")
		for i, c := range cols {
			valStr := fmt.Sprintf("%v", r[c])
			if len(valStr) > 40 {
				valStr = valStr[:37] + "..."
			}
			fmt.Printf(" %-*s |", widths[i], valStr)
		}
		fmt.Println()
	}
	printSep()
}
