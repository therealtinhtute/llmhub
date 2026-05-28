# Plan: Legacy Removal

Phase: legacy-removal
Status: ready
Wave Count: 4
Execution Owner: work
Updated At: 2026-05-28

## Goal
Delete all 13 Legacy* wrapper components and migrate their 71 consumer files to shadcn/ui primitives.

## Inputs
- All Legacy* files in `web/src/components/ui/`
- All consumer files identified in audit
- shadcn UI components already installed

## Wave 1 — High-impact wrappers (most consumers)
### T1 — Migrate LegacyToggleSwitch → Switch + Label (9 files)
- type: migration
- inputs:
  - `web/src/components/ui/LegacyToggleSwitch.tsx` (read to understand props API)
- touches:
  - `web/src/components/config/VisualConfigEditor.tsx`
  - `web/src/components/modelAlias/ModelMappingDiagramModals.tsx`
  - `web/src/features/authFiles/components/AuthFileCard.tsx`
  - `web/src/features/authFiles/components/AuthFilesPrefixProxyEditorModal.tsx`
  - `web/src/features/providers/components/ProviderResourceTable.tsx`
  - `web/src/pages/AuthFilesOAuthModelAliasEditPage.tsx`
  - `web/src/pages/AuthFilesPage.tsx`
  - `web/src/pages/LogsPage.tsx`
  - `web/src/pages/SystemPage.tsx`
- avoid:
  - changing any business logic, API calls, or state management
- steps:
  1. Read LegacyToggleSwitch to understand its props (label, checked, onChange, disabled, size)
  2. For each consumer: replace import, replace JSX with `<div className="flex items-center gap-2"><Switch checked={...} onCheckedChange={...} /><Label>...</Label></div>`
  3. Delete `LegacyToggleSwitch.tsx`
- expected outputs:
  - 9 consumer files updated, 1 legacy file deleted
- verification:
  - `grep -r "LegacyToggleSwitch" web/src/` returns no matches
  - `cd web && bun run type-check` passes (if available) or `bun run build` succeeds
- stop if:
  - LegacyToggleSwitch has complex logic beyond wrapping Switch
- escalate to:
  - user clarification

### T2 — Migrate LegacyCard → Card (8 files)
- type: migration
- inputs:
  - `web/src/components/ui/LegacyCard.tsx`
- touches:
  - `web/src/components/quota/QuotaSection.tsx`
  - `web/src/features/authFiles/components/OAuthExcludedCard.tsx`
  - `web/src/features/authFiles/components/OAuthModelAliasCard.tsx`
  - `web/src/pages/AuthFilesPage.tsx`
  - `web/src/pages/LogsPage.tsx`
  - `web/src/pages/OAuthPage.tsx`
  - `web/src/pages/PlaceholderPage.tsx`
  - `web/src/pages/SystemPage.tsx`
- avoid:
  - changing business logic
- steps:
  1. Read LegacyCard to understand props
  2. Replace imports and JSX in each consumer
  3. Delete `LegacyCard.tsx`
- expected outputs:
  - 8 files updated, 1 file deleted
- verification:
  - `grep -r "LegacyCard" web/src/` returns no matches
- stop if:
  - LegacyCard has props not directly mappable to shadcn Card
- escalate to:
  - user clarification

### T3 — Migrate LegacyModal → Dialog (8 files)
- type: migration
- inputs:
  - `web/src/components/ui/LegacyModal.tsx`
- touches:
  - `web/src/components/common/ConfirmationModal.tsx`
  - `web/src/components/config/DiffModal.tsx`
  - `web/src/components/config/VisualConfigEditorBlocks.tsx`
  - `web/src/components/modelAlias/ModelMappingDiagramModals.tsx`
  - `web/src/features/authFiles/components/AuthFileModelsModal.tsx`
  - `web/src/features/authFiles/components/AuthFilesPrefixProxyEditorModal.tsx`
  - `web/src/pages/LogsPage.tsx`
  - `web/src/pages/SystemPage.tsx`
- avoid:
  - changing modal content or behavior
- steps:
  1. Read LegacyModal to understand props (isOpen, onClose, title, footer, size)
  2. Replace with Dialog/DialogContent/DialogHeader/DialogTitle/DialogFooter
  3. Delete `LegacyModal.tsx`
- expected outputs:
  - 8 files updated, 1 file deleted
- verification:
  - `grep -r "LegacyModal" web/src/` returns no matches
- stop if:
  - LegacyModal has animation logic tied to motion library
- escalate to:
  - plan phase

## Wave 2 — Medium-impact wrappers
### T4 — Migrate LegacyInput → Input + Label (8 files)
- type: migration
- inputs:
  - `web/src/components/ui/LegacyInput.tsx`
