# 后端架构决策

本目录记录仅属于 `allinme.core-api` 的重大、长期实现选择，例如业务分层、存储、认证、错误模型或可观测性。

## 规则

- 文件名使用 `NNNN-short-title.md`，编号全局递增且不复用。
- frontmatter 至少包含 `status`、`date`；正文包含背景、决策、备选方案和后果。
- 改变 Schema-UI 跨端字段、执行语义或版本策略的决策必须写入 [`schema-ui-docs/docs/decisions/`](../../../schema-ui-docs/docs/decisions/)，本目录只能引用。
- 被替代的 ADR 保留历史并标记 `superseded`，不得删除以掩盖决策演进。

## 当前决策

- 当前尚无后端项目级 ADR。