# Product Context: LLMHub Management Center

## Platform
web

## Mode
Operate

## Target Audience
- Developers, DevOps engineers, and operators managing multi-account CLI proxy gateways.
- Users configuring OAuth credentials, quota tracking, model aliases, and dynamic provider routing for local and team AI tooling.

## Product Purpose
- Provide a high-density, reliable, and low-distraction web control plane for the LLMHub proxy server.
- Allow operators to inspect real-time connection status, manage API keys, configure quota limits, edit server configuration (visual and source YAML), and monitor provider health with zero operational overhead.

## Positioning
- The interface serves the operator's task: familiarity, legibility, and state predictability take precedence over decorative flair.
- High information density, clear status feedback, standard navigation patterns, and instant responsiveness.

## Durable Constraints
- **Theme Foundation**: Adhere to the tweakcn Walter design system (warm cream/charcoal aesthetic, deep teal primary, refined 0.3rem border radii).
- **Typography**: `Archivo` for primary UI sans, `Xanh Mono` for serif accents, and `Google Sans Code` for code/data tables.
- **Tech Stack**: React 19, Tailwind CSS v4 (`@theme inline`), Radix UI primitives, Lucide icons, Zustand state stores.
- **Build & Distribution**: Single-file bundled HTML via `vite-plugin-singlefile`, embedded directly into the Go binary static assets (`internal/managementasset/static/management.html`).
- **Testing Policy**: Strictly no new frontend test files under `web/` (per `CLAUDE.md`); verify via type checking (`tsc`), production bundling, and browser runtime inspection.

## Brand Commitments
- Project identity is **LLMHub**; no legacy upstream naming ("CLI Proxy API") on user-facing surfaces.
- Semantic state vocabulary: consistent styling for `success`, `warning`, `destructive`, `muted`, and `primary` across both light and dark modes.
- State-only motion (150–250ms transitions); zero decorative animations, floating background orbs, or orchestrated page-load choreography.
