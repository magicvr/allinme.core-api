---
title: AGENTS 模板 · 目标治理 AI 规则
status: active
created: 2026-07-18
updated: 2026-07-24
parent: null
version: 0.9.0
---

# AGENTS.md

> **使用说明**：复制到目标仓库根目录并命名为 `AGENTS.md`。  
> 将 `{{...}}` 替换为项目真实信息后生效。未使用的可选节可删除。

面向在本仓库工作的 AI 助手（及人类协作者）。**以下规则必须遵守。**

## 1. 文档真相来源

| 内容 | 路径 | 要求 |
|------|------|------|
| 目标与过程记录 | `docs/workspace-<NNN>-<slug>/` | 当前工作区内的唯一长期存储 |
| 目标树与状态 | `<workspace-root>/goal-tree.md` | **必读、必更新** |
| 核心方法论（architecture） | `docs/architecture/` | **与 Skills 同级必备**（install 默认安装）；含 principles、workspace-protocol 等 |
| 治理原则全文 | `docs/architecture/principles.md` | **必备**；P-001～P-005 权威长文；AGENTS §6/6b 为操作摘要 |
| 文档使用规范 | `docs/README.md` | **必备**（install 默认精简入口） |
| 核心模板 | `docs/templates/`（或 `{{CORE_TEMPLATES_DIR}}`） | **必备**；创建五件套与 workspace 上下文 |
| 工作区与共享资料协议 | `<workspace-root>/workspace.md`、`docs/architecture/workspace-protocol.md` | workspace.md 存在时必读；protocol **必备**；目标状态仍以该工作区根为准 |

冲突时以已验证的工作区 canonical root 与本文件为准。

## 2. 目标存储与编号

1. **工作区内扁平存储**：所有目标文件夹平铺在 `docs/workspace-<NNN>-<slug>/` 根，**禁止**用子文件夹表达父子关系。
2. **Root**：每个工作区的 `GOAL-001` 固定为总目标，其 `parent` 必须为 `null`；禁止改号。
3. **编号**：先读当前工作区 `goal-tree.md`（或扫描其 canonical root），新编号 = 当前最大编号 + 1，三位数字（如 `004`）。
4. **文件夹名**：`GOAL-NNN-short-slug`（`NNN` 三位；slug 小写英文、短横线）。
5. **`id` = 文件夹名**：`00-meta.md` 的 `id` 必须与文件夹名完全一致（如 `GOAL-004-foo-bar`）。
6. **层级唯一来源**：仅通过各目标 `00-meta.md` 的 `parent` 字段维护。
   - 值为**父目标完整 id**（含 slug，例：`GOAL-001-your-root-slug`），Root 为 `null`。
   - Root 的 slug **由项目自定**（不要照搬其他仓库的 `main-vision` 等示例名）。
   - **禁止**用目录嵌套、文件名或正文标题充当层级真相。

## 3. 目标五件套（创建时一次建齐）

```text
docs/workspace-001-example/GOAL-NNN-short-slug/
├── 00-meta.md
├── 01-decision.md
├── 02-execution.md
├── 03-audit.md
└── attachments/          # 可为空，目录必须存在
```

- 不得省略任一文件或目录。
- 若项目提供独立核心模板层，优先从 `{{CORE_TEMPLATES_DIR}}/goal-folder/` 复制；否则从 `{{GOAL_FOLDER_TEMPLATE}}` 复制（常见：`<skills-pkg>/templates/goal-folder/`；包目录名可能不是 `skills`）。

## 4. Frontmatter 最低要求

每个 Markdown 至少包含：

| 字段 | 说明 |
|------|------|
| `status` | 见下表 |
| `created` | `YYYY-MM-DD` |
| `updated` | 修改内容时更新为当日 |
| `parent` | 目标：父目标完整 id 或 `null`；非目标文件可用 `null` |
| `version` | 文档版本号 |

`00-meta.md` **必须**另含：`id`、`title`；**建议**含 `progress`（如 `50%`）。

### status 取值

