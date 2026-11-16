package embeddings

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/alexandrmotologa/embdb/internal/hnsw"
)

type Adapter interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
	Dimension() int
}

// DeterministicAdapter produces repeatable deterministic embeddings without external network access.
type DeterministicAdapter struct {
	dim int
}

func NewDeterministicAdapter(dim int) *DeterministicAdapter {
	if dim <= 0 {
		dim = 384
	}
	return &DeterministicAdapter{dim: dim}
}

func (d *DeterministicAdapter) Dimension() int {
	return d.dim
}

func (d *DeterministicAdapter) Embed(ctx context.Context, text string) ([]float32, error) {
	vec := make([]float32, d.dim)
	words := bytes.Fields([]byte(text))

	for _, w := range words {
		hash := sha256.Sum256(w)
		for i := 0; i < d.dim; i++ {
			chunk := hash[(i*4)%len(hash) : (i*4)%len(hash)+4]
			val := int32(binary.LittleEndian.Uint32(chunk))
			vec[i] += float32(val) / float32(math.MaxInt32)
		}
	}

	return hnsw.Normalize(vec), nil
}

func (d *DeterministicAdapter) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	res := make([][]float32, len(texts))
	for i, t := range texts {
		vec, err := d.Embed(ctx, t)
		if err != nil {
			return nil, err
		}
		res[i] = vec
	}
	return res, nil
}

// OllamaAdapter connects to a local Ollama instance.
type OllamaAdapter struct {
	baseURL    string
	model      string
	dim        int
	httpClient *http.Client
}

func NewOllamaAdapter(baseURL, model string, dim int) *OllamaAdapter {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "nomic-embed-text"
	}
	if dim <= 0 {
		dim = 768
	}
	return &OllamaAdapter{
		baseURL: baseURL,
		model:   model,
		dim:     dim,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (o *OllamaAdapter) Dimension() int {
	return o.dim
}

func (o *OllamaAdapter) Embed(ctx context.Context, text string) ([]float32, error) {
	type reqBody struct {
		Model  string `json:"model"`
		Prompt string `json:"prompt"`
	}
	type respBody struct {
		Embedding []float32 `json:"embedding"`
	}

	b, _ := json.Marshal(reqBody{Model: o.model, Prompt: text})
	req, err := http.NewRequestWithContext(ctx, "POST", o.baseURL+"/api/embeddings", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama connection error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	var parsed respBody
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	if len(parsed.Embedding) == 0 {
		return nil, errors.New("ollama returned empty embedding")
	}

	return parsed.Embedding, nil
}

func (o *OllamaAdapter) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	res := make([][]float32, len(texts))
	for i, t := range texts {
		vec, err := o.Embed(ctx, t)
		if err != nil {
			return nil, err
		}
		res[i] = vec
	}
	return res, nil
}
