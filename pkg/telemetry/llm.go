package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// LLMUsage holds token accounting for a single LLM call.
// Counts of zero are emitted as-is — interpretation is up to the dashboard.
type LLMUsage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

// LLMInfo describes one LLM call for span annotation. Provider and System are
// required for SigNoz dashboards to group correctly; everything else is
// best-effort. Map your framework's response (Genkit, OpenAI SDK, Anthropic
// SDK, ...) onto this struct in the calling site.
//
// Conventional values:
//   - Provider: "google" | "openai" | "anthropic" | "aws-bedrock" | "azure"
//   - System:   "gemini" | "openai" | "anthropic" | "vertexai" | "ollama"
//   - Model:    fully-qualified id (e.g. "gemini-2.5-flash", "gpt-4o-mini")
type LLMInfo struct {
	Provider     string
	System       string
	Model        string
	FinishReason string
	Usage        *LLMUsage
}

// StartLLMSpan starts a child span named "llm.<op>" — the naming convention
// SigNoz LLM dashboards filter on. End() is the caller's responsibility so
// errors can be recorded before the span closes.
func StartLLMSpan(ctx context.Context, tracer trace.Tracer, op string) (context.Context, trace.Span) {
	return tracer.Start(ctx, "llm."+op)
}

// RecordLLMAttrs annotates an LLM span with provider, model, and token usage.
// Dual-emits the Traceloop schema (which SigNoz's pre-built LLM dashboards
// query today) and the OTel GenAI semconv (forward-compatible).
//
// Empty fields are skipped — pass only what you have. No-op when info is
// completely empty.
func RecordLLMAttrs(span trace.Span, info LLMInfo) {
	attrs := make([]attribute.KeyValue, 0, 12)

	if info.Provider != "" {
		attrs = append(attrs,
			attribute.String("llm.provider", info.Provider),
			attribute.String("gen_ai.provider.name", info.Provider),
		)
	}
	if info.System != "" {
		attrs = append(attrs, attribute.String("gen_ai.system", info.System))
	}
	if info.Model != "" {
		attrs = append(attrs,
			attribute.String("llm.model_name", info.Model),
			attribute.String("llm.response.model", info.Model),
			attribute.String("gen_ai.request.model", info.Model),
			attribute.String("gen_ai.response.model", info.Model),
		)
	}
	if info.FinishReason != "" {
		attrs = append(attrs, attribute.String("gen_ai.response.finish_reasons", info.FinishReason))
	}
	if u := info.Usage; u != nil {
		attrs = append(attrs,
			attribute.Int("llm.usage.prompt_tokens", u.InputTokens),
			attribute.Int("llm.usage.completion_tokens", u.OutputTokens),
			attribute.Int("llm.usage.total_tokens", u.TotalTokens),
			attribute.Int("gen_ai.usage.input_tokens", u.InputTokens),
			attribute.Int("gen_ai.usage.output_tokens", u.OutputTokens),
		)
	}

	if len(attrs) == 0 {
		return
	}
	span.SetAttributes(attrs...)
}
