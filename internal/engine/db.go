package engine

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alexandrmotologa/embdb/internal/bm25"
	"github.com/alexandrmotologa/embdb/internal/graph"
	"github.com/alexandrmotologa/embdb/internal/hnsw"
	"github.com/alexandrmotologa/embdb/internal/metadata"
	"github.com/alexandrmotologa/embdb/internal/storage"
)

var (
	ErrNotFound       = errors.New("db: document not found")
	ErrAlreadyExists  = errors.New("db: document key already exists")
	ErrInvalidVector  = errors.New("db: vector dimension mismatch")
)

type Document struct {
	ID        uint64         `json:"id"`
	Key       string         `json:"key"`
	Text      string         `json:"text"`
	Vector    []float32      `json:"vector,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt int64          `json:"created_at"`
}

type SearchResult struct {
	ID         uint64         `json:"id"`
	Key        string         `json:"key"`
	Text       string         `json:"text"`
	Score      float32        `json:"score"`
	DenseRank  int            `json:"dense_rank,omitempty"`
	SparseRank int            `json:"sparse_rank,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type QueryResult struct {
	Columns    []string         `json:"columns"`
	Rows       []map[string]any `json:"rows"`
	TotalHits  int              `json:"total_hits"`
	DurationMs float64          `json:"duration_ms"`
}

type DBStats struct {
	TotalDocuments uint64 `json:"total_documents"`
	VectorDim      int    `json:"vector_dimension"`
	GraphNodes     int    `json:"graph_nodes"`
	GraphEdges     int    `json:"graph_edges"`
	TotalPages     uint32 `json:"total_pages"`
	Metric         uint32 `json:"metric"`
	Quantization   uint32 `json:"quantization"`
}

type DB struct {
	mu        sync.RWMutex
	path      string
	container *storage.Container
	hnsw      *hnsw.Graph
	bm25      *bm25.Index
	kg        *graph.KnowledgeGraph
	docs      map[uint64]*Document
	keyToID   map[string]uint64
	dim       int
	autoID    atomic.Uint64
}

// OpenDB opens or creates an embdb database at the specified path.
func OpenDB(path string, dim int, metric uint32, quant uint32) (*DB, error) {
	var c *storage.Container
	var err error

	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		c, err = storage.CreateContainer(path, uint32(dim), metric, quant)
	} else {
		c, err = storage.OpenContainer(path)
	}
	if err != nil {
		return nil, err
	}

	h := c.Header()
	actualDim := int(h.VectorDimension)
	if actualDim == 0 {
		actualDim = dim
	}

	cfg := hnsw.DefaultConfig(actualDim)
	cfg.Metric = h.DistanceMetric
	cfg.Quantize = (h.QuantizationMode == storage.QuantizeSQ8)

	db := &DB{
		path:      path,
		container: c,
		hnsw:      hnsw.NewGraph(cfg),
		bm25:      bm25.NewIndex(),
		kg:        graph.NewKnowledgeGraph(),
		docs:      make(map[uint64]*Document),
		keyToID:   make(map[string]uint64),
		dim:       actualDim,
	}

	return db, nil
}

// Insert adds a new document to the database.
func (db *DB) Insert(doc *Document) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if doc.Key != "" {
		if _, exists := db.keyToID[doc.Key]; exists {
			return ErrAlreadyExists
		}
	}

	if len(doc.Vector) > 0 && len(doc.Vector) != db.dim {
		return ErrInvalidVector
	}

	newID := db.autoID.Add(1)
	doc.ID = newID
	if doc.CreatedAt == 0 {
		doc.CreatedAt = time.Now().Unix()
	}

	// Index in HNSW
	if len(doc.Vector) > 0 {
		normalized := hnsw.Normalize(doc.Vector)
		if err := db.hnsw.Insert(newID, normalized); err != nil {
			return err
		}
	}

	// Index in BM25
	if doc.Text != "" {
		db.bm25.IndexDocument(newID, doc.Text)
	}

	// Index in Knowledge Graph if entity is referenced
	db.kg.AddNode(graph.Node{
		ID:         doc.Key,
		DocID:      newID,
		Type:       "Document",
		Properties: doc.Metadata,
	})

	// Record in memory document map
	db.docs[newID] = doc
	if doc.Key != "" {
		db.keyToID[doc.Key] = newID
	}

	// Write to slotted page container
	dataBytes, _ := json.Marshal(doc)
	page, err := db.container.AllocatePage(storage.PageTypeData)
	if err == nil {
		_, _ = page.InsertItem(dataBytes)
		_ = db.container.WritePage(page)
	}

	return nil
}

