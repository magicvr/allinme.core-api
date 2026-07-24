---
title: Skills · 目标治理可复用包
status: active
created: 2026-07-18
updated: 2026-07-24
parent: null
version: 1.4.0
---

# Skills

本目录提供可复制到**其他项目**的目标治理约定与模板。  
本仓库运行中的强制规则仍以根目录 [AGENTS.md](../AGENTS.md) 为准；此处是提炼后的**可复用交付物**。

Skills 是核心方法论的 **AI 消费适配器**。**核心方法论与 Skills 同级必备**（GOAL-019 D-003）：包内 [`core/`](core/) 为消费分发镜像，`install` **默认**安装到目标仓 `docs/architecture/`、`docs/templates/` 与精简 `docs/README.md`。缺 core = **不完整安装**。

在 monorepo 中，规范模板位于 [`docs/templates/`](../docs/templates/)；包内 `templates/` 与 `core/docs/templates/` 为分发镜像。机读契约以 [`docs/contracts/`](../docs/contracts/) 为 canonical，本包 `contracts/` 为逐字节镜像。

**发布与候选证据边界（GOAL-019 / v0.9.0）**：Claude Code CLI、Grok Build CLI 与 GitHub Copilot CLI `1.0.71` 均列为 `committed` 支持基线。矩阵 **`candidateRevision: v0.9.0`**；三个 CLI 的 `/govern`+`/audit` 共六单元均于 2026-07-24 **runtime-verified**（coverage ready-for-release-evidence）。Web parser 保持 `automated-verified`。VS Code 插件不作为 Copilot 重放证据来源。权威字段见 [`docs/contracts/skills-consumer-contract.json`](../docs/contracts/skills-consumer-contract.json) 与 [`docs/contracts/skills-consumer-compatibility-matrix.json`](../docs/contracts/skills-consumer-compatibility-matrix.json)。

## 产品模型（必读）

| 层级 | 是什么 | 用户怎么用 |
|------|--------|------------|
| **核心方法论** | `docs/architecture` + `docs/templates` + 精简 `docs/README` | install 从 `core/` **默认**安装；与 Skills **同级必备** |
| **主入口（primary）** | 编排器：扫描 / 意见台账 / 分类 / P-004 裁决 / 确认 / 原语 | **`/govern`** |
| **交叉入口** | 独立审计：只出意见（`source: independent`） | **`/audit`** |
| **原语（primitives）** | 创建目标、记决策、更执行、写审计 | 由编排器调用；Copilot advanced 可选 |
| **规则** | AGENTS / copilot-instructions | 结构、编号、操作细则摘要 |

生命周期：**设立 → 信息发现与就绪判断 →（可审视）→ 方案 → 实施 → 审计/整改 → 关门**。
交叉意见由 `/audit` 写入；**响应与放行**由 `/govern` 处理。

工作区协议：`/govern` 和 `/audit` 先定位当前 `docs/workspace-<NNN>-<slug>/workspace.md`，校验其 Root Goal、canonical 范围和共享资料固定引用；不匹配或多个工作区未指定焦点时 fail closed。没有显式工作区根的旧项目才按 `docs/goals/` 的 legacy 隐式单工作区工作，Skills 不会自动发现或混合外部工作区。共享资料候选库存只补充文件摘要，资料内容仍须经用户确认才能成为事实、证据或 finding 关闭依据。

| 工具 / 表面 | 安装位置 | 斜杠 | 当前契约层级 |
|------|----------|------|--------------|
| Claude Code CLI | `.claude/skills/govern/` + `audit/` | `/govern` · `/audit` | `committed / 当前候选待 runtime 验证` |
| Grok Build CLI | `.grok/skills/govern/` + `audit/` | `/govern` · `/audit` | `committed / 当前候选待 runtime 验证` |
| GitHub Copilot CLI `1.0.71` | `.github/copilot-instructions.md` + repository prompt sources | `/govern` · `/audit` | `committed / 当前候选待 runtime 验证` |

核心行为：

