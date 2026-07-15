# 版本化 Evidence 摘要

此目录保存计划、审计或发布流程明确要求提交的小型、脱敏、可复核 Evidence 摘要和 manifest。大型产物、数据库、原始日志和敏感数据不得提交；其不可变 revision、SHA-256、保留期和受控下载位置写入摘要。

## Revision evidence

`docs/tools/invoke-revision-evidence.ps1` 把任意 argv 放入本地已有、按 digest 固定的 Docker 镜像执行。运行前宿主 worktree 必须干净（允许既有 `docs/evidence/runs/` 输出）；否则直接退出且不创建 artifact。运行时强制 `--network none`、只读的精确 Git archive snapshot（不挂载宿主 worktree、`.git` 或 ignored 文件）、只读容器 rootfs、`--cap-drop ALL`、`no-new-privileges`、非 root 用户、CPU/内存/PID/file limits 和脱敏环境。snapshot 与 Go cache 位于临时 tmpfs，运行结束即销毁；Docker CLI、daemon 或固定镜像不可用时 runner 失败，绝不退回宿主执行或联网拉取。runner 同时强制 `/usr/bin/env` entrypoint、wall-clock timeout 和 bounded streaming output，并把 snapshot manifest/digests、limits、failure_kind 和 cleanup 状态写入 artifact。

每次运行必须使用唯一 UUIDv4 `evidence_run_id`，并只追加创建：

```text
docs/evidence/runs/<evidence_run_id>/evidence.json
```

`evidence.json` 使用 `schema: revision-evidence/v1`，至少记录 exact commit/tree、原始 argv、退出码、固定 image digest/image ID、隔离参数、stdout/stderr/combined SHA-256 与字节数、subject/host clean 状态和 UTC 时间。Docker preflight 或容器 bootstrap 失败时仍写入 `exit_code: 125`、`preflight_passed: false` 的闭锁 artifact，便于追踪失败尝试，但该 artifact 绝不能支持通过或关闭结论。runner 不保存原始输出；审计者必须先检查命令及输出是否包含凭据、token、个人数据或大型内容，再决定是否把必要的脱敏摘要写入审计正文。

审计 frontmatter 通过以下字段一一绑定 artifact；`evidence_run_id` 与路径 UUID 必须相同：

```yaml
evidence_run_id: 00000000-0000-4000-8000-000000000000
evidence_artifact: docs/evidence/runs/00000000-0000-4000-8000-000000000000/evidence.json
evidence_revision: git:full-commit-sha; worktree:clean
evidence_worktree_revision: git:full-commit-sha
evidence_runner: docs/tools/invoke-revision-evidence.ps1
evidence_argv_json: ["<subject-command>", "<arg-1>"]
evidence_attestation: docs/evidence/runs/00000000-0000-4000-8000-000000000000/attestation.json
```

### Signed evidence binding

Every new `audit-loop/v3` + `audit-runtime/v1` audit must declare `evidence_argv_json` as strict JSON containing the exact ordered, non-empty string argv that the runner will execute. For a closed audit, this declaration must byte-for-value match `evidence.json.argv` and the signed payload's `argv`; prose command claims are not evidence and cannot replace this field.

Every new closed `audit-loop/v3` + `audit-runtime/v1` audit must include the external `revision-evidence-attestation/v1` envelope at the exact `evidence_attestation` path. The trusted runtime/CI signer must directly execute or observe the runner; it must never blindly sign caller-supplied JSON. Its signed payload binds the repository origin, audit ID/path, run ID, artifact byte SHA-256, exact revision/tree/argv/exit code, approved image digest/image ID, and a bounded lifetime. Validation requires `AUDIT_RUNTIME_TRUSTED_KEY_SHA256` plus `AUDIT_RUNTIME_PUBLIC_KEY_PATH` or `AUDIT_RUNTIME_PUBLIC_KEY_BASE64`; the private key never enters this repository.

The runtime-context attestation is committed with the open record checkpoint. The primary `evidence.json` and its `attestation.json` are committed with the terminal audit/index transaction. Missing trust roots, signatures, exact paths, or byte bindings fail closed. Historical records before `HistoryBase` remain read-only compatible.

所有新合同下的计划审计、实施审计、整改复审及两类验收都必须绑定自己的 artifact；不得复用 run ID，不得手工改写已生成 artifact，不得把失败命令改写为成功证据。单次审计若执行多个 subject 命令，应为每个命令生成独立 run ID，并在正文列出附加 artifact；frontmatter 指向决定该审计终态的主 artifact。

其他 Evidence 可按计划或功能建立子目录，例如 `phase5/<run-id>/`。审计记录只链接 Evidence，不在 `audits/` 内存放产物。
