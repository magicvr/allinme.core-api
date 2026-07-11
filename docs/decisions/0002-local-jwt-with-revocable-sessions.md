---
status: accepted
date: 2026-07-11
---

# 使用本地账号、JWT Bearer 与可撤销会话

## 背景

demo 需要真实登录和角色授权，且前端与 API 可能运行在不同开发源。只使用不可撤销 JWT 无法在登出、账号禁用或角色变化后立即失效；完整 OIDC 身份基础设施又会增加本地联调门槛。

## 决策

- API 在 SQLite 保存本地账号与强密码哈希，内置 `viewer`、`operator`、`approver`、`admin` 演示角色。
- 登录成功签发短期 JWT Bearer；token 包含 subject、role、expiry、issued-at 和唯一 `jti`。
- 每个 `jti` 对应 SQLite session。认证同时校验 JWT 和 session 撤销/到期状态，登出撤销当前 session。
- 角色用于粗粒度授权，资源状态、所有者、申请人与审批人分离等规则由业务用例再次检查。
- 签名密钥只能来自运行环境或 secret，生产模式缺失时拒绝启动。

## 备选方案

- 服务端 Cookie session：撤销简单且适合同源浏览器，但跨源 demo 需要额外 Cookie/CORS/CSRF 配置。
- 纯 JWT：请求验证简单，但无法可靠即时撤销。
- OIDC/JWT 资源服务器：更适合生产身份治理，但不是本地 demo 的首版依赖。

## 后果

本地前端可直接使用 Authorization header，API 能演示登录、登出、角色和即时撤销。每次认证会多一次 session 查询；若未来接入 OIDC，应保留业务授权策略并替换身份验证适配器。