// Upsert inserts or updates a document.
func (db *DB) Upsert(doc *Document) error {
	db.mu.Lock()
	if doc.Key != "" {
		if existingID, exists := db.keyToID[doc.Key]; exists {
			doc.ID = existingID
			db.docs[existingID] = doc
			db.mu.Unlock()
			return nil
		}
	}
	db.mu.Unlock()
	return db.Insert(doc)
}

// Get retrieves a document by internal docID.
func (db *DB) Get(id uint64) (*Document, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	doc, exists := db.docs[id]
	if !exists {
		return nil, ErrNotFound
	}
	return doc, nil
}

// GetByKey retrieves a document by user key.
func (db *DB) GetByKey(key string) (*Document, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	id, exists := db.keyToID[key]
	if !exists {
		return nil, ErrNotFound
	}
	return db.docs[id], nil
}

// SearchVector performs dense vector search with optional metadata predicate filter.
func (db *DB) SearchVector(query []float32, k int, filter *metadata.Predicate) ([]SearchResult, error) {
	normQ := hnsw.Normalize(query)

	var hnswFilter func(id uint64) bool
	if filter != nil {
		hnswFilter = func(id uint64) bool {
			db.mu.RLock()
			doc := db.docs[id]
			db.mu.RUnlock()
			if doc == nil {
				return false
			}
			return filter.Evaluate(doc.Metadata)
		}
	}

	items, err := db.hnsw.Search(normQ, k, hnswFilter)
	if err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, len(items))
	db.mu.RLock()
	defer db.mu.RUnlock()

	for _, it := range items {
		doc := db.docs[it.ID]
		if doc == nil {
			continue
		}
		// Convert distance to similarity score
		score := 1.0 - it.Distance
		results = append(results, SearchResult{
			ID:       doc.ID,
			Key:      doc.Key,
			Text:     doc.Text,
			Score:    score,
			Metadata: doc.Metadata,
		})
	}

	return results, nil
}

// SearchBM25 performs lexical keyword search with optional metadata predicate filter.
func (db *DB) SearchBM25(queryText string, k int, filter *metadata.Predicate) ([]SearchResult, error) {
	var bm25Filter func(id uint64) bool
	if filter != nil {
		bm25Filter = func(id uint64) bool {
			db.mu.RLock()
			doc := db.docs[id]
			db.mu.RUnlock()
			if doc == nil {
				return false
			}
			return filter.Evaluate(doc.Metadata)
		}
	}

	matches := db.bm25.Search(queryText, k, bm25Filter)

	results := make([]SearchResult, 0, len(matches))
	db.mu.RLock()
	defer db.mu.RUnlock()

	for _, m := range matches {
		doc := db.docs[m.DocID]
		if doc == nil {
			continue
		}
		results = append(results, SearchResult{
			ID:       doc.ID,
			Key:      doc.Key,
			Text:     doc.Text,
			Score:    m.Score,
			Metadata: doc.Metadata,
		})
	}

	return results, nil
}

// SearchHybrid combines dense vector and sparse BM25 results using Reciprocal Rank Fusion.
func (db *DB) SearchHybrid(
	queryText string,
	queryVector []float32,
	k int,
	filter *metadata.Predicate,
	denseWeight, sparseWeight float32,
) ([]SearchResult, error) {
	denseK := k * 2
	sparseK := k * 2

	var denseItems []RankItem
	if len(queryVector) > 0 {
		denseRes, err := db.SearchVector(queryVector, denseK, filter)
		if err == nil {
			for _, r := range denseRes {
				denseItems = append(denseItems, RankItem{
					DocID:      r.ID,
					DenseScore: r.Score,
				})
			}
		}
	}

	var sparseItems []RankItem
	if queryText != "" {
		sparseRes, err := db.SearchBM25(queryText, sparseK, filter)
		if err == nil {
			for _, r := range sparseRes {
				sparseItems = append(sparseItems, RankItem{
					DocID:       r.ID,
					SparseScore: r.Score,
				})
			}
		}
	}

	fused := ReciprocalRankFusion(denseItems, sparseItems, denseWeight, sparseWeight, 60)

	if k > 0 && len(fused) > k {
		fused = fused[:k]
	}

	results := make([]SearchResult, 0, len(fused))
	db.mu.RLock()
	defer db.mu.RUnlock()

	for _, item := range fused {
		doc := db.docs[item.DocID]
		if doc == nil {
			continue
		}
		results = append(results, SearchResult{
			ID:         doc.ID,
			Key:        doc.Key,
			Text:       doc.Text,
			Score:      item.FinalScore,
			DenseRank:  item.DenseRank,
			SparseRank: item.SparseRank,
			Metadata:   doc.Metadata,
		})
	}

	return results, nil
}

