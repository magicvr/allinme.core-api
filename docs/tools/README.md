# 文档工具

- `validate.ps1`：检查 Markdown frontmatter、相对链接、`PLN` / `AUD` / `REM` 命名与元数据、plan/checklist 配对、审计/整改索引、工作流入口及 `git diff HEAD --check`。
- `validate.tests.ps1`：用独立 fixture 验证合法结构通过，且未索引审计、缺失链接和非法治理结构失败。

从仓库根目录运行：

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.tests.ps1
```
