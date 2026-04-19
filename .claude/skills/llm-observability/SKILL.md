---
name: llm-observability
description: LLM span annotation patterns — framework-agnostic telemetry.LLMInfo contract, StartLLMSpan/RecordLLMAttrs, and the Genkit OTEL plugin
user_invocable: false
---

How this project annotates LLM calls so SigNoz's LLM dashboards light up.
The pattern is agnostic of Genkit — the same `telemetry.LLMInfo` contract and
the same `StartLLMSpan`/`RecordLLMAttrs` pair work for the OpenAI SDK, Anthropic
SDK, Bedrock, Azure OpenAI, or anything else. Only the small mapper function
(`llmInfoFrom<Framework>`) changes.

## Purpose

Every call to an LLM (or embedding model) creates an OTel span so we can answer
in SigNoz:

- What did each request cost in tokens?
- Which model/provider/system finished the call, and how?
- Where is latency spent across retrieve → generate → tool-call?

The plumbing lives in `pkg/telemetry/llm.go`. Domains annotate one span per
call at the point of invocation and never import the telemetry package
transitively from `pkg/` into framework code.

---

## The `LLMInfo` contract

`pkg/telemetry/llm.go`:

```go
type LLMUsage struct {
    InputTokens  int
    OutputTokens int
    TotalTokens  int
}

type LLMInfo struct {
    Provider     string
    System       string
    Model        string
    FinishReason string
    Usage        *LLMUsage
}
```

Field meanings (copied from the godoc — keep values low-cardinality):

| Field | Meaning | Conventional values |
|---|---|---|
| `Provider` | Company / cloud selling the model | `google` \| `openai` \| `anthropic` \| `aws-bedrock` \| `azure` |
| `System` | Model family / SDK surface | `gemini` \| `openai` \| `anthropic` \| `vertexai` \| `ollama` |
| `Model` | Fully-qualified model id | `gemini-2.5-flash`, `gpt-4o-mini`, `claude-sonnet-4-5` |
| `FinishReason` | Model's stop reason string | `stop`, `length`, `tool_calls`, `content_filter` |
| `Usage` | Token accounting; `nil` allowed | `{InputTokens, OutputTokens, TotalTokens}` |

`Provider` and `System` are required for SigNoz LLM dashboards to group
correctly; every other field is best-effort. Counts of zero are emitted as-is
— interpretation is left to the dashboard.

---

## `StartLLMSpan` and `RecordLLMAttrs`

```go
func StartLLMSpan(ctx context.Context, tracer trace.Tracer, op string) (context.Context, trace.Span)
func RecordLLMAttrs(span trace.Span, info LLMInfo)
```

Conventions enforced by the helpers:

- **Span name is `"llm.<op>"`.** SigNoz LLM dashboards filter on that prefix.
  Common ops: `generate`, `embed`, `rerank`, `classify`, `stream`.
- **`End()` is the caller's responsibility** — so you can record errors and
  attributes before the span closes.
- **Empty fields are skipped.** Full no-op when the whole `LLMInfo` is empty
  — safe to call when the framework returned `nil` or erred out before
  producing a response.
- **Dual-emits two attribute schemas** — the Traceloop schema (`llm.*`) that
  SigNoz's pre-built LLM dashboards query today, and the OTel GenAI semconv
  (`gen_ai.*`) for forward compatibility as the standard stabilizes. You set
  one `LLMInfo`; the helper writes both schemas.

---

## Call-site pattern

Canonical shape from `internal/recipe/service.recipe.go`:

```go
llmCtx, llmSpan := telemetry.StartLLMSpan(ctx, s.tracer, "generate")
result, modelResp, err := s.queryPrompt.Execute(llmCtx, input)
telemetry.RecordLLMAttrs(llmSpan, llmInfoFromGenkit(modelResp))
llmSpan.End()
if err != nil {
    return nil, telemetry.SpanUnexpectedErr(span, err, "...: generate")
}
```

The ordering matters:

1. **`StartLLMSpan` first** — so the framework call runs inside the child span
   and any sub-spans the framework itself creates are attached underneath.
2. **`RecordLLMAttrs` BEFORE `End`** — attributes set after `End` are dropped.
3. **`RecordLLMAttrs` BEFORE the error check** — you want token usage recorded
   even when the response came back but the post-processing (parsing, schema
   validation) errored.
4. **`End` before returning the error** — don't leak an unclosed span.

---

## Framework mappers live in the domain package