| 值 | 含义 |
|----|------|
| `draft` | 草稿，未正式启动 |
| `active` | 进行中 |
| `blocked` | 阻塞 |
| `done` | 已完成 |
| `cancelled` | 已取消 |

## 5. 内容写作要求

| 文件 | 写什么 | 禁止 |
|------|--------|------|
| `01-decision` | 决定了什么 + 为什么；重要取舍写未选方案 | 编造未发生的决策 |
| `02-execution` | 按时间线记**事实**（做了什么、产物路径、进度评估） | 虚构未完成工作 |
| `03-audit` | 阶段复盘与**全部**审计意见（`self` / `independent`）：编号节、source、verdict、成果/偏差/findings；长文可链到 `attachments/` | 无复盘节点硬写「已完成」；独立意见只留聊天不落盘；仅附件无索引节 |

不确定标注「待确认」。语言简洁真实。

## 6. 目标可执行性与路线图（P-001）

**判定**：范围大、步骤不明、或明显需要拆成多个可独立交付的子目标 → 视为「尚不可直接执行」。

**强制顺序**：

1. **禁止**在尚不可直接执行时，直接批量创建细粒度子目标并开工。
2. **必须先**写可追踪的高层路线图：主要阶段 + 先后关系（可含完成标记）。
3. 路线图写在该目标的 `00-meta.md` 或 `01-decision.md`，并随进展更新。
4. 路线图就位后，再**按阶段**创建与执行具体子目标。
5. 已可直接执行的小目标**无需**强行补路线图。

原则以**本文件（AGENTS）第 6 节**为操作入口；**全文**以 `docs/architecture/principles.md` 为准（**完整安装必备**）。  
Skills 与核心方法论**同级必备**：缺 `docs/architecture/` 视为不完整安装，应先补 core（重跑 install 或从包内 `core/docs` 复制），不得当作可跳过。

## 6b. 治理闭环、交叉审计与信息就绪（P-002～P-005）

操作摘要如下；**全文**见 `docs/architecture/principles.md`。  
降级兜底：若 principles 文件暂时缺失，仍须遵守本小节，并在推进前报告不完整安装——**这不是「architecture 可选」产品定位**。

### P-002 · 阶段质量意识

- 目标态：设立 → 信息发现与就绪判断（P-005）→ 审视目标 → 方案/计划 → 审视方案 → 实施并记**事实** → 审视事实（可整改环）→ 关门审计后结项。
- 小目标可合并审视步骤；不得省略可验证事实与关门前结论。
- 实施事实 ≠ 实施流水账；审计须能指回证据。

### P-003 · 交叉审计与意见响应

- **独立/交叉审计**可在编排流程外进行；默认**只写审计意见**，不直接改 `status`/`progress`/方案正文。
- **编排器**汇总并响应与焦点相关的**全部**意见（`self` + `independent`），驱动修正/复审/推进。
- **开放必改门禁**：存在**未关闭**的 required / 必改 findings（不论 self 还是 independent、不论是否与其它意见冲突）时，**不得**推进该门禁对应的下一阶段，**不得**将目标标为 `done`（关门）。仅汇总意见而不关闭必改项 = 违规放行。
- 默认主入口仍为编排器；交叉审计为专用入口（如 `/audit`），非四填表并列主路径。

#### 审计意见落盘（强制）

| 项 | 约定 |
|----|------|
| **权威位置** | 被审目标的 `03-audit.md`（唯一正式台账） |
| **编号** | `A-001` 起递增，自审与独立审**共用**序列 |
| **条目头** | 至少：`source`（`self` \| `independent`）、日期、scope、`verdict`（pass \| conditional \| fail） |
| **长文** | 可放 `attachments/audit-A-00N-….md`，但 `03-audit` **必须**有对应编号节（摘要 + verdict + 链接） |
| **禁止** | 仅聊天未写入；仅附件无 `03-audit` 节；用全局目录替代目标下 `03-audit` |
| **写入** | 交叉工具直接追加，或代贴并保留 `source: independent` |

