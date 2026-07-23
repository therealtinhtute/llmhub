export const meta = {
  name: 'backport-plan-correction',
  description: 'Correct Phase 3 and Phase 4 contracts and produce an approved fan-out plan',
  phases: [
    { title: 'Resolve contracts', detail: 'adjudicate Phase 3 and audit Phase 4 with bounded scope' },
    { title: 'Revise plan', detail: 'build a smaller dependency-safe execution plan' },
    { title: 'Verify plan', detail: 'adversarially verify the corrected plan against the locked spec' },
  ],
}

const CONTRACT_SCHEMA = {
  type: 'object',
  required: ['subject', 'verdict', 'lockedBehavior', 'allowedFiles', 'tasks', 'risks', 'stopConditions'],
  properties: {
    subject: { type: 'string' },
    verdict: { type: 'string', enum: ['ready', 'needs-plan-refresh', 'blocked'] },
    lockedBehavior: { type: 'array', items: { type: 'string' } },
    allowedFiles: { type: 'array', items: { type: 'string' } },
    tasks: {
      type: 'array',
      items: {
        type: 'object',
        required: ['id', 'title', 'dependsOn', 'exclusiveFiles', 'verification'],
        properties: {
          id: { type: 'string' },
          title: { type: 'string' },
          dependsOn: { type: 'array', items: { type: 'string' } },
          exclusiveFiles: { type: 'array', items: { type: 'string' } },
          verification: { type: 'array', items: { type: 'string' } },
        },
      },
    },
    risks: { type: 'array', items: { type: 'string' } },
    stopConditions: { type: 'array', items: { type: 'string' } },
  },
}

phase('Resolve contracts')
const resolved = await parallel([
  () => agent(`Read-only contract adjudication for Phase 3 in /home/tinhpt/Lab/llmhub. Read only: .kit/planning/SPEC.md requirements 7-9, the approved source plan Phase 3, the current Phase 3 CONTEXT/PLAN, and the named translator files. Do not run tests and do not inspect Google Interactions. Resolve the previous contradiction strictly in favor of the locked SPEC: provider output_index work belongs to internal/translator/codex/claude/codex_claude_response.go with sequential fallback only when absent. Keep Antigravity in scope only if both the approved source plan and locked boundary authorize it; otherwise mark it excluded pending SPEC/phase refresh. Define the exact schema-safe expected representation for string/array/object function_call_output when translating OpenAI Responses to Claude, or identify a stop condition if current Claude content schema cannot represent it. Produce the smallest independent task split with exclusive files. Limit yourself to 25 tool calls. Return only structured output.`, {
    label: 'resolve:phase3', phase: 'Resolve contracts', schema: CONTRACT_SCHEMA, effort: 'high'
  }),
  () => agent(`Bounded read-only audit for Phase 4 in /home/tinhpt/Lab/llmhub. Read only the locked SPEC requirements 10-12, approved source plan Phase 4, current Phase 4 CONTEXT/PLAN, cached upstream files named in CONTEXT, and the exact local files named in PLAN. Do not run tests. Limit yourself to 30 tool calls. Confirm whether request declarations are available for response restoration. Define an injective collision policy or explicit duplicate rejection for namespace/tool names containing '__', unnamespaced names equal to qualified names, and existing mcp__ names. Define x_search matching against original client-declared identities. Split translator foundation and executor consumer tasks with exclusive file ownership and dependency. Return only structured output.`, {
    label: 'audit:phase4-retry', phase: 'Resolve contracts', schema: CONTRACT_SCHEMA, effort: 'high'
  }),
])

