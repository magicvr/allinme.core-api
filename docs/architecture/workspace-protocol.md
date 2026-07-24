---
title: 工作区与共享资料区协议
status: active
created: 2026-07-20
updated: 2026-07-24
parent: null
version: 0.2.1
---

# 工作区与共享资料区协议

本协议定义核心文档、Skills 和后续消费适配器共同使用的工作区边界。它保留目标五件套和目标平铺，但不再以全局 `docs/goals/` 作为当前 canonical 布局：每个显式工作区在自己的根目录内保存唯一目标状态。

## 1. 范围与术语

| 术语 | 含义 | 不是 |
|------|------|------|
| **工作区根** | `docs/workspace-<NNN>-<slug>/`，其中包含 `workspace.md`、`goal-tree.md` 与平铺的 `GOAL-*`。 | `parent` 层级、页面缓存、审计 `scope` 或第二套目标状态。 |
| **工作区上下文文档** | 工作区根内的 `workspace.md`，从 [workspace-context.md](../templates/workspace-context.md) 复制。 | 每个目标重复保存的元数据，或全局工作区注册表。 |
| **legacy 隐式单工作区** | 外部旧仓库没有显式工作区根、但保留 `docs/goals/` 与唯一 Root Goal 时的兼容模式。 | 多工作区发现、共享资料访问或跨目录自动搜索许可。 |
| **共享资料目录** | 位于工作区根之外的资料集合；常见为 `docs/shared-materials/`。 | 目标树、目标状态库或任一工作区可写的隐式公共区。 |
| **共享资料候选库存** | 重建脚本生成的路径、大小和 SHA-256 清单。 | 固定资料引用、canonical 事实、证据或用户确认。 |
| **共享资料引用** | 工作区内指向资料 ID、来源、版本和哈希的可追溯记录。 | 已确认事实、可执行指令或其他工作区的上下文通道。 |

每个工作区根内的 `goal-tree.md` 与 `GOAL-*` 五件套只承载该工作区的目标生命周期状态。共享资料候选库存或导航索引不能形成第二套状态；资料内容必须按事实准入和用户确认规则处理，不能仅因可读取而成为事实或关闭证据。

## 2. 工作区不变量

1. 一个工作区恰好绑定一个 `parent: null` 的 Root Goal；`workspace.md` 的 `root_goal` 必须与该 Root Goal 完整 ID 一致。
2. `canonical_scope` 必须等于包含该 `workspace.md` 的工作区根，例如 `docs/workspace-001-<your-slug>/`。该目录直接平铺 `GOAL-*` 与 `goal-tree.md`；层级只由 `parent` 字段表达。
3. 工作区之间不得混合目标、候选、草稿、审计意见、写入请求或 AI 上下文。多个工作区而没有明确焦点时，Skills 和消费适配器必须 fail closed，而非猜测默认工作区。
4. 平台或宿主可以提供导航，但导航缓存不能成为 canonical 目标状态；跨工作区导航字段、运行时授权和用户操作仍属于消费适配器/产品门禁。
5. 工作区上下文改变 Root Goal 绑定、canonical 范围或共享资料目录指针时，属于治理变更：必须有可追溯决定，并在受影响目标的执行记录中留下事实。

## 3. Root Goal 与串行阶段

Root Goal 表达稳定目的、初始边界和高层路线图，不要求在立项时穷尽所有未来阶段。MVP、后续阶段和扩展工作应更新该 Root Goal 的路线图并创建串行子目标；只有长期目的、成功边界或战略方向确实改变时，才记录决定后修改 Root Goal 定义。不得用工作区目录嵌套代替目标 `parent` 关系。

## 4. 工作区上下文文档

新建显式工作区时，从 `docs/templates/workspace-context.md` 复制为 `docs/workspace-<NNN>-<slug>/workspace.md`。frontmatter 的最小字段为：

| 字段 | 要求 |
|------|------|
| `id` | 工作区稳定标识；资料引用的 `workspace_id` 必须相同。 |
| `root_goal` | 当前工作区 Root Goal 的完整 ID，且该目标 `parent: null`。 |
| `canonical_scope` | 当前工作区根；格式为 `docs/workspace-<NNN>-<slug>/`。 |
| `shared_materials_catalog` | 共享资料目录的固定路径/URI，或 `none`。它只标识资料来源，不保存资料内容。 |
| `status`、`created`、`updated`、`version` | 与其他 core Markdown 一致的可追溯元信息。 |

若 `shared_materials_catalog: none`，工作区不得声明共享资料引用。对于旧仓库，只有没有显式工作区根且存在 `docs/goals/` 时才可作为 legacy 隐式单工作区处理；不得把该兼容路径复制到已迁移仓库中。

## 5. 共享资料候选库存与固定引用

用户可手工把文件复制到资料目录。若使用维护者提供的索引重建工具，生成的 `index.json` 通常只记录相对路径、字节数和 SHA-256。该清单有以下边界：

1. 它不自动分配业务 `material_id`、版本、用途、工作区归属或资料确认状态。
2. 它不读取、执行、外传或解释资料内容；索引脚本拒绝符号链接和目录逃逸。
3. 它不替代用户确认、P-004 裁决、P-005 信息门禁或任何 finding 关闭证据。

工作区只能在其上下文文档或受控的决策/执行记录中维护固定资料引用。每一条引用至少包含下列字段：

| 字段 | 要求 |
|------|------|
| `reference_id` | 工作区内唯一的引用 ID。 |
| `workspace_id` | 必须等于工作区上下文的 `id`。 |
| `material_id` | 用户确认的资料稳定标识。 |
| `source` | 可追溯的资料来源路径或 URI。 |
| `version` | 不可省略的资料版本。 |
| `sha256` | 该版本的 64 位十六进制 SHA-256 摘要。 |
| `purpose` | 该引用为何与当前工作区相关。 |
| `local_record` | 本地注释或派生记录路径；没有时明确为 `none`。 |
| `status` | `active`、`withdrawn` 或 `invalid`，不得把资料可读性写成事实已确认。 |

以下规则是 fail-closed 的：缺少 `material_id`、`source`、`version` 或有效 `sha256` 的资料不得作为共享资料引用使用；工作区不匹配、目录为 `none`、来源或摘要不一致时，Skills 和消费适配器不得读取、推理混合、引用为证据或写入目标记录。

本协议不规定资料上传、用户 CRUD、保留实现、AI 读取执行、版权/敏感数据处理流程、跨工作区导航或 Web 写入；这些由各项目产品目标在其门禁内定义和验证。

## 6. Skills 与消费适配器规则

1. 先定位用户指定或已配置的工作区 `workspace.md`，校验 Root Goal、canonical scope 和共享资料引用后，再扫描该工作区 `goal-tree.md` 与目标记录。
2. 若仓库只有一个显式工作区，消费适配器可以使用它作为当前 scope；多个工作区而未指定焦点时必须拒绝受影响读取、写入和放行。
3. 没有显式工作区根时，只能将旧 `docs/goals/` 作为 legacy 单工作区；不得自动发现、合并或写入其他目录。
4. 任何创建、决策、执行、审计或提案都必须在已验证的当前工作区内。资料候选库存只补充可核对的文件摘要，不替代固定引用或事实确认。
5. `/audit` 只在当前工作区目标台账追加 `source: independent` 意见；它不得凭资料目录或索引改变状态或关闭 finding。

## 7. 与产品适配器的交接

显式工作区与共享资料候选索引为协议层输入。资料 CRUD、AI 全文读取、跨工作区导航与 Web 写入等产品能力，由各项目自己的目标与门禁验证；本协议不自动放行这些能力。