---
status: completed
remediation_id: REM-0005
implementer: codex
scope: audit:AUD-0007,AUD-0008
source_audits: AUD-0007, AUD-0008
source_findings: AUD-0007-F001, AUD-0008-F001, AUD-0008-F002
baseline: git:596477a1bf2c9fc0d4d9f5f27e3e913c9854b4c1; worktree:dirty (pre-existing staged AUD-0007, AUD-0008, REM-0004, audit/remediation index, and validator changes)
started_at: 2026-07-14T13:24:27+08:00
completed_at: 2026-07-14T13:31:38+08:00
last_updated: 2026-07-14
related_plans: PLN-0005
---

# Active audit remediation

## Scope and boundaries

This record remediates every audit currently indexed with `remediation=required`: `AUD-0007` and `AUD-0008`. It covers the `WP-Facts` output contract and the phase-five governance validator/self-tests. Closed audit records and prior closed remediation records remain unchanged, and only a later follow-up audit may verify these findings as resolved.

## Finding remediation matrix

| Source finding | Root cause | Planned change | Validation | Result |
|---|---|---|---|---|
| `AUD-0007-F001` | The authoritative `WP-Facts` row says its output contains four external fact-source diffs even though P0-1 requires five, including architecture. The validator did not pin the exact output set. | Updated the tracked row to enumerate the five external fact sources plus plan/checklist changes, and added an exact-output validator contract. | Repository validator plus a negative fixture that removes `docs/01-architecture.md`. | completed locally; pending follow-up audit |
| `AUD-0008-F001` | Deployment evidence scanning exempted an entire clause whenever any deferral token appeared, so a later obligation in the same unsplit clause could be masked. | Added clause-local live-Evidence polarity evaluation, including English and Chinese contrast boundaries and overlap handling for negated obligations. | Validator self-test covers deferral-to-obligation, obligation-to-deferral, and Chinese transition fixtures. | completed locally; pending follow-up audit |
| `AUD-0008-F002` | Rejection wording exempted the entire line, and the relationship grammar omitted common variants such as `depends upon`. | Scoped rejection exemptions to isolated clauses, recognized `depends upon`, and failed closed for unrecognized multi-package connectors. | Validator self-test covers rejection-line contamination, `depends upon`, and an unknown connector. | completed locally; pending follow-up audit |

## Actual changes

Updated `docs/plans/PLN-0005-phase-05-attachment-lifecycle.md`, `docs/tools/validate.ps1`, and `docs/tools/validate.tests.ps1`. Added the `WP-Facts` source contract and focused negative fixtures while preserving all closed audit and prior remediation records.

## Validation results

`powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.tests.ps1` passed. `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1` passed with 50 Markdown files. `git diff HEAD --check` passed as part of the repository validator.

## Incomplete items and residual risk

Independent verification remains outstanding. Product code, phase-five binaries, migrations, filesystems, and live deployment behavior are outside these documentation-governance findings. The parser remains intentionally conservative for future grammar not covered by the current fixtures.

## Follow-up handoff

Ready for independent verification with `$backend-follow-up-audit TARGET=REM-0005`. This REM is complete locally but is not a verification result.
