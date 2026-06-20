package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	pgvector "github.com/pgvector/pgvector-go"

	"github.com/Datto27/vecsim/internal/adapters"
	"github.com/Datto27/vecsim/internal/config"
	"github.com/Datto27/vecsim/internal/embeddings"
	"github.com/Datto27/vecsim/internal/indexer"
	"github.com/Datto27/vecsim/internal/store"
)

// Handlers holds all HTTP handler methods for vecsim's API.
type Handlers struct {
	store    *store.Store
	embedder *embeddings.OllamaEmbedder
	cfg      *config.Config
}

// NewHandlers returns a Handlers struct wired with the given dependencies.
func NewHandlers(s *store.Store, e *embeddings.OllamaEmbedder, cfg *config.Config) *Handlers {
	return &Handlers{store: s, embedder: e, cfg: cfg}
}

// ─── /health ─────────────────────────────────────────────────────────────────

func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	ollamaOK := h.embedder.Health(r.Context()) == nil

	dbOK := false
	if _, err := h.store.GetMeta(r.Context(), "profile"); err == nil || errors.Is(err, store.ErrNotFound) {
		dbOK = true
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"profile":    string(h.cfg.Profile),
		"model":      h.cfg.Model,
		"dimensions": h.cfg.Dimensions,
		"ollama_ok":  ollamaOK,
		"db_ok":      dbOK,
	})
}

// ─── /items ──────────────────────────────────────────────────────────────────

func (h *Handlers) ListItems(w http.ResponseWriter, r *http.Request) {
	typ := r.URL.Query().Get("type")
	limit := queryInt(r, "limit", 20)
	offset := queryInt(r, "offset", 0)

	items, total, err := h.store.ListItems(r.Context(), typ, limit, offset)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"items":  items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *Handlers) GetItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	item, err := h.store.GetItem(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			WriteError(w, http.StatusNotFound, fmt.Sprintf("item %q not found", id))
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, item)
}

type createItemRequest struct {
	ID     string         `json:"id"`
	Label  string         `json:"label"`
	Type   string         `json:"type"`
	Fields map[string]any `json:"fields"`
	Tags   []string       `json:"tags"`
}

func (h *Handlers) CreateItem(w http.ResponseWriter, r *http.Request) {
	var req createItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.Label == "" {
		WriteError(w, http.StatusBadRequest, "label is required")
		return
	}
	if req.Type == "" {
		WriteError(w, http.StatusBadRequest, "type is required")
		return
	}
	if _, ok := adapters.Get(req.Type); !ok {
		WriteError(w, http.StatusBadRequest, fmt.Sprintf("unknown type %q (valid: %s)", req.Type, strings.Join(adapters.Types(), ", ")))
		return
	}

	// Generate ID if not provided.
	id := req.ID
	if id == "" {
		id = h.generateID(r.Context(), req.Type, req.Label)
	}

	item := adapters.Item{
		ID:     id,
		Label:  req.Label,
		Type:   req.Type,
		Fields: req.Fields,
		Tags:   req.Tags,
	}
	if item.Fields == nil {
		item.Fields = map[string]any{}
	}
	if item.Tags == nil {
		item.Tags = []string{}
	}

	if _, err := h.store.InsertItem(r.Context(), item); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Attempt to embed immediately; failure is non-fatal.
	embedded := false
	ad, _ := adapters.Get(req.Type)
	text := ad.BuildText(item.Fields)
	if vecs, err := h.embedder.Embed(r.Context(), []string{text}); err == nil {
		if setErr := h.store.SetEmbedding(r.Context(), id, pgvector.NewVector(vecs[0])); setErr == nil {
			embedded = true
		}
	}

	item.Embedded = embedded
	WriteJSON(w, http.StatusCreated, item)
}