> Contract manifest 的 `verificationStatus` 仍是历史有界事实，不能替代候选矩阵。Claude Code 机读证据保存脱敏 stream transcript；Grok Build 机读证据保留辅助 session-title `grok-build` alias 的 502 警告，但主 `grok-4.5` 调用 exit `0` 且输出实际 dispatch marker。完整发行验收仍以全部 matrix 单元、coverage、CI 与 release 证据为准。

- 编排：[`prompts/00-govern-orchestrator.md`](prompts/00-govern-orchestrator.md)
- 交叉：[`prompts/05-independent-audit.md`](prompts/05-independent-audit.md)

## 目录结构

```text
skills/
├── README.md
├── AGENTS.template.md
├── install.sh / install.ps1
├── core/                               # GOAL-019：方法论镜像 → install 默认装到 ./docs/
│   └── docs/
│       ├── README.md                   # 精简文档入口
│       ├── architecture/               # principles, workspace-protocol, overview, layout
│       └── templates/                  # 五件套 + workspace-context
├── install/
│   ├── claude/
│   │   ├── AGENTS.md
│   │   └── skills/{govern,audit}/SKILL.md
│   ├── grok/
│   │   └── skills/{govern,audit}/SKILL.md
│   └── copilot/
│       ├── copilot-instructions.md
│       └── prompts/
│           ├── govern.md               # primary
│           ├── audit.md                # cross-audit (default install)
│           └── new-goal.md …           # advanced only
├── prompts/
│   ├── 00-govern-orchestrator.md       # PRIMARY core
│   ├── 01–04 …                         # primitives
│   └── 05-independent-audit.md         # cross-audit core
├── templates/goal-folder/              # 包内模板镜像（install --all 同步到 skills 目录）
├── templates/workspace-context.md
├── contracts/                          # docs/contracts 的分发镜像
└── tests/
```

## 安装

推荐：**从 GitHub Release 下载 skills-only zip**（不是整个 monorepo），解压进目标项目，再装规则与主入口。安装脚本**离线**、不访问网络。

### 从 GitHub Release 安装（推荐 · 其他项目）

