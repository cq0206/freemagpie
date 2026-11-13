# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

@FREEMAGPIE.md
@AGENTS.md
@LESSONS.md

## What this repository is

FreeMagpie is a downstream distribution of [yetone/magpie](https://github.com/yetone/magpie) (MIT): one local gateway (127.0.0.1:3425) that speaks OpenAI Chat/Responses, Anthropic Messages and Gemini protocols, translates between them, and lets any coding agent use any provider, subscription or account behind it. This fork's goal is **minimal, enumerable, branding-only diffs** — keep up with upstream, don't diverge functionally.

### Fork workflow (read FREEMAGPIE.md before doing anything)

- `main` is fast-forward-only, always equal to `upstream/main`. **Never commit to `main`.**
- All work goes on `freemagpie`; sync with `git fetch upstream && git checkout main && git merge --ff-only upstream/main && git checkout freemagpie && git merge main`. On merge conflicts, keep only our branding changes; everything else follows upstream.
- Every diff vs upstream must be registered in FREEMAGPIE.md's 改动清单 (change list) and marked with `// freemagpie:` comments in code (audit with `git grep -n "freemagpie:"`).
- Branding changes user-visible text only. Config paths (`~/.config/magpie/`), HTTP identifiers (`{"name":"magpie"}`, `magpie-<agent>` tokens, `X-Magpie-Session`), env vars (`MAGPIE_*`), launchers (`claude-magpie`) and other documented behavior contracts stay `magpie` — changing them breaks user data compatibility and the upstream test suite (~4240 Go + ~1436 JS tests assert these strings).
- The freemagpie.com website lives in a separate repo ([cq0206/freemagpie-site](https://github.com/cq0206/freemagpie-site)); this repo's `site/` stays upstream-untouched.

## Commands

The Makefile self-guards on the module path: run `make` from the repo root (`make -C path/to/repo` works). Local toolchain (this machine): `export PATH="$HOME/sdk/go1.26.8/bin:$PATH"`, `GOFLAGS=-mod=mod`, `GOPROXY=https://goproxy.cn,direct`.

```sh
make build        # desktop build (cgo webview; native only)
make cli          # terminal-only build (CGO_ENABLED=0, -tags nogui, cross-compiles)
make test         # go vet + go test with production tags
make test-ui      # Playwright GUI tests in internal/gui/tests (Chromium + WebKit)
make dev          # dev app: UI from internal/gui/assets, reload on save, ports 3426-3429
make app          # macOS .app bundle
make install      # go install
```

Single package / single test (the full suite is slow; run per package):

```sh
go test ./internal/gateway
go test -run TestName ./internal/provider
BROWSER=webkit node --test internal/gui/tests/foo.test.cjs   # one GUI test, WebKit
```

Acceptance criteria for any change are in [docs/code-standards.md](docs/code-standards.md): `gofmt -l` on changed files, `go vet ./...` plus `GOOS=linux`/`GOOS=windows` vet/build (cross-OS builds skip `_test.go`; vet does not), `go test ./...` (the repo's guard tests live in packages a change doesn't touch), and GUI tests in both engines and every language for `internal/gui/assets` changes.

## Architecture

Go 1.26.8, module `github.com/yetone/magpie` (upstream module path is kept on purpose). `main.go` only embeds THIRD_PARTY_NOTICES and starts `internal/cli`; all commands live there.

- **`internal/`** (~90 packages) is the whole app. The important ones:
  - `internal/gateway` — the local gateway and its request pipeline: protocol translation, routing groups, fallback, session affinity.
  - `internal/provider` — providers, API keys, subscription accounts, refresh, allowance; also the movers that retire built-in subscriptions to plugins (`migrate*.go`, see AGENTS.md).
  - `internal/agent` + `internal/agentenv` — per-agent config wiring (Claude Code, Codex, Gemini CLI, …); agent env vars must go through `agentenv.Vars` so the test sandbox clears them.
  - `internal/plugin` / `internal/pluginrt` — the plugin host (plugins run on Bun) and the built-in plugin runtime.
  - `internal/masking`, `internal/pii`, `internal/redact` — privacy masking of prompts/replies.
  - `internal/gui` — tray/window/web UI shell; page code is plain JS in `internal/gui/assets` (no build step), with Playwright tests in `internal/gui/tests`.
  - `internal/tui` — terminal UI (bubbletea).
  - `internal/testenv` — the sandboxed test environment (temp HOME, pinned Go caches); tests never touch a real agent config or `~/.config/magpie`.
- **`plugins/`** — the built-in plugins (registry, gateway middleware, tools, routing, budget, search, …), registered via `plugins/all` in `main.go`.
- **`docs/subsystems/`** — design references per subsystem (responsibilities, runtime paths, state, contracts). Read the relevant one before changing a documented subsystem; update it in the same change when a documented contract changes.
- **`native/`** — native code (privacy-filter); **`site/`** — upstream's usemagpie.ai static site, kept as-is.

Rules that come up repeatedly (full versions in AGENTS.md, docs/code-standards.md and LESSONS.md): run child processes through `proc.Command`, not `exec.Command` (TestNoCommandBypassesProc); a failing/unknown read means "unknown", never "empty"; a fix must reach every sibling of the bug; a fix comes with a test that fails without it; tests never reach the network.
