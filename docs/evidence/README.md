# Evidence 记录

本目录仅保存计划、审计或发布流程明确要求提交的小型、脱敏 Evidence 摘要。大型产物、数据库、原始日志和敏感数据不得提交。

审计与验收至少在正文记录：

- 实际验证的 Git revision；
- 完整命令或测试入口；
- exit code 和结果摘要；
- 未执行项及原因；
- 必要时的产物路径、SHA-256 和外部保留位置。

Evidence 的目标是让后续审计者能够复核结论，不是建立密码学可信执行平台。仓库不要求外部 signer、runtime attestation、容器 image 证明或专用 evidence runner。对高风险结论，审计者仍应在独立上下文针对准确 revision 重新执行 subject-specific 验证。
