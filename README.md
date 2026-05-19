# notebooklm-mcp-server

A Go MCP server that exposes Google NotebookLM as a set of structured tools for AI assistants — ask questions, manage notebooks, add sources, and generate audio overviews, all through the Model Context Protocol.

---

## Quick Start

1. **Install Playwright browsers** (required once):
   ```sh
   go run github.com/playwright-community/playwright-go/cmd/playwright install chromium
   ```

2. **Build the server**:
   ```sh
   go build ./cmd/notebooklm-mcp-server
   ```

3. **Authenticate** (opens a browser window for Google sign-in):
   ```sh
   ./notebooklm-mcp-server --transport stdio
   # Then call the setup_auth tool from your MCP client
   ```

4. **Register in your MCP client** (stdio, Claude Desktop example):
   ```json
   {
     "mcpServers": {
       "notebooklm": {
         "command": "/path/to/notebooklm-mcp-server",
         "args": ["--transport", "stdio", "--profile", "standard"]
       }
     }
   }
   ```

5. **Ask a question** using the `ask_question` tool:
   ```json
   { "question": "Summarize the key findings in my research notebook." }
   ```

---

## What It Does

| Category      | Tools                                                                 | Description                                      |
|---------------|-----------------------------------------------------------------------|--------------------------------------------------|
| **Chat**      | `ask_question`                                                        | Ask questions; returns answers with citations    |
| **Library**   | `add_notebook`, `list_notebooks`, `get_notebook`, `select_notebook`  | Manage your notebook registry                    |
| **Library**   | `update_notebook`, `remove_notebook`, `search_notebooks`, `get_library_stats` | Edit, remove, search, and stats           |
| **Sources**   | `add_source`                                                          | Add a URL or inline text to a notebook           |
| **Audio**     | `generate_audio`, `get_audio_status`, `download_audio`               | Generate and download podcast-style audio        |
| **Sessions**  | `list_sessions`, `close_session`, `reset_session`                    | Inspect and manage browser sessions              |
| **Auth**      | `setup_auth`, `re_auth`, `cleanup_data`                              | Google authentication lifecycle                  |
| **Health**    | `get_health`                                                          | Liveness check                                   |

> **Total: 20 tools** across 8 categories.

---

## Configuration

All settings are read from environment variables. Defaults apply when a variable is unset.

### Core

| Variable          | Default   | Description                                       |
|-------------------|-----------|---------------------------------------------------|
| `NOTEBOOK_URL`    | _(none)_  | Default NotebookLM URL used when none is provided |
| `HEADLESS`        | `true`    | Run Chrome headlessly (`true`/`false`)            |
| `BROWSER_TIMEOUT` | `30000`   | Page-load timeout in ms                           |
| `ANSWER_TIMEOUT_MS` | `600000` | Max time to wait for a stable answer (10 min)   |

### Sessions

| Variable           | Default | Description                          |
|--------------------|---------|--------------------------------------|
| `MAX_SESSIONS`     | `10`    | Maximum concurrent browser sessions  |
| `SESSION_TIMEOUT`  | `900`   | Session idle timeout in seconds       |

### Auto-login

| Variable               | Default | Description                                   |
|------------------------|---------|-----------------------------------------------|
| `AUTO_LOGIN_ENABLED`   | `false` | Enable automatic Google sign-in               |
| `LOGIN_EMAIL`          | _(none)_ | Google account email for auto-login          |
| `LOGIN_PASSWORD`       | _(none)_ | Google account password for auto-login       |
| `AUTO_LOGIN_TIMEOUT_MS`| `120000` | Timeout for the auto-login flow (2 min)      |

> **Security note:** prefer `setup_auth` (interactive) over `AUTO_LOGIN_ENABLED` in production. Credentials stored in env vars are readable by any process.

### Stealth

Stealth mode makes browser interactions resemble human behaviour to reduce automation detection.

| Variable                | Default | Description                              |
|-------------------------|---------|------------------------------------------|
| `STEALTH_ENABLED`       | `true`  | Enable all stealth features              |
| `STEALTH_RANDOM_DELAYS` | `true`  | Insert random delays between actions     |
| `STEALTH_HUMAN_TYPING`  | `true`  | Type at human-like WPM instead of fill  |
| `STEALTH_MOUSE_MOVEMENTS` | `true` | Simulate natural mouse paths            |
| `TYPING_WPM_MIN`        | `160`   | Minimum typing speed (words per minute)  |
| `TYPING_WPM_MAX`        | `240`   | Maximum typing speed (words per minute)  |
| `MIN_DELAY_MS`          | `100`   | Minimum inter-action delay in ms         |
| `MAX_DELAY_MS`          | `400`   | Maximum inter-action delay in ms         |

### Library Defaults

These values are used as defaults when `add_notebook` fields are omitted.

| Variable                  | Default                               |
|---------------------------|---------------------------------------|
| `NOTEBOOK_DESCRIPTION`    | `"General knowledge base"`            |
| `NOTEBOOK_TOPICS`         | `"General topics"` (comma-separated)  |
| `NOTEBOOK_CONTENT_TYPES`  | `"documentation,examples"`            |
| `NOTEBOOK_USE_CASES`      | `"General research"`                  |

### Chrome Profile Strategy