const PLAN_SCHEMA = {
  type: 'object',
  required: ['summary', 'counts', 'waves', 'reviewContract', 'baselineContract', 'phaseGates', 'storyEvidence', 'pausePoints'],
  properties: {
    summary: { type: 'string' },
    counts: {
      type: 'object',
      required: ['productImplementationTasks', 'phaseClosureTasks', 'phaseGateTasks', 'finalEvidenceTasks', 'totalExecutionUnits'],
      properties: {
        productImplementationTasks: { type: 'integer' },
        phaseClosureTasks: { type: 'integer' },
        phaseGateTasks: { type: 'integer' },
        finalEvidenceTasks: { type: 'integer' },
        totalExecutionUnits: { type: 'integer' },
      },
    },
    waves: {
      type: 'array',
      items: {
        type: 'object',
        required: ['id', 'name', 'tasks', 'parallel', 'dependsOn', 'exitProof'],
        properties: {
          id: { type: 'string' },
          name: { type: 'string' },
          tasks: { type: 'array', items: { type: 'string' } },
          parallel: { type: 'boolean' },
          dependsOn: { type: 'array', items: { type: 'string' } },
          exitProof: { type: 'array', items: { type: 'string' } },
        },
      },
    },
    reviewContract: { type: 'array', items: { type: 'string' } },
    baselineContract: { type: 'array', items: { type: 'string' } },
    phaseGates: { type: 'array', items: { type: 'string' } },
    storyEvidence: { type: 'array', items: { type: 'string' } },
    pausePoints: { type: 'array', items: { type: 'string' } },
  },
}

phase('Revise plan')
const revised = await agent(`Create a corrected, minimal fan-out execution plan for the remaining CLIProxyAPI v7.2.93 backport in /home/tinhpt/Lab/llmhub. This is planning only. Read the locked SPEC, approved source plan, current roadmap and five phase plans. Use the two resolved contract objects below. Correct all prior critic findings:
1. Phase 1 closure must fingerprint or rerun the exact current six-file product tree before APPROVED check.
2. Each implementation slice gets a durable pre-slice baseline and reversible patch artifact; dependent slices declare base IDs.
3. Jitter contract clamps base wait to max first, then adds at most min(clamped/4,2s,remaining budget); cover wait below/equal/above max and zero.
4. Phase 3 output_index stays Codex-to-Claude with provider value preserved and sequential fallback only when absent.
5. Structured Claude output requires exact schema-valid fixtures; stop instead of improvising if impossible.
6. Phase 4 collision and x_search identity rules come from the bounded audit.
7. Every gate compares tracked/untracked path manifests against an exact phase allowlist before and after build commands.
8. Any unimplemented in-scope requirement fails the gate; no implicit deferrals.
9. Add the required new high-risk story packet and update validation after every phase.
10. Reviewer work is not counted as separate product/closure tasks; every implementation task has a paired read-only review report with normalized verdict/findings/reverification.
Keep the plan lean: group mechanical consumers only when they have exclusive files but allow parallel subagents within a wave. Count execution units as implementation/closure/gates/final evidence only, not review pairings. No cross-phase implementation overlap. No commits, pushes, PRs, releases, or publication. Return only structured output.\nRESOLVED CONTRACTS:\n${JSON.stringify(resolved.filter(Boolean))}`, {
  label: 'revise:lead-plan', phase: 'Revise plan', schema: PLAN_SCHEMA, effort: 'xhigh'
})

const REVIEW_SCHEMA = {
  type: 'object',
  required: ['approved', 'blockingIssues', 'nonBlockingNotes', 'correctedCounts'],
  properties: {
    approved: { type: 'boolean' },
    blockingIssues: { type: 'array', items: { type: 'string' } },
    nonBlockingNotes: { type: 'array', items: { type: 'string' } },
    correctedCounts: {
      type: 'object',
      required: ['productImplementationTasks', 'totalExecutionUnits'],
      properties: {
        productImplementationTasks: { type: 'integer' },
        totalExecutionUnits: { type: 'integer' },
      },
    },
  },
}

phase('Verify plan')
const review = await agent(`Adversarially verify the revised plan against /home/tinhpt/Lab/llmhub/.kit/planning/SPEC.md, the approved source plan, and the two resolved contracts. Reject for any spec contradiction, forbidden-surface drift, unowned shared file, unsafe concurrent edit, missing deterministic proof, missing rollback boundary, missing story validation, stale-evidence Phase 1 approval, implicit in-scope deferral, or counting reviewer roles as execution tasks. The plan is allowed to be detailed but should not inflate task count. Return only structured output.\nREVISED PLAN:\n${JSON.stringify(revised)}\nRESOLVED CONTRACTS:\n${JSON.stringify(resolved.filter(Boolean))}`, {
  label: 'verify:corrected-plan', phase: 'Verify plan', schema: REVIEW_SCHEMA, effort: 'xhigh'
})

return { resolved: resolved.filter(Boolean), revised, review }