#### 意见状态（最小约定；细节由阶段 B 提示词细化）

| 概念 | 最小判定 |
|------|----------|
| **相关意见** | 同一目标 `03-audit` 中，scope 覆盖当前推进焦点（目标整体 / 某阶段 / 某门禁）的 A-00N 条目 |
| **开放必改** | 条目中标为 required/必改、且尚未在执行/决策/后续审计中写明关闭证据的 finding |
| **已关闭** | 有可核对修正事实（路径/决策号）+ 可选复审确认；仅口头「已改」不算关闭 |
| **冲突** | 同范围下 verdict 相反，或对同一必改项一要一否（见 P-004） |

### P-004 · 用户裁决点（必须询问，禁止静默自动裁）

| 情形 | 行为 |
|------|------|
| 已有独立审计、尚无自审计 | **询问**用户是否还要自审；不自动跳过、不未问即强制 |
| 多条意见在结论/必改项上冲突 | 展示冲突 + **给建议** + **等用户决策**并留痕；未决不放行/不关门 |

**延后**：自动判定「可否跳过自审」的复杂机制（版本指纹、覆盖度算法等）。

### P-005 · 信息就绪与未知项门禁

- **允许带未知立项**：目标不必在设立时已知所有信息；但必须能写明意图、初始边界、父级与最小可验证方向。已识别的未知、假设或待验证事实不得伪装为决策、成功事实或已关闭风险。
- **信息需求登记**：在目标的 `00-meta.md` 或 `01-decision.md` 维护信息项，至少含：编号、要回答的问题/所需信息、`required` 或 `non-blocking` 级别、影响的决策或门禁、最晚需要阶段、验证/收集动作、状态、延期/复核和证据或结论。`deferred` 必须有延期理由、责任人和下一复核日期或触发条件。
- **阶段门禁**：影响方案冻结、实施、验收或关门的 `required` 信息项，必须在对应阶段前由证据关闭。经用户裁决并记录的有界实验只可进入明确的信息收集范围，信息项仍为 `collecting`；唯一能解除明确门禁的例外是用户书面接受的、有范围和复审触发条件的残余风险，接受不等于信息已经验证。
- **发现后的回流**：实施中发现新的关键未知时，暂停受影响范围，记录事实，并回到信息登记、决策或路线图；信息冲突、是否以有界实验收集信息、或是否接受残余风险由用户按 P-004 裁决。到达最晚需要阶段时，未获 residual 接受的 `deferred required` 视为开放 required 并阻断受影响门禁。
- **按规模拆分**：先登记并设定门禁；只有澄清/收集工作具有独立范围、依赖、交付证据或并行价值时，才创建“信息澄清/验证”或“信息收集”子目标。禁止为每个低风险问题机械创建两个子目标。

## 6c. 工作区与共享资料边界

先定位当前 `docs/workspace-<NNN>-<slug>/workspace.md`，再按 `docs/architecture/workspace-protocol.md`（完整安装必备）校验其 `root_goal`、`canonical_scope` 和共享资料引用。多个工作区而用户未指定焦点时必须 fail closed：

1. 工作区绑定一个 `parent: null` 的 Root Goal 与其 canonical 目标范围；它不是 `parent` 层级、审计 scope 或第二套状态。
2. 同一项目的 MVP、二阶段、三阶段等通常更新 Root Goal 路线图并建立串行子目标；只有长期目的、成功边界或战略方向实际变化时，才记录决策后改写 Root Goal 定义。
3. 没有显式工作区根、但存在 `docs/goals/` 时，只按该 legacy 目录使用隐式单工作区；禁止自动发现、读取、混合或写入其他工作区上下文。
4. 共享资料只能以匹配当前 `workspace_id` 的 `material_id`、`source`、`version` 和有效 `sha256` 固定引用。引用缺失/不匹配、资料目录为 `none` 或来源不可固定时，必须 fail closed；资料内容仍须经用户确认才可成为事实、证据或 finding 关闭依据。
5. 本协议不自动放行共享资料物理存储、用户 CRUD、AI 读取执行、跨工作区导航、Web 写入或访问安全模型；这些留给对应目标的信息门禁与验证。

