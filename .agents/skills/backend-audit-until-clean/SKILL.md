---
name: backend-audit-until-clean
description: Orchestrate the existing backend audit, remediation, and follow-up audit skills under a persistent Codex goal until the selected plans or repository are independently verified clean, with bounded cycles and stagnation circuit breakers. Use only when explicitly requested for an end-to-end audit-fix-verify loop.
---

# Audit Until Clean

Use the existing skills as the only implementation of each phase. Do not restate, shorten, or replace their workflows:

- `$backend-plan-audit`
- `$backend-full-audit`
- `$backend-fix-audit-findings`
- `$backend-follow-up-audit`

## Inputs

- `TARGET=active` by default: all active, unarchived plans.
- `TARGET=PLN-NNNN` or a comma-separated list: selected plans.
- `TARGET=repository`: the complete repository.
- `MAX_CYCLES=3` by default; accept 1 through 10.
- `MAX_STAGNANT_CYCLES=2` by default; accept 1 through 3.
- Pass optional `AUDITOR` and `FOCUS` through to the applicable underlying skills.
- `ESCALATE=stop` by default. Only `ESCALATE=repository` permits a plan loop to expand into a full audit when a systemic issue is found.

## Establish The Goal

1. Inspect the current Codex goal.
2. If no goal exists, create a persistent goal whose outcome is: independently verify the selected target with no remaining `remediation=required`, `remediation=awaiting-verification:*`, or `verification=pending` entries in its audit chain.
3. Include constraints in the goal: preserve closed AUD/REM records, use only the four skills above, respect cycle/stagnation limits, never auto-accept risk, and do not broaden scope without `ESCALATE=repository`.
4. Include verification in the goal: the latest follow-up AUD resolves every finding in scope and all gates required by the underlying skills pass.
5. Reuse a matching active goal. If an unrelated goal is active, stop and ask the user to clear or edit it; never overwrite it silently.

## Run The Bounded Loop

1. Run the initial audit:
   - plans: invoke `$backend-plan-audit` with the resolved target;
   - repository: invoke `$backend-full-audit`.
2. Read the new indexed AUD results. If no entry in this target chain has `remediation=required`, finish successfully.
3. For each cycle from 1 through `MAX_CYCLES`:
   - invoke `$backend-fix-audit-findings` with the current `remediation=required` AUD IDs;
   - read the created REM index entry; if it is blocked or not ready, stop under the blocker rule;
   - invoke `$backend-follow-up-audit` with that REM ID;
   - read the new follow-up AUD and both updated indexes;
   - if the target chain is clean, finish successfully;
   - otherwise use the new follow-up AUD entries marked `remediation=required` as the next cycle's fix target.

Never call the fix skill against both a source AUD marked `continued-by:*` and its newer follow-up AUD. The indexes determine the current queue.

## Progress And Circuit Breakers

After each follow-up, record a progress snapshot containing:

- unresolved finding count by severity;
- highest unresolved severity;
- source/root-cause mapping carried into the new follow-up AUD;
- failing validation commands or external blockers.

A cycle is stagnant when no source finding becomes resolved, the unresolved severity vector does not improve, and the same root causes or failing gates remain. Stop automation when any condition applies:

1. `MAX_CYCLES` is reached before clean verification.
2. `MAX_STAGNANT_CYCLES` consecutive cycles are stagnant.
3. The same external dependency, permission, environment, or unavailable evidence blocks two consecutive cycles.
4. Completion would require accepting risk, weakening/removing tests, lowering severity without evidence, deleting findings, changing immutable AUD/REM records, destructive operations, or authority not granted by the user.
5. A plan-scoped loop finds a systemic issue and `ESCALATE=stop`.

On a circuit breaker, pause the goal when goal controls support it, keep all records and indexes intact, and report the exact cycle history, unresolved findings, repeated evidence, and decision needed. Do not mark the goal complete.

## Completion

Complete the goal only when the newest follow-up AUD independently verifies all findings in scope, all relevant AUD entries are `none`, `verified-by:*`, `continued-by:*`, or `accepted-risk` with no active successor in scope, every REM is no longer pending, and the underlying validation gates pass.

Report the goal outcome, cycle count, AUD -> REM -> follow-up AUD chain, final verification, and any accepted residual risk. Never manufacture completion by changing index state without the corresponding underlying skill output.
