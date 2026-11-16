package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestDatabaseCompaction(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "embdb-compact-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "original.embdb")
	db, err := OpenDB(dbPath, 4, 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		_ = db.Insert(&Document{
			Key:  fmt.Sprintf("doc_%d", i),
			Text: fmt.Sprintf("Text content %d", i),
		})
	}

	compactPath := filepath.Join(tempDir, "compacted.embdb")
	if err := db.Compact(compactPath); err != nil {
		t.Fatalf("compact failed: %v", err)
	}
	_ = db.Close()

	// Verify compacted DB opens cleanly and has all 10 documents
	dbCompact, err := OpenDB(compactPath, 4, 0, 0)
	if err != nil {
		t.Fatalf("failed to open compacted db: %v", err)
	}
	defer dbCompact.Close()

	if dbCompact.Stats().TotalDocuments != 10 {
		t.Fatalf("expected 10 documents in compacted db, got %d", dbCompact.Stats().TotalDocuments)
	}
}

func TestComputeClusters(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "embdb-cluster-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "cluster.embdb")
	db, err := OpenDB(dbPath, 4, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Cluster A docs
	_ = db.Insert(&Document{Key: "legal_1", Text: "contract terms and liability", Vector: []float32{1, 0, 0, 0}})
	_ = db.Insert(&Document{Key: "legal_2", Text: "breach of contract clause", Vector: []float32{0.9, 0.1, 0, 0}})

	// Cluster B docs
	_ = db.Insert(&Document{Key: "tech_1", Text: "vector indexing and hardware registers", Vector: []float32{0, 0, 1, 0}})
	_ = db.Insert(&Document{Key: "tech_2", Text: "simd acceleration and memory buffers", Vector: []float32{0, 0, 0.9, 0.1}})

	res := db.ComputeClusters(2, 10)
	if len(res.Clusters) != 2 {
		t.Fatalf("expected 2 clusters, got %d", len(res.Clusters))
	}
	if res.TotalDocs != 4 {
		t.Fatalf("expected 4 clustered docs, got %d", res.TotalDocs)
	}
}

func TestDetectQueryIntent(t *testing.T) {
	// Technical / code / exact identifier query -> should favor BM25 sparse
	denseW, sparseW := DetectQueryIntent("error_code_500 AES-256")
	if sparseW <= denseW {
		t.Fatalf("expected sparse weight to exceed dense weight for technical code query, got dense: %f, sparse: %f", denseW, sparseW)
	}

	// Natural language conceptual question -> should favor HNSW dense
	denseW2, sparseW2 := DetectQueryIntent("what is the architectural difference between relational databases and graph models?")
	if denseW2 <= sparseW2 {
		t.Fatalf("expected dense weight to exceed sparse weight for conceptual question, got dense: %f, sparse: %f", denseW2, sparseW2)
	}
}

func TestRRF(t *testing.T) {
	dense := []RankItem{
		{DocID: 1, DenseScore: 0.95},
		{DocID: 2, DenseScore: 0.85},
		{DocID: 3, DenseScore: 0.70},
	}
	sparse := []RankItem{
		{DocID: 2, SparseScore: 4.5},
		{DocID: 4, SparseScore: 3.2},
		{DocID: 1, SparseScore: 2.1},
	}

	fused := ReciprocalRankFusion(dense, sparse, 0.5, 0.5, 60)
	if len(fused) != 4 {
		t.Fatalf("expected 4 merged items, got %d", len(fused))
	}

	// Doc 2 appears at rank 2 in dense and rank 1 in sparse -> should have high score
	if fused[0].DocID != 2 && fused[0].DocID != 1 {
		t.Fatalf("expected doc 1 or 2 to be top ranked, got doc %d", fused[0].DocID)
	}
}

func TestSQLParser(t *testing.T) {
	q, err := ParseSQL("SELECT id, text, score FROM documents WHERE category = 'legal' AND year >= 2024 ORDER BY hybrid('contract breach') LIMIT 5 OFFSET 2")
	if err != nil {
		t.Fatalf("failed to parse SQL: %v", err)
	}

	if q.Collection != "documents" {
		t.Fatalf("expected collection 'documents', got '%s'", q.Collection)
	}
	if len(q.Fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(q.Fields))
	}
	if q.Limit != 5 {
		t.Fatalf("expected limit 5, got %d", q.Limit)
	}
	if q.Offset != 2 {
		t.Fatalf("expected offset 2, got %d", q.Offset)
	}
	if q.Order.Mode != OrderModeHybrid {
		t.Fatalf("expected hybrid order mode, got %v", q.Order.Mode)
	}
	if q.Order.QueryText != "contract breach" {
		t.Fatalf("expected query 'contract breach', got '%s'", q.Order.QueryText)
	}
}

