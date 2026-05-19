# Contributing to notebooklm-mcp-server

Thank you for contributing. This document describes workflow conventions, package contracts, and the fastest path from idea to merged PR.

---

## Prerequisites

| Tool       | Version  | Purpose                        |
|------------|----------|--------------------------------|
| Go         | 1.26+    | Build and test                 |
| Playwright | Chromium | Browser automation (tests that require a live page) |

Install Playwright once after cloning:

```sh
go run github.com/playwright-community/playwright-go/cmd/playwright install chromium
```

---

## TDD Workflow

All changes must have tests. The expected cycle is **red → green → refactor**.

```sh
# Run the full suite
go test ./...

# Run a single package
go test ./internal/notebooklm/...

# Run with race detector (required before opening a PR)
go test -race ./...

# Run with coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

Tests that exercise real browser automation are marked with a build tag or skipped when `PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1` is set. Unit tests must not require a live browser.

---

## Package Conventions

| Package            | Responsibility                                        | Rules                                               |
|--------------------|-------------------------------------------------------|-----------------------------------------------------|
| `internal/config`  | Load and validate env-var configuration               | No I/O other than `os.Getenv`; never import domain packages |
| `internal/notebooklm` | Domain logic: chat, audio, sources, citations     | No HTTP handlers; use `playwright.Page` interfaces  |
| `internal/tools`   | MCP tool definitions, profiles, registry, handlers    | Definitions in `definitions.go`; handlers in separate files per tool group |
| `internal/library` | Notebook CRUD, JSON persistence                       | Pure domain model; no browser dependencies          |
| `internal/browser` | Playwright lifecycle                                  | One shared context per server instance              |
| `internal/session` | Session pool and idle-timeout management              | Thread-safe; use `sync.Mutex` or channels            |
| `internal/transport` | MCP server wire-up, stdio / HTTP transport          | Thin layer — delegates all logic to domain packages |
| `internal/stealth` | Human-like timing and input simulation                | No UI assertions; accepts `config.Config`           |
| `internal/apperrors` | Typed sentinel errors                               | Exported error types only; no wrapping logic        |

### Naming

- **Files**: `snake_case.go`
- **Exported types/functions**: `PascalCase`
- **Unexported helpers**: `camelCase`
- **Test files**: `foo_test.go` in the same package; use `package foo_test` for black-box tests
- **Constants**: group related constants under a named type (e.g., `AudioStatus`, `CitationFormat`, `ToolProfile`)

---

## How to Add a Tool

1. **Define** the tool in `internal/tools/definitions.go`:
   ```go
   func defMyNewTool() mcp.Tool {
       return mcp.NewTool("my_new_tool",
           mcp.WithDescription("One-sentence description."),
           mcp.WithString("param_name",
               mcp.Required(),
               mcp.Description("What this param does."),
           ),
       )
   }
   ```

2. **Register** it in `AllToolDefinitions()` in the same file:
   ```go
   "my_new_tool": defMyNewTool(),
   ```

3. **Add to profiles** in `internal/tools/profiles.go`:
   - `ProfileMinimal` — only for read-only, zero-side-effect tools
   - `ProfileStandard` — default; most tools belong here
   - `ProfileFull` — destructive or admin tools (e.g., `remove_notebook`, `cleanup_data`)

4. **Implement the handler** in a new or existing handler file under `internal/tools/`:
   ```go
   func handleMyNewTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
       // parse args → call domain function → return result
   }
   ```

5. **Wire** the handler in `ToolRegistry.RegisterAll()`.

6. **Write tests** for the handler (unit) and the domain function (unit + optional integration).

---

## Selector Maintenance

NotebookLM is a web app; its DOM changes without notice. All CSS selectors live in `internal/notebooklm/selectors.go` as named constants.

**When a selector breaks:**

1. Open Chrome DevTools on `notebooklm.google.com`.
2. Find the new selector for the element.
3. Update the constant in `selectors.go`.
4. If the change is locale-sensitive (e.g., a chat input aria-label), add the new locale to the appropriate slice (`ChatInputAriaLabels`, `answerPlaceholderPhrases`, etc.).
5. Add a comment explaining what the element is used for if it is not obvious.

**Do not hardcode selectors** anywhere outside `selectors.go`. Other packages must import the constant.

---

## Pull Request Checklist

Before opening a PR, verify:

- [ ] `go test -race ./...` passes locally
- [ ] New public APIs have doc comments
- [ ] Selectors changed in `selectors.go` only
- [ ] New tools are registered in `AllToolDefinitions()` and at least one profile
- [ ] `go vet ./...` reports no issues
- [ ] Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/) (`feat:`, `fix:`, `refactor:`, `test:`, `docs:`)

---

## Questions

Open an issue on GitHub. Please include:
- Go version (`go version`)
- OS and architecture
- Relevant env vars (redact credentials)
- Full error output
