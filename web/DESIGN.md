# Design System: LLMHub Walter Theme (Operate Mode)

## Overview
This design system defines the visual tokens, typography, component rules, and interaction states for the LLMHub management interface (`@web/`). It is tuned for **Operate mode**: prioritizing visual density, high legibility, accessible contrast (WCAG AA), predictable controls, and zero decorative noise.

---

## Colors

The color system uses **OKLCH** color coordinates for perceptual uniformity across light and dark themes.

### Light Mode (`:root`)
- `--background`: `oklch(0.9798 0.0045 78.2983)` (warm cream `#faf8f5`)
- `--foreground`: `oklch(0.1913 0 0)` (charcoal `#141414`)
- `--card`: `oklch(1.0000 0 0)` (`#ffffff`)
- `--card-foreground`: `oklch(0.1913 0 0)` (`#141414`)
- `--popover`: `oklch(1.0000 0 0)` (`#ffffff`)
- `--popover-foreground`: `oklch(0.1913 0 0)` (`#141414`)
- `--primary`: `oklch(0.3917 0.0748 236.0547)` (deep teal `#124b68`)
- `--primary-foreground`: `oklch(0.9809 0.0025 228.7836)` (`#f7f9fa`)
- `--secondary`: `oklch(0.9798 0.0045 78.2983)` (`#faf8f5`)
- `--secondary-foreground`: `oklch(0.1913 0 0)` (`#141414`)
- `--muted`: `oklch(0.9809 0.0025 228.7836)` (`#f7f9fa`)
- `--muted-foreground`: `oklch(0.3867 0 0)` (`#444444`)
- `--accent`: `oklch(0.9175 0.0177 234.5132)` (`#d9e6ee`)
- `--accent-foreground`: `oklch(0.3917 0.0748 236.0547)` (`#124b68`)
- `--destructive`: `oklch(0.6266 0.2173 24.9380)` (`#ef393e`)
- `--destructive-foreground`: `oklch(1.0000 0 0)` (`#ffffff`)
- `--border`: `oklch(0.9219 0 0)` (`#e5e5e5`)
- `--input`: `oklch(0.9219 0 0)` (`#e5e5e5`)
- `--ring`: `oklch(0.3917 0.0748 236.0547)` (`#124b68`)

### Dark Mode (`.dark`)
- `--background`: `oklch(0.2679 0.0036 106.6427)` (deep charcoal `#262624`)
- `--foreground`: `oklch(0.8074 0.0142 93.0137)` (warm stone `#c3c0b6`)
- `--card`: `oklch(0.2679 0.0036 106.6427)` (`#262624`)
- `--card-foreground`: `oklch(0.9818 0.0054 95.0986)` (`#faf9f5`)
- `--popover`: `oklch(0.3085 0.0035 106.6039)` (`#30302e`)
- `--popover-foreground`: `oklch(0.9211 0.0040 106.4781)` (`#e5e5e2`)
- `--primary`: `oklch(0.5200 0.1050 223.1280)`
- `--primary-foreground`: `oklch(1.0000 0 0)` (`#ffffff`)
- `--secondary`: `oklch(0.9818 0.0054 95.0986)` (`#faf9f5`)
- `--secondary-foreground`: `oklch(0.3085 0.0035 106.6039)` (`#30302e`)
- `--muted`: `oklch(0.2213 0.0038 106.7070)` (`#1b1b19`)
- `--muted-foreground`: `oklch(0.7713 0.0169 99.0657)` (`#b7b5a9`)
- `--accent`: `oklch(0.2130 0.0078 95.4245)` (`#1a1915`)
- `--accent-foreground`: `oklch(0.9663 0.0080 98.8792)` (`#f5f4ee`)
- `--destructive`: `oklch(0.6368 0.2078 25.3313)` (`#ef4444`)
- `--destructive-foreground`: `oklch(1.0000 0 0)` (`#ffffff`)
- `--border`: `oklch(0.3618 0.0101 106.8928)` (`#3e3e38`)
- `--input`: `oklch(0.4336 0.0113 100.2195)` (`#52514a`)
- `--ring`: `oklch(0.5200 0.1050 223.1280)`