Each integration owns an `observability.<domain>.go` file with a single
function `llmInfoFrom<Framework>(resp *FrameworkResponse) telemetry.LLMInfo`.
This stays inside `internal/<domain>/` — not `pkg/telemetry/` — because:

- The mapping is framework-specific and the framework's response type would
  leak into `pkg/` otherwise (`pkg/telemetry/` imports only stdlib + OTel).
- Different frameworks in the same service can coexist, each with its own
  mapper, without conflicting.

Canonical example for Genkit (`internal/recipe/observability.recipe.go`):

```go
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

Note the model-name caveat in that file's doc comment: `ai.ModelRequest`
stores the model inside `Config any`, so extracting it would require
provider-specific type assertions. Set `info.Model` explicitly when calling
`genkit.Generate` with a known model literal.

Sketch for a hypothetical OpenAI SDK adapter:

```go
func openaiInfoFromResp(resp *openai.ChatCompletion) telemetry.LLMInfo {
    info := telemetry.LLMInfo{
        Provider: "openai",
        System:   "openai",
    }
    if resp == nil {
        return info
    }
    info.Model = resp.Model
    if len(resp.Choices) > 0 {
        info.FinishReason = string(resp.Choices[0].FinishReason)
    }
    info.Usage = &telemetry.LLMUsage{
        InputTokens:  int(resp.Usage.PromptTokens),
        OutputTokens: int(resp.Usage.CompletionTokens),
        TotalTokens:  int(resp.Usage.TotalTokens),
    }
    return info
}
```

Anthropic, Bedrock, Vertex, Ollama — all follow the same shape.

---

## Genkit OTEL plugin

`cmd/http/main.go` wires the Genkit OTEL plugin alongside the app's own
telemetry:

```go
otelPlugin := opentelemetry.New(opentelemetry.Config{
    ServiceName:    "gokit",
    ForceExport:    true,
    OTLPEndpoint:   telemetry.EndpointHTTP(),
    OTLPUseHTTP:    true,
    MetricInterval: 15 * time.Second,
})

ai := genkit.Init(ctx,
    genkit.WithPlugins(&googlegenai.GoogleAI{}, otelPlugin),
    genkit.WithDefaultModel("googleai/gemini-2.5-flash"),
    genkit.WithPromptDir("./prompts"),
)
```

It exports Genkit-internal spans (LLM calls, embedding ops, retriever spans,
Genkit flow spans) over OTLP HTTP to the same collector as app spans — they
land under the same `service.name` and correlate via trace ID.

The plugin gives you **framework-internal** spans for free. You still wrap
call sites with `StartLLMSpan` + `RecordLLMAttrs` because:

- You want an **application-owned** parent span with the domain's naming
  convention (`llm.generate`, not `genkit.generate.googleai.gemini-*`).
- The Traceloop attribute set SigNoz's pre-built dashboards filter on is
  emitted by `RecordLLMAttrs`, not by the plugin.

---

## Dashboards

- SigNoz UI: `http://localhost:8080` (started via `make obs/up`).
- The "LLM Observability" dashboard filters traces by span name prefix
  `llm.*` and aggregates the Traceloop attributes (`llm.provider`,
  `llm.model_name`, `llm.usage.*`).
- Filter by `gen_ai.provider.name` / `gen_ai.usage.*` once SigNoz adopts
  the GenAI semconv view — `RecordLLMAttrs` already emits those.

---

## Dos and don'ts

**Do**

- Annotate every LLM call with an `LLMInfo`. Skipping it blanks the SigNoz
  LLM dashboard.
- Keep `Provider` and `System` low-cardinality — stick to the conventional
  values table above.
- Record attributes BEFORE closing the span, BEFORE checking the error.
- Put the framework mapper next to the service that uses it
  (`internal/<domain>/observability.<domain>.go`).

**Don't**

- Don't put raw prompt text or model response text in span attributes — PII
  risk and cardinality explosion. The model and retrieval contents belong in
  application logs or a dedicated eval/trace store, not in OTel attributes.
- Don't use high-cardinality values (user id, request id, prompt text) in
  `Provider`, `System`, or `Model` — they break SigNoz aggregation.
- Don't import a framework SDK into `pkg/telemetry/` to "simplify" the mapper.
  The domain-local mapper is the boundary that keeps `pkg/` framework-free.
- Don't call `End()` on the LLM span before `RecordLLMAttrs` — the attributes
  will be dropped.