## 7. 必须同步更新 goal-tree.md

以下任一操作后，**必须**更新当前工作区的 `goal-tree.md`：

- 新建目标
- 修改 `status` / `progress`
- 修改 `parent`（调整树）
- 完成或取消目标
- 重命名文件夹或 slug（并修正所有引用）

更新内容至少包括：**ASCII/文本树** + **状态表格**。  
只改单目标文件、不更新 goal-tree → **视为任务未完成**。

## 8. 代码与文档边界

- **目标真相源**：长期过程记录在已验证工作区的 canonical root。业务代码与 UI 可以引用目标，但不得建立全局 `docs/goals/` 或第二状态源。
- **代码布局（默认策略）**：
  - 默认：应用/库代码可在**仓库根**，或按该语言/生态惯例分布。
  - 若项目已约定子目录（如 `web/`、`app/`、`services/`）：按该约定；`{{APP_DIR}}` 仅在有约定时填写。
  - 刚装本包、文件很少时：项目性质与代码路径标为待确认，**问用户**或读已有 README/架构；目录观察只作参考。
- **语言与日期**：标题/正文跟随用户语言；slug 建议小写英文短横线；日期用会话/系统 `YYYY-MM-DD`。
- **architecture**：完整安装必须具备；缺失时先补 core 再推进治理写入。改治理元规则时先更新 `docs/architecture/` 再改实现。

## 8b. Skills 包路径

- 常见目录名 `skills/`，也可改名。
- **定位 SKILLS_PKG**：含 `prompts/00-govern-orchestrator.md`（或 `prompts/01-create-new-goal.md`）的目录。原语与模板相对该根。
- `{{SKILLS_DIR}}` = 包根相对仓库根的路径。

## 9. 交付形态（按项目裁剪）

默认：**文档驱动的目标治理**；代码与可视化应用按项目实际叠加。

1. **文档体系（本包约定）**：`docs/workspace-<NNN>-<slug>/` + 工作区内 `goal-tree.md`
2. **产品/代码（常见）**：仓库根或项目实际目录
3. **独立可视化应用（可选）**：有则按项目路径
4. **Skills 包（可选）**：`{{SKILLS_DIR}}`

## 9b. Skills 主入口（若已安装本包）

- **编排主路径**：`{{SKILLS_DIR}}/prompts/00-govern-orchestrator.md` → **`/govern`**。  
  扫描 → 意见台账 → 分类 → P-004 裁决 → 提议 → 确认 → 原语 `01`～`04`。
- **交叉审计**：`{{SKILLS_DIR}}/prompts/05-independent-audit.md` → **`/audit`**（只出意见，不改 status；响应归 `/govern`）。
- advanced 填表 slash 可选（`--with-primitives`）。
- **P-001** 以本文件第 6 节为准；**P-002～P-005** 以第 6b 节为准；**全文**以 `docs/architecture/principles.md` 为准（必备）。

## 10. 变更工作流

```text
1. 定位当前工作区 `workspace.md` → 校验 Root Goal/canonical 范围/资料引用；再读该工作区 goal-tree.md → 编号、parent、未关门目标
2. 未指定原子操作时 → 优先编排器
3. 尚不可直接执行 → 先高层路线图（P-001）；存在影响门禁的未知 → 先登记信息需求与最晚需要阶段（P-005）
4. 推进时检查相关审计意见与信息就绪门禁；P-004 情形先询问用户
5. **若存在未关闭 required/必改项 → 先响应/修正，不得假装放行或关门**
6. 创建或修改五件套
7. 更新 goal-tree.md（树 + 表）
8. 项目已有 docs/README、architecture 等时再按需更新
9. 再改代码或 Skills（路径以项目实际为准）
```

步骤 **1、3–7 强制**；编排优先；8–9 按影响面。

## 11. 正确做法与硬约束

