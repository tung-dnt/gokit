---
name: new-genkit-agent
description: Scaffold a new Genkit-powered AI agent as a flat domain package under internal/<domain>/
user_invocable: true
---

Scaffold a new AI-agent domain under `internal/<domain>/`. Use `internal/recipe/` as
the reference implementation (RAG over an in-memory vector store, Gemini via
Genkit, Dotprompt files, local vector retriever).

## Overview

Invoke this skill when adding an AI-powered endpoint that calls an LLM — RAG,
chat, classification, summarization, structured extraction, anything that goes
through `github.com/firebase/genkit/go/genkit`.

**Output:** a flat single package under `internal/<domain>/` plus one or more
Dotprompt files under `prompts/`.

**Prerequisite:** the `GEMINI_API_KEY` environment variable must be set at
runtime. `cmd/http/main.go` already wires `genkit.Init` with the
`&googlegenai.GoogleAI{}` plugin, the OTEL plugin, and a default model
(`googleai/gemini-2.5-flash`) — you inherit all of that through `a.Agent`.

## Architecture layout

Same flat-package rules as the `new-domain` skill — all layers in one package
`<domain>`, no sub-packages, file-name prefix separates concerns. The file set
differs slightly from a CRUD domain:

| File | Purpose |
|---|---|
| `module.<domain>.go` | `Module{agentAdapter}` + `NewModule(a) (*Module, error)` (returns error — init can fail) + `RegisterRoutes` |
| `adapter.agent.go` | HTTP handlers for the agent. Named `agentAdapter`, NOT `httpAdapter`. File is `adapter.agent.go`, NOT `adapter.http.go` |
| `service.<domain>.go` | `<domain>Service` holding `*genkit.Genkit`, retriever, doc store, chunker, typed prompt. |
| `domain.dto.go` | Request DTOs with `validate` + `json` + `example` tags |
| `domain.error.go` | Agent-specific sentinels: `ErrIndexingFailed`, `ErrRetrievalFailed`, etc. |
| `mapping.response.go` | Response types. The response for a typed prompt also doubles as its Dotprompt output schema |
| `observability.<domain>.go` | Framework-specific mapper `llmInfoFrom<Framework>` onto `telemetry.LLMInfo` |
| `prompts/<domain>_<op>.prompt` | Dotprompt file at repo root (NOT inside the domain package) |

See the `llm-observability` skill for why the framework-specific mapper lives in
the domain package instead of `pkg/telemetry/`.

---

## Step 1 — Write the Dotprompt file

Create `prompts/<domain>_<op>.prompt`. Genkit discovers prompts by filename stem
— `recipe_query.prompt` becomes `genkit.LookupDataPrompt(..., "recipe_query")`.

YAML frontmatter declares the model, config, and typed input/output schemas.
Handlebars body uses `{{role "system"}}` / `{{role "user"}}` blocks and may
include `{{#if <field>}}...{{/if}}` for optional inputs.

Reference — `prompts/recipe_query.prompt`:

```prompt
---
model: googleai/gemini-2.5-flash
config:
  temperature: 0.7
input:
  schema:
    question: string
    dietaryRestrictions?: string
    context: string
output:
  schema:
    answer: string, the answer to the user's question based on context
    sources?(array): string, relevant source excerpts used to form the answer
---
{{role "system"}}
You are a helpful recipe assistant. Answer questions about recipes using ONLY
the provided context. If the context doesn't contain relevant information,
say so honestly.

Context:
{{context}}

{{role "user"}}
{{question}}
{{#if dietaryRestrictions}}
Dietary restrictions: {{dietaryRestrictions}}
{{/if}}
```

The JSON tag names in your Go `queryPromptInput` struct must match the
frontmatter's `input.schema` keys exactly. Same for the output type vs
`output.schema`.

---

## Step 2 — DTOs and responses

`domain.dto.go` — HTTP request shapes with validator tags:

```go
package recipe

type IndexRecipeRequest struct {
    Content string `json:"content" validate:"required,min=1" example:"Classic guacamole: mash 3 ripe avocados..."`
}

type QueryRecipeRequest struct {
    Question            string `json:"question"                        validate:"required,min=1,max=500" example:"How do I make guacamole?"`
    DietaryRestrictions string `json:"dietaryRestrictions,omitempty"    validate:"omitempty,max=200"      example:"vegan"`
}
```

`mapping.response.go` — the `queryResponse` doubles as the Dotprompt output
schema, so its JSON tags must match the prompt's `output.schema` keys exactly:

```go
package recipe

type indexResponse struct {
    Success          bool `json:"success"`
    DocumentsIndexed int  `json:"documents_indexed"`
}

// queryResponse is the HTTP JSON response for a RAG query.
// Also used as the Dotprompt output schema.
type queryResponse struct {
    Answer  string   `json:"answer"`
    Sources []string `json:"sources,omitempty"`
}
```

---

## Step 3 — Error sentinels

`domain.error.go` — one sentinel per failure mode so the adapter can map each
to the right HTTP status and error message. Agents typically need at least two:

```go
package recipe

import "errors"

var ErrIndexingFailed  = errors.New("recipe: indexing failed")
var ErrRetrievalFailed = errors.New("recipe: retrieval failed")
```

Add more as the service grows (e.g. `ErrGenerationFailed`) — each unlocks a
distinct `writeErr` branch.

---

## Step 4 — Service

`service.<domain>.go` holds all Genkit interaction. Structure:

```go
package recipe

import (
    "context"
    "errors"
    "fmt"
    "log/slog"
    "strings"

    "github.com/firebase/genkit/go/ai"
    "github.com/firebase/genkit/go/genkit"
    "github.com/firebase/genkit/go/plugins/localvec"
    "go.opentelemetry.io/otel/trace"

    "gokit/pkg/chunker"
    "gokit/pkg/logger"
    "gokit/pkg/telemetry"
)

// queryPromptInput is the typed input for the recipe_query Dotprompt.
// Extends the HTTP request with a Context field populated from retrieval.
type queryPromptInput struct {
    Question            string `json:"question"`
    DietaryRestrictions string `json:"dietaryRestrictions,omitempty"`
    Context             string `json:"context"`
}

type recipeService struct {
    g           *genkit.Genkit
    tracer      trace.Tracer
    retriever   ai.Retriever
    docStore    *localvec.DocStore
    chunker     *chunker.Chunker
    queryPrompt *ai.DataPrompt[queryPromptInput, *queryResponse]
}

func newRecipeService(g *genkit.Genkit, tracer trace.Tracer) (*recipeService, error) {
    if err := localvec.Init(); err != nil {
        return nil, fmt.Errorf("localvec init: %w", err)
    }

    embedder := genkit.LookupEmbedder(g, "googleai/text-embedding-004")
    if embedder == nil {
        return nil, errors.New("embedder text-embedding-004 not found")
    }

    docStore, retriever, err := localvec.DefineRetriever(g, "recipeStore", localvec.Config{
        Embedder: embedder,
    }, nil)
    if err != nil {
        return nil, fmt.Errorf("define retriever: %w", err)
    }

    queryPrompt := genkit.LookupDataPrompt[queryPromptInput, *queryResponse](g, "recipe_query")
    if queryPrompt == nil {
        return nil, errors.New("prompt recipe_query not found")
    }

    return &recipeService{
        g: g, tracer: tracer,
        retriever: retriever, docStore: docStore,
        chunker: chunker.New(200, 50), queryPrompt: queryPrompt,
    }, nil
}
```

Indexing flow — chunk → wrap as `ai.Document` → push into the local vector
store. Always join the sentinel with the underlying error via `errors.Join`
before passing to `telemetry.SpanUnexpectedErr`:

```go
func (s *recipeService) indexRecipes(ctx context.Context, in IndexRecipeRequest) (*indexResponse, error) {
    ctx, span := s.tracer.Start(ctx, "recipe.recipeService.indexRecipes")
    defer span.End()

    logger.FromContext(ctx).InfoContext(ctx, "indexing recipe content")

    segments := s.chunker.Chunk(in.Content)
    docs := make([]*ai.Document, len(segments))
    for i, seg := range segments {
        docs[i] = ai.DocumentFromText(seg, nil)
    }

    if err := localvec.Index(ctx, docs, s.docStore); err != nil {
        return nil, telemetry.SpanUnexpectedErr(span, errors.Join(ErrIndexingFailed, err), "recipe.recipeService.indexRecipes")
    }
    return &indexResponse{Success: true, DocumentsIndexed: len(docs)}, nil
}
```

Query flow — retrieve, flatten doc text, execute the typed prompt inside an
LLM child span. Note the ordering: `RecordLLMAttrs` runs BEFORE `End` so token
attributes are attached to the span, and BEFORE the error check so we always
record usage even on partial failures:

```go
func (s *recipeService) queryRecipes(ctx context.Context, req QueryRecipeRequest) (*queryResponse, error) {
    ctx, span := s.tracer.Start(ctx, "recipe.recipeService.queryRecipes")
    defer span.End()

    logger.FromContext(ctx).InfoContext(ctx, "querying recipes", slog.String("question", req.Question))

    resp, err := genkit.Retrieve(ctx, s.g,
        ai.WithRetriever(s.retriever),
        ai.WithTextDocs(req.Question),
    )
    if err != nil {
        return nil, telemetry.SpanUnexpectedErr(span, errors.Join(ErrRetrievalFailed, err), "recipe.recipeService.queryRecipes: retrieve")
    }

    var cb strings.Builder
    for _, doc := range resp.Documents {
        for _, part := range doc.Content {
            if text := part.Text; text != "" {
                cb.WriteString(text)
                cb.WriteString("\n\n")
            }
        }
    }

    input := queryPromptInput{
        Question:            req.Question,
        DietaryRestrictions: req.DietaryRestrictions,
        Context:             cb.String(),
    }

    llmCtx, llmSpan := telemetry.StartLLMSpan(ctx, s.tracer, "generate")
    result, modelResp, err := s.queryPrompt.Execute(llmCtx, input)
    telemetry.RecordLLMAttrs(llmSpan, llmInfoFromGenkit(modelResp))
    llmSpan.End()
    if err != nil {
        return nil, telemetry.SpanUnexpectedErr(span, err, "recipe.recipeService.queryRecipes: generate")
    }
    return result, nil
}
```

---

## Step 5 — Framework-to-telemetry mapper

`observability.<domain>.go` — one small function that translates the
framework's response type into the framework-agnostic `telemetry.LLMInfo`.
Stays inside the domain package so `pkg/telemetry/` never learns about Genkit.

```go
package recipe

import (
    "github.com/firebase/genkit/go/ai"

    "gokit/pkg/telemetry"
)

// llmInfoFromGenkit maps a Genkit ModelResponse onto the framework-agnostic
// telemetry.LLMInfo shape. Model name is omitted: ai.ModelRequest stores it
// inside Config (any), so extraction would require provider-specific type
// assertions. Set it explicitly when calling genkit.Generate with a known model.
func llmInfoFromGenkit(resp *ai.ModelResponse) telemetry.LLMInfo {
    info := telemetry.LLMInfo{
        Provider: "google",
        System:   "gemini",
    }
    if resp == nil {
        return info
    }
    info.FinishReason = string(resp.FinishReason)
    if u := resp.Usage; u != nil {
        info.Usage = &telemetry.LLMUsage{
            InputTokens:  u.InputTokens,
            OutputTokens: u.OutputTokens,
            TotalTokens:  u.TotalTokens,
        }
    }
    return info
}
```

---

## Step 6 — Adapter

`adapter.agent.go` — thin HTTP layer. `writeErr` maps every domain sentinel to
a status code and a scrubbed error message. Never leak raw error strings to
the client — they often contain upstream model/vendor text:

```go
package recipe

import (
    "errors"
    "net/http"

    "gokit/internal/app"
    router "gokit/pkg/http"
    "gokit/pkg/logger"
)

type agentAdapter struct {
    svc *recipeService
    val app.Validator
}

func newAgentAdapter(svc *recipeService, val app.Validator) *agentAdapter {
    return &agentAdapter{svc: svc, val: val}
}

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

// @Summary      Index recipe content
// @Tags         recipes
// @Accept       json
// @Produce      json
// @Param        body  body      IndexRecipeRequest  true  "Recipe content to index"
// @Success      201   {object}  indexResponse
// @Failure      400   {object}  map[string]string
// @Failure      422   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /agents/recipes/index [post]
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

// @Summary      Query recipes
// @Tags         recipes
// @Router       /agents/recipes/query [post]
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
```

`@Tags` is the domain noun (recipes). `@Router` path is `/agents/<domain>/<op>`
because `/agents` is the nested group prefix in main.

---

## Step 7 — Module

`module.<domain>.go` — unlike CRUD modules, `NewModule` returns an error
because retriever definition, embedder lookup, and prompt lookup can all fail:

```go
package recipe

import (
    "fmt"

    "gokit/internal/app"
    router "gokit/pkg/http"
)

type Module struct {
    agentAdapter *agentAdapter
}

func NewModule(a *app.App) (*Module, error) {
    svc, err := newRecipeService(a.Agent, a.Tracer.Tracer("recipe"))
    if err != nil {
        return nil, fmt.Errorf("recipe module: %w", err)
    }
    adapter := newAgentAdapter(svc, a.Validator)
    return &Module{agentAdapter: adapter}, nil
}

func (m *Module) RegisterRoutes(g *router.Group) {
    g.POST("/index", m.agentAdapter.indexRecipeHandler)
    g.POST("/query", m.agentAdapter.queryRecipeHandler)
}
```

---

## Step 8 — Wire into main.go

All agents live under the `/agents` nested group. Build the module BEFORE
registering routes so the error can bubble out of `run()`:

```go
recipeMod, err := recipe.NewModule(a)
if err != nil {
    return fmt.Errorf("recipe module: %w", err)
}

r.Group("/v1", func(g *router.Group) {
    g.Prefix("/api")
    g.ANY("/swagger/", httpSwagger.WrapHandler)
    g.Group("/users", user.NewModule(a).RegisterRoutes)
    g.Group("/agents", func(g *router.Group) {
        g.Group("/recipes", recipeMod.RegisterRoutes)
        // g.Group("/<next-domain>", <next>Mod.RegisterRoutes)
    })
})
```

---

## Checklist

- [ ] `prompts/<domain>_<op>.prompt` created with matching input/output schemas
- [ ] `internal/<domain>/domain.dto.go` — request DTOs with validator tags
- [ ] `internal/<domain>/domain.error.go` — `ErrIndexingFailed`, `ErrRetrievalFailed`, or equivalent sentinels
- [ ] `internal/<domain>/mapping.response.go` — response types; any type used as a prompt output schema has JSON tags that match the `.prompt` output
- [ ] `internal/<domain>/service.<domain>.go` — service struct + `new<Domain>Service(g, tracer) (*<domain>Service, error)` + CRUD/RAG methods with LLM spans
- [ ] `internal/<domain>/observability.<domain>.go` — `llmInfoFromGenkit` (or `llmInfoFrom<Framework>`)
- [ ] `internal/<domain>/adapter.agent.go` — `agentAdapter` + `writeErr` + handlers with Swagger annotations under `/agents/<domain>/...`
- [ ] `internal/<domain>/module.<domain>.go` — `NewModule(a) (*Module, error)` + `RegisterRoutes`
- [ ] `cmd/http/main.go` — build module with error handling, register under `/agents/<domain>`
- [ ] `GEMINI_API_KEY` documented / available in the target environment
- [ ] `make check` passes
