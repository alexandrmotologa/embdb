package server

import (
	"fmt"
	"net/http"
	"runtime"
	"sync/atomic"
)

type MetricsTracker struct {
	hybridQueries atomic.Uint64
	vectorQueries atomic.Uint64
	bm25Queries   atomic.Uint64
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	stats := s.db.Stats()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	fmt.Fprintf(w, "# HELP embdb_documents_total Total number of indexed documents in the database.\n")
	fmt.Fprintf(w, "# TYPE embdb_documents_total gauge\n")
	fmt.Fprintf(w, "embdb_documents_total %d\n\n", stats.TotalDocuments)

	fmt.Fprintf(w, "# HELP embdb_vector_dimensions Dimensionality of the dense vectors.\n")
	fmt.Fprintf(w, "# TYPE embdb_vector_dimensions gauge\n")
	fmt.Fprintf(w, "embdb_vector_dimensions %d\n\n", stats.VectorDim)

	fmt.Fprintf(w, "# HELP embdb_pages_total Total number of 4KB slotted pages allocated on disk.\n")
	fmt.Fprintf(w, "# TYPE embdb_pages_total gauge\n")
	fmt.Fprintf(w, "embdb_pages_total %d\n\n", stats.TotalPages)

	fmt.Fprintf(w, "# HELP embdb_graph_nodes_total Total number of entities in the knowledge graph.\n")
	fmt.Fprintf(w, "# TYPE embdb_graph_nodes_total gauge\n")
	fmt.Fprintf(w, "embdb_graph_nodes_total %d\n\n", stats.GraphNodes)

	fmt.Fprintf(w, "# HELP embdb_graph_edges_total Total number of relationships in the knowledge graph.\n")
	fmt.Fprintf(w, "# TYPE embdb_graph_edges_total gauge\n")
	fmt.Fprintf(w, "embdb_graph_edges_total %d\n\n", stats.GraphEdges)

	fmt.Fprintf(w, "# HELP embdb_memory_allocated_bytes Go runtime heap allocated memory in bytes.\n")
	fmt.Fprintf(w, "# TYPE embdb_memory_allocated_bytes gauge\n")
	fmt.Fprintf(w, "embdb_memory_allocated_bytes %d\n\n", m.Alloc)

	fmt.Fprintf(w, "# HELP embdb_memory_sys_bytes Total system memory obtained from OS in bytes.\n")
	fmt.Fprintf(w, "# TYPE embdb_memory_sys_bytes gauge\n")
	fmt.Fprintf(w, "embdb_memory_sys_bytes %d\n\n", m.Sys)
}
