---
status: completed
remediation_id: REM-0006
implementer: codex
scope: audit:AUD-0009
source_audits: AUD-0009
source_findings: AUD-0009-F001
baseline: git:fb787629ce892900e8fe806a6f527d8b839ee5ef; worktree:dirty (pre-existing AUD-0009 and remediation index changes)
started_at: 2026-07-14T00:00:00+08:00
completed_at: 2026-07-14T14:19:41+08:00
last_updated: 2026-07-14
related_plans: PLN-0005
---

# WP-Facts exact-output remediation

## Scope and boundaries

This record remediates `AUD-0009-F001`, which found that the tracked `WP-Facts` evidence cell checked for required paths but accepted unrecognized extra outputs. Closed audit records remain unchanged; independent verification belongs to a later follow-up audit.

## Finding remediation matrix

| Source finding | Root cause | Planned change | Validation | Result |
|---|---|---|---|---|
| `AUD-0009-F001` | `Test-PhaseFiveFactsOutput` searched for required substrings without parsing the output contract as a bounded set. | Parse the evidence cell into canonical output tokens, require exactly the five fact sources plus `plan/checklist`, and reject unknown, duplicate, or ambiguous tokens. Add negative fixtures for each rejection path. | `docs/tools/validate.tests.ps1`; `docs/tools/validate.ps1`; repository validator; `git diff HEAD --check`. | completed locally; pending follow-up audit |

## Implementation and evidence

Updated `docs/tools/validate.ps1` to tokenize path-like WP-Facts outputs, reject unknown/non-canonical tokens, reject duplicates, and require all six canonical outputs. Added self-test fixtures in `docs/tools/validate.tests.ps1` for missing, unknown, duplicate, and ambiguous outputs. No source audit finding is marked resolved by this remediation record.

Validation evidence:

- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.tests.ps1`: passed.
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1`: passed, validating 52 Markdown files.
- `git diff HEAD --check`: passed.

## Incomplete items and residual risk

Independent follow-up verification is required. This scope covers the phase-five documentation validator and fixtures only; product binaries, migrations, filesystems, and live deployment behavior remain outside the finding.

## Follow-up handoff

Ready for independent verification with `$backend-follow-up-audit TARGET=REM-0006`. This REM is complete locally but is not a verification result.
