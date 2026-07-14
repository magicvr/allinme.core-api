---
status: closed
audit_id: AUD-0009
auditor: codex
audit_type: follow-up
scope: follow-up:REM-0005
subject: rem-0005-active-audits
baseline: git:fb787629ce892900e8fe806a6f527d8b839ee5ef; worktree:clean
started_at: 2026-07-14T13:47:32+08:00
completed_at: 2026-07-14T13:58:55+08:00
last_updated: 2026-07-14
related_audits: AUD-0007,AUD-0008
related_remediations: REM-0005
supersedes: none
related_plans: PLN-0005
---

# REM-0005 follow-up audit

## Scope and method

This audit independently verifies `REM-0005` against source findings `AUD-0007-F001`, `AUD-0008-F001`, and `AUD-0008-F002`. The baseline is the current clean `HEAD`; the review covers the tracked `PLN-0005` work-package contract, `docs/tools/validate.ps1`, its self-tests, and the remediation and source-audit records. Product binaries, migrations, and live deployment behavior are outside the source findings and remain unverified.

## Verification matrix

| Source finding | Claimed remediation | Code/evidence inspected | Independent test | Verdict |
|---|---|---|---|---|
| `AUD-0007-F001` | Enumerate five external fact sources plus plan/checklist changes and validate the output contract. | `PLN-0005` `WP-Facts` row; `Test-PhaseFiveFactsOutput` in `docs/tools/validate.ps1`; the missing-architecture fixture in `docs/tools/validate.tests.ps1`. | A temporary fixture adding an unrecognized `docs/99-untracked-fact.md` output was accepted with validator exit 0. The required-source negative fixture still failed as expected. | `partially-resolved`; required sources are pinned, but the claimed exact output set is not enforced. See `AUD-0009-F001`. |
| `AUD-0008-F001` | Evaluate live-evidence polarity per clause, including English and Chinese contrast boundaries and overlap handling. | `Get-PhaseFiveStatementClauses`, `Get-PhaseFiveLiveEvidencePattern`, and `Test-PhaseFiveDeploymentClauses`; the remediation's bidirectional fixtures. | A temporary fixture using the actual Chinese transition `但是` between a deferral and a live supervisor obligation was rejected with `P0 deployment evidence clause must stop at contract fixtures`. | `resolved` |
| `AUD-0008-F002` | Limit rejection exemptions to isolated clauses, recognize `depends upon`, and fail closed on unknown multi-package connectors. | `Get-PhaseFiveDependencyClauses` and `Test-PhaseFiveDependencyStatements`; rejection-contamination and `depends upon` fixtures. | A temporary fixture using the unlisted connector `WP-Release is coupled to WP-Unknown` was rejected with `dependency statement could not be fully parsed`; the declared `depends upon` and rejection-contamination checks also pass. | `resolved` |

## Findings

### AUD-0009-F001 - WP-Facts accepts unrecognized extra outputs

- Maps to: `AUD-0007-F001`
- Severity: medium
- Evidence: `Test-PhaseFiveFactsOutput` checks that each required path occurs in the `WP-Facts` evidence cell, but it does not parse the cell into an allowlisted set or reject additional paths. An independent temporary fixture adding `docs/99-untracked-fact.md` passed the validator with exit 0.
- Impact: A work package can add an unreviewed fact source while still satisfying the validator. The five required P0-1 sources cannot be omitted, but the output contract is not exact as claimed by the remediation.
- Recommendation: Parse the `WP-Facts` evidence cell into canonical tokens and reject any output outside the five required fact sources plus the explicitly allowed plan/checklist diff token. Add a negative fixture for an unknown extra source and duplicate/ambiguous source spelling.
- Owner: phase-five protocol owner / Evidence tooling owner
- Disposition: open

## Validation results

- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.tests.ps1`: passed.
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1`: passed, validating 50 Markdown files.
- `git diff HEAD --check`: passed through the repository validator.
- Independent Chinese same-clause and alternate unknown-connector fixtures produced the expected non-zero validator results.

## Incomplete items and residual risk

`AUD-0009-F001` remains open. The active remediation queue is the exact-output contract gap; no claim is made about phase-five binaries, migrations, filesystem behavior, or live deployment profiles.

## Closure conclusion

`AUD-0009` is closed as an audit process, with remediation required for its new finding. `REM-0005` is partially verified. `AUD-0007` continues with the new finding; `AUD-0008` is fully verified. A subsequent remediation and follow-up audit must address the exact `WP-Facts` output set without appending to either closed source audit.
