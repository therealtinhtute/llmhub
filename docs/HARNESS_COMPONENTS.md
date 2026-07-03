# Harness Components

This taxonomy maps the Harness surface currently installed in `llmhub`.

It uses two component frameworks:

- Runtime Substrate responsibilities: the 11 responsibility areas the harness
  should cover.
- NexAU decomposition: the implementation surfaces that influence agent
  behavior.

Status values:

- **Covered**: the repository has an explicit file, command, or durable record
  for this responsibility.
- **Partial**: the repository has some support, but the support is incomplete,
  manual, or not yet measured.
- **Missing**: no meaningful support exists yet.

## Responsibility Map

| # | Responsibility | Status | Harness Surface | Evidence | Gap |
| --- | --- | --- | --- | --- | --- |
| 1 | Task specification | Covered | `AGENTS.md`, `docs/FEATURE_INTAKE.md`, `docs/templates/story.md`, `docs/templates/spec-intake.md`, `docs/templates/high-risk-story/*`, `docs/stories/*`, durable intake/story records | Requests are classified by type and lane before implementation; normal and high-risk work have templates and durable story rows. | Keep story packets synchronized with product docs. |
| 2 | Context selection | Covered | `AGENTS.md`, `docs/CONTEXT_RULES.md`, `docs/ARCHITECTURE.md`, `docs/decisions/*`, `docs/product/README.md`, `scripts/bin/harness-cli score-context` | Context rules define phase-by-lane reads and retrieval triggers; context scoring compares recorded trace reads against those rules. | Future automation could enforce context selection instead of only measuring it. |
| 3 | Tool access | Covered | `scripts/bin/harness-cli`, `docs/TOOL_REGISTRY.md`, `scripts/README.md`, `scripts/schema/003-tool-registry.sql`, `scripts/schema/005-tool-extensions.sql`, durable tool records | The Harness CLI exposes a machine-readable tool manifest through `query tools`; external tools can be registered, checked, queried, and removed. | Permission profiles and usage analytics remain future work. |
| 4 | Project memory | Covered | `docs/HARNESS.md`, `docs/decisions/*`, `docs/GLOSSARY.md`, `docs/HARNESS_BACKLOG.md`, `docs/stories/*`, durable decision/backlog/trace records | Decisions, backlog, stories, and traces preserve durable knowledge across tasks. | Future work should summarize old traces. |
| 5 | Task state | Covered | `scripts/bin/harness-cli query matrix`, `docs/TEST_MATRIX.md`, durable intake/story/trace records | Durable records track intake, story status, proof columns, and task traces. | Add lifecycle checks so in-progress stories cannot be forgotten. |
| 6 | Observability | Partial | `docs/TRACE_SPEC.md`, `scripts/bin/harness-cli trace`, `scripts/bin/harness-cli score-trace`, `scripts/bin/harness-cli query traces`, `scripts/bin/harness-cli query friction`, `docs/HARNESS_MATURITY.md` | Traces are scored when recorded, can be rescored by command, and can be reviewed with friction context. | No dashboard or benchmark ingestion exists in this repo. |
| 7 | Failure attribution | Partial | `docs/HARNESS_COMPONENTS.md`, `docs/TRACE_SPEC.md`, `docs/HARNESS_BACKLOG.md`, `scripts/bin/harness-cli query friction`, trace friction/error fields | Failures can be tied to files, components, friction, backlog proposals, and linked intake context. | No automated attribution from benchmark failures to harness components exists yet. |
| 8 | Verification | Covered | `docs/TEST_MATRIX.md`, `docs/templates/validation-report.md`, `scripts/bin/harness-cli query matrix`, `scripts/bin/harness-cli story verify`, `scripts/bin/harness-cli story verify-all`, `scripts/bin/harness-cli trace`, `scripts/bin/harness-cli score-trace`, `.github/workflows/release.yml` | Stories can store and run proof commands individually or in batch, traces warn when linked story verification has not passed, and trace quality can be checked mechanically. | Benchmark ingestion remains future work. |
| 9 | Permissions | Partial | `AGENTS.md`, `docs/HARNESS.md`, `docs/FEATURE_INTAKE.md`, `docs/ARCHITECTURE.md`, `scripts/install.sh`, `scripts/install-local.sh` | Policy describes when agents may update docs and when to ask before architecture or workflow changes; installer scripts encode operational safety. | Permissions are instruction-level only; no enforced policy layer or command allowlist exists. |
| 10 | Entropy auditing | Covered | `docs/HARNESS_BACKLOG.md`, `docs/HARNESS_AUDIT.md`, `docs/IMPROVEMENT_PROTOCOL.md`, `docs/HARNESS_MATURITY.md`, `scripts/bin/harness-cli audit`, `scripts/bin/harness-cli propose`, durable backlog/friction records | Growth rule captures friction, audit detects drift and entropy score, backlog items compare predicted impact to actual outcome, and proposal generation can create reviewable backlog items. | Automated repair remains future work. |
| 11 | Intervention recording | Covered | `scripts/schema/004-intervention.sql`, `scripts/bin/harness-cli intervention add`, `scripts/bin/harness-cli query interventions`, `docs/TRACE_SPEC.md`, `docs/decisions/*`, `docs/stories/*`, durable intervention records | Human, reviewer, CI, and agent interventions are separate durable records and can be filtered by trace, story, or type. | Capture is still manual and advisory. |

