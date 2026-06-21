package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	pgvector "github.com/pgvector/pgvector-go"

	"github.com/Datto27/GOSim/internal/adapters"
	"github.com/Datto27/GOSim/internal/config"
	"github.com/Datto27/GOSim/internal/embeddings"
	"github.com/Datto27/GOSim/internal/indexer"
	"github.com/Datto27/GOSim/internal/ingest"
	"github.com/Datto27/GOSim/internal/rerank"
	"github.com/Datto27/GOSim/internal/schema"
	"github.com/Datto27/GOSim/internal/store"
)

// candidatePool bounds how many items are pulled from pgvector for hybrid
// re-ranking. When structured weights are active, ranking can be driven by
// fields that are semantically distant, so the whole collection must be
// considered rather than only the semantic top-N.
const candidatePool = 10000

// validateWeights rejects any negative weight value.
func validateWeights(weights map[string]float64) error {
	for k, v := range weights {
		if v < 0 {
			return fmt.Errorf("weight for %q must not be negative", k)
		}
	}
	return nil
}

// fetchLimitFor returns how many candidates to pull before re-ranking: the
// whole collection (capped) when structured weights are active, otherwise just
// the requested limit.
func fetchLimitFor(limit int, weights map[string]float64) int {
	if rerank.HasStructuralWeight(weights) {
		return candidatePool
	}
	return limit
}

// Handlers holds all HTTP handler methods for gosim's API.
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

	// Update the collection's schema with this item, preserving existing weights.
	sch, err := h.recordSchema(r.Context(), req.Type, []map[string]any{item.Fields})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Attempt to embed immediately; failure is non-fatal.
	embedded := false
	text := schema.EmbeddingText(item.Fields, sch)
	if vecs, err := h.embedder.Embed(r.Context(), []string{text}); err == nil {
		if setErr := h.store.SetEmbedding(r.Context(), id, pgvector.NewVector(vecs[0])); setErr == nil {
			embedded = true
		}
	}

	item.Embedded = embedded
	WriteJSON(w, http.StatusCreated, item)
}

// recordSchema detects the structure of the given items, merges it into the
// collection's stored schema (initializing default weights for a new
// collection, preserving user weights otherwise), persists it, and returns the
// merged schema for immediate use.
func (h *Handlers) recordSchema(ctx context.Context, typ string, fieldMaps []map[string]any) (schema.Schema, error) {
	detected := schema.Detect(fieldMaps)
	existing, weights, err := h.store.GetCollection(ctx, typ)
	if err != nil {
		return nil, err
	}
	merged := schema.Merge(existing, detected)
	weights = schema.EnsureWeights(merged, weights) // fill defaults for any unset parameter
	if err := h.store.UpsertCollection(ctx, typ, merged, weights); err != nil {
		return nil, err
	}
	return merged, nil
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
	ID        string             `json:"id"`
	Label     string             `json:"label"`
	Type      string             `json:"type"`
	CrossType bool               `json:"cross_type"`
	Limit     int                `json:"limit"`
	Weights   map[string]float64 `json:"weights,omitempty"`
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
	if err := validateWeights(req.Weights); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
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

	// Use the collection's persisted per-parameter weights unless overridden.
	sch, persisted, err := h.store.GetCollection(r.Context(), item.Type)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(req.Weights) == 0 {
		req.Weights = persisted
	}

	typeFilter := item.Type
	if req.CrossType {
		typeFilter = ""
	}

	results, err := h.store.SearchByVector(r.Context(), vec, typeFilter, item.ID, fetchLimitFor(req.Limit, req.Weights))
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	results = rerank.Rerank(*item, results, sch, req.Weights, req.Limit)

	WriteJSON(w, http.StatusOK, searchResponse{Query: item, Results: results})
}

type searchEmbedRequest struct {
	Text      string             `json:"text"`
	Object    map[string]any     `json:"object,omitempty"`
	Type      string             `json:"type"`
	CrossType bool               `json:"cross_type"`
	Limit     int                `json:"limit"`
	Weights   map[string]float64 `json:"weights,omitempty"`
}