| Variable                       | Default  | Description                                          |
|--------------------------------|----------|------------------------------------------------------|
| `NOTEBOOK_PROFILE_STRATEGY`    | `auto`   | `auto`, `isolated`, or `single`                      |
| `NOTEBOOK_CLONE_PROFILE`       | `false`  | Clone the base profile for each isolated instance    |
| `NOTEBOOK_CLEANUP_ON_STARTUP`  | `true`   | Remove stale profile instances at startup            |
| `NOTEBOOK_CLEANUP_ON_SHUTDOWN` | `true`   | Remove profile instances at clean shutdown           |
| `NOTEBOOK_INSTANCE_TTL_HOURS`  | `72`     | Hours before an instance profile is considered stale |
| `NOTEBOOK_INSTANCE_MAX_COUNT`  | `20`     | Maximum number of live instance profiles             |

**Profile strategies:**

| Strategy   | Behaviour                                                        |
|------------|------------------------------------------------------------------|
| `auto`     | Re-uses a single shared Chrome profile; default for most setups  |
| `single`   | Always uses one fixed profile regardless of sessions             |
| `isolated` | Creates a fresh profile clone per session (highest isolation)    |

### Data Directories (platform defaults)

| Platform | Path                                             |
|----------|--------------------------------------------------|
| Linux    | `~/.local/share/notebooklm-mcp/`                 |
| macOS    | `~/Library/Application Support/notebooklm-mcp/` |
| Windows  | `%APPDATA%\notebooklm-mcp\`                      |

---

## Tool Profiles

Use `--profile` to limit which tools are exposed to the MCP client.

| Profile    | Tools included                                                                                              | Use Case                              |
|------------|-------------------------------------------------------------------------------------------------------------|---------------------------------------|
| `minimal`  | `ask_question`, `get_health`, `list_notebooks`, `get_notebook`                                              | Read-only assistant                   |
| `standard` | All minimal tools + `select_notebook`, `update_notebook`, `search_notebooks`, `get_library_stats`, `list_sessions`, `close_session`, `reset_session`, `setup_auth`, `add_source`, `generate_audio`, `get_audio_status`, `download_audio` | Everyday use (default) |
| `full`     | All standard tools + `add_notebook`, `remove_notebook`, `re_auth`, `cleanup_data`                          | Admin / CI pipelines                  |

Unknown profile values fall back to `full`.

---

## CLI

```
Usage: notebooklm-mcp-server [flags]
       notebooklm-mcp-server config <command> [args]

Flags:
  -transport string       Transport mode: stdio or http (default "stdio")
  -port int               HTTP port — only used with http transport (default 3000)
  -host string            HTTP bind address — only used with http transport (default "127.0.0.1")
  -account string         Account name for profile isolation (creates a subdirectory)
  -profile string         Tool profile: minimal, standard, or full (default "standard")
  -disabled-tools string  Comma-separated tool names to disable at runtime

Config subcommands:
  config get <key>          Print the stored value of a config key
  config set <key> <value>  Write a value to settings.json
  config reset              Delete settings.json and restore defaults
```

**Recognized keys for `config get/set`:** `headless`, `stealth_enabled`, `data_dir`, `notebook_url`, `profile_strategy`.

### HTTP Transport

When `--transport http` is used, the server exposes:

| Endpoint     | Method            | Description                          |
|--------------|-------------------|--------------------------------------|
| `/mcp`       | POST / GET / DELETE | Streamable HTTP MCP endpoint       |
| `/healthz`   | GET               | Liveness check (JSON)                |

---

## Architecture

```
cmd/
  notebooklm-mcp-server/   # main — wires all components, parses CLI flags

internal/
  config/        # Config struct, Load(), env-var overrides, BrowserOptions
  auth/          # Google auth state persistence and validation
  browser/       # Playwright browser lifecycle and shared context manager
  session/       # Session pool: create, reuse, idle-timeout, close
  notebooklm/    # Domain logic: chat, sources, audio, citations, selectors
  tools/         # MCP tool definitions (20 tools), profiles, registry, handlers
  library/       # Notebook library: CRUD, search, stats, JSON persistence
  resources/     # MCP resource registrations (notebook URIs)
  transport/     # MCP server transport: stdio and Streamable HTTP
  stealth/       # Human-like typing, mouse movements, random delays
  apperrors/     # Typed errors: rate limit, auth, timeout
  utils/         # Structured logger
```

**Dependency order at startup:**

```
config → library → browser → auth → session → notebooklm → tools → resources → transport
```

**Answer stability algorithm** (`internal/notebooklm/chat.go`):  
The server polls the page every 750 ms and requires **3 consecutive identical reads** before treating an answer as stable. Placeholder phrases (40+ across 8 languages) and rate-limit messages are filtered before counting.

**Citation extraction** (`internal/notebooklm/citations.go`):  
Four output formats are supported: `none`, `inline`, `footnotes`, `json`.

**CSS selectors** (`internal/notebooklm/selectors.go`):  
All UI selectors are defined as named constants. When the primary `textarea.query-box-input` selector fails, 10 locale-specific `aria-label` fallbacks are tried in order.

---

## Requirements

| Requirement         | Version / Notes                                       |
|---------------------|-------------------------------------------------------|
| Go                  | 1.26+                                                 |
| Playwright browsers | Chromium (install with `playwright install chromium`) |
| Google account      | Required for NotebookLM access                        |
| NotebookLM access   | Free or Plus account at notebooklm.google.com         |

---

## License

MIT
