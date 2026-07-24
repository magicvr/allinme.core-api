---
title: 提示词 · 创建新目标
status: active
created: 2026-07-18
updated: 2026-07-24
parent: null
version: 0.5.0
role: primitive
---

# 01 · 创建新目标（原语 / primitive）

## 说明

供 [00-govern-orchestrator.md](00-govern-orchestrator.md) 在用户确认「需要新建目标」后调用；也可高级直调。  
日常默认路径请用编排器。

交付物：正确编号的目标文件夹 + 完整五件套 + 已同步的 `goal-tree.md`。  
若尚无显式工作区（S0），须**先** scaffold 工作区再创建 Root（见步骤 0）。

---

## 提示词正文

```markdown
# 角色

你是本项目的目标治理协作者。遵守 `AGENTS.md` 和/或 `.github/copilot-instructions.md`。  
P-001 与 P-005 以 AGENTS 为准；**全文**见 `docs/architecture/principles.md`（与 Skills 同级必备）。缺失时报告不完整安装，不得当作「可选跳过」。

# 任务

按已确认信息创建一个新目标：五件套齐全，goal-tree 已更新。  
创建 Root 且尚无工作区时：先 scaffold 工作区骨架。

# 用户输入（缺项先问清再写）

- 目标标题：【用户语言】
- 英文短 slug（小写、短横线）：【如 improve-auth】；**Root 与工作区 slug 必须用户确认，禁止静默占位**
- 父目标完整 ID：【如 GOAL-001-my-root-slug；Root 则 null】
- 工作区：【当前 `docs/workspace-<NNN>-<slug>/workspace.md` 的 id 与路径；若尚无则先收集 workspace-slug 再 scaffold】
- 一句话概述：
- 初始成功标准（2～5 条可验证项；可标“暂定”）：
- 是否需要高层路线图？【是 / 否】
- 已识别的信息需求 / 假设：【I-00N 等；无则写“当前未识别”】
- 是否存在到期 required 信息门禁？【是 / 否】
- 初始状态：draft 或 active（默认 draft）
- 今日日期：【YYYY-MM-DD】

# 步骤

0. **工作区骨架（仅当需要新建显式工作区 / S0）**  
   - 确认用户给出的 `workspace-slug` 与 Root slug（禁止擅自 `main-vision` 等）。  
   - 创建 `docs/workspace-001-<workspace-slug>/`（首工作区 NNN=`001`，除非用户指定其他编号）。  
   - 从模板复制 `workspace.md`：优先 `docs/templates/workspace-context.md`，否则 `<SKILLS_PKG>/core/docs/templates/workspace-context.md` 或 `<SKILLS_PKG>/templates/workspace-context.md`。  
   - 写入 frontmatter：`id`、`root_goal`（即将创建的 Root 完整 id）、`canonical_scope`（该工作区路径，以 `/` 结尾）、`shared_materials_catalog`（默认 `none`）、status/created/updated/version。  
   - 确保 `goal-tree.md` 存在（可先空壳，步骤 8 写满）。  
   - **不要**在新项目默认使用 legacy `docs/goals/`。

1. 定位当前工作区 `workspace.md` 与 `goal-tree.md`：校验 `root_goal` / `canonical_scope`。  
   - 创建**非 Root** 子目标时：工作区必须已存在且绑定正确。  
   - 仅当用户明确维护旧仓、且无显式工作区根时，才使用 legacy `docs/goals/`。  
2. 新编号 = 当前工作区最大编号 + 1（三位）。Root 固定 GOAL-001。  
3. 创建 `<workspace-root>/GOAL-NNN-<slug>/`（平铺）。  
4. 一次写入五件套：`00-meta` / `01-decision` / `02-execution` / `03-audit` / `attachments/`。  
5. 模板源：优先 `docs/templates/goal-folder/`；否则 `<SKILLS_PKG>/core/docs/templates/goal-folder/` 或 `<SKILLS_PKG>/templates/goal-folder/`。  
6. Frontmatter：status, created, updated, parent, version；meta 另含 id、title（建议 progress）。Root 的 slug = 用户确认名。  
7. 正文：meta 概述/成功标准/路线图/信息概览；decision 取舍与信息项；execution 仅事实；audit 可写尚未到复盘节点。  
8. 更新 `goal-tree.md`（树 + 表）。若本目标为 Root，确保 `workspace.md` 的 `root_goal` 与 id 一致。  
9. 如需，在父目标文档轻量提及新子目标。

# 完成标准

- [ ] 编号无冲突；id = 文件夹名  
- [ ] 五件套齐全；parent 为完整 id 或 null  
- [ ] goal-tree 树与表已更新  
- [ ] 显式工作区：workspace.md 存在且 Root/canonical 一致；新目标未越界  
- [ ] 未在未确认 slug 时静默命名工作区或 Root  
- [ ] 大目标已写路线图（若适用）  
- [ ] 已识别信息项已登记；到期 required 未伪装成已验证  
- [ ] 内容真实，无虚构完成项  
```

---

## 使用注意事项

- S0 由编排器主导：先确认 slug → scaffold 工作区 → 再本原语建 Root。  
- 缺信息时先确认并登记（P-005）。  
- Root：`GOAL-001` + `parent: null`；slug 由用户定。