1. 打开本仓库 [Releases](https://github.com/magicvr/goal-governance/releases)，下载与 tag 对应的  
   `goal-governance-skills-vX.Y.Z.zip`（可对照同目录的 `.sha256` 校验）。  
   包内含 Skills 适配器 + **core 方法论镜像**（prompts、install、`core/`、模板/契约），**不含** monorepo dogfood 过程树、`web/` 或 `artifacts/`，**不含** `tech-stack.md`。
2. 在目标项目根目录解压，使包内容落在 `./skills/`（或你选择的目录名）：

```bash
# 示例：已下载 zip 到当前目录
unzip goal-governance-skills-vX.Y.Z.zip
# zip 根目录名为 goal-governance-skills-vX.Y.Z/ — 重命名为 skills 便于默认参数
mv goal-governance-skills-vX.Y.Z skills
```

```powershell
Expand-Archive .\goal-governance-skills-vX.Y.Z.zip -DestinationPath .
Rename-Item .\goal-governance-skills-vX.Y.Z skills
```

3. 安装宿主入口（默认 `/govern` + `/audit`）**并默认安装 core → `./docs/`**：

```bash
bash ./skills/install.sh --all --skills-dir ./skills
# 或单宿主：--claude / --grok / --copilot（同样会装 core）
```

```powershell
.\skills\install.ps1 -All -SkillsDir .\skills
# 或：-Claude / -Grok / -Copilot（同样会装 core）
```

4. 确认 `docs/architecture/principles.md` 等已存在；再建立 `docs/workspace-001-<slug>/`（`workspace.md` + `goal-tree.md`），调用 **`/govern`**（交叉审计用 **`/audit`**）。

> 维护者正式发布：推 **annotated** `v*` tag → CI pack → Environment **`release` 审批** → 硬 `release_evidence --mode release` 通过后自动 `gh release create` 并挂 zip / sha256 / evidence。详见 [docs/releases/README.md](../docs/releases/README.md)。  
> 本地调试 zip：`python scripts/pack_skills_release.py --version X.Y.Z --output-dir dist/`。  
> 尚未对齐矩阵/`candidateRevision` 的工作树**不要**推正式 tag；门禁失败则**不会**创建 Release。

### 0. 从源码树复制包（开发者 / 无 Release 时）

```bash
cp -R /path/to/goal-governance/skills ./skills
```

```powershell
Copy-Item -Recurse path\to\goal-governance\skills .\skills
```

### 1. 手动安装

**默认安装面**（与脚本一致）：每个所列安装产物都装 **`/govern` + `/audit`**。这描述可复制文件，不提升上表的声明/承诺层级或 `unverified` 运行时状态；填表类 advanced slash 仍为可选。

#### Claude Code

```text
install/claude/AGENTS.md
  →  <repo>/AGENTS.md
install/claude/skills/govern/SKILL.md
  →  <repo>/.claude/skills/govern/SKILL.md
install/claude/skills/audit/SKILL.md
  →  <repo>/.claude/skills/audit/SKILL.md
```

```bash
mkdir -p .claude/skills/govern .claude/skills/audit
cp ./skills/install/claude/AGENTS.md ./AGENTS.md
cp ./skills/install/claude/skills/govern/SKILL.md .claude/skills/govern/SKILL.md
cp ./skills/install/claude/skills/audit/SKILL.md .claude/skills/audit/SKILL.md
```

#### Grok Build

```text
install/grok/skills/govern/SKILL.md
  →  <repo>/.grok/skills/govern/SKILL.md
install/grok/skills/audit/SKILL.md
  →  <repo>/.grok/skills/audit/SKILL.md
```

（建议同时有根 `AGENTS.md` 作项目规则；可与 Claude 共用。）

```bash
mkdir -p .grok/skills/govern .grok/skills/audit
cp ./skills/install/grok/skills/govern/SKILL.md .grok/skills/govern/SKILL.md
cp ./skills/install/grok/skills/audit/SKILL.md .grok/skills/audit/SKILL.md
```

#### GitHub Copilot

```text
install/copilot/copilot-instructions.md
  →  .github/copilot-instructions.md
install/copilot/prompts/govern.md
  →  .github/prompts/govern.prompt.md
install/copilot/prompts/audit.md
  →  .github/prompts/audit.prompt.md
```

| Wrapper | 斜杠 | 何时安装 |
|---------|------|----------|
| govern.md | `/govern` | **默认**（主入口） |
| audit.md | `/audit` | **默认**（交叉审计） |
| new-goal … write-audit | advanced | 仅 `--with-primitives` |

### 2. 脚本安装

| 参数 | 作用 |
|------|------|
| `--claude` / `-Claude` | `AGENTS.md` + `.claude/skills/govern` + **`audit`** + **core → docs/** |
| `--grok` / `-Grok` | `.grok/skills/govern` + **`audit`** + **core → docs/** |
| `--copilot` / `-Copilot` | copilot-instructions + `govern`/`audit` prompts + **core → docs/** |
| `--with-primitives` / `-WithPrimitives` | 可选：四个 advanced 填表 slash（new-goal 等） |
| `--all` / `-All` | Claude + Grok + Copilot + prompts/templates/contracts + **core** |
| `--init-workspace` / `-InitWorkspace` | 可选：scaffold `docs/workspace-NNN-slug/`（**须**同时给 slug） |
| `--workspace-slug` / `-WorkspaceSlug` | 与 init-workspace 联用；小写短横线；**禁止静默默认** |
| `--root-slug` / `-RootSlug` | 与 init-workspace 联用 → 计划中的 `GOAL-001-<slug>` |
| `--root-title` / `-RootTitle` | 可选；计划中 Root 标题 |
| `--workspace-nnn` / `-WorkspaceNnn` | 可选；默认 `001` |
| `--skills-dir` / `-SkillsDir` | 默认 `./skills` |

```bash
bash ./skills/install.sh --all --skills-dir ./skills
bash ./skills/install.sh --claude --skills-dir ./skills
bash ./skills/install.sh --grok --skills-dir ./skills
bash ./skills/install.sh --copilot --skills-dir ./skills
```

```powershell
.\skills\install.ps1 -All -SkillsDir .\skills
.\skills\install.ps1 -Claude -SkillsDir .\skills
.\skills\install.ps1 -Grok -SkillsDir .\skills
.\skills\install.ps1 -Copilot -SkillsDir .\skills
```

可选：安装同时 scaffold 工作区骨架（**不**创建 Root 五件套；slug 必须显式给出）：

```bash
bash ./skills/install.sh --all --skills-dir ./skills \
  --init-workspace --workspace-slug my-product --root-slug product-vision \
  --root-title "Product vision"
```

```powershell
.\skills\install.ps1 -All -SkillsDir .\skills `
  -InitWorkspace -WorkspaceSlug my-product -RootSlug product-vision `
  -RootTitle 'Product vision'
```

安装后：使用 **`/govern`** 推进（若已 scaffold，则创建 Root 五件套）；需要交叉审计时用 **`/audit`**。

## 最小可运行集（消费方）

| 必备 | 来源 |
|------|------|
| 根 `AGENTS.md`（或 copilot-instructions） | install |
| `/govern` + `/audit` + `skills/prompts/*` | install + 包 |
| **`docs/architecture/`**（principles、workspace-protocol、overview、directory-layout） | install 从 `core/` |
| **`docs/templates/`** + 精简 **`docs/README.md`** | install 从 `core/` |
| `docs/workspace-…/workspace.md` + `goal-tree` | `/govern` S0，或 install `--init-workspace`（slug **显式**） |
| Root 五件套 | `/govern` / 原语 01 创建（init-workspace **不**代建） |

| 不要期望随包出现 | 原因 |
|------------------|------|
| monorepo dogfood `GOAL-*` 树 | 过程数据 |
| `tech-stack.md` | 实现栈，非方法论 |
| 完整 monorepo `docs/README` / standalone 测试 | 维护者路径 |

## 在其他项目中快速启用

1. 安装规则 + `/govern` + `/audit`（**同时默认安装 core → `docs/`**）。  
2. 核对 `docs/architecture/principles.md` 存在。  
3. 从 `docs/templates/workspace-context.md` 建立 `docs/workspace-001-<slug>/workspace.md` 与 `goal-tree.md`。  
4. 调用 `/govern`：引导总目的 / Root。  
5. 调用 `/audit`：独立审计意见（不改 status）。

## 核心约定（摘要）

| 规则 | 说明 |
|------|------|
| 核心 + Skills | 同级必备；仅装适配器不算完整 |
| 扁平存储 | 目标平铺在当前 `docs/workspace-<NNN>-<slug>/` 根 |
| 编号 | `GOAL-001` 为 Root；slug 自定 |
| 层级 | 仅 `parent` 字段 |
| 总览 | 变更后更新 `goal-tree.md` |
| 五件套 | meta / decision / execution / audit / attachments |
| 工作区 | `workspace.md` 绑定 Root Goal 与 canonical 范围；legacy `docs/goals/` 仅旧仓兼容 |
| 共享资料 | 固定 `material_id` / source / version / SHA-256；非第二状态 |
| 信息就绪 | 可带未知立项；I-00N 与阶段门禁 |
| 包目录名 | 常为 `skills/`，可改名；按含 `prompts/00-…` 定位 |

## 测试

```bash
# 结构契约 +（Windows）PowerShell 隔离安装冒烟（F-018）
python skills/tests/test_skills_orchestrator.py
```

```powershell
# 仅跑隔离安装冒烟（不经过 unittest）
powershell -NoProfile -ExecutionPolicy Bypass -File .\skills\tests\test_install_ps1_isolated.ps1
```

Windows 上隔离安装冒烟断言 `/govern`+`/audit`+**core docs 落点**，且不含填表 advanced slash、不含 `tech-stack`。`install.sh` 的真实执行仍依赖 bash 环境（本仓库 Windows 主证据以 PS1 为准）。

## 尚未包含

- Marketplace 完整包  
- 编号 / parent 自动校验工具  
- 自动在无维护者授权时创建 GitHub Release（tag CI 仅 pack + 上传 artifact）

当前交付：**core 方法论镜像（默认 install）+ Skills 适配 + `/govern`/`/audit` + 原语 01～05 + 多宿主安装 + 可选 `--init-workspace` + 模板/契约镜像 + pack zip**。monorepo `docs/` 仍为维护者 canonical 上游。
