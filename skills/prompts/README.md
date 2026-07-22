---
title: Skills · 提示词模板
status: active
created: 2026-07-18
updated: 2026-07-19
parent: null
version: 0.5.0
---

# prompts/ · 目标治理提示词

## 默认用户路径

| 文件 | 角色 | 用途 |
|------|------|------|
| [00-govern-orchestrator.md](00-govern-orchestrator.md) | **primary** | 扫描 goal-tree 与审计意见 → 分类 → P-004 裁决 → 提议确认 → 原语 |
| [05-independent-audit.md](05-independent-audit.md) | **independent-audit** | 交叉审计：只出意见（`source: independent`），不改状态 |

| 入口 | 宿主 |
|------|------|
| `/govern` | Claude / Grok skill；Copilot `govern.prompt.md` |
| `/audit` | Claude / Grok skill；Copilot `audit.prompt.md` |

**生命周期**：设立 → 信息发现与就绪判断 →（可审视）→ 方案 → 实施 → 审计/整改 → 关门。
**交叉意见**由 `/audit` 写入；**响应与放行**由 `/govern` 处理。

## 原语（primitives / advanced）

编排器在用户确认后调用；熟练用户也可直调。**不是**默认产品主菜单。

| 文件 | 角色 | 用途 |
|------|------|------|
| [01-create-new-goal.md](01-create-new-goal.md) | primitive | 创建新目标（五件套 + goal-tree） |
| [02-record-decision.md](02-record-decision.md) | primitive | 记录决策 |
| [03-update-execution.md](03-update-execution.md) | primitive | 更新执行时间线与进度 |
| [04-write-audit.md](04-write-audit.md) | primitive | 自审 / 阶段复盘 / 响应记录（结构化意见） |

## 使用方式

### 主路径

1. `/govern` 或打开 `00-govern-orchestrator.md`「提示词正文」。
2. 先扫描与意见台账，再提议；确认后写入。
3. 需要交叉审计时用 `/audit`（建议另开会话）。

### 原语直调（高级）

1. 打开对应 `01`～`04`，复制「提示词正文」。
2. 缺项先确认；落盘遵守 `03-audit` 规则（P-003）。

## 设计原则

| 原则 | 说明 |
|------|------|
| 目的优先 | 辅助达到目的，而非辅助填表 |
| 主入口 + 交叉入口 | `/govern` 生命周期；`/audit` 独立意见 |
| 原语可组合 | 01～04 保证文档结构一致 |
| 信息就绪 | P-005：登记未知、最晚需要阶段、证据与残余风险；按规模拆信息工作 |
| 遵守 AGENTS | 扁平存储、parent、goal-tree、P-001～P-005 |
| 真实 | 禁止编造进度与空话 |

## 与其他交付物

| 路径 | 角色 |
|------|------|
| [../../docs/templates/goal-folder/](../../docs/templates/goal-folder/) | 核心 canonical 目标模板（本仓库） |
| [../AGENTS.template.md](../AGENTS.template.md) | 规则正文 |
| [../templates/goal-folder/](../templates/goal-folder/) | Skills 分发模板镜像 |
| [../install/](../install/) | 各宿主 skill / slash 安装源 |