func (h *Handlers) DeleteItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	deleted, err := h.store.DeleteItem(r.Context(), id)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !deleted {
		WriteError(w, http.StatusNotFound, fmt.Sprintf("item %q not found", id))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── /search ─────────────────────────────────────────────────────────────────

type searchRequest struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Type      string `json:"type"`
	CrossType bool   `json:"cross_type"`
	Limit     int    `json:"limit"`
}

type searchResponse struct {
	Query   *adapters.Item       `json:"query"`
	Results []store.SearchResult `json:"results"`
}

func (h *Handlers) Search(w http.ResponseWriter, r *http.Request) {
	var req searchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}

	// Resolve query item by ID (preferred) or label.
	var (
		item *adapters.Item
		err  error
	)
	if req.ID != "" {
		item, err = h.store.GetItem(r.Context(), req.ID)
	} else if req.Label != "" {
		item, err = h.store.GetItemByLabel(r.Context(), req.Label, req.Type)
	} else {
		WriteError(w, http.StatusBadRequest, "provide 'id' or 'label'")
		return
	}
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "query item not found")
			return
		}
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	vec, err := h.store.GetEmbedding(r.Context(), item.ID)
	if err != nil {
		if errors.Is(err, store.ErrNotEmbedded) {
			WriteError(w, http.StatusUnprocessableEntity, fmt.Sprintf("item %q has not been indexed yet", item.ID))
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	typeFilter := item.Type
	if req.CrossType {
		typeFilter = ""
	}

	results, err := h.store.SearchByVector(r.Context(), vec, typeFilter, item.ID, req.Limit)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, searchResponse{Query: item, Results: results})
}

type searchEmbedRequest struct {
	Text      string `json:"text"`
	Type      string `json:"type"`
	CrossType bool   `json:"cross_type"`
	Limit     int    `json:"limit"`
}

func (h *Handlers) SearchByText(w http.ResponseWriter, r *http.Request) {
	var req searchEmbedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.Text == "" {
		WriteError(w, http.StatusBadRequest, "'text' is required")
		return
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}

	vecs, err := h.embedder.Embed(r.Context(), []string{req.Text})
	if err != nil {
		WriteError(w, http.StatusBadGateway, "embed failed: "+err.Error())
		return
	}

	typeFilter := req.Type
	if req.CrossType {
		typeFilter = ""
	}

	results, err := h.store.SearchByVector(r.Context(), pgvector.NewVector(vecs[0]), typeFilter, "", req.Limit)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"query":   req.Text,
		"results": results,
	})
}

// ─── /stats ───────────────────────────────────────────────────────────────────

func (h *Handlers) Stats(w http.ResponseWriter, r *http.Request) {
	result, err := h.store.Stats(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, result)
}

// ─── /index ───────────────────────────────────────────────────────────────────

func (h *Handlers) Index(w http.ResponseWriter, r *http.Request) {
	go func() {
		if err := indexer.Run(context.Background(), h.store, h.embedder, "all", 20, nil); err != nil {
			_ = err // logged nowhere; consider adding structured logging here
		}
	}()
	WriteJSON(w, http.StatusAccepted, map[string]string{"status": "indexing started"})
}

// ─── helpers ──────────────────────────────────────────────────────────────────

var nonAlphanumRE = regexp.MustCompile(`[^a-z0-9]+`)

// slug converts a label into a URL-friendly lowercase slug.
func slug(label string) string {
	s := strings.ToLower(label)
	s = nonAlphanumRE.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

// generateID derives a type-prefixed ID from label, appending a numeric
// suffix to avoid collisions.
func (h *Handlers) generateID(ctx context.Context, typ, label string) string {
	base := typ + ":" + slug(label)
	id := base
	for i := 2; i < 100; i++ {
		if _, err := h.store.GetItem(ctx, id); err != nil {
			return id
		}
		id = fmt.Sprintf("%s-%d", base, i)
	}
	return id
}

func queryInt(r *http.Request, key string, defaultVal int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 0 {
		return defaultVal
	}
	return v
}
