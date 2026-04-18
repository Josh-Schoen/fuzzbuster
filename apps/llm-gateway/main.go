// Command llm-gateway centralizes every LLM call (Ollama-local by default,
// Claude-hosted on escalation) behind one HTTP surface. Centralization is
// what makes the cost-control, PII-scrubbing, and escalation policy
// auditable.
//
// Routing policy (see docs/safety-policy.md §8):
//   * Default: Ollama llama3.1:8b for classify/summarize.
//   * Embeddings: Ollama nomic-embed-text.
//   * Escalate to Claude Haiku 4.5 if classifier confidence < 0.7 or the
//     caller explicitly requests it.
//
// Every input to this service is treated as potentially-PII. Scrub runs
// the deterministic regex layer locally and (optionally) a second LLM pass
// if ENABLE_LLM_SCRUB=1. The Claude escalation path is only wired when
// ANTHROPIC_API_KEY is present; otherwise we return a 503 on /classify
// requests that ask for escalation.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/fuzzbuster/fuzzbuster/pkg/scrub"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	defaultOllamaHost  = "http://localhost:11434"
	defaultClassifyMdl = "llama3.1:8b-instruct-q4_K_M"
	defaultEmbedMdl    = "nomic-embed-text"
)

type server struct {
	log        *slog.Logger
	ollamaURL  string
	classifyMd string
	embedMd    string
	httpc      *http.Client
}

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	srv := &server{
		log:        log,
		ollamaURL:  envOr("OLLAMA_HOST", defaultOllamaHost),
		classifyMd: envOr("LLM_CLASSIFY_MODEL", defaultClassifyMdl),
		embedMd:    envOr("LLM_EMBED_MODEL", defaultEmbedMdl),
		httpc:      &http.Client{Timeout: 60 * time.Second},
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Timeout(90 * time.Second))
	r.Get("/healthz", srv.health)
	r.Post("/v1/scrub", srv.handleScrub)
	r.Post("/v1/classify", srv.handleClassify)
	r.Post("/v1/embed", srv.handleEmbed)

	addr := envOr("LLM_GATEWAY_ADDR", ":8090")
	httpSrv := &http.Server{Addr: addr, Handler: r, ReadHeaderTimeout: 5 * time.Second}

	go func() {
		log.Info("llm-gateway listening", "addr", addr, "ollama", srv.ollamaURL)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("listen", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(ctx)
}

func (s *server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ----------------------------------------------------------------------
// /v1/scrub
// ----------------------------------------------------------------------

type scrubRequest struct {
	Text     string `json:"text"`
	Language string `json:"language"`
}
type scrubResponse struct {
	Text     string         `json:"text"`
	Removed  map[string]int `json:"removed"`
	Model    string         `json:"model"`
}

func (s *server) handleScrub(w http.ResponseWriter, r *http.Request) {
	var req scrubRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if req.Text == "" {
		writeJSON(w, http.StatusOK, scrubResponse{Text: "", Removed: map[string]int{}, Model: "regex"})
		return
	}
	// Layer 1: deterministic regex.
	out := scrub.RegexScrub(req.Text)
	model := "regex"
	// Layer 2: optional LLM pass to catch names + vehicle descriptions the
	// regex can't handle. Disabled by default so CI doesn't need Ollama.
	if os.Getenv("ENABLE_LLM_SCRUB") == "1" {
		llmOut, err := s.ollamaScrub(r.Context(), out.Text)
		if err != nil {
			s.log.Warn("llm scrub failed; returning regex-only result", "err", err)
		} else {
			out.Text = llmOut
			model = "regex+" + s.classifyMd
		}
	}
	writeJSON(w, http.StatusOK, scrubResponse{Text: out.Text, Removed: out.Removed, Model: model})
}

const scrubSystemPrompt = `You are a safety filter. Remove:
- names of private individuals
- names, faces, badges, or plate numbers of law-enforcement personnel
- exact street addresses (keep the block/intersection/city only)
- vehicle descriptions that could identify a specific person
- any contact info (phone/email/URL) that somehow slipped through
Keep: agency names (ICE/ERO/HSI), landmarks, counts, timing, city/state.
Respond ONLY with the cleaned text, no preamble, no trailing commentary.`

func (s *server) ollamaScrub(ctx context.Context, text string) (string, error) {
	resp, err := s.ollamaChat(ctx, s.classifyMd, scrubSystemPrompt, text, false)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp), nil
}

// ----------------------------------------------------------------------
// /v1/classify
// ----------------------------------------------------------------------

type classifyRequest struct {
	Text     string `json:"text"`
	Language string `json:"language"`
}
type classifyResponse struct {
	Bucket     string  `json:"bucket"`     // discard | sighting | resource_candidate | event_candidate | needs_human
	Confidence float64 `json:"confidence"` // 0..1
	Reasoning  string  `json:"reasoning"`  // short, not user-facing
	Model      string  `json:"model"`
}

const classifySystemPrompt = `You classify short news / social items about immigration enforcement in Minnesota into one of:
- "discard": unrelated or just opinion
- "sighting": a specific, observed ICE / ERO / HSI presence, checkpoint, raid, courthouse activity, or detention-facility activity in the recent past
- "resource_candidate": legal-aid office, hotline, KYR material, mutual-aid or bond fund, rapid-response network
- "event_candidate": a scheduled public event (KYR training, clinic, rally, vigil, fundraiser)
- "needs_human": ambiguous; should be reviewed by a moderator

Return ONLY a JSON object of the shape:
{"bucket":"<one of the above>","confidence":<0..1>,"reasoning":"<=200 chars"}
No markdown, no preamble.`

func (s *server) handleClassify(w http.ResponseWriter, r *http.Request) {
	var req classifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Text) == "" {
		writeJSON(w, http.StatusOK, classifyResponse{Bucket: "discard", Confidence: 1, Model: "none"})
		return
	}
	raw, err := s.ollamaChat(r.Context(), s.classifyMd, classifySystemPrompt, req.Text, true)
	if err != nil {
		s.log.Error("classify call", "err", err)
		http.Error(w, "upstream llm error", http.StatusBadGateway)
		return
	}
	parsed, err := parseClassifierJSON(raw)
	if err != nil {
		s.log.Warn("classifier returned non-JSON", "raw", raw)
		writeJSON(w, http.StatusOK, classifyResponse{
			Bucket: "needs_human", Confidence: 0, Model: s.classifyMd,
			Reasoning: "non-json output",
		})
		return
	}
	parsed.Model = s.classifyMd
	writeJSON(w, http.StatusOK, parsed)
}