// SearchByText finds the nearest items to ad-hoc input that is not a stored
// item: either free "text" or a structured "object" whose fields are embedded
// on the fly. When an object is given, its structured fields also participate
// in weighted re-ranking; for free text only the "semantic" weight applies.
func (h *Handlers) SearchByText(w http.ResponseWriter, r *http.Request) {
	var req searchEmbedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.Limit <= 0 {
		req.Limit = 10
	}
	if err := validateWeights(req.Weights); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Load the target collection's schema/weights so an object query is embedded
	// from its text fields and re-ranked by the same per-parameter weights.
	sch, persisted, err := h.store.GetCollection(r.Context(), req.Type)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(req.Weights) == 0 {
		req.Weights = persisted
	}

	// Build the embedding input from either an object or free text.
	var (
		text      string
		queryItem adapters.Item
		queryEcho any
	)
	switch {
	case len(req.Object) > 0:
		text = schema.EmbeddingText(req.Object, sch)
		queryItem = adapters.Item{Type: req.Type, Fields: req.Object}
		queryEcho = req.Object
	case req.Text != "":
		text = req.Text
		queryEcho = req.Text
	default:
		WriteError(w, http.StatusBadRequest, "provide 'text' or 'object'")
		return
	}

	vecs, err := h.embedder.Embed(r.Context(), []string{text})
	if err != nil {
		WriteError(w, http.StatusBadGateway, "embed failed: "+err.Error())
		return
	}

	typeFilter := req.Type
	if req.CrossType {
		typeFilter = ""
	}

	results, err := h.store.SearchByVector(r.Context(), pgvector.NewVector(vecs[0]), typeFilter, "", fetchLimitFor(req.Limit, req.Weights))
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// For free text there is no structured query item, so only the "semantic"
	// weight has any effect; for an object, its fields are compared too.
	results = rerank.Rerank(queryItem, results, sch, req.Weights, req.Limit)

	WriteJSON(w, http.StatusOK, map[string]any{
		"query":   queryEcho,
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

// ─── /collections ─────────────────────────────────────────────────────────────

func (h *Handlers) ListCollections(w http.ResponseWriter, r *http.Request) {
	types, err := h.store.ListCollections(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"collections": types})
}

func (h *Handlers) GetCollection(w http.ResponseWriter, r *http.Request) {
	typ := r.PathValue("type")
	sch, weights, err := h.store.GetCollection(r.Context(), typ)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(sch) == 0 {
		WriteError(w, http.StatusNotFound, fmt.Sprintf("collection %q not found", typ))
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"type": typ, "schema": sch, "weights": weights})
}

type weightsRequest struct {
	Weights map[string]float64 `json:"weights"`
}

// SetCollectionWeights merges the supplied weights into a collection's stored
// weights and persists them.
func (h *Handlers) SetCollectionWeights(w http.ResponseWriter, r *http.Request) {
	typ := r.PathValue("type")
	var req weightsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if err := validateWeights(req.Weights); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	_, existing, err := h.store.GetCollection(r.Context(), typ)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if existing == nil {
		existing = map[string]float64{}
	}
	for k, v := range req.Weights {
		existing[k] = v
	}

	if err := h.store.SetCollectionWeights(r.Context(), typ, existing); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			WriteError(w, http.StatusNotFound, fmt.Sprintf("collection %q not found (import it first)", typ))
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"type": typ, "weights": existing})
}

// ─── /index ───────────────────────────────────────────────────────────────────

func (h *Handlers) Index(w http.ResponseWriter, r *http.Request) {
	go func() {
		if err := indexer.Run(context.Background(), h.store, h.embedder, "all", 20, false, nil); err != nil {
			_ = err // logged nowhere; consider adding structured logging here
		}
	}()
	WriteJSON(w, http.StatusAccepted, map[string]string{"status": "indexing started"})
}

// ─── /import ──────────────────────────────────────────────────────────────────

type importRequest struct {
	Type       string           `json:"type"`
	LabelField string           `json:"label_field"`
	IDField    string           `json:"id_field"`
	TagsField  string           `json:"tags_field"`
	Items      []map[string]any `json:"items"`
}

// Import bulk-loads a collection. The body may be either a bare JSON array of
// objects (with the collection name in the ?type= query parameter) or an
// object {"type", "items":[...], "label_field", ...}. Items are inserted, then
// embedded in the background; the call returns 202 immediately.
func (h *Handlers) Import(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "read body: "+err.Error())
		return
	}

	opts := ingest.Options{Type: r.URL.Query().Get("type")}
	var raw []map[string]any

	if arr, arrErr := decodeArray(body); arrErr == nil {
		raw = arr
	} else {
		var req importRequest
		if err := json.Unmarshal(body, &req); err != nil {
			WriteError(w, http.StatusBadRequest, "body must be a JSON array or an object with an \"items\" array")
			return
		}
		raw = req.Items
		opts.Type = firstNonEmpty(req.Type, opts.Type)
		opts.LabelField = req.LabelField
		opts.IDField = req.IDField
		opts.TagsField = req.TagsField
	}

	if opts.Type == "" {
		WriteError(w, http.StatusBadRequest, "a collection 'type' is required (in the body or as ?type=)")
		return
	}
	if len(raw) == 0 {
		WriteError(w, http.StatusBadRequest, "no items to import")
		return
	}

	items, err := ingest.Normalize(raw, opts)
	if err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Detect and persist the collection's schema before embedding.
	fieldMaps := make([]map[string]any, len(items))
	for i, item := range items {
		fieldMaps[i] = item.Fields
	}
	if _, err := h.recordSchema(r.Context(), opts.Type, fieldMaps); err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	inserted, skipped := 0, 0
	for _, item := range items {
		ok, err := h.store.InsertItem(r.Context(), item)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if ok {
			inserted++
		} else {
			skipped++
		}
	}

	// Embed the newly imported collection in the background.
	typ := opts.Type
	go func() {
		if err := indexer.Run(context.Background(), h.store, h.embedder, typ, 20, false, nil); err != nil {
			_ = err // see note on Index: no structured logging here yet
		}
	}()

	WriteJSON(w, http.StatusAccepted, map[string]any{
		"type":     typ,
		"inserted": inserted,
		"skipped":  skipped,
		"status":   "indexing started",
	})
}

// decodeArray attempts to parse body as a bare JSON array of objects.
func decodeArray(body []byte) ([]map[string]any, error) {
	var arr []map[string]any
	if err := json.Unmarshal(body, &arr); err != nil {
		return nil, err
	}
	return arr, nil
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// generateID derives a type-prefixed ID from label, appending a numeric
// suffix to avoid collisions.
func (h *Handlers) generateID(ctx context.Context, typ, label string) string {
	base := typ + ":" + ingest.Slug(label)
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
