package recipe

import (
	"errors"
	"net/http"

	"gokit/internal/app"
	router "gokit/pkg/http"
	"gokit/pkg/logger"
)

// agentAdapter handles HTTP requests for the recipe AI agent domain.
type agentAdapter struct {
	svc *recipeService
	val app.Validator
}

func newAgentAdapter(svc *recipeService, val app.Validator) *agentAdapter {
	return &agentAdapter{svc: svc, val: val}
}

// writeErr maps recipe domain errors to HTTP responses, logs once with trace
// correlation, and never leaks raw error strings to clients.
func (m *agentAdapter) writeErr(r *http.Request, w http.ResponseWriter, err error) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	switch {
	case errors.Is(err, ErrIndexingFailed):
		log.ErrorContext(ctx, "recipe indexing failed", "error", err)
		router.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "indexing failed"})
	case errors.Is(err, ErrRetrievalFailed):
		log.ErrorContext(ctx, "recipe retrieval failed", "error", err)
		router.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "retrieval failed"})
	default:
		log.ErrorContext(ctx, "recipe request failed", "error", err)
		router.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
}

// indexRecipeHandler ingests recipe content into the vector store.
//
//	@Summary      Index recipe content
//	@Tags         recipes
//	@Accept       json
//	@Produce      json
//	@Param        body  body      IndexRecipeRequest  true  "Recipe content to index"
//	@Success      201   {object}  indexResponse
//	@Failure      400   {object}  map[string]string
//	@Failure      422   {object}  map[string]string
//	@Failure      500   {object}  map[string]string
//	@Router       /agents/recipes/index [post]
func (m *agentAdapter) indexRecipeHandler(w http.ResponseWriter, r *http.Request) {
	var req IndexRecipeRequest
	if !router.Bind(m.val, w, r, &req) {
		return
	}
	resp, err := m.svc.indexRecipes(r.Context(), req)
	if err != nil {
		m.writeErr(r, w, err)
		return
	}
	router.WriteJSON(w, http.StatusCreated, resp)
}

// queryRecipeHandler answers a question using RAG over indexed recipe content.
//
//	@Summary      Query recipes
//	@Tags         recipes
//	@Accept       json
//	@Produce      json
//	@Param        body  body      QueryRecipeRequest  true  "Recipe query"
//	@Success      200   {object}  queryResponse
//	@Failure      400   {object}  map[string]string
//	@Failure      422   {object}  map[string]string
//	@Failure      500   {object}  map[string]string
//	@Router       /agents/recipes/query [post]
func (m *agentAdapter) queryRecipeHandler(w http.ResponseWriter, r *http.Request) {
	var req QueryRecipeRequest
	if !router.Bind(m.val, w, r, &req) {
		return
	}
	resp, err := m.svc.queryRecipes(r.Context(), req)
	if err != nil {
		m.writeErr(r, w, err)
		return
	}
	router.WriteJSON(w, http.StatusOK, resp)
}
