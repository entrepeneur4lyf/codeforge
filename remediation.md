# Remediation Plan and Task Tracker

Use this checklist to drive production-readiness. Mark items complete as we land edits. Grouped by area with priorities.

## Completed
- [x] TUI: Implement paste handling in `internal/tui/components/chat/editor.go` (file paths attach; text appends)
- [x] LSP: Guard nil writers and uninitialized clients in `internal/lsp/transport.go`
- [x] Config: Make `config.WorkingDirectory()` non-panicking with `os.Getwd()` fallback
- [x] Security: Restrict Web/WebSocket origins to localhost by default (`internal/web/server.go`, `internal/api/server.go`)
- [x] TUI: Consolidate to single model selection dialog (removed legacy `internal/tui/components/dialogs/model.go`; normalized events)

## TUI
- [ ] Make sure TUI is used by defaut so that the user doesn't have to enter "codeforge tui"
- [ ] High: Replace hardcoded version in status bar with build-time version (ldflags) (`internal/tui/components/status/status.go`)
- [ ] Medium: Fully implement inline markdown relying on Glammour

## LLM Providers / Core LLM Layer
- [ ] High: Create shared usage metering adapter so all providers emit consistent `ApiStreamUsageChunk`
- [ ] High: Standardize error mapping (auth, rate-limit, timeout, retriable) and plug into `internal/llm/retry.go`
- [ ] Medium: Add provider conformance tests (streaming, usage, retries, timeouts) across OpenAI/Anthropic/Gemini/OpenRouter

## CLI (codeforge, mcp-manage)
- [ ] High: Replace scattered `os.Exit` with returned errors; exit once at root; print errors to stderr
- [ ] Medium: Replace `fmt.Printf/Println` with structured logger; add verbosity flag (`--verbose`)

## Embeddings
- [ ] Medium: Switch `log.Printf` to centralized logger with levels in `internal/embeddings/embeddings.go`
- [ ] Medium: Ensure consistent request timeouts/retries/backoff; expose via config
- [ ] Medium: Confirm provider selection and reindex behavior documented and observable in logs

## Web/API
- [ ] High: Make allowed origins configurable (allowlist) instead of hardcoded prefixes (`internal/web/server.go`)
- [ ] Medium: Guard static UI serving (`internal/api/server.go`): feature-flag or detect `./web/dist` and log clear message if missing
- [ ] Medium: Audit direct FS fallbacks in web server file endpoints; prefer permission-managed path (`permissions.FileOperationManager`) and make fallback opt-in

## History / Auditability
- [ ] High: Replace in-memory `HistoryService` (tools) with persistent store (sqlite) behind the same interface (`internal/llm/tools/history_stub.go`)

## Configuration
- [ ] High: Unify config file path/format between Web UI settings writer and core viper config; ensure single canonical `~/.codeforge.json`

## Testing & QA
- [ ] High: Add e2e path covering: TUI chat send, streaming, file read/write via permissions, embeddings search, and LSP diagnostics
- [ ] Medium: Security tests for CORS/WS origins; reject non-allowed origins
- [ ] Medium: Load test streaming handlers (backpressure, channel closure, no panics)

## Docs
- [ ] Medium: Document runtime version injection, origin allowlist config, logging levels, and embedding provider selection

## Security / Permissions
- [ ] Medium: Review default permission settings; ensure safe defaults; verify audit logging is enabled and documented

## Misc Cleanup
- [ ] Low: Remove or finalize placeholder content in `internal/llm/tools/bash.go` (test plan snippet)
- [ ] Low: Move `test_ai_commit.go` to a `tools/` or `examples/` directory or guard with build tags to avoid confusion

---

Owner legend: assign owners as you pick up tasks (e.g., `@owner`), and update status inline.

- Format for status updates: `- [x] <task> — @owner (PR #123)`
- Keep tasks atomic; if a task grows, split it into subtasks beneath the parent.
