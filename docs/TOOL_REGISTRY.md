# Tool Registry

The Harness CLI exposes a machine-readable tool manifest through:

```bash
scripts/bin/harness-cli query tools --summary
scripts/bin/harness-cli query tools --json
scripts/bin/harness-cli query tools --responsibility Verification
```

External project tools can be registered with:

```bash
scripts/bin/harness-cli tool register \
  --name deploy-check \
  --command ./scripts/deploy-check.sh \
  --description "Verify deploy health before release" \
  --responsibility Verification \
  --args "env:enum:required:staging,production"
```

Use `--force` only when the command is intentionally unavailable on the current
machine. Remove registered tools with:

```bash
scripts/bin/harness-cli tool remove --name deploy-check
```

## Compiled Harness Commands

| Command | Responsibility | Purpose | Arguments |
| --- | --- | --- | --- |
| `init` | Task state | Create the harness database. | none |
| `migrate` | Task state | Apply pending schema migrations. | none |
| `import brownfield` | Project memory | Seed durable records from markdown state. | none |
| `intake` | Task specification | Record a feature intake classification. | `--type`, `--summary`, `--lane` |
| `story add` | Task state | Create a durable story record. | `--id`, `--title`, `--lane`, optional `--verify` |
| `story update` | Task state | Update story status, proof flags, evidence, or verification command. | `--id`, optional proof/status fields |
| `story verify` | Verification | Run one story `verify_command` and record pass/fail. | story id |
| `story verify-all` | Verification | Run all configured story verification commands and skip stories without one. | none |
| `decision add` | Project memory | Create a durable decision record. | `--id`, `--title`, optional `--doc`, `--verify` |
| `decision verify` | Verification | Run one decision verification command. | decision id |
| `backlog add` | Entropy auditing | Record a harness improvement proposal. | `--title`, optional pain/suggestion/risk/predicted fields |
| `backlog close` | Entropy auditing | Close a backlog item with outcome evidence. | `--id`, optional `--status`, `--outcome` |
| `tool register` | Tool access | Register an external project tool. | `--name`, `--command`, `--description`, `--responsibility`, optional `--args`, `--force` |
| `tool remove` | Tool access | Remove a registered external tool. | `--name` |
| `intervention add` | Intervention recording | Record a human, reviewer, CI, or agent intervention. | `--type`, `--description`, `--source`, optional `--trace`, `--story`, `--impact` |
| `trace` | Observability | Record an agent execution trace and print trace quality. | `--summary`, optional trace fields |
| `score-trace` | Observability | Score trace detail against lane requirements. | optional `--id` |
| `score-context` | Context selection | Score trace reads against compiled context rules. | trace id |
| `audit` | Entropy auditing | Run drift checks and compute entropy score. | none |
| `propose` | Entropy auditing | Generate improvement proposals from friction, interventions, and audit findings. | optional `--commit` |
| `query matrix` | Task state | Show durable story proof matrix. | optional `--numeric` |
| `query backlog` | Entropy auditing | Show harness improvement backlog. | optional `--open`, `--closed` |
| `query decisions` | Project memory | Show durable decision records. | none |
| `query intakes` | Task specification | Show recent intake records. | none |
| `query traces` | Observability | Show recent trace records. | none |
| `query friction` | Failure attribution | Show traces with harness friction. | none |
| `query tools` | Tool access | Show compiled and registered tool entries. | optional `--json`, `--summary`, `--responsibility` |
| `query interventions` | Intervention recording | Show intervention records. | optional `--trace`, `--story`, `--type` |
| `query stats` | Task state | Show durable record counts. | none |
| `query sql` | Tool access | Run arbitrary SQL against `harness.db`. | SQL text |

## Validation Rules

- Tool names must be unique among registered tools.
- Descriptions must be 10-200 characters.
- Responsibilities must match the Runtime Substrate responsibility list.
- `--args` entries must use `name:type:required` or
  `name:type:required:help`, with `required` or `optional` as the third field.
- Commands must exist as a path or on `PATH`, unless `--force` is supplied.
