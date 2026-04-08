# Decoupled Harness Implementation: Script-Based Provisioning

## Motivation

Today, every harness implementation lives as compiled Go code inside the scion binary (`pkg/harness/`). Each harness performs a similar set of operations — writing config files, injecting auth credentials, rewriting settings JSON/YAML/TOML — but the specifics are unique per harness. This means:

1. **Adding a new harness requires Go expertise and a fork/PR** — even though the harness logic is mostly file manipulation and string templating, not systems programming.
2. **The binary grows with every harness** — Claude, Gemini, OpenCode, Codex each add ~300-500 lines of Go code for what is essentially JSON/YAML/TOML rewriting.
3. **The plugin system (`pkg/plugin/`) is heavyweight** — hashicorp/go-plugin requires building a separate Go binary, RPC serialization, and process lifecycle management. This is appropriate for long-running services (message brokers) but overkill for file-writing provisioning logic.
4. **Harness images already ship a Python interpreter** — every scion container image includes Python for `sciontool` and utility scripts. A harness provisioning script would have zero additional dependencies.

The core insight: **harness provisioning is inherently a scripting problem, not a systems programming problem.** The operations are: read config, write files, template strings, move things into place. This is what scripting languages excel at.

## Proposal

Replace the compiled Go harness implementations with **Python provisioning scripts** that live inside each harness-config directory. The scion binary provides a stable, well-defined interface (environment variables and a JSON manifest), invokes the script, and the script handles all harness-specific file manipulation.

### What Moves Out of Go

| Current Go Method | Replacement |
|---|---|
| `Provision()` | `provision.py provision` — writes harness-specific config files |
| `InjectAgentInstructions()` | `provision.py inject-instructions` — writes to harness-specific location |
| `InjectSystemPrompt()` | `provision.py inject-prompt` — writes to harness-specific location |
| `ResolveAuth()` | `provision.py resolve-auth` — selects auth method, returns env vars and file mappings |
| `ApplyAuthSettings()` | `provision.py apply-auth` — updates native harness config with resolved auth |
| `ApplyTelemetrySettings()` | `provision.py apply-telemetry` — writes telemetry config (e.g., Codex TOML) |

### What Stays in Go

| Concern | Why it stays |
|---|---|
| `Name()`, `DefaultConfigDir()`, `SkillsDir()`, `GetInterruptKey()`, `GetEmbedDir()` | Static metadata — declared in `config.yaml`, not logic |
| `GetCommand()` | Simple command construction — declarable in `config.yaml` |
| `GetEnv()` | Simple env var mapping — declarable in `config.yaml` |
| `AdvancedCapabilities()` | Capability advertisement — declarable in `config.yaml` |
| `HasSystemPrompt()` | Simple file existence check — can be derived from config |
| Auth gathering, validation, overlay | Cross-cutting concern shared by all harnesses (`auth.go`) |
| Container launch, volume mounting, image resolution | Runtime layer (`pkg/runtime/`, `pkg/agent/run.go`) |
| Template/harness-config loading and merging | Config layer (`pkg/config/`) |

## Design

### Harness-Config Directory Structure (Extended)

```
~/.scion/harness-configs/claude/
  config.yaml              # Declarative metadata (existing, extended)
  provision.py             # Provisioning script (NEW)
  home/                    # Base home directory files (existing)
    .bashrc
    .claude/
      settings.json
```

### Extended `config.yaml` Schema

The existing `config.yaml` fields are preserved. New fields capture metadata that is currently returned by Go methods:

