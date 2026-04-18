// Command llm-gateway centralizes all LLM calls (Ollama-local + Claude-hosted)
// behind one gRPC surface. Centralization is what makes the cost-control,
// PII-scrubbing, and escalation policy auditable.
//
// Routing policy (see docs/safety-policy.md §8):
//   * Default: Ollama llama3.1:8b for classify/summarize/extract.
//   * Embeddings: Ollama nomic-embed-text.
//   * Translation: Ollama qwen2.5:7b for ES↔EN; escalate elsewhere.
//   * Escalate to Claude Haiku 4.5 if:
//       - classifier confidence < 0.7
//       - translation source is not in {en, es}
//       - the moderation service flags the item
//   * Escalate to Claude Sonnet 4.5 only when a human moderator clicks
//     "assist".
//
// All inputs hitting Claude are PII-scrubbed locally first.
package main

import (
	"log/slog"
	"os"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("llm-gateway skeleton: gRPC server lands in week 3-4")

	// TODO(week-3):
	//   * Wire ollama-go client (OLLAMA_HOST=http://ollama:11434).
	//   * Wire anthropic-sdk-go (ANTHROPIC_API_KEY).
	//   * Implement ModerationService.{Scrub, Classify} from
	//     proto/fuzzbuster/v1/moderation.proto.
	//   * Add per-job cost ledger so the orchestrator can budget.
}
