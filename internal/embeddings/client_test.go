package embeddings

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFormatVector(t *testing.T) {
	got := FormatVector([]float32{0.1, -0.2, 3})
	want := "[0.1,-0.2,3]"
	if got != want {
		t.Errorf("FormatVector = %q, want %q", got, want)
	}
}

func TestFormatVector_Empty(t *testing.T) {
	if got := FormatVector(nil); got != "[]" {
		t.Errorf("FormatVector(nil) = %q, want \"[]\"", got)
	}
}

func TestEmbedDocument_SendsPrefixedInputAndParsesResponse(t *testing.T) {
	var capturedBody embedRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embed" {
			t.Errorf("expected POST /api/embed, got %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		json.NewEncoder(w).Encode(embedResponse{Embeddings: [][]float32{{0.1, 0.2, 0.3}}})
	}))
	defer srv.Close()

	c := New(srv.URL, "nomic-embed-text")
	vec, err := c.EmbedDocument(context.Background(), "a great resume")
	if err != nil {
		t.Fatalf("EmbedDocument: %v", err)
	}
	if !strings.HasPrefix(capturedBody.Input, prefixDocument) {
		t.Errorf("expected input to be prefixed with %q, got %q", prefixDocument, capturedBody.Input)
	}
	if len(vec) != 3 || vec[0] != 0.1 {
		t.Errorf("unexpected embedding: %v", vec)
	}
}

func TestEmbedQuery_UsesQueryPrefix(t *testing.T) {
	var capturedBody embedRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedBody)
		json.NewEncoder(w).Encode(embedResponse{Embeddings: [][]float32{{0.5}}})
	}))
	defer srv.Close()

	c := New(srv.URL, "nomic-embed-text")
	if _, err := c.EmbedQuery(context.Background(), "a job description"); err != nil {
		t.Fatalf("EmbedQuery: %v", err)
	}
	if !strings.HasPrefix(capturedBody.Input, prefixQuery) {
		t.Errorf("expected input to be prefixed with %q, got %q", prefixQuery, capturedBody.Input)
	}
}

func TestEmbed_ErrorsOnNonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("model not found"))
	}))
	defer srv.Close()

	c := New(srv.URL, "nomic-embed-text")
	if _, err := c.EmbedDocument(context.Background(), "text"); err == nil {
		t.Error("expected an error for a non-200 response")
	}
}

func TestEmbed_ErrorsOnUnreachableServer(t *testing.T) {
	c := New("http://127.0.0.1:1", "nomic-embed-text") // port 1 is reserved and should refuse connections
	if _, err := c.EmbedDocument(context.Background(), "text"); err == nil {
		t.Error("expected an error for an unreachable server")
	}
}
