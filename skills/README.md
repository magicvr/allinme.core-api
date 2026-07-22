---
title: Skills · 目标治理可复用包
status: active
created: 2026-07-18
updated: 2026-07-22
parent: null
version: 1.3.0
---

# Skills

本目录提供可复制到**其他项目**的目标治理约定与模板。  
本仓库运行中的强制规则仍以根目录 [AGENTS.md](../AGENTS.md) 为准；此处是提炼后的**可复用交付物**。

Skills 是核心方法论与文档协议的消费适配器，不是独立真相源。在本仓库中，规范模板位于 [`docs/templates/goal-folder/`](../docs/templates/goal-folder/) 与可选的 [`docs/templates/workspace-context.md`](../docs/templates/workspace-context.md)；本包内对应模板是用于离线复制和安装脚本的同步镜像。消费适配器的机读版本/兼容声明以 [`docs/contracts/`](../docs/contracts/) 为 canonical，本包 `contracts/` 是逐字节分发镜像。安装到其他仓库后，镜像必须自包含可用。

**发布与候选证据边界（GOAL-008 D-010 / D-011）**：Claude Code CLI、Grok Build CLI 与 GitHub Copilot CLI `1.0.71` 均列为 `committed` 支持基线。`v0.7.0` 的六个 CLI 入口与 Web CI 证据是已归档的历史发布事实；GOAL-010 修改行为源后，当前矩阵明确为 `candidateRevision: unreleased`，三个 CLI 的 `/govern`、`/audit` 共六个入口保持 `pending-runtime-validation` 且不引用旧证据。Web parser 保持已有的 `automated-verified` CI 证据。VS Code 插件不作为 Copilot 重放证据来源。权威字段见 [`docs/contracts/skills-consumer-contract.json`](../docs/contracts/skills-consumer-contract.json) 与 [`docs/contracts/skills-consumer-compatibility-matrix.json`](../docs/contracts/skills-consumer-compatibility-matrix.json)。

## 产品模型（必读）

| 层级 | 是什么 | 用户怎么用 |
|------|--------|------------|
| **主入口（primary）** | 编排器：扫描 / 意见台账 / 分类 / P-004 裁决 / 确认 / 原语 | **`/govern`** |
| **交叉入口** | 独立审计：只出意见（`source: independent`） | **`/audit`** |
| **原语（primitives）** | 创建目标、记决策、更执行、写审计 | 由编排器调用；Copilot advanced 可选 |
| **规则** | AGENTS / copilot-instructions | 结构、编号、P-001～P-005、goal-tree |

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
├── templates/goal-folder/              # docs/templates 的五件套分发镜像
├── templates/workspace-context.md      # workspace-<NNN>-<slug>/workspace.md 分发镜像
├── contracts/                          # docs/contracts 的分发镜像
└── tests/
```

## 安装

推荐：**从 GitHub Release 下载 skills-only zip**（不是整个 monorepo），解压进目标项目，再装规则与主入口。安装脚本**离线**、不访问网络。

### 从 GitHub Release 安装（推荐 · 其他项目）

1. 打开本仓库 [Releases](https://github.com/magicvr/goal-governance/releases)，下载与 tag 对应的  
   `goal-governance-skills-vX.Y.Z.zip`（可对照同目录的 `.sha256` 校验）。  
   包内**只有** Skills 分发面（prompts、install 适配、模板/契约镜像、安装脚本），**不含** dogfood 过程树、`web/` 或 `artifacts/`。
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

3. 安装宿主入口（默认 `/govern` + `/audit`）：

```bash
bash ./skills/install.sh --all --skills-dir ./skills
# 或单宿主：--claude / --grok / --copilot
```

```powershell
.\skills\install.ps1 -All -SkillsDir .\skills
# 或：-Claude / -Grok / -Copilot
```

4. 在目标仓库建立工作区与 `goal-tree.md` 后，在对应 AI 宿主中调用 **`/govern`**（交叉审计用 **`/audit`**）。

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
| `--claude` / `-Claude` | `AGENTS.md` + `.claude/skills/govern` + **`audit`**（`/govern` + `/audit`） |
| `--grok` / `-Grok` | `.grok/skills/govern` + **`audit`**（`/govern` + `/audit`） |
| `--copilot` / `-Copilot` | copilot-instructions + **默认双入口** `govern` + `audit` prompts |
| `--with-primitives` / `-WithPrimitives` | 可选：四个 advanced 填表 slash（new-goal 等） |
| `--all` / `-All` | Claude + Grok + Copilot + prompts/templates/contracts |
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

安装后：使用 **`/govern`** 推进；需要交叉审计时用 **`/audit`**，再用 `/govern` 响应意见。

## 在其他项目中快速启用

1. 安装规则 + `/govern` + `/audit` skill（见上）。  
2. 建立 `docs/workspace-001-<slug>/goal-tree.md`（可先空）。
3. 从 `templates/workspace-context.md` 创建 `docs/workspace-001-<slug>/workspace.md`，绑定 Root Goal 与该工作区根；旧 `docs/goals/` 只用于 legacy 单工作区兼容。
4. 调用 `/govern`：扫描并引导总目的，或分析未关门目标的下一步。
5. 调用 `/audit`：对指定目标写独立审计意见（不改 status）。

## 核心约定（摘要）

| 规则 | 说明 |
|------|------|
| 扁平存储 | 目标平铺在当前 `docs/workspace-<NNN>-<slug>/` 根（本包约定） |
| 编号 | `GOAL-001` 为 Root；slug 自定 |
| 层级 | 仅 `parent` 字段 |
| 总览 | 变更后更新 `goal-tree.md` |
| 五件套 | meta / decision / execution / audit / attachments |
| 工作区 | `docs/workspace-<NNN>-<slug>/workspace.md` 绑定 Root Goal 与 canonical 范围；仅 legacy `docs/goals/` 缺失显式根时为隐式单工作区 |
| 共享资料 | 只使用匹配工作区的固定 `material_id` / source / version / SHA-256 引用；不成为第二状态或事实捷径 |
| 信息就绪 | 可带未知立项；登记 I-00N、阶段门禁、证据与用户接受的残余风险 |
| 代码布局 | 普遍在仓库根；子目录仅项目自定 |
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

Windows 上 `test_install_ps1_isolated_all_produces_govern_and_audit` 会在临时目录执行 `install.ps1 -All`，断言默认产出 `/govern`+`/audit` 且不含填表 advanced slash。`install.sh` 的真实执行仍依赖 bash 环境（本仓库 Windows 主证据以 PS1 为准）。

## 尚未包含

- Marketplace 完整包  
- 编号 / parent 自动校验工具  
- 自动在无维护者授权时创建 GitHub Release（tag CI 仅 pack + 上传 artifact）

当前交付：**核心协议的 Skills 适配规则 + 编排主入口 `/govern` + 交叉入口 `/audit` + 文档原语 01～05 + 多宿主安装脚本 + 模板与机读契约分发镜像 + 版本化 zip 打包入口（`scripts/pack_skills_release.py`）**。核心 canonical 方法论、模板与契约见仓库 `docs/` 层。
