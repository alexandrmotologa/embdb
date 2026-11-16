package graph

import (
	"bytes"
	"testing"
)

func TestKnowledgeGraphAndTraversal(t *testing.T) {
	kg := NewKnowledgeGraph()

	kg.AddNode(Node{ID: "doc:1", Type: "Document", Properties: map[string]any{"title": "Master Service Agreement"}})
	kg.AddNode(Node{ID: "entity:AcmeCorp", Type: "Company"})
	kg.AddNode(Node{ID: "entity:Alice", Type: "Person"})
	kg.AddNode(Node{ID: "doc:2", Type: "Document", Properties: map[string]any{"title": "Invoice #402"}})

	kg.AddEdge(Edge{From: "doc:1", To: "entity:AcmeCorp", Relation: "MENTIONS", Weight: 1.0})
	kg.AddEdge(Edge{From: "entity:Alice", To: "entity:AcmeCorp", Relation: "WORKS_AT", Weight: 0.9})
	kg.AddEdge(Edge{From: "doc:2", To: "entity:AcmeCorp", Relation: "ISSUED_TO", Weight: 1.0})

	if kg.NodeCount() != 4 {
		t.Fatalf("expected 4 nodes, got %d", kg.NodeCount())
	}
	if kg.EdgeCount() != 3 {
		t.Fatalf("expected 3 edges, got %d", kg.EdgeCount())
	}

	// 2-hop traversal from doc:1
	subgraph := kg.TraverseBFS([]string{"doc:1"}, 2, 0.5)
	if len(subgraph.Nodes) != 4 {
		t.Fatalf("expected all 4 connected nodes in 2 hops, got %d", len(subgraph.Nodes))
	}
	if len(subgraph.Edges) != 3 {
		t.Fatalf("expected 3 edges traversed, got %d", len(subgraph.Edges))
	}

	// Serialization and deserialization test
	var buf bytes.Buffer
	if err := kg.Serialize(&buf); err != nil {
		t.Fatalf("serialize failed: %v", err)
	}

	kg2 := NewKnowledgeGraph()
	if err := kg2.Deserialize(&buf); err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}

	if kg2.NodeCount() != 4 || kg2.EdgeCount() != 3 {
		t.Fatalf("deserialized graph count mismatch: %d nodes, %d edges", kg2.NodeCount(), kg2.EdgeCount())
	}
}
