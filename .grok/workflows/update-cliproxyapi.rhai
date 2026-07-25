let meta = #{
    name: "update-cliproxyapi-parity",
    description: "Diff and update llmhub with latest CLIProxyAPI upstream changes (backend Go + frontend)",
    phases: [
        #{ title: "Diff", detail: "compare current code with latest upstream" },
        #{ title: "UpdateBackend", detail: "implement Go backend updates" },
        #{ title: "UpdateFrontend", detail: "implement React frontend updates" },
        #{ title: "Verify", detail: "run tests and build" },
        #{ title: "Check", detail: "final gate and report" }
    ]
};

let target = if args == () { () } else { args.target }; // default to current llmhub vs latest upstream

if target == () { pause("verification", "Pass args.target — the current branch or commit to update from."); }

phase("Diff");
let diff_job = #{
    prompt: "Compare current llmhub code (at " + target + ") with latest CLIProxyAPI upstream v7.2.96. List all gaps in backend Go (auth, executor, translator, credential handling) and frontend (management panel, model display, docs). Use read_file, grep, git diff to inspect code. Output as structured findings.",
    label: "diff",
    capability_mode: "read-only",
    output_schema: #{ "type": "object", "properties": #{ "findings": #{ "type": "array", "items": #{ "type": "object" } } } }
};
let diff_result = agent(diff_job.prompt, diff_job);

phase("UpdateBackend");
let backend_job = #{
    prompt: "Implement the backend Go updates from the diff findings (credential concurrency, token estimation, model routing, WebSocket fixes, etc.). Use search_replace or edit files in llmhub. Output the edited files and run go test ./internal/... after.",
    label: "backend",
    capability_mode: "execute",
    output_schema: #{ "type": "object", "properties": #{ "edits": #{ "type": "array" } } }
};
let backend_result = agent(backend_job.prompt, backend_result);

phase("UpdateFrontend");
let frontend_job = #{
    prompt: "Implement the frontend React updates from the diff findings (model display, AIUsage showcase, management panel UI). Use search_replace or edit React/Vite files. Output the edited components and run make build-web after.",
    label: "frontend",
    capability_mode: "execute",
    output_schema: #{ "type": "object", "properties": #{ "edits": #{ "type": "array" } } }
};
let frontend_result = agent(frontend_job.prompt, frontend_result);

phase("Verify");
let verify_job = #{
    prompt: "Run tests and build after updates: go test ./... , make build-web , make build. Report any failures.",
    label: "verify",
    capability_mode: "execute",
    output_schema: #{ "type": "object", "properties": #{ "result": #{ "type": "string" } } }
};
let verify_result = agent(verify_job.prompt, verify_result);

phase("Check");
let check_job = #{
    prompt: "Final check: verify all updates passed parity and no regressions. Report summary.",
    label: "check",
    capability_mode: "read-only",
    output_schema: #{ "type": "object", "properties": #{ "summary": #{ "type": "string" } } }
};
let check_result = agent(check_job.prompt, check_result);

complete(#{ summary: "Update complete", changes: diff_result.output.findings, backend_edits: backend_result.output.edits, frontend_edits: frontend_result.output.edits });
