---
id: GOAL-042-optimize-readme
doc: execution
status: active
parent: GOAL-040-docs-quality
created: 2026-03-01
updated: 2026-07-19
version: 0.2.0
---

# 执行记录 · GOAL-042

## 时间线

> 涉及 P-005 信息项时，记录本次收集/验证的实际动作、I-00N、级别、证据路径，以及新发现的未知。计划中的收集动作必须明确标为计划，不能把 `open`、`deferred` 或 `accepted-residual` 写成已验证事实。

### 2026-03-01 · 目标立项

- 从模板复制本目标五件套，设定 `parent: GOAL-040-docs-quality`。
- 在决策中确认 README 定位：入口信息 + 链到 docs（见 [01-decision.md](01-decision.md) D-001）。
- 同步更新当前工作区 `goal-tree.md` 登记本目标。

### 2026-03-05 · 重写根 README 骨架

- 按 D-001 重写根 `README.md`：简介、快速开始、目录树、文档链接四段。
- 删除原 README 中与 `docs/architecture/` 重复的架构长文（约 80 行），改为链接。
- 按 D-002 将快速开始收敛为 3 条命令；Windows / macOS 差异用一行注释标明。

### 2026-03-12 · 自测快速开始

- 在干净目录按 README 执行：venv 创建、依赖安装、服务启动均成功。
- 发现一处过期路径：`docs/setup.md` 已不存在，改为链到 `docs/README.md`（已改）。
- 尚未请协作者做首次启动确认；与 `docs/README.md` 的交叉链接仍待逐条核对。

## 待办

1. 核对根 README 与 `docs/README.md` 全部交叉链接
2. 请一名协作者按 README 完成首次启动并记录结果

## 进度评估

**约 60%**：结构与快速开始已落地并自测通过；交叉链接核对与协作者验证未完成。
