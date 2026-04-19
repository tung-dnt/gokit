package recipe

import (
	"fmt"

	"gokit/internal/app"
	router "gokit/pkg/http"
)

// Module exposes recipe AI agent endpoints over HTTP.
type Module struct {
	agentAdapter *agentAdapter
}

// NewModule wires the recipe service from the shared App container.
func NewModule(a *app.App) (*Module, error) {
	svc, err := newRecipeService(a.Agent, a.Tracer.Tracer("recipe"))
	if err != nil {
		return nil, fmt.Errorf("recipe module: %w", err)
	}
	adapter := newAgentAdapter(svc, a.Validator)
	return &Module{agentAdapter: adapter}, nil
}

// RegisterRoutes mounts all recipe agent endpoints onto g.
func (m *Module) RegisterRoutes(g *router.Group) {
	g.POST("/index", m.agentAdapter.indexRecipeHandler)
	g.POST("/query", m.agentAdapter.queryRecipeHandler)
}