## NexAU Cross-Reference

| Component | Harness Equivalent | Status | Notes |
| --- | --- | --- | --- |
| System prompts | `AGENTS.md` plus Harness policy docs | Covered | `AGENTS.md` is the stable shim; `docs/HARNESS.md`, `docs/FEATURE_INTAKE.md`, and `docs/CONTEXT_RULES.md` carry evolving operating instructions. |
| Tool descriptions | `docs/TOOL_REGISTRY.md`, `scripts/README.md`, `docs/HARNESS.md`, `docs/TRACE_SPEC.md`, `scripts/bin/harness-cli query tools` | Covered | Commands are documented in a standalone registry and exposed as compiled plus registered tool manifest entries. |
| Tool implementations | `scripts/bin/harness-cli`, `scripts/schema/*` | Covered | The Rust CLI binary is the primary durable-layer implementation and stable repo-local entrypoint. |
| Middleware | Feature intake workflow and installer safety logic | Partial | The intake process and installers mediate work, but there is no runtime middleware enforcing policies. |
| Skills | `docs/templates/*`, `docs/FEATURE_INTAKE.md`, `docs/CONTEXT_RULES.md`, `docs/TRACE_SPEC.md` | Partial | Reusable procedures exist as markdown, not executable or installable agent skills. |
| Sub-agents | None in this repository | Missing | No delegated specialist agents or sub-agent protocols exist. |
| Long-term memory | Harness database, `docs/decisions/*`, `docs/stories/*`, `docs/HARNESS_BACKLOG.md`, `docs/GLOSSARY.md` | Covered | Durable records and markdown decisions preserve task history and project vocabulary. |

## File Inventory

Representative tracked Harness files are mapped to Runtime Substrate
responsibilities.

| File | Primary Responsibility | Secondary Responsibilities |
| --- | --- | --- |
| `AGENTS.md` | Context selection | Task specification, permissions |
| `README.md` | Task specification | Project memory |
| `.github/workflows/release.yml` | Verification | Tool access |
| `docs/ARCHITECTURE.md` | Permissions | Context selection, task specification |
| `docs/CONTEXT_RULES.md` | Context selection | Permissions, task specification |
| `docs/FEATURE_INTAKE.md` | Task specification | Permissions, context selection |
| `docs/GLOSSARY.md` | Project memory | Context selection |
| `docs/HARNESS.md` | Task specification | Project memory, task state, permissions |
| `docs/HARNESS_AUDIT.md` | Entropy auditing | Verification, task state |
| `docs/HARNESS_BACKLOG.md` | Entropy auditing | Project memory, failure attribution |
| `docs/HARNESS_COMPONENTS.md` | Failure attribution | Observability, entropy auditing |
| `docs/HARNESS_MATURITY.md` | Entropy auditing | Observability, verification |
| `docs/IMPROVEMENT_PROTOCOL.md` | Entropy auditing | Failure attribution, permissions |
| `docs/README.md` | Project memory | Context selection |
| `docs/TEST_MATRIX.md` | Verification | Task state |
| `docs/TOOL_REGISTRY.md` | Tool access | Context selection, verification |
| `docs/TRACE_SPEC.md` | Observability | Failure attribution, intervention recording |
| `docs/decisions/*` | Project memory | Task specification, permissions, verification |
| `docs/product/README.md` | Task specification | Project memory |
| `docs/stories/*` | Task specification | Project memory, verification |
| `docs/stories/high-risk/*` | Task specification | Permissions, verification |
| `docs/templates/*` | Task specification | Context selection, verification |
| `scripts/README.md` | Tool access | Context selection |
| `scripts/bin/harness-cli` | Tool access | Task state, observability |
| `scripts/install.sh` | Tool access | Permissions |
| `scripts/install-local.sh` | Tool access | Permissions |
| `scripts/schema/001-init.sql` | Task state | Observability, project memory |
| `scripts/schema/002-story-verify.sql` | Verification | Task state, project memory |
| `scripts/schema/003-tool-registry.sql` | Tool access | Project memory |
| `scripts/schema/004-intervention.sql` | Intervention recording | Failure attribution |
| `scripts/schema/005-tool-extensions.sql` | Tool access | Verification, task state |

## Coverage Summary

- Covered: 8/11 responsibilities.
- Partial: 3/11 responsibilities.
- Missing: 0/11 responsibilities.

Covered responsibilities:

- Task specification.
- Context selection.
- Tool access.
- Project memory.
- Task state.
- Verification.
- Entropy auditing.
- Intervention recording.

Partial responsibilities:

- Observability.
- Failure attribution.
- Permissions.

The current Harness refresh converts tool access, entropy auditing, and
intervention recording into covered responsibilities with a registry, drift
audit, proposal loop, and intervention schema. Later work should focus on
benchmark ingestion, component-level attribution, permission enforcement, and
tool usage analytics.
