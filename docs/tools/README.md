# 文档工具

- `validate.ps1`：检查 Markdown frontmatter、相对链接、`PLN` / `IMP` / `AUD` / `REM` 命名与元数据、对象引用图、plan/checklist 配对、revision-bound 计划审计、实施/审计/整改索引、验收矩阵与 verdict 状态机、subject-specific 独立验证、状态跳转引用、真实 subject revision、最新 IMP、计划漂移、IMP/REM effective revision 链、source 终态快照、审计工作流契约和跨提交只追加历史，并运行 `git diff HEAD --check`。
- `validate-audit-workflows.ps1`：只读检查两个闭环及其原子 prompt/skill 是否保留真实 runtime task 隔离、事务化 open/subject/terminal commit、持久 loop state 双 CAS、严格串行 child、clean `governance_revision`、足够 cycle budget、single-transition child 路由和 `TARGET`/`ADVANCE_SET`/`PEER_SET` 分离。
- `validate-governance-history.ps1`：从显式 `HistoryBase` 到 `HEAD` 检查新建以及基线中已存在的 open AUD/REM/IMP；终态记录必须有合法 terminal commit，AUD 的 evidence revision 必须位于 open 之前，非 blocked/superseded REM/IMP 必须形成严格的 open/result/terminal 祖先链，并限制各阶段提交路径。它还直接固定 HistoryBase 到 HEAD 的 runtime attestation blob，即使 record 文件未变化也拒绝替换，并拒绝仓库内 `git` 可执行文件劫持。
- `validate-runtime-attestations.ps1`：相对 `HistoryBase` 拒绝任何新增但未签名、降级合同或复用 `execution_context_id` 的 AUD/REM/IMP；使用外部固定公钥验证单次 runtime attestation，绑定 repository、record ID/path、task/parent、scope、baseline 与 context，并要求独立审计列出精确 signed source 集合。仓库不得保存私钥。
- `validate-evidence-attestations.ps1`：相对 `HistoryBase` 对新增关闭 AUD 验证外部 `revision-evidence-attestation/v1`，把签名绑定到 artifact 原始字节 SHA-256、run/revision/tree/argv/exit/image 与审计路径；缺少 trust root、签名或直接 runner 观察时失败关闭。
- `validate.tests.ps1`：用独立 fixture 验证合法计划、实施、审计、整改和验收结构通过，并拒绝未绑定 revision 的计划审计、治理命令冒充 subject 验证、partial REM 重复排队、倒置时间、多计划单 verdict、旧 IMP 验收、伪造状态跳转、漏列 effective REM、脏验收基线、失效对象引用和非法治理结构。
- `reserve-governance-record.ps1`：在 Git common directory 共享互斥区内分配 `AUD` / `REM` / `IMP` 编号，通过 common directory 中的持久 reservation 与原子 `CreateNew` 预留目标文件，避免 linked worktree 或并发执行复用同一 ID。
- `invoke-governance-transaction.ps1`：要求空 index、精确文件 allowlist 和无关改动为零；在 Git common directory 共享锁内创建提交，并以当前 HEAD 与 `refs/allinme/governance-head` 双 CAS 原子推进分支和全仓治理链，拒绝 linked worktree 治理分叉。
- `update-loop-run-state.ps1` / `governance-loop-run.schema.json`：把闭环不可变 workflow/集合/模式/上限、cycle 和 per-plan fingerprint/stagnation/blocker 状态持久化到 Git common directory；初始化绑定当前治理 revision，更新时同时校验 generation 与 previous governance SHA。
- `invoke-revision-evidence.ps1`：从指定 commit 生成不含宿主 `.git`/ignored 文件的净化 archive snapshot，在固定 digest 镜像、显式安全 entrypoint、资源/超时/输出上限中执行 subject-specific 命令，输出 exact revision、tree、snapshot manifest、argv、exit code 和运行后 clean 状态。

从仓库根目录运行：

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate-audit-workflows.ps1
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1 -HistoryBase <merge-base-full-sha>
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate-governance-history.ps1 -HistoryBase <merge-base-full-sha>
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate-runtime-attestations.ps1 -HistoryBase <merge-base-full-sha>
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate-evidence-attestations.ps1 -HistoryBase <merge-base-full-sha>
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.tests.ps1
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate-governance-history.tests.ps1
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/reserve-governance-record.ps1 -Kind AUD -Suffix 20260714-codex-plan-example
& docs/tools/invoke-governance-transaction.ps1 -ExpectedHead (git rev-parse HEAD) -Paths @('docs/audits/records/AUD-NNNN-example.md','docs/audits/README.md') -Message 'audit: open AUD-NNNN'
& docs/tools/update-loop-run-state.ps1 -Operation Read -RunId <stable-run-id>
& docs/tools/invoke-revision-evidence.ps1 -Revision HEAD -Command git -CommandArgs @('rev-parse', 'HEAD')
```

CI 必须使用完整 Git 历史，并通过 `AUDIT_HISTORY_BASE` 或 `-HistoryBase` 传入 PR merge-base/push 前一 revision，同时运行当前树校验和独立治理历史校验；否则不能证明新记录实际经历了 open/subject/terminal 三阶段，也不能证明终态 AUD/REM/IMP 在多提交分支中未被改写。新增治理记录还要求仓库外配置 `AUDIT_RUNTIME_PUBLIC_KEY_BASE64` 与 `AUDIT_RUNTIME_TRUSTED_KEY_SHA256`；缺失时签名 validator 必须失败关闭。