```yaml
# Existing fields
harness: claude
image: scion-claude:latest
user: scion

# New declarative metadata (replaces simple Go getters)
config_dir: .claude                # DefaultConfigDir()
skills_dir: .claude/skills         # SkillsDir()
interrupt_key: Escape              # GetInterruptKey()
instructions_file: .claude/CLAUDE.md   # Target for InjectAgentInstructions()
system_prompt_file: .claude/system-prompt.md  # Target for InjectSystemPrompt()

# Command construction (replaces GetCommand())
command:
  base: ["claude", "--no-chrome", "--dangerously-skip-permissions"]
  resume_flag: "--continue"
  task_flag: "--message"           # or null if task is positional
  system_prompt_flag: "--system-prompt"

# Environment variables (replaces GetEnv())
env_template:
  CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC: "1"

# Capability advertisement (replaces AdvancedCapabilities())
capabilities:
  max_turns: true
  max_model_calls: false
  max_duration: true
  system_prompt: native            # native | downgrade | none
  agent_instructions: true
  auth:
    api_key: true
    auth_file: false
    vertex_ai: true
  telemetry:
    enabled_config: true
    native_emitter: true

# Hook dialect specification (from harness-plugin-challenges.md Tier 1)
dialect:
  event_name_field: event_type
  mappings:
    tool_use:
      event: tool-start
      fields:
        tool_name: .tool_name
    # ...
```

With this extended config, a **thin Go "script harness"** can implement the `api.Harness` interface purely from `config.yaml` data, delegating only the complex operations (provisioning, auth resolution, settings rewriting) to `provision.py`.

### Script Interface

The provisioning script receives context via a JSON manifest on **stdin** and returns results on **stdout** (for commands that produce output). Errors are signaled via non-zero exit code and stderr.

#### Commands

**`provision`** — Called during agent creation (replaces `Harness.Provision()`)

```bash
echo '$MANIFEST' | python3 provision.py provision
```

Manifest:
```json
{
  "command": "provision",
  "agent_name": "researcher",
  "agent_dir": "/path/to/.scion/agents/researcher",
  "agent_home": "/path/to/.scion/agents/researcher/home",
  "agent_workspace": "/path/to/.scion/agents/researcher/workspace",
  "harness_config": { /* full HarnessConfigEntry */ },
  "scion_config": { /* full ScionConfig from scion-agent.json */ }
}
```

The script performs whatever file operations are needed. For Claude, this means updating `.claude.json` with workspace paths.

**`inject-instructions`** — Write agent instructions to harness-specific location

```bash
echo '$MANIFEST' | python3 provision.py inject-instructions
```

Manifest:
```json
{
  "command": "inject-instructions",
  "agent_home": "/path/to/home",
  "content": "Instructions content here...",
  "target_file": ".claude/CLAUDE.md"
}
```

