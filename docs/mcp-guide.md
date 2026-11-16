# Model Context Protocol (MCP) Guide

embdb implements a Model Context Protocol server compliant with the JSON-RPC 2.0 specification. This enables autonomous coding agents such as Claude Code, Antigravity, and Cursor to attach directly to local semantic memory.

## Transports

1. **Standard I/O (stdio)**: Default mode when invoking `embdb mcp`. Messages are exchanged as newline-delimited JSON over standard input and output streams.
2. **HTTP / Server-Sent Events (SSE)**: Available when running `embdb serve` on port 8383 at `http://localhost:8383/mcp`.

## Available Tools

### 1. `embdb_search`

Executes hybrid, dense vector, or BM25 lexical searches against indexed documents.

* Arguments:
  * `query` (string, required): Search query text.
  * `mode` (string, optional): One of `"hybrid"` (default), `"vector"`, or `"bm25"`.
  * `limit` (integer, optional): Maximum results to return (default 5).
  * `category` (string, optional): Metadata category filter.

### 2. `embdb_index`

Indexes a new document or updates an existing document in the database.

* Arguments:
  * `key` (string, required): Unique document identifier.
  * `text` (string, required): Full text content.
  * `category` (string, optional): Category tag for filtering.

### 3. `embdb_graph_traverse`

Expands connected knowledge graph nodes from a seed document or entity identifier.

* Arguments:
  * `start_node` (string, required): Node identifier to start from.
  * `max_hops` (integer, optional): Maximum exploration depth (default 2).

### 4. `embdb_stats`

Inspects database metrics, allocated pages, total document count, and vector dimensions.

* Arguments: None.

## Agent Configuration

### Claude Desktop / Antigravity / Cursor

Add the following block to your agent configuration file:

```json
{
  "mcpServers": {
    "embdb": {
      "command": "embdb",
      "args": ["mcp", "--db", "agent_memory.embdb"]
    }
  }
}
```
