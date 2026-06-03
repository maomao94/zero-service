# Implement: 优化 Trellis spec 文档

## Preconditions

- User has reviewed planning artifacts or explicitly approved proceeding.
- Task has been started with `python3 ./.trellis/scripts/task.py start .trellis/tasks/06-03-optimize-trellis-spec-docs` before editing `.trellis/spec/**` further.
- Read order for implementation: `prd.md` -> `design.md` -> this file -> affected spec files.

## Ordered Checklist

1. Baseline audit
   - Run `wc -l .trellis/spec/backend/*.md .trellis/spec/guides/*.md`.
   - Run `rg -n "coding-standards|go-zero-conventions|socketiox|error-handling|antsx|\.\./" .trellis/spec`.
   - Read the long target files before editing.
   - For each target file, identify the AI routing question it should answer: "when should an agent read this file?"

2. Fix stale error-code links
   - Replace every `../../../code.md` or textual `code.md` canonical-reference in backend specs.
   - Link user-facing mapping to `../../../docs/error-codes.md`.
   - Link enum definitions to `../../../third_party/extproto.proto`.

3. Clean `coding-standards.md`
   - Keep AI collaboration, safety, naming summary, Git rules.
   - Replace duplicated code generation / go-zero details with links to `go-zero-conventions.md`.
   - Replace Markdown LSP troubleshooting detail with a shorter note unless it remains project-critical.

4. Clean or split `socketiox-guidelines.md`
   - Split contract scenarios into `socketiox-contracts.md` only if the main file can then serve as a shorter routing/API guideline.
   - Preserve all payload contracts, response semantics, and tests required.
   - Remove repeated template text where a compact matrix is enough.
   - Add a short "When to read" note to both files if split.
   - Update `backend/index.md` if a new file is created.

5. Clean `error-handling.md`
   - Preserve gateway/RPC behavior and recommended factories.
   - Collapse repetitive examples while keeping one Wrong/Correct pair per important pattern.
   - Link to `../../../docs/error-codes.md` and `../../../third_party/extproto.proto` for canonical mappings instead of duplicating them.

6. Clean `antsx-invoke-guidelines.md`
   - Preserve core signatures, selection rules, cancellation behavior, panic protection, and tests.
   - Compress duplicated explanations and keep examples short.

7. Review `guides/*.md`
   - Remove backend implementation detail duplicated from code-spec files.
   - Keep each guide as a checklist with links to relevant backend specs.
   - Update `guides/index.md` if triggers change.

8. Link and index pass
   - Update `backend/index.md` descriptions and reading guidance.
   - For each spec row, include enough context for AI routing: topic, trigger, and canonical-source intent.
   - Update all relative links after any split/rename.
   - Ensure `.trellis/spec/` top level remains `backend/` and `guides/` only.

9. Validation
   - Run `rg -n "\.\./coding-standards|\.\./go-zero-conventions|code\.md" .trellis/spec` and expect no stale matches.
   - Run `rg -n "\]\([^)]*\.md\)" .trellis/spec` to inspect Markdown links manually.
   - Run `wc -l .trellis/spec/backend/*.md .trellis/spec/guides/*.md` and confirm long files are justified by contract density or were split/compressed.
   - Review `backend/index.md` and `guides/index.md` as an AI routing table: no row should force reading unrelated files.
   - Run `git diff --check`.
   - Run `lsp_diagnostics` on changed Markdown files if Markdown LSP is configured; otherwise record that `.md` LSP is unavailable.
   - Run `git diff -- .trellis/spec .trellis/tasks/06-03-optimize-trellis-spec-docs` and review for accidental content loss.

## Risky Files / Rollback Points

- `.trellis/spec/backend/socketiox-guidelines.md`: high risk because it contains multiple concrete protocol contracts.
- `.trellis/spec/backend/error-handling.md`: medium risk because it encodes project-wide error behavior.
- `.trellis/spec/backend/antsx-invoke-guidelines.md`: medium risk because it includes concurrency semantics and tests.
- `.trellis/spec/backend/index.md`: low risk but must stay accurate for discoverability.

Rollback by reverting only the affected spec file and its index entry; avoid broad reset commands.
