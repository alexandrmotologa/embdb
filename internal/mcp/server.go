package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/alexandrmotologa/embdb/internal/embeddings"
	"github.com/alexandrmotologa/embdb/internal/engine"
	"github.com/alexandrmotologa/embdb/internal/metadata"
)

type Server struct {
	mu       sync.RWMutex
	db       *engine.DB
	embedder embeddings.Adapter
	tools    []Tool
}

func NewServer(db *engine.DB, embedder embeddings.Adapter) *Server {
	if embedder == nil {
		embedder = embeddings.NewDeterministicAdapter(db.Stats().VectorDim)
	}

	s := &Server{
		db:       db,
		embedder: embedder,
	}
	s.registerTools()
	return s
}

func (s *Server) registerTools() {
	s.tools = []Tool{
		{
			Name:        "embdb_search",
			Description: "Perform hybrid, vector, or keyword search across the local embdb vector database.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"query": {
						Type:        "string",
						Description: "Text query for lexical and semantic matching.",
					},
					"mode": {
						Type:        "string",
						Description: "Search mode: 'hybrid' (default), 'vector', or 'bm25'.",
					},
					"limit": {
						Type:        "integer",
						Description: "Maximum number of search results to return (default 5).",
					},
					"category": {
						Type:        "string",
						Description: "Optional metadata category filter.",
					},
				},
				Required: []string{"query"},
			},
		},
		{
			Name:        "embdb_index",
			Description: "Index or upsert a new text document into the local embdb database.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"key": {
						Type:        "string",
						Description: "Unique string key for the document.",
					},
					"text": {
						Type:        "string",
						Description: "Full text content of the document.",
					},
					"category": {
						Type:        "string",
						Description: "Optional category tag for metadata filtering.",
					},
				},
				Required: []string{"key", "text"},
			},
		},
		{
			Name:        "embdb_graph_traverse",
			Description: "Traverse connected entities in the knowledge graph for multi-hop RAG context expansion.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"start_node": {
						Type:        "string",
						Description: "Seed node or document key to start traversal from.",
					},
					"max_hops": {
						Type:        "integer",
						Description: "Maximum traversal depth (default 2).",
					},
				},
				Required: []string{"start_node"},
			},
		},
		{
			Name:        "embdb_stats",
			Description: "Retrieve database health metrics, total document count, vector dimensions, and graph stats.",
			InputSchema: InputSchema{
				Type:       "object",
				Properties: map[string]Property{},
			},
		},
	}
}

// HandleRequest processes an incoming JSON-RPC request and returns a response.
func (s *Server) HandleRequest(req *JSONRPCRequest) *JSONRPCResponse {
	switch req.Method {
	case "initialize":
		res := map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    "embdb",
				"version": "1.0.0",
			},
		}
		return NewSuccessResponse(req.ID, res)

	case "notifications/initialized":
		return nil

	case "ping":
		return NewSuccessResponse(req.ID, map[string]any{})

	case "tools/list":
		return NewSuccessResponse(req.ID, map[string]any{
			"tools": s.tools,
		})

	case "tools/call":
		var callParams struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &callParams); err != nil {
			return NewErrorResponse(req.ID, CodeInvalidParams, "failed to parse tool arguments")
		}

		res, err := s.callTool(callParams.Name, callParams.Arguments)
		if err != nil {
			return NewErrorResponse(req.ID, CodeInternalError, err.Error())
		}
		return NewSuccessResponse(req.ID, res)

	default:
		return NewErrorResponse(req.ID, CodeMethodNotFound, fmt.Sprintf("method %s not found", req.Method))
	}
}

func (s *Server) callTool(name string, args map[string]any) (CallToolResult, error) {
	ctx := context.Background()

	switch name {
	case "embdb_search":
		query, _ := args["query"].(string)
		mode, _ := args["mode"].(string)
		if mode == "" {
			mode = "hybrid"
		}
		limit := 5
		if l, ok := args["limit"].(float64); ok && l > 0 {
			limit = int(l)
		}

		var filter *metadata.Predicate
		if cat, ok := args["category"].(string); ok && cat != "" {
			p := metadata.Eq("category", cat)
			filter = &p
		}

		var results []engine.SearchResult
		var err error

		switch mode {
		case "vector":
			vec, errEmbed := s.embedder.Embed(ctx, query)
			if errEmbed != nil {
				return TextResult(""), fmt.Errorf("embedding failed: %w", errEmbed)
			}
			results, err = s.db.SearchVector(vec, limit, filter)
		case "bm25":
			results, err = s.db.SearchBM25(query, limit, filter)
		default: // hybrid
			vec, _ := s.embedder.Embed(ctx, query)
			results, err = s.db.SearchHybrid(query, vec, limit, filter, 0.5, 0.5)
		}

		if err != nil {
			return TextResult(""), err
		}

		out, _ := json.MarshalIndent(results, "", "  ")
		return TextResult(string(out)), nil

	case "embdb_index":
		key, _ := args["key"].(string)
		text, _ := args["text"].(string)
		category, _ := args["category"].(string)

		if key == "" || text == "" {
			return TextResult(""), errors.New("key and text parameters are required")
		}

		meta := make(map[string]any)
		if category != "" {
			meta["category"] = category
		}

		vec, _ := s.embedder.Embed(ctx, text)
		doc := &engine.Document{
			Key:      key,
			Text:     text,
			Vector:   vec,
			Metadata: meta,
		}

		if err := s.db.Upsert(doc); err != nil {
			return TextResult(""), err
		}

		return TextResult(fmt.Sprintf("Successfully indexed document '%s' (ID %d)", key, doc.ID)), nil

	case "embdb_graph_traverse":
		startNode, _ := args["start_node"].(string)
		maxHops := 2
		if h, ok := args["max_hops"].(float64); ok && h > 0 {
			maxHops = int(h)
		}

		kg := s.db.KnowledgeGraph()
		subgraph := kg.TraverseBFS([]string{startNode}, maxHops, 0.1)

		out, _ := json.MarshalIndent(subgraph, "", "  ")
		return TextResult(string(out)), nil

	case "embdb_stats":
		stats := s.db.Stats()
		out, _ := json.MarshalIndent(stats, "", "  ")
		return TextResult(string(out)), nil

	default:
		return TextResult(""), ErrToolNotFound
	}
}

// ServeStdio runs the standard I/O JSON-RPC loop for Claude Code / Antigravity / Cursor.
func (s *Server) ServeStdio(r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			errResp := NewErrorResponse(nil, CodeParseError, "parse error")
			b, _ := json.Marshal(errResp)
			fmt.Fprintf(w, "%s\n", string(b))
			continue
		}

		resp := s.HandleRequest(&req)
		if resp != nil {
			b, err := json.Marshal(resp)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "%s\n", string(b))
		}
	}
	return scanner.Err()
}

// HandleHTTP handles HTTP POST JSON-RPC requests.
func (s *Server) HandleHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var req JSONRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		resp := NewErrorResponse(nil, CodeParseError, "parse error")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp := s.HandleRequest(&req)
	w.Header().Set("Content-Type", "application/json")
	if resp != nil {
		json.NewEncoder(w).Encode(resp)
	}
}
