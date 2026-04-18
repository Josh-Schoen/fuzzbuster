// Command moderation owns the safety-critical state machine:
//   * PII-scrubbing user-submitted notes (delegates the LLM call to the
//     llm-gateway, but the policy of WHAT to remove lives here).
//   * Crowd-verify thresholds (auto-hide at deny:confirm > 2:1, N >= 5).
//   * Human moderator dashboard (Tailwind + htmx, very thin).
//   * Audit trail in moderation_decisions.
//
// This service is the only one with write access to sighting/event/resource
// status columns. The api service can read; the orchestrator can insert
// `pending`; only this service can promote to `published` or demote to
// `hidden`.
package main

import (
	"log/slog"
	"os"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("moderation skeleton: state machine + dashboard land in week 5")

	// TODO(week-5):
	//   * gRPC client to llm-gateway for ModerationService.Scrub.
	//   * Postgres-backed CrowdVerify.Vote (+ rate limit).
	//   * htmx moderator dashboard at :9090/moderate (auth via fiscal-sponsor
	//     SSO; no public endpoint).
	//   * Background job: every 1m, recompute deny:confirm ratios, emit
	//     moderation_decisions, flip status.
}
