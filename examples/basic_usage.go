package main

import (
	"context"
	"fmt"
	"os"

	"github.com/alexandrmotologa/embdb/internal/embeddings"
	"github.com/alexandrmotologa/embdb/internal/engine"
	"github.com/alexandrmotologa/embdb/internal/storage"
)

func main() {
	dbPath := "demo.embdb"
	defer os.Remove(dbPath)
	defer os.Remove(dbPath + "-wal")

	// 1. Open or create embedded database
	// Dimensions: 128, Metric: Cosine, Quantization: SQ8
	db, err := engine.OpenDB(dbPath, 128, storage.MetricCosine, storage.QuantizeSQ8)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// 2. Initialize embedding provider
	embedder := embeddings.NewDeterministicAdapter(128)

	// 3. Insert documents
	documents := []struct {
		Key      string
		Category string
		Text     string
	}{
		{
			Key:      "doc_legal_1",
			Category: "legal",
			Text:     "Non-disclosure agreement and liability indemnification clauses for software vendors.",
		},
		{
			Key:      "doc_security_2",
			Category: "security",
			Text:     "Zero trust architecture requiring mutual TLS authentication and hardware security tokens.",
		},
		{
			Key:      "doc_database_3",
			Category: "database",
			Text:     "Single binary vector search engine using HNSW graph traversal and BM25 lexical indexing.",
		},
	}

	for _, d := range documents {
		vec, _ := embedder.Embed(context.Background(), d.Text)
		err := db.Insert(&engine.Document{
			Key:    d.Key,
			Text:   d.Text,
			Vector: vec,
			Metadata: map[string]any{
				"category": d.Category,
			},
		})
		if err != nil {
			panic(err)
		}
	}

	// 4. Hybrid Search with Reciprocal Rank Fusion
	queryText := "vector search and lexical indexing"
	queryVec, _ := embedder.Embed(context.Background(), queryText)

	results, err := db.SearchHybrid(queryText, queryVec, 5, nil, 0.5, 0.5)
	if err != nil {
		panic(err)
	}

	fmt.Println("=== Hybrid Search Results ===")
	for i, r := range results {
		fmt.Printf("[%d] %s (Score: %.4f)\n", i+1, r.Key, r.Score)
		fmt.Printf("    %s\n", r.Text)
	}

	// 5. Semantic SQL Query with Metadata Filtering
	sqlQuery := "SELECT id, key, text FROM documents WHERE category = 'database' ORDER BY hybrid('HNSW indexing') LIMIT 5"
	sqlRes, err := db.ExecuteSQL(sqlQuery, queryVec)
	if err != nil {
		panic(err)
	}

	fmt.Println("\n=== SQL Query Results ===")
	for _, row := range sqlRes.Rows {
		fmt.Printf("- Key: %v, Text: %v\n", row["key"], row["text"])
	}

	// 6. Inspect Database Statistics
	stats := db.Stats()
	fmt.Printf("\n=== Database Statistics ===\n")
	fmt.Printf("Total Documents: %d\n", stats.TotalDocuments)
	fmt.Printf("Allocated Pages: %d (4KB per page)\n", stats.TotalPages)
}
