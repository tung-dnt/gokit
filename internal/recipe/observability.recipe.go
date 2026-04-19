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