- touches:
  - `web/src/components/config/VisualConfigEditor.tsx`
  - `web/src/components/modelAlias/ModelMappingDiagramModals.tsx`
  - `web/src/features/authFiles/components/AuthFilesPrefixProxyEditorModal.tsx`
  - `web/src/pages/AuthFilesPage.tsx`
  - `web/src/pages/ConfigPage.tsx`
  - `web/src/pages/LoginPage.tsx`
  - `web/src/pages/LogsPage.tsx`
  - `web/src/pages/OAuthPage.tsx`
- avoid:
  - changing validation or form logic
- steps:
  1. Read LegacyInput to understand props (label, hint, error, type, etc.)
  2. Replace with `<div><Label>...</Label><Input .../>{hint && <p className="text-sm text-muted-foreground">...</p>}{error && <p className="text-sm text-destructive">...</p>}</div>`
  3. Delete `LegacyInput.tsx`
- expected outputs:
  - 8 files updated, 1 file deleted
- verification:
  - `grep -r "LegacyInput" web/src/` returns no matches
- stop if:
  - LegacyInput has complex validation integration
- escalate to:
  - user clarification

### T5 — Migrate LegacyEmptyState → EmptyState utility (8 files)
- type: migration
- inputs:
  - `web/src/components/ui/LegacyEmptyState.tsx`
- touches:
  - `web/src/components/quota/QuotaSection.tsx`
  - `web/src/features/authFiles/components/AuthFileModelsModal.tsx`
  - `web/src/features/authFiles/components/OAuthExcludedCard.tsx`
  - `web/src/features/authFiles/components/OAuthModelAliasCard.tsx`
  - `web/src/pages/AuthFilesOAuthExcludedEditPage.tsx`
  - `web/src/pages/AuthFilesOAuthModelAliasEditPage.tsx`
  - `web/src/pages/AuthFilesPage.tsx`
  - `web/src/pages/LogsPage.tsx`
  - New: `web/src/components/ui/EmptyState.tsx`
- avoid:
  - over-engineering the EmptyState component
- steps:
  1. Read LegacyEmptyState to understand props (icon, title, description, action)
  2. Create `EmptyState.tsx` — simple component with icon, title, description, optional action button, using Tailwind utilities
  3. Update 8 consumer files to import from new EmptyState
  4. Delete `LegacyEmptyState.tsx`
- expected outputs:
  - New EmptyState.tsx created, 8 files updated, 1 file deleted
- verification:
  - `grep -r "LegacyEmptyState" web/src/` returns no matches
  - New `EmptyState.tsx` exists
- stop if:
  - LegacyEmptyState has complex render logic
- escalate to:
  - user clarification

### T6 — Migrate LegacySelect → Select (6 files)
- type: migration
- inputs:
  - `web/src/components/ui/LegacySelect.tsx`
- touches:
  - `web/src/components/config/VisualConfigEditorBlocks.tsx`
  - `web/src/components/config/VisualConfigEditor.tsx`
  - `web/src/features/providers/components/OpenAIBrandToolbar.tsx`
  - `web/src/features/providers/sheets/forms/BaseProviderForm.tsx`
  - `web/src/pages/AuthFilesPage.tsx`
  - `web/src/pages/LoginPage.tsx`
- avoid:
  - changing option values or selection logic
- steps:
  1. Read LegacySelect props
  2. Replace with Select/SelectTrigger/SelectContent/SelectItem
  3. Delete `LegacySelect.tsx`
- expected outputs:
  - 6 files updated, 1 file deleted
- verification:
  - `grep -r "LegacySelect" web/src/` returns no matches
- stop if:
  - LegacySelect has custom rendering beyond shadcn Select capability
- escalate to:
  - user clarification

## Wave 3 — Lower-impact wrappers
### T7 — Migrate LegacySelectionCheckbox → Checkbox + Label (5 files)
- type: migration
- inputs:
  - `web/src/components/ui/LegacySelectionCheckbox.tsx`
- touches:
  - `web/src/features/authFiles/components/AuthFileCard.tsx`
  - `web/src/features/providers/components/OpenAIBrandToolbar.tsx`
  - `web/src/features/providers/sheets/forms/ModelDiscoveryPanel.tsx`
  - `web/src/pages/AuthFilesOAuthExcludedEditPage.tsx`
  - `web/src/pages/LoginPage.tsx`
- steps:
  1. Read LegacySelectionCheckbox props
  2. Replace with Checkbox + Label composition
  3. Delete `LegacySelectionCheckbox.tsx`
- expected outputs:
  - 5 files updated, 1 file deleted
- verification:
  - `grep -r "LegacySelectionCheckbox" web/src/` returns no matches
- stop if:
  - custom selection state logic
- escalate to:
  - user clarification

### T8 — Migrate LegacyLoadingSpinner → Loader2 icon (5 files)
- type: migration
- inputs:
  - `web/src/components/ui/LegacyLoadingSpinner.tsx`