**正确做法**

- 层级：平铺文件夹 + `parent` 完整 id。
- 改 status/progress/parent/新建：同步 goal-tree 树与表。
- 大目标：先路线图，再按阶段建子目标。
- 执行/审计：只写有证据的事实；计划单独标注。
- 代码布局与 Root slug：默认见第 8 节；以用户/项目约定为准（`web/` 等为可选约定示例）。
- Skills 包：按内容定位 SKILLS_PKG。
- P-001：本文件第 6 节；P-002～P-005：第 6b 节；architecture 原则全文**必备**（与 Skills 同级）。
- 空仓 S0：先 scaffold `docs/workspace-001-<用户确认 slug>/`（workspace.md + goal-tree），再创建 Root；禁止静默默认 slug。
- 目标可带未知立项，但信息项、阶段门禁、证据与残余风险接受必须可追踪；按工作量而非固定“两子目标”拆分。
- 交叉审计意见由编排器统一响应；冲突与「是否自审」问用户并给建议。

**硬约束**

- Root 编号保持 `GOAL-001`；`parent: null`。
- 新建一次建齐五件套。
- 决策、执行、审计只记录真实内容。
- 不静默自动裁决 P-004 情形；独立审计默认不直接改目标状态。
- 不得以“以后再说”绕过 required 信息门禁；残余风险只有用户书面接受并留痕后才可解除其明确范围内的门禁。
- 正式审计意见必须落在被审目标 `03-audit.md`（可链附件）；未落盘意见不作为放行依据。
- 未关闭的 required/必改 findings 存在时，禁止推进对应门禁或 `status: done`。

## 12. 完成前检查清单

- [ ] 编号未冲突；`id` = 文件夹名
- [ ] `parent` 为完整父 id 或 `null`
- [ ] 五件套齐全（若新建）
- [ ] 大目标路线图已写/更新（若适用）
- [ ] 若存在工作区上下文，Root Goal/canonical 范围已校验；共享资料引用未被当成跨工作区状态或未确认事实
- [ ] 已识别的未知项已登记；本次要推进的阶段没有开放 required 信息门禁，或残余风险已获用户书面接受
- [ ] `goal-tree.md` 已同步
- [ ] `updated` / `progress` / `status` 与事实一致
- [ ] 若涉及推进/放行：相关审计意见已汇总；P-004 已询问用户（若适用）
- [ ] 无未关闭的 required/必改 findings（或已获用户书面接受并留痕的 residual 清单）

## 写法对照（简表）

| 推荐 | 说明 |
|------|------|
| 平铺 + `parent: GOAL-001-<slug>` | 完整 id |
| 改 progress 同时改 goal-tree | 两处一致 |
| 大目标先路线图 | 再按阶段立项 |
| 计划与已完成分开写 | 时间线只记事实 |
| 复制模板后改真实 id | 例如勿留 GOAL-042 |
| 按 prompts 文件定位包目录 | 包名可以是 `skills` 或其他 |
| 独立审只出意见；编排器响应 | P-003 |
| 审计意见写入被审目标 `03-audit.md`（A-00N + source） | P-003 落盘 |
| 冲突 / 是否自审 → 问用户 + 建议 | P-004 |


## 快速链接（按项目填写）

- 文档说明：`{{DOCS_README_PATH}}`
- 目标树：`{{WORKSPACE_ROOT}}/goal-tree.md`
- Root Goal：`{{WORKSPACE_ROOT}}/{{ROOT_GOAL_FOLDER}}/00-meta.md`
- 核心模板目录：`{{CORE_TEMPLATES_DIR}}`（若项目采用独立核心层）
- 架构说明：`{{ARCHITECTURE_PATH}}`
- 治理原则：AGENTS 第 6 / 6b 节；`docs/architecture/principles.md`（必备，P-001～P-005 全文）
- 代码/应用布局：仓库根为常见默认；若已约定子目录则填 `{{APP_DIR}}`（可空）
- Skills 目录：`{{SKILLS_DIR}}`
