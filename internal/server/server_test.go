package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/alexandrmotologa/embdb/internal/embeddings"
	"github.com/alexandrmotologa/embdb/internal/engine"
)

func TestServerEndpoints(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "embdb-server-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.embdb")
	db, err := engine.OpenDB(dbPath, 16, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	embedder := embeddings.NewDeterministicAdapter(16)
	srv := NewServer(db, embedder, nil)
	handler := srv.Routes()

	// 1. Health check
	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from health, got %d", w.Code)
	}

	// 2. Stats
	reqStats := httptest.NewRequest("GET", "/api/stats", nil)
	wStats := httptest.NewRecorder()
	handler.ServeHTTP(wStats, reqStats)
	if wStats.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from stats, got %d", wStats.Code)
	}

	// 3. Insert Document
	docBody, _ := json.Marshal(map[string]any{
		"key":      "doc_terms",
		"text":     "General terms of service and commercial agreement",
		"metadata": map[string]any{"category": "legal"},
	})
	reqDoc := httptest.NewRequest("POST", "/api/documents", bytes.NewReader(docBody))
	wDoc := httptest.NewRecorder()
	handler.ServeHTTP(wDoc, reqDoc)
	if wDoc.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created from documents, got %d", wDoc.Code)
	}

	// 4. Hybrid Search
	searchBody, _ := json.Marshal(map[string]any{
		"query": "commercial agreement",
		"mode":  "hybrid",
		"limit": 5,
	})
	reqSearch := httptest.NewRequest("POST", "/api/search", bytes.NewReader(searchBody))
	wSearch := httptest.NewRecorder()
	handler.ServeHTTP(wSearch, reqSearch)
	if wSearch.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from search, got %d", wSearch.Code)
	}

	// 5. Semantic SQL
	sqlBody, _ := json.Marshal(map[string]any{
		"sql": "SELECT id, key, text FROM documents WHERE category = 'legal' ORDER BY hybrid('agreement') LIMIT 5",
	})
	reqSQL := httptest.NewRequest("POST", "/api/query/sql", bytes.NewReader(sqlBody))
	wSQL := httptest.NewRecorder()
	handler.ServeHTTP(wSQL, reqSQL)
	if wSQL.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from sql query, got %d: %s", wSQL.Code, wSQL.Body.String())
	}
}
