package recipe

import (
	"net/http"

	"restful-boilerplate/internal/app"
	router "restful-boilerplate/pkg/http"
)

// agentAdapter handles HTTP requests for the recipe AI agent domain.
type agentAdapter struct {
	svc *recipeService
	val app.Validator
}

func newAgentAdapter(svc *recipeService, val app.Validator) *agentAdapter {
	return &agentAdapter{svc: svc, val: val}
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
		router.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
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
		router.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	router.WriteJSON(w, http.StatusOK, resp)
}