### Semantic Status Tokens
These tokens provide consistent state coloring across all pages, badges, and alerts:
- **Success (Green/Emerald)**:
  - Light: `--success: oklch(0.58 0.16 142.5)` (`#15803d`), `--success-foreground: oklch(0.98 0 0)`
  - Dark: `--success: oklch(0.72 0.16 142.5)` (`#4ade80`), `--success-foreground: oklch(0.18 0 0)`
  - Badge recipe: `text-success bg-success/12 border-success/30`
- **Warning (Amber/Orange)**:
  - Light: `--warning: oklch(0.65 0.17 72.0)` (`#b45309`), `--warning-foreground: oklch(0.98 0 0)`
  - Dark: `--warning: oklch(0.78 0.16 72.0)` (`#fbbf24`), `--warning-foreground: oklch(0.18 0 0)`
  - Badge recipe: `text-warning bg-warning/12 border-warning/30`
- **Destructive (Red)**:
  - Light: `--destructive: oklch(0.6266 0.2173 24.9380)`, `--destructive-foreground: #ffffff`
  - Dark: `--destructive: oklch(0.6368 0.2078 25.3313)`, `--destructive-foreground: #ffffff`
  - Badge recipe: `text-destructive bg-destructive/10 border-destructive/30`

---

## Typography

- **Primary Sans (`--font-sans`)**: `'Bricolage Grotesque', Geist, ui-sans-serif, sans-serif, system-ui`
  - Used for all interface chrome, headings, navigation, form inputs, buttons, and standard labels.
- **Display Serif (`--font-serif`)**: `'Xanh Mono', Lora, ui-serif, serif`
  - Used strictly for brand title accents (logo abbreviation and top splash heading). Never used for form labels, table cells, or body text.
- **Monospace (`--font-mono`)**: `'Google Sans Code', 'Geist Mono', ui-monospace, monospace`
  - Used for API keys, model identifiers, endpoints, HTTP status codes, numbers, and YAML/JSON code editors.

### Scale & Hierarchy
- Headings: `text-2xl` (24px) to `text-3xl` (28px) bold, tight leading (`leading-tight`).
- Body: `text-sm` (14px) / `leading-normal` (20px).
- Compact/Table data: `text-xs` (12px) to `text-[13px]`.
- Badges/Meta: `text-[11px]` font-semibold with tabular numerals (`tabular-nums`).

---

## Radii & Elevation

- Base radius: `--radius: 0.3rem` (4.8px)
- `--radius-sm`: `calc(var(--radius) - 4px)` (1px ~ 2px)
- `--radius-md`: `calc(var(--radius) - 2px)` (3px)
- `--radius-lg`: `var(--radius)` (4.8px)
- `--radius-xl`: `calc(var(--radius) + 4px)` (8.8px)

### Elevation
Subtle, soft shadows with low blur opacity to keep data tables grounded:
- Default shadow: `0 0px 4px -1px hsl(0 0% 0% / 0.05), 0 1px 2px -2px hsl(0 0% 0% / 0.05)`
- Modal/Dropdown shadow: `var(--shadow-lg)`

---

## Component Rules

### Interactive Focus States
- All interactive elements (`<button>`, `<input>`, `<select>`, `<textarea>`, custom toggles, and modal close buttons) **must** declare visible focus rings:
  `outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background`
- Never rely solely on `focus-visible:border-ring` without an outline or ring.

### Touch Targets
- On mobile and tablet viewports, all interactive touch targets must measure at least **44x44px** (or use negative margins / padding to meet the 44px hit area).

### Status Badges & Chips
- Status indicators must use semantic tokens:
  - Active/Verified: `text-success bg-success/15 border-success/30`
  - Warning/Pending/Amber: `text-warning bg-warning/15 border-warning/30`
  - Error/Disabled/Red: `text-destructive bg-destructive/10 border-destructive/30`
  - Neutral/Inactive: `text-muted-foreground bg-muted border-border`
- **Never** use hardcoded light-theme utilities like `bg-emerald-100 text-emerald-700` or `bg-amber-100 text-amber-700` without dark variants.

---

## Motion Principles

- **State-Only**: Transitions exist to convey state changes (hover, active press, modal entrance, toast notification).
- **Speed**: Transitions must complete within **150ms – 250ms**.
- **Timing Curve**: `var(--ease-out)` (`cubic-bezier(0.23, 1, 0.32, 1)`).
- **Forbidden**: Continuous decorative animations, floating ambient background orbs, animated watermarks, and orchestrated page-load sequences.
