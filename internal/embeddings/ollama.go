// Package embeddings provides a thin HTTP client over Ollama's embedding,
// health, and model-pull APIs.
package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	embedTimeout  = 120 * time.Second
	healthTimeout = 10 * time.Second
)

// OllamaEmbedder generates embeddings via a local Ollama instance.
type OllamaEmbedder struct {
	baseURL string
	model   string
	dims    int
	client  *http.Client
}

// New returns an OllamaEmbedder targeting baseURL using model, which is
// expected to produce vectors of dims dimensions.
func New(baseURL, model string, dims int) *OllamaEmbedder {
	return &OllamaEmbedder{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		model:   model,
		dims:    dims,
		client:  &http.Client{},
	}
}

// Model returns the embedding model name.
func (e *OllamaEmbedder) Model() string {
	return e.model
}

// Dimensions returns the expected embedding dimensionality.
func (e *OllamaEmbedder) Dimensions() int {
	return e.dims
}

type embedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

// Embed sends inputs to Ollama's /api/embed endpoint and returns one vector
// per input, in the same order.
func (e *OllamaEmbedder) Embed(ctx context.Context, inputs []string) ([][]float32, error) {
	ctx, cancel := context.WithTimeout(ctx, embedTimeout)
	defer cancel()

	body, err := json.Marshal(embedRequest{Model: e.model, Input: inputs})
	if err != nil {
		return nil, fmt.Errorf("embeddings: embed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/api/embed", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("embeddings: embed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embeddings: embed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embeddings: embed: ollama returned %s: %s", resp.Status, strings.TrimSpace(string(msg)))
	}

	var out embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("embeddings: embed: %w", err)
	}

	if len(out.Embeddings) != len(inputs) {
		return nil, fmt.Errorf("embeddings: embed: expected %d embeddings, got %d", len(inputs), len(out.Embeddings))
	}

	return out.Embeddings, nil
}

// Health checks whether the Ollama server is reachable.
func (e *OllamaEmbedder) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, healthTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, e.baseURL+"/api/version", nil)
	if err != nil {
		return fmt.Errorf("embeddings: health: %w", err)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("embeddings: health: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("embeddings: health: ollama returned %s", resp.Status)
	}

	return nil
}

type pullRequest struct {
	Model  string `json:"model"`
	Stream bool   `json:"stream"`
}

type pullProgress struct {
	Status    string `json:"status"`
	Digest    string `json:"digest,omitempty"`
	Total     int64  `json:"total,omitempty"`
	Completed int64  `json:"completed,omitempty"`
}

// Pull downloads model, invoking onProgress for each status update Ollama
// streams back. It blocks until the pull completes or fails.
func (e *OllamaEmbedder) Pull(ctx context.Context, model string, onProgress func(status string, completed, total int64)) error {
	body, err := json.Marshal(pullRequest{Model: model, Stream: true})
	if err != nil {
		return fmt.Errorf("embeddings: pull: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/api/pull", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("embeddings: pull: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("embeddings: pull: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("embeddings: pull: ollama returned %s: %s", resp.Status, strings.TrimSpace(string(msg)))
	}

	dec := json.NewDecoder(resp.Body)
	for {
		var p pullProgress
		if err := dec.Decode(&p); err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("embeddings: pull: %w", err)
		}

		if p.Status == "error" {
			return fmt.Errorf("embeddings: pull: ollama reported an error pulling %q", model)
		}

		if onProgress != nil {
			onProgress(p.Status, p.Completed, p.Total)
		}

		if p.Status == "success" {
			return nil
		}
	}
}
