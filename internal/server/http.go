package server

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alexandrmotologa/embdb/internal/embeddings"
	"github.com/alexandrmotologa/embdb/internal/engine"
	"github.com/alexandrmotologa/embdb/internal/mcp"
)

type Server struct {
	db        *engine.DB
	embedder  embeddings.Adapter
	mcpServer *mcp.Server
	distFS    fs.FS
	sseMu     sync.Mutex
	sseChans  []chan string
}

func NewServer(db *engine.DB, embedder embeddings.Adapter, distFS fs.FS) *Server {
	if embedder == nil {
		embedder = embeddings.NewDeterministicAdapter(db.Stats().VectorDim)
	}

	s := &Server{
		db:        db,
		embedder:  embedder,
		mcpServer: mcp.NewServer(db, embedder),
		distFS:    distFS,
		sseChans:  make([]chan string, 0),
	}
	return s
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// REST Endpoints
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/stats", s.handleStats)
	mux.HandleFunc("POST /api/documents", s.handleInsertDocument)
	mux.HandleFunc("GET /api/documents", s.handleListDocuments)
	mux.HandleFunc("POST /api/search", s.handleSearch)
	mux.HandleFunc("POST /api/query/sql", s.handleSQL)
	mux.HandleFunc("GET /api/graph/traverse", s.handleGraphTraverse)
	mux.HandleFunc("GET /api/vectors/scatter", s.handleVectorScatter)

	// MCP HTTP Endpoint
	mux.HandleFunc("POST /mcp", s.mcpServer.HandleHTTP)

	// Server-Sent Events
	mux.HandleFunc("GET /events", s.handleSSE)

	// Embedded Static Frontend Studio
	if s.distFS != nil {
		fileServer := http.FileServer(http.FS(s.distFS))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/mcp" || r.URL.Path == "/events" {
				http.NotFound(w, r)
				return
			}

			// Check if file exists in distFS
			cleanPath := strings.TrimPrefix(r.URL.Path, "/")
			if cleanPath == "" {
				cleanPath = "index.html"
			}
			if f, err := s.distFS.Open(cleanPath); err == nil {
				f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}

			// Fallback to index.html for SPA client-side routing
			if indexFile, err := s.distFS.Open("index.html"); err == nil {
				defer indexFile.Close()
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				stat, _ := indexFile.Stat()
				http.ServeContent(w, r, "index.html", stat.ModTime(), indexFile.(io.ReadSeeker))
				return
			}

			http.NotFound(w, r)
		})
	}

	return s.corsMiddleware(mux)
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.jsonResponse(w, http.StatusOK, map[string]any{
		"status":  "healthy",
		"engine":  "embdb",
		"version": "1.0.0",
		"time":    time.Now().Unix(),
	})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	stats := s.db.Stats()
	s.jsonResponse(w, http.StatusOK, stats)
}

func (s *Server) handleInsertDocument(w http.ResponseWriter, r *http.Request) {
	type docReq struct {
		Key      string         `json:"key"`
		Text     string         `json:"text"`
		Vector   []float32      `json:"vector,omitempty"`
		Metadata map[string]any `json:"metadata,omitempty"`
	}

	var req docReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Text == "" {
		s.jsonError(w, http.StatusBadRequest, "text field is required")
		return
	}

	if len(req.Vector) == 0 {
		vec, err := s.embedder.Embed(r.Context(), req.Text)
		if err == nil {
			req.Vector = vec
		}
	}

	doc := &engine.Document{
		Key:      req.Key,
		Text:     req.Text,
		Vector:   req.Vector,
		Metadata: req.Metadata,
	}

	if err := s.db.Upsert(doc); err != nil {
		s.jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.broadcastEvent(fmt.Sprintf(`{"event": "document_indexed", "key": "%s", "id": %d}`, doc.Key, doc.ID))
	s.jsonResponse(w, http.StatusCreated, doc)
}

func (s *Server) handleListDocuments(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}

	query := fmt.Sprintf("SELECT * FROM documents LIMIT %d", limit)
	result, err := s.db.ExecuteSQL(query, nil)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, result)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	type searchReq struct {
		Query        string    `json:"query"`
		Vector       []float32 `json:"vector,omitempty"`
		Mode         string    `json:"mode"`
		Limit        int       `json:"limit"`
		DenseWeight  float32   `json:"dense_weight"`
		SparseWeight float32   `json:"sparse_weight"`
	}

	var req searchReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid search request")
		return
	}

	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Mode == "" {
		req.Mode = "hybrid"
	}
	if req.DenseWeight == 0 && req.SparseWeight == 0 {
		req.DenseWeight = 0.5
		req.SparseWeight = 0.5
	}

	ctx := r.Context()
	var results []engine.SearchResult
	var err error

	switch req.Mode {
	case "vector":
		if len(req.Vector) == 0 {
			req.Vector, _ = s.embedder.Embed(ctx, req.Query)
		}
		results, err = s.db.SearchVector(req.Vector, req.Limit, nil)
	case "bm25":
		results, err = s.db.SearchBM25(req.Query, req.Limit, nil)
	default: // hybrid
		if len(req.Vector) == 0 && req.Query != "" {
			req.Vector, _ = s.embedder.Embed(ctx, req.Query)
		}
		results, err = s.db.SearchHybrid(req.Query, req.Vector, req.Limit, nil, req.DenseWeight, req.SparseWeight)
	}

	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]any{
		"query":   req.Query,
		"mode":    req.Mode,
		"count":   len(results),
		"results": results,
	})
}