- touches:
  - `web/src/components/common/SecondaryScreenShell.tsx`
  - `web/src/features/authFiles/components/AuthFileCard.tsx`
  - `web/src/features/authFiles/components/AuthFilesPrefixProxyEditorModal.tsx`
  - `web/src/pages/AuthFilesOAuthExcludedEditPage.tsx`
  - `web/src/router/ProtectedRoute.tsx`
- steps:
  1. Read LegacyLoadingSpinner
  2. Replace with `<Loader2 className="animate-spin" />` from lucide-react
  3. Delete `LegacyLoadingSpinner.tsx`
- expected outputs:
  - 5 files updated, 1 file deleted
- verification:
  - `grep -r "LegacyLoadingSpinner" web/src/` returns no matches
- stop if:
  - spinner has size/color variants beyond className
- escalate to:
  - user clarification

### T9 — Migrate LegacyCollapsible → Collapsible (3 files)
- type: migration
- inputs:
  - `web/src/components/ui/LegacyCollapsible.tsx`
- touches:
  - `web/src/features/providers/sheets/forms/AmpcodeForm.tsx`
  - `web/src/features/providers/sheets/forms/BaseProviderForm.tsx`
  - `web/src/features/providers/sheets/ResourceDetailView.tsx`
- steps:
  1. Read LegacyCollapsible (uses HTML `<details>`)
  2. Replace with shadcn Collapsible/CollapsibleTrigger/CollapsibleContent
  3. Delete `LegacyCollapsible.tsx`
- expected outputs:
  - 3 files updated, 1 file deleted
- verification:
  - `grep -r "LegacyCollapsible" web/src/` returns no matches
- stop if:
  - `<details>` behavior is needed (native open/close without JS)
- escalate to:
  - user clarification

## Wave 4 — Single-consumer wrappers + autocomplete
### T10 — Migrate LegacyAutocompleteInput → Combobox pattern (2 files)
- type: migration
- inputs:
  - `web/src/components/ui/LegacyAutocompleteInput.tsx`
- touches:
  - `web/src/pages/AuthFilesOAuthExcludedEditPage.tsx`
  - `web/src/pages/AuthFilesOAuthModelAliasEditPage.tsx`
- steps:
  1. Read LegacyAutocompleteInput to understand filtering and selection logic
  2. Replace with Input + Popover + filtered list pattern (preserve filtering logic)
  3. Delete `LegacyAutocompleteInput.tsx`
- expected outputs:
  - 2 files updated, 1 file deleted
- verification:
  - `grep -r "LegacyAutocompleteInput" web/src/` returns no matches
- stop if:
  - complex keyboard navigation or async filtering
- escalate to:
  - user clarification

### T11 — Migrate LegacySheet → Sheet (1 file)
- type: migration
- inputs:
  - `web/src/components/ui/LegacySheet.tsx`
- touches:
  - `web/src/features/providers/sheets/ProviderSheet.tsx`
- steps:
  1. Replace with shadcn Sheet components
  2. Delete `LegacySheet.tsx`
- expected outputs:
  - 1 file updated, 1 file deleted
- verification:
  - `grep -r "LegacySheet" web/src/` returns no matches

### T12 — Migrate LegacySkeleton → Skeleton (1 file)
- type: migration
- inputs:
  - `web/src/components/ui/LegacySkeleton.tsx`
- touches:
  - `web/src/features/providers/ProvidersWorkbenchPage.tsx`
- steps:
  1. Replace with shadcn Skeleton (adjust sizing props)
  2. Delete `LegacySkeleton.tsx`
- expected outputs:
  - 1 file updated, 1 file deleted
- verification:
  - `grep -r "LegacySkeleton" web/src/` returns no matches

### T13 — Migrate LegacyTable → Table (1 file)
- type: migration
- inputs:
  - `web/src/components/ui/LegacyTable.tsx`
- touches:
  - `web/src/features/providers/components/ProviderResourceTable.tsx`
- steps:
  1. Replace with shadcn Table components
  2. Delete `LegacyTable.tsx`
- expected outputs:
  - 1 file updated, 1 file deleted
- verification:
  - `grep -r "LegacyTable" web/src/` returns no matches

### T14 — Final legacy verification
- type: test
- steps:
  1. `grep -r "Legacy" web/src/components/ui/` — should return no files
  2. `grep -r "from.*Legacy" web/src/` — should return no imports
  3. `cd web && bun run build` — should succeed
- expected outputs:
  - Zero Legacy references, successful build
- verification:
  - All grep commands return empty
  - Build exits 0

## Risks / Watch-fors
- Some consumer files import multiple Legacy components — handle all in one edit pass
- LegacyInput's label/hint/error pattern needs a consistent replacement approach across 8 files
- LegacyAutocompleteInput may have async filtering — read implementation before migrating
