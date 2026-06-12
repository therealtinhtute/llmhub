/**
 * Tailwind v4 class map for quota components.
 * Replaces the former SCSS module with plain utility strings.
 */

export const quotaStyles = {
  // Page layout
  container: 'flex flex-col gap-4',
  pageHeader: 'flex flex-col gap-2',
  pageTitle: 'text-[28px] font-bold text-foreground m-0',
  description: 'text-sm text-muted-foreground m-0',

  // Header actions
  headerActions:
    'flex gap-2 flex-wrap items-center justify-end min-w-0 max-md:w-full max-md:justify-stretch',
  titleWrapper: 'flex items-center gap-2 leading-6',
  countBadge:
    'inline-flex items-center justify-center h-6 min-w-6 px-2 rounded-full text-[13px] font-semibold text-primary bg-primary/10 box-border',
  tabCountBadge:
    'inline-flex items-center justify-center h-4 min-w-4 px-1 rounded-full text-[10px] font-semibold text-primary bg-primary/10 box-border',

  // Error box
  errorBox:
    'p-3 bg-destructive/10 border border-destructive text-destructive text-sm',

  // Grid layouts (all providers share the same pattern)
  antigravityGrid:
    'grid gap-3 [grid-template-columns:repeat(auto-fill,minmax(380px,1fr))] max-md:[grid-template-columns:1fr]',
  claudeGrid:
    'grid gap-3 [grid-template-columns:repeat(auto-fill,minmax(380px,1fr))] max-md:[grid-template-columns:1fr]',
  codexGrid:
    'grid gap-3 [grid-template-columns:repeat(auto-fill,minmax(380px,1fr))] max-md:[grid-template-columns:1fr]',
  geminiCliGrid:
    'grid gap-3 [grid-template-columns:repeat(auto-fill,minmax(380px,1fr))] max-md:[grid-template-columns:1fr]',
  kimiGrid:
    'grid gap-3 [grid-template-columns:repeat(auto-fill,minmax(380px,1fr))] max-md:[grid-template-columns:1fr]',
  xaiGrid:
    'grid gap-3 [grid-template-columns:repeat(auto-fill,minmax(380px,1fr))] max-md:[grid-template-columns:1fr]',
  quotaGrid:
    'grid gap-3 [grid-template-columns:repeat(auto-fill,minmax(380px,1fr))] max-md:[grid-template-columns:1fr]',

  // View-mode toggle
  viewModeToggle:
    'inline-flex gap-1 items-center p-[3px] rounded-full bg-muted/92 border border-border/88 shadow-[inset_0_1px_0_rgba(255,255,255,0.16)] max-md:flex-auto max-md:w-full',
  viewModeButton: 'rounded-full border-transparent bg-transparent text-muted-foreground shadow-none max-md:flex-1',
  viewModeButtonActive: 'bg-primary border-primary text-white shadow-[0_8px_18px_-14px_rgba(0,0,0,0.45)]',
  refreshAllButton: 'rounded-full max-md:w-full max-md:justify-center',

  // Quota section (inside card)
  quotaSection: 'flex flex-col gap-2 pt-2 mt-1 border-t border-dashed border-border',
  quotaRow: 'flex flex-col gap-1',
  quotaRowHeader:
    'flex items-center justify-between gap-2 min-w-0 max-md:flex-col max-md:items-start',
  quotaModel:
    'text-[13px] font-semibold text-foreground whitespace-nowrap overflow-hidden text-ellipsis flex-1 min-w-0 max-md:whitespace-normal',
  quotaBar: 'h-1 bg-secondary rounded-full overflow-hidden',
  quotaBarFill: 'h-full',
  quotaMeta:
    'flex items-center gap-2 text-[12px] text-muted-foreground whitespace-nowrap max-md:justify-start',
  quotaPercent: 'font-semibold text-foreground',
  quotaReset: 'text-[var(--text-tertiary,hsl(var(--muted-foreground)))]',
  quotaAmount: 'text-muted-foreground',
  quotaMessage:
    'text-[12px] text-[var(--text-tertiary,hsl(var(--muted-foreground)))] text-center py-2',
  quotaMessageAction:
    'w-full border-none bg-none cursor-pointer underline text-[12px] text-[var(--text-tertiary,hsl(var(--muted-foreground)))] text-center py-2 hover:not-disabled:text-foreground disabled:cursor-not-allowed disabled:opacity-60 disabled:no-underline',
  quotaError:
    'text-[12px] text-destructive bg-destructive/8 border border-destructive px-2 py-1',
  quotaWarning:
    'text-[12px] text-amber-700 bg-amber-100 border border-amber-400/30 px-2 py-1',

  // Codex plan row
  codexPlan: 'flex items-center gap-1.5 text-[12px] text-muted-foreground',
  codexPlanLabel: 'text-[var(--text-tertiary,hsl(var(--muted-foreground)))]',
  codexPlanValue: 'font-semibold text-foreground capitalize',
  overagePlanValue: 'flex min-w-0 flex-1 items-center justify-between gap-2 font-semibold text-foreground',
  overageToggle:
    'shrink-0 border border-border bg-background px-2 py-0.5 text-[11px] font-semibold text-foreground hover:not-disabled:bg-muted disabled:cursor-not-allowed disabled:opacity-55',
  premiumPlanValue:
    'relative inline-flex items-center font-bold text-[12px] px-2 py-0.5 rounded-full overflow-visible isolate [background:radial-gradient(circle_at_18%_24%,rgba(255,255,255,0.96)_0%,rgba(255,255,255,0.72)_18%,rgba(255,255,255,0)_42%),linear-gradient(135deg,#fff9e3_0%,#ffe07f_52%,#e0aa14_100%)] border border-[rgba(217,165,22,0.72)] shadow-[0_1px_3px_rgba(133,92,0,0.16),0_0_0_1px_rgba(255,255,255,0.22)_inset,0_0_16px_rgba(255,214,98,0.28)] text-[#6b4b00] [text-shadow:0_1px_0_rgba(255,255,255,0.55)] capitalize',

  // Card styles (per-provider gradient tints)
  claudeCard:
    '[background-image:linear-gradient(180deg,rgba(251,236,228,0.18),rgba(251,236,228,0))]',
  antigravityCard:
    '[background-image:linear-gradient(180deg,rgba(224,247,250,0.12),rgba(224,247,250,0))]',
  codexCard:
    '[background-image:linear-gradient(180deg,rgba(234,231,255,0.18),rgba(234,231,255,0))]',
  geminiCliCard:
    '[background-image:linear-gradient(180deg,rgba(224,232,255,0.2),rgba(224,232,255,0))]',
  kimiCard:
    '[background-image:linear-gradient(180deg,rgba(220,232,255,0.2),rgba(220,232,255,0))]',
  xaiCard:
    '[background-image:linear-gradient(180deg,rgba(243,244,246,0.22),rgba(243,244,246,0))]',

  // File card
  fileCard:
    'bg-background border border-border p-3 flex flex-col gap-2',
  cardHeader: 'flex items-center gap-2 min-h-7',
  typeBadge: 'px-2.5 py-1 rounded-xl text-[12px] font-semibold whitespace-nowrap shrink-0',
  fileName: 'text-sm font-semibold text-foreground break-all leading-[1.4]',

  // Pagination
  pagination:
    'flex justify-center items-center gap-4 mt-4 pt-3 border-t border-border',
  pageInfo:
    'text-[13px] text-muted-foreground px-4 py-1 bg-muted',

  // Warning overlay / modal
  warningOverlay:
    'fixed inset-0 bg-black/50 flex items-center justify-center z-[1000]',
  warningModal:
    'bg-background p-4 max-w-[400px] text-center shadow-lg',

  // Unused controls classes kept for config compat (no-op strings)
  antigravityControls: '',
  claudeControls: '',
  codexControls: '',
  geminiCliControls: '',
  kimiControls: '',
  xaiControls: '',
  antigravityControl: '',
  claudeControl: '',
  codexControl: '',
  geminiCliControl: '',
  kimiControl: '',
  xaiControl: '',
  pageSizeSelect: '',
  statsInfo: '',
  headerActions2: '',
} as const;

export type QuotaStyleMap = typeof quotaStyles;
