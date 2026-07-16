# 文档工具

- `validate.ps1`：默认文档结构门禁，检查 Markdown frontmatter、相对链接、计划/checklist 配对、AUD/REM/IMP 命名与索引引用、必要审计矩阵、终态记录不可改写，以及 `git diff HEAD --check`。
- `validate.tests.ps1`：用最小 fixture 验证默认文档 validator 的结构检查和主要失败路径。
- `validate-audit-workflows.ps1`：可选的治理工作流维护检查，验证 9 个 prompt/skill 的映射与闭环约束；只在修改 `.github/prompts/backend-*`、`.agents/skills/backend-*` 或该脚本时运行，不是产品 PR 默认门禁。
- `reserve-governance-record.ps1`：在本机并发创建记录时预留 AUD/REM/IMP 编号；不参与审计结论可信度证明。

默认从仓库根目录运行：

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.tests.ps1
```

维护治理工作流时额外运行：

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate-audit-workflows.ps1
```

这些工具只防止机械结构错误，不尝试理解任意自然语言计划的业务语义，也不证明审计者、运行环境或历史命令是可信的。
