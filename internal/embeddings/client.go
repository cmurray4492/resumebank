// Package embeddings talks to a local Ollama server to generate text
// embeddings for the candidate/job matching (RAG) features.
package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// nomic-embed-text is an asymmetric embedding model: text being indexed
// must be prefixed "search_document: " and text being searched for must be
// prefixed "search_query: ", or retrieval quality suffers. See
// https://ollama.com/library/nomic-embed-text.
const (
	prefixDocument = "search_document: "
	prefixQuery    = "search_query: "
)

type Client struct {
	baseURL string
	model   string
	http    *http.Client
}

func New(baseURL, model string) *Client {
	return &Client{
		baseURL: baseURL,
		model:   model,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// EmbedDocument embeds text that will be stored and searched against later
// (a candidate's resume, a job's description).
func (c *Client) EmbedDocument(ctx context.Context, text string) ([]float32, error) {
	return c.embed(ctx, prefixDocument+text)
}

// EmbedQuery embeds text pasted in at search time (a job description an
// employer is matching against candidates, or a resume a candidate is
// matching against jobs).
func (c *Client) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	return c.embed(ctx, prefixQuery+text)
}

type embedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type embedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

func (c *Client) embed(ctx context.Context, input string) ([]float32, error) {
	body, err := json.Marshal(embedRequest{Model: c.model, Input: input})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/embed", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling Ollama at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("Ollama returned %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decoding Ollama response: %w", err)
	}
	if len(parsed.Embeddings) == 0 {
		return nil, fmt.Errorf("Ollama returned no embeddings")
	}
	return parsed.Embeddings[0], nil
}

// FormatVector renders v as a pgvector literal (e.g. "[0.1,0.2,0.3]") for
// use with a "$1::vector" query parameter.
func FormatVector(v []float32) string {
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = strconv.FormatFloat(float64(f), 'f', -1, 32)
	}
	return "[" + strings.Join(parts, ",") + "]"
}
