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
    'grid gap-3 [grid-template-columns:repeat(auto-fill,minmax(300px,1fr))] max-md:[grid-template-columns:1fr]',
  claudeGrid:
    'grid gap-3 [grid-template-columns:repeat(auto-fill,minmax(300px,1fr))] max-md:[grid-template-columns:1fr]',
  codexGrid:
    'grid gap-3 [grid-template-columns:repeat(auto-fill,minmax(300px,1fr))] max-md:[grid-template-columns:1fr]',
  geminiCliGrid:
    'grid gap-3 [grid-template-columns:repeat(auto-fill,minmax(300px,1fr))] max-md:[grid-template-columns:1fr]',
  kimiGrid:
    'grid gap-3 [grid-template-columns:repeat(auto-fill,minmax(300px,1fr))] max-md:[grid-template-columns:1fr]',
  xaiGrid:
    'grid gap-3 [grid-template-columns:repeat(auto-fill,minmax(300px,1fr))] max-md:[grid-template-columns:1fr]',
  quotaGrid:
    'grid gap-3 [grid-template-columns:repeat(auto-fill,minmax(300px,1fr))] max-md:[grid-template-columns:1fr]',

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
  quotaCardTabs: 'flex flex-col gap-2',
  quotaCardTabsList:
    'flex w-full justify-start gap-0 border-b border-border bg-transparent p-0 overflow-x-auto',
  quotaCardTabsTrigger:
    'min-h-[28px] px-2 py-1 gap-1.5 text-[11px] font-semibold text-muted-foreground border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:text-primary hover:text-foreground',
  quotaCardTabsContent: 'flex flex-col gap-2 pt-1',
  quotaCardActionRow: 'flex justify-end pt-1',

  // Codex plan row
  codexPlan: 'flex flex-wrap items-center gap-x-3 gap-y-1 text-[12px] text-muted-foreground',
  codexPlanItem: 'inline-flex items-center gap-1.5',
  codexPlanLabel: 'text-[var(--text-tertiary,hsl(var(--muted-foreground)))]',
  codexPlanValue: 'font-semibold text-foreground capitalize',

  // Codex manual reset credits
  codexResetCredits: 'flex flex-col gap-1 border border-border bg-muted/40 px-2 py-1.5',
  codexResetCreditsTitle:
    'text-[11px] font-semibold text-[var(--text-tertiary,hsl(var(--muted-foreground)))]',
  codexResetCreditRow: 'flex items-center justify-between gap-2 text-[11px]',
  codexResetCreditLabel: 'text-muted-foreground',
  codexResetCreditTime: 'font-medium text-foreground tabular-nums',
  codexResetCreditsError: 'text-[11px] text-destructive',
  overagePlanValue: 'flex min-w-0 flex-1 items-center justify-between gap-2 font-semibold text-foreground',
  overageToggle:
    'shrink-0 rounded-md border border-border bg-background px-2 py-0.5 text-[11px] font-semibold text-foreground hover:not-disabled:bg-muted disabled:cursor-not-allowed disabled:opacity-55',
  premiumPlanValue:
    'inline-flex items-center font-bold text-[12px] px-2 py-[2px] bg-amber-500/15 border border-amber-500/30 text-amber-600 capitalize',
  kiroInfoRow:
    'flex items-center justify-between gap-2 text-[12px] text-muted-foreground min-w-0',
  kiroInfoValue: 'flex items-center gap-1.5 min-w-0 justify-end',
  kiroChip:
    'inline-flex items-center rounded-full border border-border bg-muted/70 px-2 py-[2px] text-[11px] font-semibold text-foreground whitespace-nowrap',
  kiroChipMuted:
    'inline-flex items-center rounded-full border border-border bg-muted/60 px-2 py-[2px] text-[11px] font-semibold text-muted-foreground whitespace-nowrap',
  kiroChipSuccess:
    'inline-flex items-center rounded-full border border-emerald-500/25 bg-emerald-500/10 px-2 py-[2px] text-[11px] font-semibold text-emerald-700 whitespace-nowrap',
  kiroOverageRow:
    'flex items-center justify-between gap-3 text-[12px] text-muted-foreground min-w-0',
  kiroRuntimeFooter:
    'flex items-center gap-1.5 pt-2 mt-1 border-t border-border/60 text-[12px] text-muted-foreground',

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
    'bg-background border border-border rounded-md p-3 flex flex-col gap-2',
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
