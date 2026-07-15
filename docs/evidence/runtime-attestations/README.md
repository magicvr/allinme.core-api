# Runtime context attestations

This directory stores immutable, signed envelopes supplied by the runtime adapter. Repository code and audit agents must never possess the signing key.

Every governance record added after the validation `HistoryBase` keeps the repository's current workflow contract and references one envelope as `runtime_context_attestation`. Independent records also list the exact source envelopes in `source_context_attestations`. Historical unattested records remain read-only compatible. The signed payload binds the repository origin, exact record ID/path, execution context, runtime task reference, task and parent identities, record scope/type, governance baseline, and a lifetime of at most 24 hours. Independent records must list the exact signed source runtime refs as well as the exact source envelopes.

Validation requires an external trust anchor through `AUDIT_RUNTIME_TRUSTED_KEY_SHA256` and either `AUDIT_RUNTIME_PUBLIC_KEY_PATH` or `AUDIT_RUNTIME_PUBLIC_KEY_BASE64`. If the runtime adapter or trust anchor is unavailable, creation and acceptance of new governance records must stop; a UUID or editable runtime reference is not evidence of isolation.

Envelope shape is defined by `../../tools/runtime-context-attestation.schema.json`. The base64 payload is signed exactly as supplied with RSA/SHA-256. Each envelope is single-use and must be committed with the open governance checkpoint.

## Revision evidence attestations

Every `status: closed` audit added after an explicit validation `HistoryBase` with `governance_contract: audit-loop/v3` and `workflow_contract_revision: audit-runtime/v1` must also reference:

```text
evidence_attestation: docs/evidence/runs/<evidence_run_id>/attestation.json
```

The envelope follows `../../tools/revision-evidence-attestation.schema.json` and uses the same externally supplied RSA/SHA-256 public key and `AUDIT_RUNTIME_TRUSTED_KEY_SHA256` trust anchor as runtime-context attestations. The private key must never enter the repository, an audit task, or a child agent. Historical records already present at `HistoryBase` remain read-only compatible; new closed records fail closed when the key, trust anchor, envelope, signature, or required binding is absent.

The signed payload binds the canonical repository origin, audit ID and exact record path, run ID, exact artifact path, SHA-256 of the artifact's original bytes, evidence commit/tree, ordered argv, exit code, the approved image `docker.io/library/golang@sha256:349ad04971da5f200a537641ae2c70774a592ca21fad4b513b65f813f546781a`, image ID, and an issuance interval no longer than 24 hours. The envelope and attestation ID are single-use for one audit record.

The external signer must directly execute the isolated runner or obtain the artifact and run facts through a trusted observation channel under its control. It must recompute the artifact byte hash and compare the signed execution fields with the observed runner result. It must reject blind-signing arbitrary JSON, hashes, image claims, or payloads submitted only by the audit author; otherwise the signature is merely a signed author assertion rather than independent evidence.