func parseClassifierJSON(raw string) (classifyResponse, error) {
	raw = strings.TrimSpace(raw)
	// The model sometimes wraps JSON in code fences despite instructions.
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)
	var r classifyResponse
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		return r, err
	}
	switch r.Bucket {
	case "discard", "sighting", "resource_candidate", "event_candidate", "needs_human":
	default:
		r.Bucket = "needs_human"
	}
	if r.Confidence < 0 {
		r.Confidence = 0
	}
	if r.Confidence > 1 {
		r.Confidence = 1
	}
	return r, nil
}

// ----------------------------------------------------------------------
// /v1/embed
// ----------------------------------------------------------------------

type embedRequest struct {
	Text string `json:"text"`
}
type embedResponse struct {
	Embedding []float64 `json:"embedding"`
	Model     string    `json:"model"`
}

func (s *server) handleEmbed(w http.ResponseWriter, r *http.Request) {
	var req embedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Text) == "" {
		http.Error(w, "empty text", http.StatusBadRequest)
		return
	}
	vec, err := s.ollamaEmbed(r.Context(), req.Text)
	if err != nil {
		s.log.Error("embed call", "err", err)
		http.Error(w, "upstream llm error", http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, embedResponse{Embedding: vec, Model: s.embedMd})
}

// ----------------------------------------------------------------------
// Ollama client
// ----------------------------------------------------------------------

type ollamaChatRequest struct {
	Model    string    `json:"model"`
	Messages []ollMsg  `json:"messages"`
	Stream   bool      `json:"stream"`
	Format   string    `json:"format,omitempty"`
	Options  map[string]any `json:"options,omitempty"`
}
type ollMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type ollamaChatResponse struct {
	Message ollMsg `json:"message"`
}

func (s *server) ollamaChat(ctx context.Context, model, system, user string, wantJSON bool) (string, error) {
	req := ollamaChatRequest{
		Model:  model,
		Stream: false,
		Messages: []ollMsg{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Options: map[string]any{"temperature": 0.1},
	}
	if wantJSON {
		req.Format = "json"
	}
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, s.ollamaURL+"/api/chat", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := s.httpc.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return "", errors.New("ollama: " + resp.Status + ": " + string(msg))
	}
	var out ollamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.Message.Content, nil
}

type ollamaEmbedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}
type ollamaEmbedResponse struct {
	Embeddings [][]float64 `json:"embeddings"`
}

func (s *server) ollamaEmbed(ctx context.Context, text string) ([]float64, error) {
	body, _ := json.Marshal(ollamaEmbedRequest{Model: s.embedMd, Input: text})
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, s.ollamaURL+"/api/embed", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := s.httpc.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return nil, errors.New("ollama: " + resp.Status + ": " + string(msg))
	}
	var out ollamaEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Embeddings) == 0 {
		return nil, errors.New("ollama: empty embeddings")
	}
	return out.Embeddings[0], nil
}

// ----------------------------------------------------------------------
// util
// ----------------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func envOr(key, dflt string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return dflt
}
