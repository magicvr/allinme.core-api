# 实施记录管理

`records/` 保存按计划执行实际交付的 `IMP-NNNN`。IMP 连接计划、checklist、代码/测试变更和后续实施审计。

## 规则

- 文件名：`IMP-NNNN-YYYYMMDD-<implementer>-plan-<plan-id-subject>.md`。
- 实施前必须存在当前、未漂移且 verdict 为 `ready` 的计划验收 AUD。
- 一次实施尝试对应一份 IMP；需要继续或重做时创建新 IMP，不改写终态记录。
- `status` 使用 `in-progress`、`completed`、`partial` 或 `blocked`。
- IMP 必须记录计划/checklist 映射、实际变更、验证命令与结果、未完成项、剩余风险和 `result_revision`。
- 计划范围或验收标准变化时停止实施并重新进入计划审计，而不是在 IMP 中事后调整标准。
- `completed` 只表示实施声称完成，必须由不同上下文执行实施审计和完成验收。
- 完成验收按实际 IMP、已验证 REM 和 Git revision 派生有效结果，不能按编号猜测或遗漏较新的失败记录。

模板：[templates/implementation-record.md](./templates/implementation-record.md)。实施入口：`$backend-implement-plan`；实施审计：`$backend-implementation-audit`；完成验收：`$backend-implementation-acceptance-audit`。

## 当前索引

暂无实施记录。