func (s *Server) handleSQL(w http.ResponseWriter, r *http.Request) {
	type sqlReq struct {
		SQL    string    `json:"sql"`
		Vector []float32 `json:"vector,omitempty"`
	}

	var req sqlReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid sql request body")
		return
	}

	result, err := s.db.ExecuteSQL(req.SQL, req.Vector)
	if err != nil {
		s.jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, result)
}

func (s *Server) handleGraphTraverse(w http.ResponseWriter, r *http.Request) {
	startNode := r.URL.Query().Get("start")
	if startNode == "" {
		s.jsonError(w, http.StatusBadRequest, "missing 'start' query parameter")
		return
	}

	hops := 2
	if hStr := r.URL.Query().Get("hops"); hStr != "" {
		if h, err := strconv.Atoi(hStr); err == nil && h > 0 {
			hops = h
		}
	}

	subgraph := s.db.KnowledgeGraph().TraverseBFS([]string{startNode}, hops, 0.1)
	s.jsonResponse(w, http.StatusOK, subgraph)
}

func (s *Server) handleVectorScatter(w http.ResponseWriter, r *http.Request) {
	// Project indexed documents to 3D coordinates for WebGL visualization
	type scatterPoint struct {
		ID   uint64  `json:"id"`
		Key  string  `json:"key"`
		Text string  `json:"text"`
		X    float32 `json:"x"`
		Y    float32 `json:"y"`
		Z    float32 `json:"z"`
	}

	var points []scatterPoint
	docRes, err := s.db.ExecuteSQL("SELECT * FROM documents LIMIT 500", nil)
	if err == nil {
		for _, row := range docRes.Rows {
			id, _ := row["id"].(uint64)
			key, _ := row["key"].(string)
			text, _ := row["text"].(string)

			// Simple 3-axis pseudo projection
			doc, _ := s.db.Get(id)
			var x, y, z float32
			if doc != nil && len(doc.Vector) >= 3 {
				x = doc.Vector[0] * 10.0
				y = doc.Vector[1] * 10.0
				z = doc.Vector[2] * 10.0
			}

			points = append(points, scatterPoint{
				ID:   id,
				Key:  key,
				Text: text,
				X:    x,
				Y:    y,
				Z:    z,
			})
		}
	}

	s.jsonResponse(w, http.StatusOK, points)
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan string, 10)
	s.sseMu.Lock()
	s.sseChans = append(s.sseChans, ch)
	s.sseMu.Unlock()

	defer func() {
		s.sseMu.Lock()
		for i, c := range s.sseChans {
			if c == ch {
				s.sseChans = append(s.sseChans[:i], s.sseChans[i+1:]...)
				break
			}
		}
		s.sseMu.Unlock()
	}()

	fmt.Fprintf(w, "data: {\"event\": \"connected\"}\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case msg := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		}
	}
}

func (s *Server) broadcastEvent(data string) {
	s.sseMu.Lock()
	defer s.sseMu.Unlock()
	for _, ch := range s.sseChans {
		select {
		case ch <- data:
		default:
		}
	}
}

func (s *Server) jsonResponse(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) jsonError(w http.ResponseWriter, code int, msg string) {
	s.jsonResponse(w, code, map[string]any{
		"error":   true,
		"message": msg,
	})
}
