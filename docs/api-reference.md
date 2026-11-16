# embdb REST API Reference

The embedded HTTP server starts by running:

```bash
embdb serve --db data.embdb --port 8383
```

All responses return standard JSON payloads.

---

### `GET /api/health`

Returns engine status and version.

**Response (200 OK):**
```json
{
  "engine": "embdb",
  "status": "healthy",
  "time": 1731796500,
  "version": "1.0.0"
}
```

---

### `GET /api/stats`

Returns storage allocation and index statistics.

**Response (200 OK):**
```json
{
  "total_documents": 20,
  "vector_dimension": 384,
  "graph_nodes": 20,
  "graph_edges": 0,
  "total_pages": 3,
  "metric": 0,
  "quantization": 0
}
```

---

### `POST /api/documents`

Inserts or updates a document.

**Request:**
```json
{
  "key": "faq_auth_01",
  "text": "Multi-factor authentication requires hardware token validation.",
  "metadata": {
    "category": "security"
  }
}
```

**Response (201 Created):**
```json
{
  "id": 1,
  "key": "faq_auth_01",
  "text": "Multi-factor authentication requires hardware token validation.",
  "created_at": 1731796510
}
```

---

### `POST /api/search`

Executes hybrid, dense vector, or BM25 search.

**Request:**
```json
{
  "query": "authentication hardware tokens",
  "mode": "hybrid",
  "limit": 5,
  "dense_weight": 0.5,
  "sparse_weight": 0.5
}
```

**Response (200 OK):**
```json
{
  "query": "authentication hardware tokens",
  "mode": "hybrid",
  "count": 1,
  "results": [
    {
      "id": 1,
      "key": "faq_auth_01",
      "text": "Multi-factor authentication requires hardware token validation.",
      "score": 0.0163,
      "dense_rank": 1,
      "sparse_rank": 1,
      "metadata": {
        "category": "security"
      }
    }
  ]
}
```

---

### `POST /api/query/sql`

Executes a semantic SQL query.

**Request:**
```json
{
  "sql": "SELECT id, key, text FROM documents WHERE category = 'security' ORDER BY hybrid('authentication') LIMIT 5"
}
```

---

### `GET /events`

Server-Sent Events stream delivering live database update notifications.