func TestDatabaseEngineIntegration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "embdb-engine-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.embdb")
	db, err := OpenDB(dbPath, 4, 0, 0)
	if err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	doc1 := &Document{
		Key:    "doc_legal_1",
		Text:   "Non-disclosure agreement and liability confidentiality clause",
		Vector: []float32{1.0, 0.0, 0.0, 0.0},
		Metadata: map[string]any{
			"category": "legal",
			"priority": 1,
		},
	}
	doc2 := &Document{
		Key:    "doc_legal_2",
		Text:   "Intellectual property assignment and patent licensing terms",
		Vector: []float32{0.0, 1.0, 0.0, 0.0},
		Metadata: map[string]any{
			"category": "legal",
			"priority": 2,
		},
	}
	doc3 := &Document{
		Key:    "doc_tech_1",
		Text:   "High performance distributed vector search algorithms",
		Vector: []float32{0.0, 0.0, 1.0, 0.0},
		Metadata: map[string]any{
			"category": "tech",
			"priority": 1,
		},
	}

	if err := db.Insert(doc1); err != nil {
		t.Fatal(err)
	}
	if err := db.Insert(doc2); err != nil {
		t.Fatal(err)
	}
	if err := db.Insert(doc3); err != nil {
		t.Fatal(err)
	}

	// 1. Vector Search
	vecResults, err := db.SearchVector([]float32{1.0, 0.0, 0.0, 0.0}, 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(vecResults) == 0 || vecResults[0].Key != "doc_legal_1" {
		t.Fatalf("expected doc_legal_1 as top vector match, got %v", vecResults)
	}

	// 2. BM25 Search
	bmResults, err := db.SearchBM25("patent licensing", 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(bmResults) == 0 || bmResults[0].Key != "doc_legal_2" {
		t.Fatalf("expected doc_legal_2 as top bm25 match, got %v", bmResults)
	}

	// 3. Hybrid Search
	hybridResults, err := db.SearchHybrid("confidentiality", []float32{1.0, 0.0, 0.0, 0.0}, 2, nil, 0.6, 0.4)
	if err != nil {
		t.Fatal(err)
	}
	if len(hybridResults) == 0 || hybridResults[0].Key != "doc_legal_1" {
		t.Fatalf("expected doc_legal_1 as top hybrid match, got %v", hybridResults)
	}

	// 4. SQL Execution
	sqlRes, err := db.ExecuteSQL("SELECT id, key, text FROM documents WHERE category = 'legal' ORDER BY bm25('patent') LIMIT 5", nil)
	if err != nil {
		t.Fatalf("SQL query failed: %v", err)
	}
	if sqlRes.TotalHits == 0 {
		t.Fatal("expected SQL hits, got 0")
	}
	if sqlRes.Rows[0]["key"] != "doc_legal_2" {
		t.Fatalf("expected doc_legal_2 in SQL query, got %v", sqlRes.Rows[0]["key"])
	}

	stats := db.Stats()
	if stats.TotalDocuments != 3 {
		t.Fatalf("expected 3 documents in stats, got %d", stats.TotalDocuments)
	}

	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	// Reopen database and verify persistent documents
	db2, err := OpenDB(dbPath, 4, 0, 0)
	if err != nil {
		t.Fatalf("failed to reopen DB: %v", err)
	}
	defer db2.Close()

	stats2 := db2.Stats()
	if stats2.TotalDocuments != 3 {
		t.Fatalf("expected 3 documents after reopening, got %d", stats2.TotalDocuments)
	}
}
