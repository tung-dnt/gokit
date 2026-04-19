package recipe

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/localvec"
	"go.opentelemetry.io/otel/trace"

	"restful-boilerplate/pkg/chunker"
	"restful-boilerplate/pkg/logger"
	"restful-boilerplate/pkg/telemetry"
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
		g:           g,
		tracer:      tracer,
		retriever:   retriever,
		docStore:    docStore,
		chunker:     chunker.New(200, 50),
		queryPrompt: queryPrompt,
	}, nil
}

func (s *recipeService) indexRecipes(ctx context.Context, in IndexRecipeRequest) (*indexResponse, error) {
	ctx, span := s.tracer.Start(ctx, "recipe.recipeService.indexRecipes")
	defer span.End()

	logger.FromContext(ctx).Info("indexing recipe content")

	segments := s.chunker.Chunk(in.Content)
	docs := make([]*ai.Document, len(segments))
	for i, seg := range segments {
		docs[i] = ai.DocumentFromText(seg, nil)
	}

	if err := localvec.Index(ctx, docs, s.docStore); err != nil {
		return nil, telemetry.SpanErr(span, err, "recipe.recipeService.indexRecipes: index")
	}

	return &indexResponse{
		Success:          true,
		DocumentsIndexed: len(docs),
	}, nil
}

func (s *recipeService) queryRecipes(ctx context.Context, req QueryRecipeRequest) (*queryResponse, error) {
	ctx, span := s.tracer.Start(ctx, "recipe.recipeService.queryRecipes")
	defer span.End()

	logger.FromContext(ctx).Info("querying recipes", "question", req.Question)

	resp, err := genkit.Retrieve(ctx, s.g,
		ai.WithRetriever(s.retriever),
		ai.WithTextDocs(req.Question),
	)
	if err != nil {
		return nil, telemetry.SpanErr(span, err, "recipe.recipeService.queryRecipes: retrieve")
	}

	// Extract text context from retrieved documents.
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

	result, _, err := s.queryPrompt.Execute(ctx, input)
	if err != nil {
		return nil, telemetry.SpanErr(span, err, "recipe.recipeService.queryRecipes: generate")
	}

	return result, nil
}