// ExecuteSQL executes a structured semantic query.
func (db *DB) ExecuteSQL(queryStr string, queryVector []float32) (*QueryResult, error) {
	start := time.Now()
	parsed, err := ParseSQL(queryStr)
	if err != nil {
		return nil, err
	}

	var rawResults []SearchResult
	switch parsed.Order.Mode {
	case OrderModeHybrid:
		rawResults, err = db.SearchHybrid(parsed.Order.QueryText, queryVector, parsed.Limit+parsed.Offset, parsed.Where, 0.5, 0.5)
	case OrderModeBM25:
		rawResults, err = db.SearchBM25(parsed.Order.QueryText, parsed.Limit+parsed.Offset, parsed.Where)
	case OrderModeVector:
		rawResults, err = db.SearchVector(queryVector, parsed.Limit+parsed.Offset, parsed.Where)
	default:
		// Full collection scan with WHERE filter
		db.mu.RLock()
		for _, doc := range db.docs {
			if parsed.Where == nil || parsed.Where.Evaluate(doc.Metadata) {
				rawResults = append(rawResults, SearchResult{
					ID:       doc.ID,
					Key:      doc.Key,
					Text:     doc.Text,
					Metadata: doc.Metadata,
				})
			}
		}
		db.mu.RUnlock()
	}

	if err != nil {
		return nil, err
	}

	// Apply Offset
	if parsed.Offset > 0 {
		if parsed.Offset < len(rawResults) {
			rawResults = rawResults[parsed.Offset:]
		} else {
			rawResults = nil
		}
	}
	// Apply Limit
	if parsed.Limit > 0 && len(rawResults) > parsed.Limit {
		rawResults = rawResults[:parsed.Limit]
	}

	rows := make([]map[string]any, 0, len(rawResults))
	for _, res := range rawResults {
		row := make(map[string]any)
		for _, f := range parsed.Fields {
			switch f {
			case "*":
				row["id"] = res.ID
				row["key"] = res.Key
				row["text"] = res.Text
				row["score"] = res.Score
				row["metadata"] = res.Metadata
			case "id":
				row["id"] = res.ID
			case "key":
				row["key"] = res.Key
			case "text":
				row["text"] = res.Text
			case "score":
				row["score"] = res.Score
			default:
				if res.Metadata != nil {
					if val, ok := res.Metadata[f]; ok {
						row[f] = val
					}
				}
			}
		}
		rows = append(rows, row)
	}

	duration := float64(time.Since(start).Microseconds()) / 1000.0
	return &QueryResult{
		Columns:    parsed.Fields,
		Rows:       rows,
		TotalHits:  len(rows),
		DurationMs: duration,
	}, nil
}

func (db *DB) KnowledgeGraph() *graph.KnowledgeGraph {
	return db.kg
}

func (db *DB) Stats() DBStats {
	db.mu.RLock()
	defer db.mu.RUnlock()

	h := db.container.Header()
	return DBStats{
		TotalDocuments: uint64(len(db.docs)),
		VectorDim:      db.dim,
		GraphNodes:     db.kg.NodeCount(),
		GraphEdges:     db.kg.EdgeCount(),
		TotalPages:     h.TotalPages,
		Metric:         h.DistanceMetric,
		Quantization:   h.QuantizationMode,
	}
}

func (db *DB) Sync() error {
	return db.container.Sync()
}

func (db *DB) Close() error {
	return db.container.Close()
}
