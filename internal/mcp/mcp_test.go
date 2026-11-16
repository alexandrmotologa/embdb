package mcp

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexandrmotologa/embdb/internal/embeddings"
	"github.com/alexandrmotologa/embdb/internal/engine"
)

func TestMCPServer(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "embdb-mcp-test-*")
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
	srv := NewServer(db, embedder)

	// 1. Initialize
	initReq := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
	}
	resp := srv.HandleRequest(initReq)
	if resp == nil || resp.Error != nil {
		t.Fatalf("initialize failed: %v", resp)
	}

	// 2. Tools List
	listReq := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
	}
	listResp := srv.HandleRequest(listReq)
	if listResp == nil || listResp.Error != nil {
		t.Fatalf("tools/list failed: %v", listResp)
	}

	// 3. Tools Call: Index document
	idxArgs, _ := json.Marshal(map[string]any{
		"name": "embdb_index",
		"arguments": map[string]any{
			"key":      "faq_billing",
			"text":     "Enterprise invoicing and billing questions answered",
			"category": "finance",
		},
	})
	callReq := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "tools/call",
		Params:  idxArgs,
	}
	idxResp := srv.HandleRequest(callReq)
	if idxResp == nil || idxResp.Error != nil {
		t.Fatalf("embdb_index failed: %v", idxResp)
	}

	// 4. Tools Call: Search
	searchArgs, _ := json.Marshal(map[string]any{
		"name": "embdb_search",
		"arguments": map[string]any{
			"query": "billing questions",
			"mode":  "hybrid",
			"limit": 5,
		},
	})
	searchReq := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      4,
		Method:  "tools/call",
		Params:  searchArgs,
	}
	searchResp := srv.HandleRequest(searchReq)
	if searchResp == nil || searchResp.Error != nil {
		t.Fatalf("embdb_search failed: %v", searchResp)
	}
	toolRes := searchResp.Result.(CallToolResult)
	if len(toolRes.Content) == 0 || !strings.Contains(toolRes.Content[0].Text, "faq_billing") {
		t.Fatalf("expected search result to contain faq_billing, got: %s", toolRes.Content[0].Text)
	}

	// 5. Test Stdio loop
	var inBuf bytes.Buffer
	var outBuf bytes.Buffer
	pingReq, _ := json.Marshal(JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      5,
		Method:  "ping",
	})
	inBuf.WriteString(string(pingReq) + "\n")

	if err := srv.ServeStdio(&inBuf, &outBuf); err != nil {
		t.Fatalf("ServeStdio error: %v", err)
	}

	var pingResp JSONRPCResponse
	if err := json.Unmarshal(outBuf.Bytes(), &pingResp); err != nil {
		t.Fatalf("failed to unmarshal ping response: %v", err)
	}
	if pingResp.Error != nil {
		t.Fatalf("ping returned error: %v", pingResp.Error)
	}
}