Note: `target_file` comes from `config.yaml` so most scripts can use a generic "write content to target" implementation. Custom scripts can override for harness-specific behavior (e.g., Claude's casing cleanup).

**`inject-prompt`** — Write system prompt to harness-specific location

Same pattern as `inject-instructions` with `target_file` from config.

**`resolve-auth`** — Select authentication method from available credentials

```bash
echo '$MANIFEST' | python3 provision.py resolve-auth
```

Manifest:
```json
{
  "command": "resolve-auth",
  "auth_config": {
    "explicit_type": "api-key",
    "anthropic_api_key": "sk-...",
    "google_cloud_project": "",
    "google_cloud_region": "",
    "adc_path": "",
    "oauth_creds_path": ""
  }
}
```

Response (stdout):
```json
{
  "method": "api-key",
  "env_vars": {
    "ANTHROPIC_API_KEY": "sk-..."
  },
  "file_mappings": {}
}
```

**`apply-auth`** — Update harness-native config files after auth resolution

```bash
echo '$MANIFEST' | python3 provision.py apply-auth
```

Manifest includes `agent_home` and `resolved_auth`. Script modifies settings files in-place (e.g., Claude writes API key fingerprint to `.claude.json`, Gemini writes `selectedType` to `settings.json`).

**`apply-telemetry`** — Reconcile telemetry configuration

```bash
echo '$MANIFEST' | python3 provision.py apply-telemetry
```

Used by harnesses like Codex that have native OTEL configuration files (writes `[otel]` section to `config.toml`).

#### Error Handling

- Exit code 0: success
- Exit code 1: error — stderr is captured and surfaced to user
- Exit code 2: unsupported command — harness does not implement this operation (equivalent to a no-op)

### Go-Side Implementation: `ScriptHarness`

A single Go struct replaces all individual harness implementations:

```go
// pkg/harness/script_harness.go

type ScriptHarness struct {
    config     api.HarnessConfigEntry  // Parsed config.yaml
    scriptPath string                   // Path to provision.py
    configDir  string                   // Path to harness-config directory
}

func (h *ScriptHarness) Name() string {
    return h.config.Harness
}

func (h *ScriptHarness) DefaultConfigDir() string {
    return h.config.ConfigDir  // New field from extended config.yaml
}

func (h *ScriptHarness) GetCommand(task string, resume bool, baseArgs []string) []string {
    // Build from config.yaml command spec
    cmd := append([]string{}, h.config.Command.Base...)
    if resume && h.config.Command.ResumeFlag != "" {
        cmd = append(cmd, h.config.Command.ResumeFlag)
    }
    // ... task flag handling
    return cmd
}

func (h *ScriptHarness) Provision(ctx context.Context, agentName, agentDir, agentHome, agentWorkspace string) error {
    manifest := ProvisionManifest{
        Command:       "provision",
        AgentName:     agentName,
        AgentDir:      agentDir,
        AgentHome:     agentHome,
        AgentWorkspace: agentWorkspace,
    }
    return h.runScript(ctx, manifest)
}

func (h *ScriptHarness) ResolveAuth(auth api.AuthConfig) (*api.ResolvedAuth, error) {
    manifest := AuthManifest{Command: "resolve-auth", AuthConfig: auth}
    out, err := h.runScriptWithOutput(context.Background(), manifest)
    if err != nil { return nil, err }
    var resolved api.ResolvedAuth
    return &resolved, json.Unmarshal(out, &resolved)
}

func (h *ScriptHarness) runScript(ctx context.Context, manifest interface{}) error {
    data, _ := json.Marshal(manifest)
    cmd := exec.CommandContext(ctx, "python3", h.scriptPath, manifest.Command)
    cmd.Stdin = bytes.NewReader(data)
    cmd.Stderr = &stderrBuf
    return cmd.Run()
}
```

### Updated Harness Factory

```go
func New(harnessName string) api.Harness {
    // 1. Check for script-based harness config with provision.py
    if hcDir, err := config.FindHarnessConfigDir(harnessName, ...); err == nil {
        scriptPath := filepath.Join(hcDir.Path, "provision.py")
        if _, err := os.Stat(scriptPath); err == nil {
            return &ScriptHarness{
                config:     hcDir.Config,
                scriptPath: scriptPath,
                configDir:  hcDir.Path,
            }
        }
    }

    // 2. Check built-in harnesses (during migration)
    switch harnessName {
    case "claude": return &ClaudeCode{}
    // ...
    }

    // 3. Check go-plugin registry
    if pluginManager != nil && pluginManager.HasPlugin(PluginTypeHarness, harnessName) {
        h, _ := pluginManager.GetHarness(harnessName)
        return h
    }

    // 4. Fall back to generic (config.yaml-only, no script)
    return &Generic{}
}
```

### Where Scripts Execute

Scripts execute on the **host** (or broker) during provisioning, **not inside the container**. This is the same execution context as the current Go harness code. The script has direct filesystem access to the agent home directory being composed.

An alternative considered was running scripts inside the container via `sciontool init`, but this was rejected (see Alternatives section).

## Interaction with Existing Plugin System

The script-based approach and go-plugin approach serve different needs:

| Dimension | Script Harness | Go-Plugin Harness |
|---|---|---|
| **Language** | Python (or any interpreter) | Go (or any language via gRPC) |
| **Complexity** | File manipulation, config rewriting | Complex logic, external service integration |
| **Distribution** | Drop files in harness-config directory | Build and install a binary |
| **Process model** | Subprocess per invocation | Long-running RPC server |
| **When to use** | 90% of harnesses — "my CLI needs these config files" | Edge cases — auth flows requiring OAuth dance, custom API calls |

The go-plugin system remains available for cases that genuinely need compiled code or long-running processes. Script harnesses are the **recommended default** for new harnesses.

### Priority Order in `harness.New()`

1. **Script harness** — harness-config directory has `provision.py`
2. **Built-in Go harness** — compiled into binary (during migration; eventually removed)
3. **Go-plugin harness** — discovered via plugin manager
4. **Generic** — config.yaml-only, no provisioning logic

## Relationship to Sideloading

Scion already supports **binary sideloading** via `SCION_DEV_BINARIES` — mounting local binaries into the container at `/opt/scion/bin`. Combined with script-based harnesses, this enables a fully external harness workflow:

1. **Container image**: Use a generic base image (e.g., `scion-base`) or any image with Python + the target CLI
2. **Harness logic**: Provide `config.yaml` + `provision.py` in a harness-config directory
3. **Hook dialect**: Provide `dialect.yaml` for event normalization (per harness-plugin-challenges.md Tier 1)
4. **CLI binary**: Sideload the harness CLI binary into the container

This means a community contributor can add support for a new coding agent (e.g., Cursor, Aider, Continue) without:
- Writing any Go code
- Building a custom container image
- Understanding the scion plugin RPC system
- Forking the scion repository

## Alternatives Considered

### Alternative A: Execute Scripts Inside Container via `sciontool init`

Instead of running `provision.py` on the host during provisioning, run it inside the container as part of `sciontool init`'s startup sequence.

**Pros:**
- Script has access to the exact runtime environment (same Python, same paths)
- No host Python dependency
- Could handle runtime-only configuration (things that depend on container state)

**Cons:**
- Provisioning currently runs **before** the container starts — the agent home is composed on the host and then mounted. Moving this into the container changes the execution model significantly.
- `sciontool init` already has a complex startup sequence (UID setup, git clone, services, hooks, heartbeat). Adding provisioning logic increases fragility.
- Error handling is harder — a provisioning failure inside the container requires the container to exit, be inspected, and restarted.
- Some provisioning steps (like auth resolution) feed back into the `RunConfig` that launches the container. Chicken-and-egg: you can't run the script inside the container if the container's config depends on the script's output.

**Verdict:** Rejected for primary provisioning. However, a lightweight **post-start hook** (see Phase 3) could handle runtime-specific adjustments that genuinely need the container environment.

### Alternative B: Use Starlark Instead of Python

[Starlark](https://github.com/google/starlark-go) is a Python-like language embeddable in Go. Scripts would execute in-process with controlled capabilities.

**Pros:**
- No external Python dependency
- Sandboxed execution — script cannot access network, arbitrary filesystem, etc.
- Deterministic — no version skew between Python installations

**Cons:**
- Starlark is a restricted subset of Python — no `import`, no standard library, no `json` module without explicit host injection
- Writing a Starlark harness would require learning a new (albeit similar) language
- We'd need to expose filesystem operations, JSON parsing, TOML writing, etc. as Starlark built-in functions — essentially building a scripting SDK
- The scion container images already ship Python. Host-side execution is the only case where Python might not be available, and that's solvable by documenting it as a requirement or falling back to built-in Go harnesses.

**Verdict:** Deferred. If host-side Python proves problematic, Starlark is a viable fallback. The script interface (JSON stdin/stdout) is language-agnostic, so switching interpreters later is straightforward.

### Alternative C: Declarative-Only (No Scripts)

Extend `config.yaml` to be fully declarative — express all provisioning as templated file writes:

```yaml
provision:
  files:
    - target: .claude.json
      template: |
        { "projects": { "{{ .AgentWorkspace }}": {} } }
    - target: .claude/CLAUDE.md
      content_from: instructions
```

**Pros:**
- No external interpreter needed
- Easy to validate and test
- Configuration-as-code

**Cons:**
- Some provisioning logic is genuinely procedural — Claude's `.claude.json` merges with existing content, Codex's TOML rewriting removes and rebuilds sections, Gemini's settings update nested JSON paths
- A sufficiently powerful template language becomes Turing-complete (see: Helm charts)
- Auth resolution involves conditional logic that doesn't map cleanly to templates

**Verdict:** Partially adopted. Simple file writes (instructions, system prompt) can be declarative via `config.yaml` fields (`instructions_file`, `system_prompt_file`). Complex provisioning remains scripted. The `ScriptHarness` Go implementation uses declarative config for simple operations and delegates to `provision.py` only when the script exists and the operation requires it.

### Alternative D: Keep Everything in Go, Improve Plugin Authoring

Double down on the go-plugin approach: improve the reference harness, add scaffolding, make it easier to write Go plugins.

**Pros:**
- Single technology stack
- Type safety across the boundary
- Existing infrastructure

**Cons:**
- Go expertise is a hard requirement for harness authors
- Plugin binaries must be compiled for the target platform
- The go-plugin RPC overhead is unnecessary for file-writing operations
- Most harness logic is 50-100 lines of "read JSON, modify field, write JSON" — this is scripting, not systems programming

**Verdict:** Go-plugin remains available for complex cases but is not the recommended path for typical harnesses.

## Open Questions

### Q1: Host Python Dependency

Script harnesses require Python 3 on the host (or broker) machine. Is this acceptable?

- **macOS**: Ships with Python 3 (or easily installed via Homebrew)
- **Linux servers/brokers**: Almost universally available
- **Container environments**: Always available (scion images include Python)
- **Mitigation**: Built-in Go harnesses remain available as fallback. The script approach is opt-in per harness-config.

**Recommendation:** Document Python 3.8+ as a soft requirement for script-based harnesses. Fall back to built-in Go harness if Python is unavailable and a built-in exists.

### Q2: Script Versioning and Compatibility

How do we handle changes to the manifest schema?

- **Option A**: Version field in manifest (`"schema_version": "1"`), scripts check and fail gracefully
- **Option B**: Semantic versioning of the script interface, scripts declare compatibility
- **Option C**: Keep the manifest additive-only — new fields are optional, old scripts ignore them

**Recommendation:** Option C (additive-only) for simplicity. Include a `schema_version` field for future-proofing but don't enforce it initially.

### Q3: Script Testing

How do harness script authors test their `provision.py`?

- The JSON manifest format is self-contained — scripts can be tested with fixture JSON files
- A `scion harness test <name>` command could scaffold a temporary agent directory and run the script against it
- Integration tests can run the script in a temporary directory and verify the output files

### Q4: Migration Path for Built-in Harnesses

Should we migrate existing built-in harnesses (Claude, Gemini, etc.) to script-based implementations?

- **Yes**: Proves the approach, reduces binary size, simplifies maintenance
- **No**: Built-in harnesses work fine; only use scripts for new harnesses
- **Hybrid**: Migrate one harness (e.g., OpenCode, which has the simplest Provision) as a proof of concept, then evaluate

**Recommendation:** Hybrid. Migrate OpenCode first (no Provision logic, simple auth), then Codex (moderate complexity), then evaluate whether Claude and Gemini benefit enough to justify migration.

### Q5: Script Execution Security

Scripts execute with the same privileges as the scion binary. Is additional sandboxing needed?

- Scripts are installed by the user (or admin) into `~/.scion/harness-configs/`
- Same trust model as go-plugin binaries or any user-installed software
- Container-side execution would provide isolation but has the chicken-and-egg problems described in Alternative A

**Recommendation:** No additional sandboxing for v1. Same trust model as the existing plugin system.

### Q6: Declarative Fallback for Simple Harnesses

Some harnesses (like Generic) have no provisioning logic at all. Should the `ScriptHarness` handle the fully-declarative case (no `provision.py`) as well?

**Recommendation:** Yes. If `config.yaml` provides all necessary metadata and no `provision.py` exists, `ScriptHarness` should handle simple operations (instruction/prompt injection) purely from config. This subsumes the current `Generic` harness.

### Q7: Sciontool Integration

`sciontool init` currently doesn't call any harness provisioning code — that happens before the container starts. Should `sciontool` gain awareness of script harnesses for post-start adjustments?

**Recommendation:** Defer. The existing hook system (`sciontool hook`) already supports post-start lifecycle events. If a harness needs runtime adjustments, it can use hooks. Adding a second provisioning path inside the container increases complexity.

## Implementation Plan

### Phase 1: Extended Config Schema and ScriptHarness Skeleton

**Goal:** Ship the `ScriptHarness` Go implementation that reads from `config.yaml` and can delegate to `provision.py`.

1. Extend `HarnessConfigEntry` in `pkg/api/types.go` (or `settings_v1.go`) with new declarative fields: `config_dir`, `skills_dir`, `interrupt_key`, `instructions_file`, `system_prompt_file`, `command`, `capabilities`
2. Implement `ScriptHarness` in `pkg/harness/script_harness.go`:
   - All simple getters read from extended `config.yaml`
   - `Provision()`, `ResolveAuth()`, `ApplyAuthSettings()`, `ApplyTelemetrySettings()` delegate to `provision.py` via subprocess
   - `InjectAgentInstructions()` and `InjectSystemPrompt()` use declarative config for the simple case (write content to `target_file`), delegate to script for complex cases
3. Update harness factory to check for `provision.py` before built-in harness lookup
4. Write comprehensive tests for `ScriptHarness` with mock scripts
5. Update `SeedHarnessConfig()` to copy `provision.py` alongside existing embed files

### Phase 2: Migrate OpenCode as Proof of Concept

**Goal:** Validate the script approach with the simplest existing harness.

1. Write `provision.py` for OpenCode harness (auth resolution only — no Provision logic)
2. Write extended `config.yaml` for OpenCode with all declarative fields
3. Remove `pkg/harness/opencode.go` and update factory
4. Run full test suite, fix any gaps in the script interface
5. Document the script authoring experience — what was easy, what was painful

### Phase 3: Migrate Codex

**Goal:** Validate with a harness that has real provisioning complexity (TOML rewriting, auth file writing).

1. Write `provision.py` for Codex covering: `provision` (TOML config), `resolve-auth`, `apply-auth` (auth.json), `apply-telemetry` (TOML otel section)
2. Extended `config.yaml` for Codex
3. Remove `pkg/harness/codex.go` and `codex_config.go`
4. Verify telemetry reconciliation works correctly through the script interface

### Phase 4: Evaluate Claude and Gemini Migration

**Goal:** Decide whether to migrate the most complex harnesses.

1. Assess complexity of Claude's `.claude.json` manipulation and auth resolution
2. Assess complexity of Gemini's nested `settings.json` updates and multi-method auth
3. If complexity is manageable, migrate; if the Go code is cleaner, keep it
4. Document the decision criteria for future harness authors

### Phase 5: Community Harness Template and Documentation

**Goal:** Make it easy for external contributors to add harnesses.

1. Create a template harness-config directory with annotated `config.yaml` and `provision.py`
2. Add `scion harness init <name>` command that scaffolds from the template
3. Add `scion harness test <name>` command for local testing
4. Write contributor documentation: "Adding a New Harness to Scion"
5. Integrate with declarative dialect spec (from harness-plugin-challenges.md Tier 1) so script harnesses can also declare their hook event mapping

## Summary

Script-based harness provisioning extracts the "file manipulation" concern out of compiled Go and into Python scripts that live alongside harness configuration. This preserves the existing architecture (host-side provisioning, container launch, sciontool supervision) while dramatically lowering the barrier to adding new harness support. The approach is additive — built-in Go harnesses and go-plugin harnesses continue to work — and can be adopted incrementally, one harness at a time.
