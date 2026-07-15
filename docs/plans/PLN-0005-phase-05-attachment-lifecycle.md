---
status: active
plan_id: PLN-0005
owner: 后端团队
created: 2026-07-13
last_updated: 2026-07-14
applies_to: implementation roadmap phase 5 attachments
---

# 阶段五：附件生命周期开发计划

配套路线：[实施路线图](../06-implementation-roadmap.md)。本阶段在阶段四订单、退款和看板闭环之上交付“先上传、后绑定、鉴权下载、可持久化恢复”的本地附件能力；页面 YAML、页面下发、真实对象存储、病毒扫描服务、CDN、断点续传、多文件单请求上传和公开分享链接不进入本轮。

## 1. 阶段目标

建立可重复演示、可审计且不暴露本地路径的附件生命周期：

- `operator`/`admin` 上传单个 PDF、PNG 或 JPEG，服务端限制大小、识别实际内容、生成存储键并计算 SHA-256；
- 上传先产生当前用户拥有、24 小时有效的未绑定附件，HTTP 响应返回可供阶段六 Schema-UI UploadAction 使用的稳定附件 ID；
- 创建订单时在单一 SQLite 事务内校验并绑定附件；编辑移除使用持久组 operation、文件隔离和最终订单事务，拒绝他人、过期、删除中或已绑定到其他订单的附件；
- 所有已认证角色只能通过受保护 API 下载已绑定附件，响应不泄露绝对路径或服务端文件名；
- 未绑定附件可由创建者删除，过期附件可通过确定性清理命令回收；
- 数据库、文件系统、reset、进程在 rename/commit 边界退出后的重启恢复、失败注入和并发竞争均有自动化证据。

阶段完成不代表 UploadAction 页面映射或附件页面已实现。阶段六才创建订单附件页面、UploadAction 映射和完整 Schema-UI 页面配置；本阶段只冻结并交付供阶段六消费的上传、订单绑定和下载 HTTP 契约。

## 2. 范围与非目标

### 2.1 本阶段范围

- additive schema v7 与附件元数据 repository；
- `internal/files` 本地文件适配器及可注入失败边界；
- 单文件 multipart 上传、已绑定附件下载、未绑定附件删除；
- 订单 create/edit 的 `attachmentIds` 绑定语义和订单附件摘要；5A 只开放 create 绑定，5B 再开放 edit add/reorder/remove；
- order create 幂等 digest/snapshot v2，同时继续读取并重放历史 v1；
- 24 小时过期清理、durable intent/tombstone 恢复、development seed/reset 和磁盘错误验证；
- CORS、route metadata、真实 JWT、重启与完整质量门禁。

### 2.2 明确非目标

- 不新增订单删除 HTTP endpoint。路线图中的“订单删除清理”在本阶段实现为可复用的 repository/file cleanup 原语和测试，不扩大当前订单 API surface；未来引入订单删除时必须复用该原语。
- 不支持对象存储、外部 URL、预签名 URL、公开下载、缩略图、图片转换、全文解析或病毒查杀。
- 不支持单请求多文件、分片上传、续传、Range、在线预览或跨用户共享未绑定附件。
- 不以客户端 `Content-Type`、扩展名或原始文件名决定存储路径或允许类型。
- 不在阶段五创建页面 YAML，也不把页面能力纳入 `/readyz`；页面 readiness 属于阶段六。
- 本阶段保证进程退出/重启后的恢复，不承诺主机掉电后目录项 rename 的跨平台持久性；若未来要求掉电安全，必须另行设计目录持久化和平台证据。

## 3. 事实源与开工决策

[架构](../01-architecture.md)持有 SQLite 原子阶段与文件系统编排边界；[领域模型](../05-domain-model.md)继续持有附件所有权、绑定、过期、访问和内部订单清理语义；[目标 HTTP API](../03-http-api-target.md)持有待实现 endpoint 范围；本计划在 active 期间持有 multipart、DTO、错误、文件编排、幂等升级、迁移和发布门禁。实现及集成门禁通过后，最终契约一次性迁入[当前 HTTP API](../03-http-api.md)。

P0 必须交叉复核并冻结以下决策；若需要改变，先修改本计划和对应事实源，不能只勾 checklist。P0 的交付拆成可独立审查的工作包，每个工作包都要有 owner、输入依赖、出口条件、最大 timebox 和固定 Evidence；全部阻塞工作包完成前保持 No-Go。P0 只交付事实源、接口/状态机/DDL 草案、fixture 与测试规格、工具 schema，以及明确标记为可抛弃且不得作为 release artifact 的最小 spike；可执行 v7 migration、可发布 capability binary、完整 startup coordinator/verify 命令、规模 wall-clock 与真实业务进程 smoke 均属于 M1A 或后续门禁。P0 spike 可以进入独立实验 revision 供方案证伪，但不得被索引为不可变发布产物、不得写业务数据，也不得作为 M1A 完成 Evidence：

1. schema 目标为 `PRAGMA user_version = 7`，本地/CI 临时产物路径为 `artifacts/phase5/v7/<artifact-kind>/<revision>/`，原始日志位于其 `evidence/` 子目录。`/artifacts/` 被 Git 忽略，只能作为生成缓存，不能成为唯一 Evidence。发布链包含 v6 gate/bootstrap、v7 schema-only、5A feature、5B feature 四类产物，分别绑定不可变源码 revision、完整构建命令、文件名与各自 SHA-256。P0 冻结不可由运行环境覆盖的编译期 capability API、生成文件布局、互斥测试规格和五个 build tag：`phase5_test`、`phase5_v6_gate`、`phase5_v7_schema_only`、`phase5_v7_5a`、`phase5_v7_5b`；其生产实现和真实 release binary 从 M1A 开始交付。缺失、未知或同时选择多个 capability 时构建失败。`phase5_test` 只提供全仓 test/vet/race 的确定装配，不得生成或批准部署产物；四种发布 tag 分别选择 `artifactKind`、支持 schema 和允许数据阶段，startup coordinator 必须直接消费编译值执行 gate。binary manifest 只是该编译值的只读导出，并必须绑定源码 revision、binary SHA-256、manifest SHA-256 和 run ID；不得让同一普通 `go build` 只靠外部 manifest 声明成不同产物。“同一候选 revision”分别约束最终 5A 或 5B feature binary，不得把历史迁移产物伪装成同一 revision。
2. 附件 ID 为 `att_` 加 32 位小写十六进制，使用 `crypto/rand`；客户端文件名永不参与存储键。
3. 首版只允许 `application/pdf`、`image/png`、`image/jpeg`，单文件最大 `10 MiB`；multipart 总请求上限固定为 `10 MiB + 64 KiB`，只允许一个名为 `file` 的文件 part，拒绝其他字段、重复 part 和多文件；结构验证采用第 20 项的增量协议。
4. 类型由服务端内容探测决定：PNG/JPEG 还需通过解码头校验，PDF 至少校验 `%PDF-` 签名和非空主体。part MIME 缺失或为 `application/octet-stream` 时允许由服务端探测决定；其他显式 MIME 必须与探测结果一致。原始扩展名不参与授权且不因与内容不一致单独拒绝，服务端始终按探测类型生成固定扩展名。
5. 原始文件名安全校验必须直接读取每个 part 的原始 `Content-Disposition` header，使用严格解析器且不得以 `multipart.Part.FileName()` 的平台相关 basename 结果作为安全输入。只允许一个 `filename` 或一个 `filename*` 参数，二者并存、重复参数、解析失败、非 `UTF-8''` 的扩展编码或非法 percent-encoding 均返回 `400`；解码后先做 NFC 规范化，再要求有效 UTF-8、无 NUL、无 `/` 或 `\\`、无任何 Unicode `Cc`、无双向 override/isolate 字符 U+202A..U+202E/U+2066..U+2069、不是 `.`/`..`，长度为 1..255 UTF-8 bytes。下载的最终值固定为 `attachment; filename="<ascii-fallback>"; filename*=UTF-8''<encoded-name>`，参数顺序不可交换；`filename` 始终使用双引号，fallback 只保留 `[A-Za-z0-9._-]`，每段其他字符折叠为单个 `_`，结果为空时使用 `attachment`，再截断到 120 ASCII bytes，因此最终值不含需要二次转义的引号或反斜杠。`filename*` 对 NFC 名称的 UTF-8 bytes 按 RFC 5987 attr-char 字节集（字母、数字及 ASCII 字符 ! # $ & + - . ^ _ | ~ 和反引号 0x60）原样保留，其余逐 byte 使用大写十六进制 `%HH`，不得使用 `+` 代替空格。ASCII、Unicode、引号/反斜杠输入、控制字符拒绝、空 fallback 和截断都必须有完整 header 字符串 fixture；任何响应 header 都不得直接拼接原值。
6. 内部持久状态为 `STAGING`、`UPLOADED`、`BOUND`、`REMOVING`、`DELETING`。只有 `UPLOADED` 和 `BOUND` 属于稳定业务状态；其余状态只用于 durable recovery，不作为成功 DTO 暴露。未绑定附件 `expiresAt = createdAt + 24h`；绑定关系与顺序放在独立 `order_attachments` 映射表，绑定后清空过期时间。
7. 每个订单最多 10 个附件，`attachmentIds` 必须唯一并保留输入顺序。创建订单时字段缺失与空数组等价；5B 编辑时字段缺失表示保留现状，显式数组表示最终集合，显式空数组表示移除全部附件。5A 的既有 `PATCH /api/v1/orders/{orderId}` 不开放 `attachmentIds` 输入：省略该字段必须完整保留现有映射与顺序，显式传入按严格未知字段返回 `400 INVALID_REQUEST`；普通订单字段编辑成功后仍返回完整附件摘要。
8. 新绑定附件必须由当前主体创建、未过期、状态为 `UPLOADED`；已绑定到当前订单的附件可在 edit 最终集合中保留；外部主体、其他订单、过期或删除中的附件统一按不可用附件处理，不泄露存在性。
9. order create 新请求使用 digest/snapshot v2。先由 `normalizeCreateV2` 复用现有业务规范化：`customerName`、每个 item 的 `sku/name` 仅执行现有 `strings.TrimSpace`，`currency` 和 item/attachment 顺序保持，order 字符串不额外做 Unicode normalization；`attachmentIds` 缺失或 JSON `null` 在严格 DTO 边界统一为非 nil 空数组，元素必须是已验证的小写 attachment ID。随后 `encodeCreateDigestV2` 只编码显式 typed struct：顶层字段顺序固定为 `operation/customerName/currency/items/attachmentIds`，每个 item 固定为 `sku/name/quantity/unitPrice`，不得使用 map、`omitempty`、缩进或尾随换行。字符串和整数精确采用 Go `encoding/json.Marshal` 的 UTF-8 JSON 字节规则：引号、反斜杠、控制字符、`<>&`、U+2028/U+2029 按该实现转义；整数为 int64 的最短十进制形式，无加号、前导零或指数；数组保持输入顺序，未知字段在 HTTP decoder 处拒绝，digest encoder 不表示 missing/unknown。摘要只对该字节序列执行 SHA-256。历史 snapshot v1 和其 digest 永不改写。幂等查询先按主体/method/route/key 取得记录再按 `snapshotVersion` 分派：命中 v1 时只有缺失、`null` 或空 `attachmentIds` 可按旧 digest 重放，任何非空数组一律返回 `409 IDEMPOTENCY_CONFLICT`；v2 使用新 digest。未知版本返回 internal，禁止放宽 v1 decoder 兼容 v2。P0 fixture 必须固定完整 bytes 与 digest，覆盖 Unicode 组合/分解形式、转义、HTML 字符、U+2028/U+2029、整数边界、空 attachment 数组、输入 JSON 字段顺序差异和 attachment 顺序差异。
10. 订单列表只新增 `attachmentCount`，不加载完整附件；详情、首次 create、edit 和履约 Action 返回 `attachments` 摘要数组。摘要固定包含 `id/fileName/contentType/sizeBytes/sha256/createdAt`，不包含绝对路径、存储键、创建者内部 ID 或过期清理字段。
11. 上传成功返回 `201` 和顶层 `id`，同时返回与订单一致的附件摘要以及 `expiresAt`；首版响应禁止顶层 `url` 或任何公开/本地 URL，确保当前优先读取 `url` 的 Schema-UI UploadAction 只能以 `id` 写入表单值。删除未绑定附件成功返回 `204`；下载成功返回原始 bytes。
12. 下载仅允许已绑定附件，并重新执行订单查看授权。当前订单查看对所有 authenticated 角色开放，因此四类角色均可下载；未认证为 `401`，不可见或不存在统一为 `404`。
13. 删除 endpoint 只允许 `operator/admin` 删除本人创建的 `UPLOADED` 附件；本人已绑定附件返回 `409 STATE_CONFLICT`，他人附件和不存在附件统一为 `404`。
14. SQLite 与文件系统之间使用持久化状态、`attachment_file_op_groups` 和 `attachment_file_ops` intent/tombstone 编排，不能只依赖进程内 defer 补偿。上传在 final rename 前先提交 `STAGING` 元数据；删除、过期清理和 edit-remove 在移动文件前先通过 CAS 取得唯一 operation token 并提交 `PREPARED` intent。多附件 edit/order-delete 必须属于一个持久组，记录目标订单、期望版本、组 phase 和期望明细数。只有最终业务事务提交后才能把组和明细标为 `COMMITTED`；所有 trash 文件名必须与 operation token 可验证关联。
15. 文件存储只接受服务端生成的固定 basename，并以启动时打开且全生命周期持有的 Go 1.26 `os.Root` 访问 `temp/final/trash`；禁止把客户端文本拼接为路径。`os.Root` 只提供 root-relative containment 基线，不等价于 no-follow、普通文件、同文件系统或 Windows share-mode 保证。平台适配接口必须冻结并实现 `OpenFinalForRead`、`RenameNoReplace`、`RestoreNoReplace`、`PurgeTokenFile` 和文件身份读取语义；附件根目录及其父目录必须由服务账号独占写入，能够以同一 OS 身份或更高权限并发改写该目录的主体不在应用层 TOCTOU 防护承诺内。打开后必须以句柄确认普通文件并记录轻量身份：Unix 使用 `st_dev/st_ino`，Windows 使用 volume serial + file ID；hash 后到绑定事务若重新打开，必须比较身份、size、mtime 和冻结摘要。平台原型必须证明 containment、同卷、目标不存在、symlink/reparse 拒绝、普通文件和 rename 前后身份关联；若标准库不足，平台适配层使用等价 syscall。静态 `Clean`/前缀判断、单独采用 `os.Root` 或先 `Lstat` 后按裸路径操作都不能作为完整安全证据。
16. 下载 header 按状态冻结：`200` 包含探测后的 `Content-Type`、精确 `Content-Length`、安全 `Content-Disposition`、强 ETag `\"sha256-<64 lowercase hex>\"`、`Accept-Ranges: none`、`X-Content-Type-Options: nosniff`、`Cache-Control: private, no-store` 和完整 body；`304` 保留 `Content-Disposition/ETag/Accept-Ranges/nosniff/Cache-Control`，不发送 `Content-Type/Content-Length/Content-Range/body`。`If-None-Match` 支持 `*` 和逗号分隔的强/弱标签并按弱比较命中；首版不解析 Range，任意 `Range` header 在完整性校验与条件请求处理后被忽略并返回完整 `200`，不得伪装为标准 `416 Range Not Satisfiable`；`HEAD` 保持 `405`。必须用真实 `httptest.Server` 断言线上可观察 header，而不是只断言 recorder 内部值。
17. 所有完整 size/SHA-256/type 校验共享一个进程级 integrity admission pool，固定 8 槽、每槽最多一个 10 MiB 复用缓冲，即最多 80 MiB 文件内容缓冲，不含 Go/runtime 与 HTTP 额外开销。download 每请求占一槽；create bind 每请求顺序校验附件且任一时刻只占一槽，不得为 10 个附件并行分配 100 MiB；cleanup 最多使用 2 个 worker，离线 verify 默认 2 个 worker且不得超过 8 槽。HTTP 未取得槽位不排队，返回 `503 SERVICE_UNAVAILABLE` 与 `Retry-After: 1`；admin 命令只允许在自身总 timeout 内可取消等待，取消、读取失败和关闭必须释放槽位。下载使用同一个安全打开的普通文件句柄，在提交任何 `200/304` 响应头前完成完整校验，并在“读到 `BOUND` 映射、完成订单授权并打开身份匹配的 final 句柄”处线性化；在线性化点后即使 edit 进入 `REMOVING`，该在途下载仍允许完成，只有此前观察到 `REMOVING` 的请求返回 `503`。P0 必须分别在 Windows/Linux 用真实句柄冻结“读句柄保持打开 → 并发 `final -> trash` rename → 原句柄继续读”的平台语义和 Windows `CreateFile` share mode；若 Windows 不能在保持读句柄时 rename，edit-remove 必须稳定返回 unavailable/重试并保持 PREPARED，不得宣称跨平台并行成功。永久元数据/内容损坏返回 `500 INTERNAL_ERROR`，临时 open/read/锁/admission/资源不足返回 `503`；响应提交后的客户端取消或 socket 写失败只记录安全传输分类并中止。响应和日志不得包含本地路径。
18. 路由关闭能力使用测试装配字段 `DisableAttachmentRoutes`，关闭 upload/download/delete 三条附件路由，但不关闭订单 API；它不是环境变量或生产 feature flag。
19. CORS 的允许方法必须包含 `DELETE`；actual response 的 `Access-Control-Expose-Headers` 固定为 `X-Request-ID, Content-Disposition, ETag, Accept-Ranges`。`200/304` 和 JSON 错误均保留合法 Origin 的 CORS header；`DisableAttachmentRoutes` 后三条附件 path 的 preflight 与 actual request 都回到 `404`。
20. 上传采用增量 multipart 协议：验证首个 `file` part 后创建可识别 temp 并流式写入，同时继续读取至 EOF 以证明没有额外字段、重复 `file` 或第二个文件；后续结构错误进入统一清理路径。正常协议/校验失败必须立即删除可识别 temp；只有真实进程退出或 sync/rename/DB 边界失败允许保留可识别 temp 供恢复；匿名、越界或无法绑定 attachment ID 的 temp 必须拒绝并清理/隔离。上传读取使用 `http.ResponseController.SetReadDeadline` 设置请求级绝对 read deadline；生产装配固定为 30 秒，handler 只允许通过非环境变量的依赖注入覆盖 duration 供测试使用。所有 `ResponseWriter` 包装器必须实现 `Unwrap() http.ResponseWriter` 以穿透到真实连接。真实 `httptest.Server` 用短 duration 验证穿透和 `408`，另以装配测试断言生产值精确为 30 秒；超时按上述失败类别处理 temp，不使用真实等待 30 秒作为常规测试门禁。
21. 错误分类冻结为：持久元数据约束损坏、非法状态、hash/size/type 永久不一致为 `500 INTERNAL_ERROR`；临时 open/read/write/sync/rename/remove、SQLite/文件锁竞争、下载 semaphore 满和尚未收敛的 operation 为 `503 SERVICE_UNAVAILABLE` + `Retry-After: 1`；已验证不存在或按不可见策略隐藏为 `404 NOT_FOUND`；本人稳定 `BOUND` 删除为 `409 STATE_CONFLICT`；上传 read deadline 为 `408 REQUEST_TIMEOUT`。日志使用同一分类且不记录绝对路径。
22. 在使用 maintenance lock 前先升级 `internal/applock` 为稳定锁对象和显式 `Shared/Exclusive` 模式：锁文件创建后持久保留，`Close` 只解锁/关句柄而不删除路径；Unix 使用非阻塞 `flock(LOCK_SH|LOCK_EX)`，Windows 使用持久 handle + `LockFileEx` 的共享/独占 byte-range lock。API 在打开 DB/root、schema gate、完整性扫描、恢复和构造 service/handler 之前取得 `API exclusive -> maintenance exclusive`，对外服务前释放 maintenance 并继续持有 API lock；`migrate/reset/seed` 要求 API 停止并按同一顺序取得两把独占锁；`cleanup-attachments` 只取得 maintenance shared。`bootstrap-admin` 冻结为 maintenance-only 特殊入口：允许 API 已在运行，不取得 API lock，只取得 maintenance exclusive，打开 DB 后执行 manifest 对应 schema gate，并仅在 production、users 空表时写 bootstrap 用户；不得打开附件 root、执行 migration/recovery/scan 或构造业务 handler。它与 API 的普通 SQLite 写通过现有 busy timeout/writer serialization 竞争，与 cleanup/migrate/reset/seed 由 maintenance lock 互斥。锁忙统一分类为 unavailable/明确 CLI 失败，不得等待形成隐式死锁。三进程 close/reacquire，以及 API/migrate/cleanup/reset/seed/bootstrap-admin 的完整真实子进程互斥矩阵和 API 在线时 bootstrap-admin 成功/空表竞争/失败释放是 P0/M1A 阻塞门禁。
23. 运行期 cleanup 不接管任何其他 owner 的 `STAGING`、`REMOVING` 或 `PREPARED`；它只领取稳定过期 `UPLOADED`、自己新建的 operation、可幂等完成的 `COMMITTED`、旧 temp 和 orphan final。遗留 `STAGING/PREPARED` 仅由尚未对外服务、持有独占 maintenance lock 的启动恢复处理，从而避免超时 owner 与接管者同时 rename。原 owner 每次文件操作前仍须重读并验证自己的 token/phase，发现变化立即停止。orphan final 必须在 SQLite 写事务中重新确认 attachment 与活动 operation 均不存在后插入唯一 `ORPHAN_FINAL` claim；上传的 `STAGING` 写事务必须反向确认同 ID 无 claim。SQLite writer serialization、`attachment_file_ops(attachment_id)` 唯一性和隔离前 token 复核共同保证同名 upload/cleanup 只有一个 owner。阶段五不引入常驻 cleanup 调度 goroutine；第 29 项的安全停服 watchdog 是唯一常驻例外，且不得执行或接管 cleanup。
24. order create 幂等顺序固定为：规范化请求并 lookup → 命中记录时按 snapshot version 比较 digest 并直接重放 → miss 时在事务外只校验文件内容/摘要/类型/身份并预构造候选 projection → 写事务第一条写入完整、最终的 v2 snapshot reservation，使用 `INSERT INTO idempotency_keys (...) VALUES (...) ON CONFLICT (principal_user_id, method, route, idempotency_key) DO NOTHING`，不得使用宽泛的 `INSERT OR IGNORE`，也不引入 pending 行或额外两阶段 schema → loser 按 `RowsAffected=0` 立即读取 winner 并重放，不再校验 owner/state/expiry → 只有 reservation winner 才批量重验 owner/state/expiry/轻量身份、创建订单和映射并提交。事务外文件校验、候选 projection 构造或重新打开身份复核的每个失败出口，在返回文件错误前必须按同一 scope/key 再做一次 winner lookup；若 late winner 已出现，立即按 winner snapshot version/digest 重放或返回幂等冲突，只有 winner 仍不存在时才能返回原文件错误。winner 后续失败时 reservation 随事务回滚，loser 可在下一次尝试中重新成为 winner。正常唯一键竞争和 SQLite busy 在固定 5 秒预算内重试 winner 读取；预算耗尽才返回稳定 `503`，不得用先前附件状态覆盖 winner 结果。v7 不改变 `idempotency_keys` 的非空列契约；P0 必须冻结完整字段值、`RowsAffected` 分支、late-winner 文件失败和 winner 回滚后的双连接 fixture。
25. snapshot v2 选择“首次 projection 冻结”方案：`internal/order` 的 create 用例返回显式 `CreateResult`，同时携带领域订单与首次冻结的 Order response projection；handler 对首次和重放都直接编码该 projection，不按重放时角色重新计算 capability。v1 保持现有 decoder/默认字段路径，P0/M3 计入 service/repository/handler 接口重构。
26. P0-1 必须先同步架构、路线图、领域模型、目标 API 和验证矩阵，完成后才能编写 schema/API：阶段五只交付由受信任 admin maintenance 编排直接调用的订单删除 cleanup 原语，不新增订单删除 HTTP。原语只允许仍为 `DRAFT` 且 expected version 匹配的订单，失败分类顺序固定为资源存在性 → version → 状态 → 退款历史；任一 `refunds` 或 `refund_idempotency_keys` 历史在取得附件 ownership 前稳定拒绝且不改变 DB/文件事实。订单删除成功后永久保留 order-create v1/v2 `idempotency_keys` snapshot；相同 scope/key 继续重放首次冻结响应而不查询当前订单、不得创建第二个订单，后续按订单 ID 查询仍返回 not found。目标 API 使用显式允许角色并声明无删除 endpoint；领域模型承接附件状态、上限、owner/expiry/绑定、下载、清理限制和幂等保留语义；架构先冻结每个 SQLite 原子阶段的事务边界与跨文件系统恢复责任。transport 细节仍由本计划持有，实施完成后迁入当前 API。
27. 5A 若作为可部署候选，必须交付最小 `cleanup-attachments`：处理过期 `UPLOADED`、5A 单项 `COMMITTED` tombstone 和旧可识别 temp，并记录可重复执行的运维调度证据；复杂 group/orphan 接管仍留在 5B。5A 的“可部署”只表示单进程、受控的 demo/内部环境候选，不构成生产或多副本部署批准；本阶段没有磁盘配额或未绑定附件 admission 控制，不能把 `ENOSPC` 收敛证据当作容量保护。没有该命令、未演练定期执行、未冻结 `ENOSPC` 为 `503` 且无残留，或未记录该环境边界时，5A 只能作为短期内部候选，禁止部署到持续可访问环境。
28. M1A 开工前固定一套真实进程 crash harness 协议：边界名称、强制退出方式、重启命令、预期 DB/文件系统状态和 evidence 路径。M1A-10、M1B-5、M4-9 必须复用同一格式，函数级 failure injection 只能补充定位，不能替代进程退出证据。
29. 运行期 unresolved 采用仓库自洽的应用驱动契约：生产 watchdog 每 5 秒检查一次，clock/ticker 可注入；unresolved 集合只包含超过阈值仍需恢复所有权或阻止安全服务的 `STAGING`、`PREPARED`、`REMOVING`，以及仍存在 final/状态不一致、不能归类为“业务已提交且只待 purge”的 `COMMITTED`。`DELETING + COMMITTED` 且 final 已不可下载、只剩 token trash/tombstone purge 或 metadata 幂等删除的条目属于 purge backlog，不触发 API watchdog，其年龄只由 cleanup 指标和告警管理。任一 unresolved operation 年龄严格超过 5 分钟时，在最多一个 tick 内把 `/readyz` 置为 not-ready，并只调用一次注入的 shutdown callback。callback 停止接收新请求、给予在途请求 15 秒宽限期，随后让 `cmd/api` 以固定退出码 `75` 退出；重复检测不得重复关闭资源或重复回调，无 probe 流量也必须触发。该 watchdog 只负责安全停服，不执行恢复或 cleanup。`cleanup-attachments` 由外部调度执行：仓库必须交付并实测至少一个受支持的单机调度示例，固定每 15 分钟一次、单次超时 10 分钟、禁止重叠、非零退出触发可观察告警；测试必须覆盖纯 purge COMMITTED 超过 5 分钟不自停、含 final 的异常 COMMITTED 触发、cleanup 超时、错过一次调度和 shutdown 竞争。没有调度与进程监督证据时不得批准持续部署。
30. 启动同步扫描限定为结构门禁：恢复全部 journal 后，按 500 行批次检查 schema、组/明细完整性、状态/映射计数、服务端 basename、final 存在性、普通文件身份和记录 size；不得在启动路径读取所有稳定文件计算 SHA-256 或重新探测 content type。扫描使用注入 clock，生产总预算固定 30 秒硬上限，超时 fail-closed 并输出不含路径的诊断码；完整 hash/type 校验保留在 bind、download、cleanup 和离线 `admin verify-attachments --full` 中。verify 要求 API 停止并按 `API exclusive -> maintenance exclusive` 取得双独占锁，只读打开 DB/root，不执行 migration、recovery 或修复；仅允许 v7 feature admin，按 500 行批次和统一 integrity pool 完成 size/hash/type/identity 校验，支持取消并输出机器可解析摘要。退出码固定为 clean `0`、发现完整性问题 `2`、调用/I/O 失败 `1`、capability/schema gate `78`。支持规模、冷/热缓存基线和超预算运维步骤按第 35 项冻结。

31. P0-1 的事实源同步必须明确覆盖 `docs/05-domain-model.md`、`docs/03-http-api-target.md`、`docs/06-implementation-roadmap.md` 和 `docs/04-validation.md`，并在 Evidence 逐项记录替换文本与行号；该独立文档变更进入可引用 revision 前不得编写 v7 schema、附件路由或订单 DTO。
32. stable applock 原型必须冻结可复用的 `Acquire(path, Mode)`/释放 API，并先迁移现有 API、reset、seed、migrate、cleanup、bootstrap-admin 入口，再让附件恢复依赖 maintenance lock；不能把现有独占锁实现视为已满足前置条件。
33. M1A 开工前固定可版本化的真实进程 crash harness/fixture/脚本，落在 `internal/testkit/phase5` 或等价受审计目录，明确边界名、强制退出、Windows/Linux 运行方式、重启命令、预期 DB/文件状态、ENOSPC 注入、监督器/调度 profile 和 Evidence 路径。systemd/timer 与 Windows Task Scheduler/Service 中至少一个必须成为实测支持部署 profile，另一平台只能标为开发/CI 兼容，除非同样完成监督、调度与恢复证据。
34. 退出契约拆成两个不可混用的状态机。构造期从取得第一把锁开始，到 listener 成功创建且 server ownership 交给运行期为止，只产生 `StartupError`：schema/capability gate 为 `78`，listen 或其他构造失败为 `1`；watchdog 不是构造期输入。外部停止若先到且尚无固有构造错误，只取消当前 stage、逆序释放并退出 `0`；若 stage 已产生固有错误，该 `StartupError` 获胜。进入运行期后才启用一次性 shutdown-reason arbiter/typed `RunResult`，优先级固定为监听后 serve/运行失败 `1` > watchdog restart `75` > 外部 SIGTERM/普通 Close `0`；重复原因幂等，较低优先级不能覆盖已选定原因。真实子进程必须覆盖每个 startup stage 的外部停止、listen 失败与停止竞争、watchdog/运行失败/停止竞争和重复触发。
35. 启动 30 秒预算是硬安全上限。P0 必须先冻结至少一套支持规模（附件数、映射数、operation 数和平均 metadata 成本），提供可生成 fixture，比较相同记录数、不同文件大小下的冷/热缓存 wall-clock，记录失败诊断码和离线修复命令；只有实测后才能决定是否同时把 30 秒声明为 SLO。
36. P0 交付拆成八个独立工作包，并按下表把 P0-1 至 P0-25 恰好映射到一个主 owner。owner 是可追责角色，开始执行时必须在 Evidence 中补充具体人员；reviewer/批准人不能与主 owner 为同一人。工作包可并行审查，但全部出口条件满足前不得进入 M1A，也不得把 ROM 当作阶段日期承诺。
37. 若 NFC 使用 `golang.org/x/text/unicode/norm` 或其他新增依赖，P0 必须记录模块版本、`go mod tidy`、`go.mod/go.sum` diff 和许可证/供应链检查；禁止用自制 Unicode 规范化算法替代成熟依赖。
38. 所有构造期入口共用一个 startup coordinator/typed gate API，统一覆盖 `NewAPI`、`NewAuthenticatedAPI`、`Handler()`、embedded entry、migrate、seed、reset、cleanup 和 verify 的允许阶段。契约冻结为：构造期只返回 `StartupError{Stage, Kind, ExitCode, Err}`；listener 成功并完成 ownership handoff 后，运行结束只返回 `RunResult{Reason, ExitCode, Err}`，其中 runtime `ExitReason` 只区分 serve/运行失败、watchdog restart 和外部停止，schema gate 只属于 `StartupError`。统一完整生命周期为“取得 API/maintenance lock → 打开 DB/root → migration/schema gate → recovery → integrity scan → service/handler 构造 → 创建 listener → 释放 maintenance → handoff/serve”；任一步失败都按逆序关闭 listener、handler/service、root、DB，再释放 maintenance/API lock。`Handler()` 只能在 coordinator 达到 ready 状态后取得，测试只能注入依赖和 clock，不能跳过阶段。schema-only admin 的 `migrate` 子命令是受限 migration path：在打开业务 service 前允许读取 v6/v7 并仅执行 v6→v7 或幂等 v7 校验；v6 bootstrap admin 只允许 v0..v6 并停在 v6。production `bootstrap-admin` 是第 22 项冻结的 maintenance-only 特殊 path，只复用 config、maintenance lock、DB open、compiled capability/schema gate 和逆序释放，不得进入附件 root/recovery/scan/service 生命周期。`verify-attachments --full` 是 API 停止后的 v7 feature-only 只读 path，取得双独占锁后进入 DB/root 与完整性扫描，但不得执行 migration/recovery/cleanup/service。其他 admin 子命令和 API 只允许其 compiled capability 声明的 schema/data。部署顺序、每步允许的 schema/data、失败恢复点、退出码和各 binary SHA-256 必须形成可执行链路。
39. P0 必须交付机器可解析的 Evidence/manifest 最小工具链，放在 `internal/testkit/phase5` 或等价受审计目录：固定 JSON schema、全局唯一 run ID、Owner、Date/Timezone、Revision、Environment、Command、Artifact/Log Path、Result、Failure/Not-Run Reason、Artifact Kind、Process Exit Code、Content SHA-256、Download URL、Retention Until、Manifest Path/URL 和 Manifest SHA-256。manifest 本体不自包含自己的 SHA；其 SHA 记录在 tracked run index。validator 必须核对 revision、artifact kind、文件/manifest SHA-256、退出码、路径/URL 可取得性及 manifest 与 run 记录的一致性。核心 provenance 字段不得写 N/A；仅 schema 标记为 optional 的非产物字段可写 JSON `null`，并同时提供结构化 `notApplicableReason`。`internal/testkit/phase5/testdata/evidence/` 保存可在 clean checkout 复核的正反样例；真实 run 的小型 manifest/index 从 M1A 起提交到 `docs/evidence/phase5/<run-id>/`。默认大型存储后端冻结为当前仓库 GitHub Actions 的 `phase5-evidence-<run-id>` artifact，工作流使用 `actions/upload-artifact@v4`、`retention-days: 180`，下载页 URL 固定关联 GitHub Actions run/artifact ID，读权限为仓库 Actions read，上传责任人为 release owner。P0-23 必须先验证仓库/组织策略确实允许 180 天；若不允许，release owner 必须在 `docs/evidence/phase5/storage-profile.json` 冻结具备至少 180 天保留和只读下载 URL 的替代存储后端后才能继续，上传失败或 URL/SHA 校验失败保持 No-Go，禁止降级为本机 `artifacts/`。最终部署批准 bundle 至少保留到对应支持版本生命周期结束。P0-14 产生可复核的真实基线样例；P0-19 产生 `contract-fixture` 或 `disposable-spike`；P0-22 只产生 `artifactKind=contract-fixture` 的目标 profile/配置/命令/退出码/告警 acceptance 样例。真实监督器、cleanup 调度、watchdog/recovery、ENOSPC 与部署 profile run 只能由 5A-D 或 5B-4 产生；仅有 Markdown 勾选或第三方报告不能作为完成 Evidence。
40. 协议变更集合以本计划中的单一标识 `P5-CRITICAL` 为事实源，当前成员为 P0-1、P0-2、P0-3、P0-5、P0-7、P0-10、P0-12、P0-13、P0-15、P0-17、P0-18、P0-19、P0-20、P0-23、P0-25；checklist 只引用该标识，不再复制成员列表。任何事务顺序、状态、DTO、错误码、digest/header 字节、Evidence schema/storage、锁/启动语义、integrity admission 或平台能力变化，必须在同一独立 revision 同步更新 plan、checklist、事实源、原子验收矩阵和 Evidence 模板，记录受影响里程碑并重新触发相关 gate；禁止通过实现分支或发布说明局部放宽门禁。
41. P0-25 必须交付 tracked、机器可解析的 requirements-to-test/evidence matrix 和 validator。每个 checklist 父项至少有一个稳定 acceptance ID `<parent-id>.A<n>`，字段至少包含 owner、test/command、artifact kind、适用平台、预期结果、实际 result、run ID 与结构化 N/A 原因；父项只有在全部适用 acceptance 完成后才能勾选。validator 必须拒绝无子项、重复 ID、缺字段、父项已完成但子项未完成，以及矩阵与 `P5-CRITICAL` 漂移。

P0 tracked work-package 表如下。`输入 revision` 在工作包开始时必须替换为完整 commit；最大 timebox 单位为开发 effort 人日，是单包上限而非 elapsed 日期承诺，超时必须停止扩范围并触发重估。

| 工作包 | 主 P0 条目 | owner / reviewer | 输入 revision 与阻塞项 | 最大 timebox | 出口 Evidence 与变更批准 |
|---|---|---|---|---:|---|
| WP-Facts | P0-1 | 阶段五协议 owner / domain+API reviewer | 本计划修订 revision；无代码前置 | 1 人日 | `docs/01-architecture.md`、`docs/05-domain-model.md`、`docs/03-http-api-target.md`、`docs/06-implementation-roadmap.md`、`docs/04-validation.md` 与 plan/checklist diff、行号、独立 revision；协议 owner 批准 |
| WP-Schema-Recovery | P0-2、P0-4、P0-9、P0-10 | storage/recovery owner / runtime reviewer | WP-Facts；迁移与状态机草案 | 3 人日 | migration/trigger、真值表、crash 边界与双进程 fixture；协议 owner 批准 |
| WP-HTTP-Order | P0-3、P0-5、P0-6、P0-7、P0-8、P0-11、P0-12 | HTTP/order owner / security reviewer | WP-Facts；现有 v1 fixture | 3 人日 | multipart/DTO/幂等/header/error 固定 fixture；协议 owner 批准 |
| WP-Lock | P0-13 | runtime lock owner / storage reviewer | WP-Facts；现有 applock revision | 2 人日 | 锁 API/入口迁移规格、三进程 fixture 与可抛弃跨平台 spike；runtime owner 批准 |
| WP-Baseline-Evidence | P0-14、P0-23、P0-24、P0-25 | Evidence/tooling owner / release reviewer | WP-Facts；本计划修订 revision；留存 profile | 3 人日 | schema、validator、clean-checkout 样例、原子验收矩阵、真实 run index 与协议变更清单；release owner 批准 |
| WP-Files | P0-15 | file adapter owner / security reviewer | WP-Lock 接口；Go 1.26 | 3 人日 | os.Root、平台适配接口、身份、rename/share mode、reparse/symlink 原型；security reviewer 批准 |
| WP-Runtime | P0-17、P0-18、P0-20 | startup/runtime owner / app reviewer | WP-Lock；startup 类型契约 | 3 人日 | coordinator/verify 接口与状态机、进程 harness/规模 fixture 规格、可抛弃 spike；真实 cmd/api 与 wall-clock 留给 M1A；runtime owner 批准 |
| WP-Release | P0-16、P0-19、P0-21、P0-22 | release owner / Evidence reviewer | WP-Facts、WP-Schema-Recovery、WP-HTTP-Order、WP-Lock、WP-Baseline-Evidence、WP-Files、WP-Runtime 的已批准输出；Windows/Linux 环境 owner 与目标 profile 输入 | 2 人日 | 入口/版本矩阵、四发布产物 manifest schema/命令 fixture、目标 profile 配置与 acceptance contract、逐包重估与 DAG 校验；真实 release artifact/profile run 留给 M1A/5A-D/5B；阶段五协议 owner 批准 |

## 4. 数据模型与文件布局

### 4.1 schema v7

`attachments` 持有：

- `id`、`created_by_user_id`、`status`；
- `original_name`、`content_type`、`extension`、`size_bytes`、`sha256`；
- nullable `expires_at`；
- `created_at`、`updated_at`。

`order_attachments` 单独持有绑定与顺序：

- `order_id`、`attachment_id`、`position`、`bound_at`；
- `attachment_id` 主键保证单附件最多绑定一个订单，`UNIQUE(order_id, position)` 保证单订单位置唯一；
- edit 最终事务先删除当前订单映射，再按最终有序集合批量重建，因此两项交换、循环置换和满 10 项逆序都不依赖临时 position。

`attachment_cleanup_observations` 持有不可直接信任文件时间时的隔离期事实：

- `(kind, basename)` 唯一标识 `TEMP/ORPHAN_FINAL` 候选，保存 `first_seen_at`、最近文件身份与 size；所有时间为 UTC 秒精度；
- 首次发现只写 observation，不删除文件；只有同一身份连续存在满 1 小时（temp）或 24 小时（orphan final）且仍无附件/活动 operation 引用时，才能进入 claim/清理；
- 身份变化、`first_seen_at` 位于未来或检测到时钟回拨时重置为当前观察时间；文件消失或重新成为合法引用时删除 observation。裸 `mtime`、文件名时间和管理员触碰时间都不是删除依据。

DDL 可表达约束冻结为：

- `attachments.status` 只允许 `STAGING/UPLOADED/BOUND/REMOVING/DELETING`；`size_bytes` 为 `1..10485760`；`sha256` 使用 32-byte BLOB；`order_attachments.position` 只允许 `0..9`；
- `STAGING/UPLOADED` 的 `expires_at` 非 NULL，`BOUND/REMOVING/DELETING` 的 `expires_at` 为 NULL；
- `created_by_user_id`、`order_attachments.order_id` 与 `order_attachments.attachment_id` 外键均使用 `ON DELETE RESTRICT`，未来订单删除必须先调用附件清理原语；现有 `refunds.order_id` 与 `refund_idempotency_keys.order_id` 保持 RESTRICT 语义，不新增级联删除。ORDER_DELETE 在准备事务内先断言两表均无目标订单记录，存在任何退款历史时返回稳定 state conflict 且不得创建 group/op 或移动文件；`attachment_file_op_groups.order_id`、`attachment_file_ops.original_order_id` 和历史 `attachment_file_ops.attachment_id` 不建立会级联或 RESTRICT 的外键，它们是恢复快照/墓碑标识，必须保留到 purge 完成；`group_id` 仅在组生命周期内受约束。BUILDING→PREPARED trigger、最终事务 token 校验和启动扫描负责补足一致性，并固定覆盖“订单已删除但 COMMITTED tombstone 仍存在”的 fixture；
- `UPLOADED/STAGING/DELETING` 最终不得存在映射，`BOUND/REMOVING` 最终必须恰有一个映射。SQLite 无法把该跨表、事务结束时 invariant 完整表达为可延迟 trigger，因此 repository 在每个写事务提交前执行集合校验，启动扫描和损坏 fixture 作为第二道门禁；不得声称由简单 DDL 自动保证；
- 原始文件名 1..255 UTF-8 bytes 由应用层在写入前校验，不能误用 SQLite 字符数代替 byte 长度；
- 按 `order_attachments.order_id, position` join `attachments` 稳定读取；
- 按 `status, expires_at, id` 批量清理过期未绑定附件；
- 按 `created_by_user_id, status` 校验 owner 删除和绑定。

`attachment_file_op_groups` 至少持有 `group_id/kind/order_id/expected_order_version/expected_count/phase/created_at/updated_at`；`kind` 只允许 `EDIT_REMOVE/ORDER_DELETE`，`phase` 增加事务内组装态 `BUILDING`，并只允许 `BUILDING/PREPARED/COMMITTED`。`attachment_file_ops` 作为 durable intent/tombstone，至少持有 `operation_id/group_id/attachment_id/kind/phase/original_status/original_order_id/original_position/original_bound_at/original_expires_at/source_name/trash_name/created_at/updated_at`；`kind` 只允许 `DELETE_UNBOUND/EXPIRE_UNBOUND/EDIT_REMOVE/ORDER_DELETE/ORPHAN_FINAL`，`phase` 只允许 `PREPARED/COMMITTED`。其中 `original_order_id/original_position/original_bound_at` 是从 `order_attachments` 冻结的恢复快照，不是 `attachments` 当前列。DDL 负责枚举、NULL 组合、长度、唯一性、外键、组装态不可被恢复查询读取，以及每个 attachment ID 同时最多一个 operation claim；不声称普通 `CHECK` 可以表达跨表计数、组/明细 kind 一致、绑定状态映射或派生 basename。

跨表完整性采用“`BUILDING` + 转相 trigger”方案：repository 在同一 SQLite 写事务中插入 `BUILDING` 组、全部明细和附件状态 CAS，最后执行单条 `BUILDING -> PREPARED`。明细写入/转相 trigger 验证普通 operation 存在对应附件行、只有 `ORPHAN_FINAL` 可无附件行，以及有组明细的 kind 与 `BUILDING` 组 kind 一致；组的 `BEFORE UPDATE OF phase` trigger 在 `BUILDING -> PREPARED` 时验证明细数等于 `expected_count`，不满足则 `RAISE(ABORT, ...)`。`source_name/trash_name` 由 repository 解析 attachment ID、固定扩展名和 operation token 后逐字段重算并比较；启动恢复先执行同一完整性扫描，遇到绕过 trigger 或损坏数据时拒绝启动。组装事务回滚后不得留下 `BUILDING`；所有恢复和运行查询只接受 `PREPARED/COMMITTED`。P0 必须提交可执行 migration 草案、trigger SQL、事务顺序及使用 `ignore_check_constraints`/直接破坏 fixture 的损坏行测试，不能用笼统“DDL 级组完整性”代替证据。

迁移只增加 schema，不扫描或改写现有订单、退款和幂等 snapshot。当前 v6 实际没有附件表或附件数据，因此无需迁移旧 Attachment；migration 必须覆盖 fresh v0、标准 v6、幂等 v7、存在空的非预期旧附件对象和存在旧附件数据五类 fixture，后两类直接拒绝并回滚，禁止猜测迁移。fresh v0 不能由 v7 schema-only 或 feature binary 隐式执行历史迁移：先使用不可变 `v6-gate` revision 的 bootstrap admin 执行现有 `0001..0006` 并在 `user_version=6` 停止，再切换到 v7 schema-only admin 执行 v6→v7。v7 migration 前先交付一个仍运行 v6 schema、但在打开业务 service/构造 handler/监听前强制 schema/object gate 的过渡 API binary；随后构建不可变 schema-only `api/admin`，停服并备份后由该 admin 的专用 `migrate` 子命令执行 v6→v7，再由同一源码 revision 的 api/admin 分别用自身 SHA-256 验证空数据 gate；执行迁移前后必须是同一个 admin SHA，API 验证使用该 revision 对应的 API SHA。schema-only artifact 仅用于迁移与首次附件写入前的前滚准备；该 revision 的构造期 gate 固定执行：

```sql
SELECT EXISTS(
  SELECT 1 FROM attachments
  UNION ALL SELECT 1 FROM order_attachments
  UNION ALL SELECT 1 FROM attachment_file_op_groups
  UNION ALL SELECT 1 FROM attachment_file_ops
  UNION ALL SELECT 1 FROM attachment_cleanup_observations
  UNION ALL SELECT 1 FROM idempotency_keys WHERE snapshot_version >= 2
  LIMIT 1
) AS forbidden_phase5_data;
```

除 schema-only admin 的专用 `migrate` 子命令外，schema 非 v7 或查询返回 1 时，`NewAPI`、认证 API 构造、嵌入式入口和其他 admin 子命令返回同一 typed schema-gate error，真实 binary 映射为固定退出码 `78`；不能只把检查放在 `Run` 或 `/readyz`。查询中的附件表集合包含 `attachment_cleanup_observations`。`migrate` 必须在业务 service/root 恢复装配前走独立 typed migration gate，只接受 v6→v7 或幂等 v7 校验，并在失败时保留迁移前恢复点。该 artifact 不能作为阶段五启用后的运行回退版本。功能启用后的回退仍只允许停写并恢复同一时点的 DB/WAL/SHM 与附件目录成套备份；未经 schema gate 的历史 v6 binary 只能视为禁止部署的运维前提，不能宣称具备技术阻断。

发布矩阵在 P0 冻结为以下四行；表内 `<revision>` 在 M1A/5A/5B 产物生成时必须替换为完整 commit，构建必须从该 revision 的干净工作树执行，并把命令、`go version`、文件大小和 SHA-256 写入同目录 manifest。P0 只校验表结构、命令 fixture、capability API 与负向测试规格，不生成这些不可变 release binary：

| artifact kind | 文件名与构建命令 | 支持 schema / 允许数据 | 启动拒绝条件 | 顺序、备份与回退 |
|---|---|---|---|---|
| `v6-gate` | `phase5-v6-gate-api-<goos>-<goarch><ext>` 与 `phase5-v6-bootstrap-admin-<goos>-<goarch><ext>`；从不可变 v6 gate revision 分别执行 `go build -trimpath -tags phase5_v6_gate -o <api-file> ./cmd/api`、`go build -trimpath -tags phase5_v6_gate -o <admin-file> ./cmd/admin` | API 仅 v6 且 `sqlite_schema` 不存在 manifest 列出的任何阶段五 table/index/trigger，`idempotency_keys` 不存在 `snapshot_version >= 2`；bootstrap admin 只允许 v0..v6 并停在 v6 | API：版本不为 v6、存在任一阶段五对象或 snapshot v2 时退出 78；bootstrap admin 对 >v6 或对象冲突退出 78 | fresh 第 1 步由该 admin 执行 v0→v6；legacy 第 1 步部署 API gate；v6 迁移前成套备份；失败时恢复 fresh 目录或继续运行该不可变 v6 revision |
| `v7-schema-only` | `phase5-v7-schema-api-<goos>-<goarch><ext>` 与 `phase5-v7-schema-admin-<goos>-<goarch><ext>`；分别执行 `go build -trimpath -tags phase5_v7_schema_only -o <api-file> ./cmd/api`、`go build -trimpath -tags phase5_v7_schema_only -o <admin-file> ./cmd/admin` | API/其他 admin 仅 v7 且五张附件表为空、无 snapshot v2；admin `migrate` 在业务装配前允许 v6→v7 或幂等校验 v7 | API/其他 admin：schema 非 v7、任一附件行或 snapshot v2 时退出 78；`migrate` 对非 v6/v7 退出 78，对兼容性校验或执行失败退出 1，成功退出 0 | 第 2 步停服后由同一个 admin SHA 执行并复核迁移，再用同源码 revision 的 API 自身 SHA 验证；首次附件写入前可恢复迁移前成套备份 |
| `v7-5a-feature` | `phase5-v7-5a-api-<goos>-<goarch><ext>` 与 `phase5-v7-5a-admin-<goos>-<goarch><ext>`；分别执行 `go build -trimpath -tags phase5_v7_5a -o <api-file> ./cmd/api`、`go build -trimpath -tags phase5_v7_5a -o <admin-file> ./cmd/admin` | v7；允许 5A 状态，5B group/kind fail-closed；产物、CI 输出和部署说明必须标注 `internal/demo-only` | schema 非 v7、无法恢复或出现不支持 group/kind | 第 3 步启用；功能写入后的回退只允许恢复同一时点成套备份，不得换回 schema-only binary |
| `v7-5b-feature` | `phase5-v7-5b-api-<goos>-<goarch><ext>` 与 `phase5-v7-5b-admin-<goos>-<goarch><ext>`；分别执行 `go build -trimpath -tags phase5_v7_5b -o <api-file> ./cmd/api`、`go build -trimpath -tags phase5_v7_5b -o <admin-file> ./cmd/admin` | v7；允许完整 5A/5B group、kind、REMOVING、ORDER_DELETE 与 ORPHAN_FINAL 数据；同样标注 `internal/demo-only` | schema 非 v7、出现未知 group/kind/state、恢复或完整性门禁失败 | 5B 最终候选启用；可从 5A 停服备份后前进，写入 5B 数据后不得回退到 5A/schema-only binary，只能恢复同一时点成套备份 |

每行原始 Evidence 先生成到 `artifacts/phase5/v7/<artifact-kind>/<revision>/evidence/`，随后按第 39 项写入受版本控制的 run index 并上传到已冻结的保留存储；本机 ignored 路径本身不可用于批准。五个 capability tag 必须互斥，并由无 tag/未知 tag/多 tag 的编译失败测试和 binary 自报告 capability 测试证明；startup gate 对自报告值执行实际 schema/data 拒绝，manifest 不能覆盖 binary capability。确定性质量矩阵固定为：`go test -tags phase5_test ./... -count=1`、`go vet -tags phase5_test ./...`、`go test -race -tags phase5_test ./... -count=1`；并对四个发布 tag 分别执行 `go test -tags <tag> ./... -count=1`、`go vet -tags <tag> ./...` 以及表中 api/admin build。`.github/workflows/ci.yml` 必须执行该矩阵；不得依赖开发者临时选择 tag，`phase5_test` 产物不得上传为 release/deployment artifact。Evidence 记录部署命令、数据库路径、备份标识、允许数据扫描结果、退出码、下载 URL、保留截止时间和回退演练。每个 binary 通过只读接口输出机器可验证的 capability manifest；部署前 validator 必须核对 artifact kind、源码 revision、run ID、binary/manifest SHA-256、schema/data scan，并以“错误 capability binary + 错误 schema/数据”负向 smoke 证明拒绝。`v7-5a-feature`、`v7-5b-feature` 及其 CI/release artifact 的 manifest、README 和部署文档必须同时标注 `internal/demo-only`，不得仅依赖文件名或人工说明；文件名只作为辅助标识。历史过渡产物以各自 revision/hash 关联；最终 5A/5B test/race/smoke 的“同一候选 revision”不追溯约束迁移产物。

入口/版本矩阵必须在 P0 固定为可执行 fixture，并由真实 binary smoke 验证：

| 初始状态 | 允许命令/产物 | 预期结果 | 备份与失败恢复 |
|---|---|---|---|
| `v0 fresh` | `v6-bootstrap-admin migrate`，随后 `v7-schema-admin migrate` | 先精确停在 v6，再到 v7 empty；任一步成功 0 | 记录空目录/DB 初始标识；失败删除本次 fresh 数据目录或恢复已记录 v6 中间点 |
| `v6 legacy`，无阶段五对象/data | `v6-gate-api` smoke，停服备份，再 `v7-schema-admin migrate` | gate/API 0，迁移 0，schema-only API 0 | v6 DB/WAL/SHM 与附件目录成套备份；失败恢复该备份 |
| `v7 empty` | 同一 `v7-schema-admin migrate` 重复执行，随后 schema-only API | 幂等 0；不得写业务数据 | 保留迁移前恢复点；失败不得改变 v7 schema/data |
| `v7 with 5A data` | `v7-5a-feature` | 支持状态启动 0；schema-only API 78 | 功能写入后只允许成套备份恢复，不回退 binary |
| `v7 with 5B data` | `v7-5b-feature` | 完整 group/kind/state 启动 0；5A/schema-only API 78 | 写入 5B 数据后只允许成套备份恢复，不回退到 5A/schema-only binary |
| `v7 unsupported data/object` | 任一不声明该 capability 的 binary | 构造期退出 78，不能监听 | 只读诊断并恢复/修复，禁止自动猜测迁移 |
| 非 v0..v7 或对象冲突 | bootstrap/schema-only migrate | 退出 78；执行期 SQL/兼容性失败退出 1 | 保持原数据并引用明确恢复点 |

### 4.2 文件目录

在 `DATA_DIR` 下使用固定受控目录：

```text
attachments/
  final/
  temp/
  trash/
```

配置层派生绝对 clean path；生产仍只配置 `DATA_DIR`，不新增可指向任意位置的附件目录环境变量。进程启动时验证 `DATA_DIR/attachments` 及父目录不存在非预期 reparse/symlink，打开并持有 `os.Root` 后只使用相对服务端键；文件适配器使用同卷 rename，并拒绝任何逃逸、reparse point 或非普通文件目标。部署文档必须要求附件目录及父目录不对不受信任本地用户开放写权限。

### 4.3 durable recovery 状态机

上传固定为：

1. 在确定性 temp 名称中流式写入、sync、close、探测并计算摘要；正常协议/校验失败立即清理；进程退出或 sync/rename/DB 边界失败允许留下可由 attachment ID 识别的 temp；任何匿名、越界或无法绑定 attachment ID 的 temp 都必须拒绝并清理/隔离；
2. 提交 `STAGING` 元数据后才允许 `temp -> final` rename；
3. rename 成功后 CAS `STAGING -> UPLOADED`，只有该提交成功后返回 `201`；
4. 恢复时，`STAGING + temp only` 重新验证后完成 rename/晋级，`STAGING + final only` 重新验证后晋级，temp/final 同时存在且内容相同时删除 temp 后晋级；两处内容不同或路径不安全视为 internal。两处均不存在时仅可删除从未对外可见的 STAGING 行，并记录损坏 evidence；稳定 `UPLOADED/BOUND` 缺失或不安全 final 一律拒绝启动，不能删除元数据掩盖损坏。

删除、过期清理和移除固定为：

1. 先在数据库事务中 CAS 取得唯一 operation token：未绑定附件转为 `DELETING`，已绑定移除转为 `REMOVING`，同时写 `PREPARED` 并保存原状态；未取得 token 的 loser 不得移动文件；
2. 使用 token 将 final 隔离到 trash；任一步失败都由 `PREPARED` 恢复逻辑将文件和原状态一起恢复；
3. DELETE/expiry 在隔离成功后提交 `COMMITTED`；edit-remove 在同一 order writer transaction 内更新订单/items、绑定新增附件、把移除项 `REMOVING -> DELETING` 并将对应 operation 标为 `COMMITTED`；
4. 业务成功只以第 3 步提交为准。随后 purge trash 并删除附件/tombstone；purge 失败不得把已提交业务事实改报失败，保留不可下载的 `DELETING + COMMITTED` 供重试；
5. `PREPARED` 只由独占启动恢复或原 owner restore，`COMMITTED` 可由原 owner 或运行期 cleanup 幂等完成。恢复自身失败保持原 phase 并返回 unavailable；API 启动恢复无法清空遗留项时拒绝启动。

订单删除 cleanup 原语复用 `ORDER_DELETE` 分组 operation，对同一订单的全部 BOUND 附件一次取得 ownership；阶段五只允许受信任 admin maintenance 编排通过 repository/service 直接调用，不注册订单删除路由，也不授权普通 API actor。调用必须携带 expected version；准备事务按资源存在性 → version → 订单状态 → 退款历史顺序校验，只允许 `DRAFT`，并查询 `refunds` 与 `refund_idempotency_keys`，任一失败都稳定拒绝且不得创建 group/op、改变附件状态或移动文件。通过后准备阶段只允许创建 PREPARED group/op、把全部附件 `BOUND -> REMOVING` 并隔离文件，不得删除订单或标记 COMMITTED。store 必须暴露单一最终写事务入口，在同一 SQLite commit 中重新验证订单存在、expected version、`DRAFT`、无退款历史与全部 token，删除 `order_attachments` 和订单聚合行、把附件 `REMOVING -> DELETING`，并把 group/op 标为 `COMMITTED`；`idempotency_keys` 中历史 order-create v1/v2 snapshot 永久保留且不以外键级联。只有该事务提交后才能报告订单删除成功。删除后相同 scope/key 必须继续按 snapshot version/digest 重放首次冻结响应、不查询当前订单且不得创建第二个订单；独立订单读取仍返回 not found。最终事务失败、version/状态变化、退款在准备与最终事务之间出现或进程退出时订单仍存在，整组按 PREPARED restore；直接测试必须覆盖非 DRAFT、version 冲突、退款/退款幂等、跨删除前后相同 key 重试、多附件和恢复。任何恢复分支都不得让稳定 `BOUND` 元数据长期指向缺失 final，也不得让 `DELETING/REMOVING` 文件可下载。

文件事实真值表对所有 delete/expire/edit-remove/order-delete 统一冻结：

| phase | final | trash | 唯一动作与结果 |
|---|---|---|---|
| `PREPARED` | 有 | 无 | 验证 final 后恢复原 DB 状态并删除 operation；视为文件已 restore |
| `PREPARED` | 无 | 有 | 验证 trash 后 rename 回 final，再恢复整组原 DB 状态 |
| `PREPARED` | 有 | 有且相同 | 保留 final、删除 token trash，再恢复整组；不得任选内容 |
| `PREPARED` | 有 | 有但不同 | 保留全部 evidence，启动失败；运行请求返回 `503`，人工处理 |
| `PREPARED` | 无 | 无 | 保留 operation/evidence，启动失败；不得删除元数据伪装成功 |
| `COMMITTED` | 无 | 有 | purge trash，再删除附件 metadata、明细 operation 和已完成组 |
| `COMMITTED` | 无 | 无 | 视为 purge 已完成，幂等删除 metadata/tombstone |
| `COMMITTED` | 有 | 无 | 验证 final 与冻结 metadata 一致后重新隔离并 purge；不一致则启动失败/运行时 `503` |
| `COMMITTED` | 有 | 有且相同 | 验证后 purge 两份并完成 tombstone；不得留下可下载副本 |
| `COMMITTED` | 有 | 有但不同 | 保留全部 evidence，启动失败；不得猜测哪份正确 |

组 operation 必须整组决策：任一明细处于无法安全 restore/finalize 的组合时，不得把组标为完成；订单删除中“部分文件缺失”属于损坏并失败，不得部分删除订单附件 metadata。

### 4.4 代码与事务边界

- `internal/files` 只负责受控路径、单句柄读写、探测、rename/restore/purge 和故障注入，不解析 multipart、不执行 SQL；
- 附件状态、绑定、operation journal 和订单删除清理原语由 `internal/order` 用例与 `internal/store` 持久化入口共同持有；order create 使用单一事务入口，edit-remove 的 service 明确编排“准备事务 → 文件隔离 → 最终订单事务 → purge”，handler 不直接串联事务；
- 大文件完整 SHA-256 校验在取得 SQLite 写锁前完成；写事务内只做批量元数据验证、CAS、从 `os.Root` 重新打开后的平台文件身份/size/mtime 复核和 snapshot 保存，避免对多个 10 MiB 文件重复哈希；
- `internal/httpapi` 只负责 multipart/JSON、认证授权、短路和错误映射；`internal/app` 负责文件根目录、启动恢复和 DB + file readiness 组合。

## 5. HTTP 契约冻结范围

| Method | Path | 允许角色 | 成功结果 |
|---|---|---|---|
| `POST` | `/api/v1/attachments` | `operator`、`admin` | `201`，返回未绑定附件 ID、摘要和 `expiresAt` |
| `GET` | `/api/v1/attachments/{attachmentId}` | authenticated | `200`，下载已绑定文件 |
| `DELETE` | `/api/v1/attachments/{attachmentId}` | `operator`、`admin` | `204`，删除本人未绑定附件 |

### 5.1 上传短路与错误

固定短路顺序：已注册 route/method → CORS → authentication → role → multipart Content-Type/boundary → 总 body 上限与 30 秒 read deadline → 增量读取首个 part 并严格解析原始 Content-Disposition → attachment ID/可识别 temp 创建 → 流式大小限制、SHA-256 与内容探测 → 继续读取至 EOF 验证无额外 part → STAGING 提交 → final rename → UPLOADED 提交。part 结构校验与 temp 写入允许交织，任何后续结构错误都必须走可识别 temp 清理。

- 非 multipart 或错误媒体类型：`415 UNSUPPORTED_MEDIA_TYPE`；
- boundary、part 结构、文件名或 multipart 语法错误：`400 INVALID_REQUEST`；
- 文件或总 body 超限：`413 PAYLOAD_TOO_LARGE`；
- 类型不允许、内容与声明不一致或空文件：`422 VALIDATION_FAILED`，detail 指向 `file`；
- 文件系统临时失败：`503 SERVICE_UNAVAILABLE`；
- 失败后不得留下可下载记录、final 文件或无法由清理器识别的匿名 temp 文件。

成功响应字段顺序固定如下，且不得出现 `url`：

```json
{"id":"att_00000000000000000000000000000001","fileName":"invoice-示例.pdf","contentType":"application/pdf","sizeBytes":128,"sha256":"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef","createdAt":"2026-01-01T00:00:00Z","expiresAt":"2026-01-02T00:00:00Z"}
```

重复或非法附件 ID 的 error detail 指向 `attachmentIds[i]`（指向首次发生错误或后一个重复项），超过 10 个指向 `attachmentIds`，上传内容类错误统一指向 `file`；对他人/过期/其他订单/恢复中附件使用相同“不可用”消息，不暴露具体原因。

### 5.2 下载

下载按 attachment ID 读取元数据，要求 `BOUND` 且关联订单存在、可查看；`REMOVING` 在旧订单摘要中仍可见但下载固定返回 `503`。取得 8 槽 semaphore 后通过持久 `os.Root` 和服务端派生 basename 安全打开一次，把内容读入 10 MiB 有界缓冲并验证普通文件身份、size、SHA-256 和实际类型。attachment ID 语法非法、记录不存在或按不可见策略隐藏统一返回 `404`；一旦已有可见附件 metadata，final 缺失、不安全、签名/type/size/SHA-256 不一致均属于永久损坏并返回 JSON `500`；临时 open/read/锁、semaphore 或资源不足返回 JSON `503` 与 `Retry-After: 1`。只有完整性验证通过后才处理条件请求：`If-None-Match` 命中返回 `304`；否则忽略 `Range`、返回带 `Accept-Ranges: none` 的完整 `200` 与完整 bytes。逐状态 header 服从第 3 节第 16 项冻结矩阵。响应提交后的客户端取消或写失败只中止传输并写安全日志。

### 5.3 删除

删除短路顺序：route/method → CORS → authentication → role → attachment ID → owner/状态 → CAS 获取 operation ownership/写 PREPARED → 文件隔离 → COMMITTED。非法 attachment ID、不可见或不存在统一为 `404`；本人已绑定返回 `409 STATE_CONFLICT`；恢复中状态返回稳定 unavailable；重复删除在记录已不存在时返回 `404`，不伪装成幂等成功。只有 COMMITTED 后返回 `204`，后续 purge 失败由 tombstone 重试。

## 6. 订单绑定与幂等兼容

### 6.1 create

`POST /api/v1/orders` 新增可选 `attachmentIds`。service 先执行幂等 lookup/version dispatch；只有不存在记录时才完成事务外文件事实校验并进入单一写事务，语句顺序不得交换：

1. 规范化请求并按主体/method/route/key 查询现有幂等记录；命中 v1/v2 时比较对应 digest 并直接返回冻结 projection，不访问当前附件；
2. 不存在记录时，写事务外按输入顺序批量读取附件并用安全句柄验证完整文件 size/hash/type，禁止逐 ID 查询；此处只冻结文件事实，不以 owner/state/expiry 决定最终竞争结果。任一读取、hash/type、身份复核或 projection 构造失败时，返回错误前必须按同一 scope/key 再查 winner；late winner 已出现则按其 snapshot version/digest 重放或冲突，仍无 winner 才返回文件错误；
3. 预构造 request digest、order ID、时间和候选 response projection；写事务第一条写入完整 v2 snapshot reservation：`INSERT INTO idempotency_keys(principal_user_id, method, route, idempotency_key, request_digest, order_id, snapshot_version, snapshot_json, snapshot_digest, created_at) VALUES (?, ?, ?, ?, ?, ?, 2, ?, ?, ?) ON CONFLICT (principal_user_id, method, route, idempotency_key) DO NOTHING`。该行从创建时就是最终可重放记录，不允许 NULL/占位字段或 pending 状态；
4. `RowsAffected=0` 的 loser 立即读取 winner，按 winner snapshot version 比较并重放，不再验证当前附件。winner 尚未可见或 SQLite busy 时只在固定 5 秒预算内重试，预算耗尽返回 `503`；
5. 只有 reservation winner 才在同一事务内批量重验唯一性、数量、owner、状态、expiry、元数据和轻量文件身份，创建订单/items，把 `order_attachments` 按 `0..n-1` 写入并 CAS `UPLOADED -> BOUND`；
6. 订单、items、映射和附件状态变更与已写入的 v2 snapshot reservation 在同一事务提交；不得再执行第二次“保存 snapshot”。任一步失败整体回滚，reservation 不残留，附件保持原 `UPLOADED` 状态。

digest v2 canonical JSON 精确形状为 `{"operation":"POST /api/v1/orders","customerName":"...","currency":"CNY","items":[{"sku":"...","name":"...","quantity":1,"unitPrice":1}],"attachmentIds":[]}`，摘要为第 3 节第 9 项定义的 `normalizeCreateV2` 后 `encodeCreateDigestV2` UTF-8 bytes 的 SHA-256；`attachmentIds` 保留输入顺序，缺失、`null` 和空数组都编码为 `[]`。encoder 必须使用显式 struct 布局和 `json.Marshal`，不得使用 map、通用 canonicalizer 或依赖输入 JSON 字段顺序。固定 fixture 同时断言规范化后的 typed value、精确 encoded bytes 和 digest。同 key/v2 digest 重放返回首次 snapshot；相同 key 但 attachmentIds 顺序或其他事实不同返回 `409 IDEMPOTENCY_CONFLICT`。历史 v1 记录继续按旧算法重放且不读取当前附件，但同 key 请求含任何非空 `attachmentIds` 必须在 v1 digest 比较前返回冲突。

snapshot metadata 中 `snapshotVersion=2`；JSON 顶层仍为 `{"order":{...}}`，内部字段顺序固定为 `id/customerName/status/paymentStatus/currency/totalAmount/availableRefundAmount/version/createdAt/updatedAt/canEdit/canAdvance/canCancel/canRequestRefund/canApproveRefund/attachmentCount/items/attachments`。它保存首次 `201` 的完整 Order response projection；新订单固定为 `UNPAID`、`availableRefundAmount=0`、退款 capability 为 false，其他 capability 按首次主体保存。create service 返回显式 `CreateResult`，handler 不对重放结果重新调用当前主体 capability 计算。v2 重放只验证 scope/request digest/snapshot digest 与 JSON 内部不变量，不查询当前订单、退款、附件元数据或文件。v1 decoder 保持原结构并为新增附件字段固定映射 `attachmentCount=0/attachments=[]`；未知版本或损坏 snapshot 返回 internal。

### 6.2 edit

5A 期间，`PATCH /api/v1/orders/{orderId}` 只接受既有订单字段；请求省略 `attachmentIds` 时必须保留 `order_attachments` 全部映射与顺序，显式传入该字段由严格 JSON decoder 返回 `400 INVALID_REQUEST`。普通 PATCH、详情和履约 Action 的成功响应始终返回完整附件摘要，并以“create bind → 普通 PATCH 修改订单字段 → detail/download/restart”smoke 证明无损。

5B 启用附件编辑后，`attachmentIds` 缺失时保留当前集合，显式数组定义最终集合。edit 继续要求 DRAFT 和 order version，但文件隔离前的准备事务与最终订单事务之间不存在持续 SQLite writer fence，采用乐观最终校验：

- 当前订单附件按 position 读取并验证；
- 保留项必须已绑定到当前订单；新增项必须是当前主体拥有的有效 `UPLOADED`；
- 准备事务先验证目标 order version/refund aggregate，创建记录 `expected_order_version/expected_count` 的组，批量 `BOUND -> REMOVING` 并写入组内 PREPARED；取得全部 ownership 后才逐项隔离到 trash；部分 rename 失败必须整组 restore；
- `REMOVING` 窗口内，列表 `attachmentCount`、详情和 Action 响应仍按旧订单集合包含该摘要，下载返回 `503`；其他订单/退款写可先提交，另一个 edit 可按正常 version 竞争，但不得取得同一附件 ownership；
- 最终 order writer transaction 必须重新验证 order version、DRAFT、退款聚合、组完整性和全部 operation token，再更新订单/items；随后删除该订单旧映射并按最终集合重建 `order_attachments`，CAS 新增附件为 `BOUND`、移除附件为 `DELETING`，并将组与明细标为 `COMMITTED`。若 version/refund/state 已变化，原 edit 返回对应 `409/422/503`，并在返回前整组 restore，restore 失败升级为 `503` 且保留 PREPARED evidence；
- 最终事务成功后 purge trash；purge 失败不回滚已提交业务事实，但必须留下可重试、不可下载的 COMMITTED tombstone evidence；
- 与 refund create/approve、附件 delete/cleanup 的跨连接竞争必须得到稳定的 version、state 或 unavailable 分类，不能产生双重绑定、订单引用缺失文件或退款不变量破坏。

### 6.3 DTO

- 附件摘要字段顺序固定为 `id/fileName/contentType/sizeBytes/sha256/createdAt`；
- Order DTO 固定字段顺序为 `id/customerName/status/paymentStatus/currency/totalAmount/availableRefundAmount/version/createdAt/updatedAt/canEdit/canAdvance/canCancel/canRequestRefund/canApproveRefund/attachmentCount/items/attachments`；列表省略 `items/attachments`，但保留 `attachmentCount`；
- 订单列表通过当前页批量查询 `attachmentCount`；详情、首次 create、edit、履约 Action 返回按 position 稳定排序的 `attachments`；
- 历史 create snapshot v1 重放：`attachmentCount=0`、`attachments=[]`，不查询当前附件；
- 新 v2 snapshot 重放：返回首次 snapshot 中的附件摘要，不查询当前文件或当前订单附件状态。

## 7. 清理、seed 与运行边界

### 7.1 过期与残留清理

`cleanup-attachments` 使用注入 clock，分批处理：

- `UPLOADED` 且 `expiresAt <= now` 的附件；
- 可幂等完成的 `COMMITTED` operation；运行期不按 stale 时间接管其他 owner 的 `STAGING/REMOVING/PREPARED`；
- 可由 attachment ID 识别的 temp；首次发现只写 `attachment_cleanup_observations`，同一文件身份连续观察满 1 小时后才可清理；
- 名称/扩展名合法的 orphan final 候选；首次发现只写 observation，同一文件身份连续观察满 24 小时后才可进入 claim。扫描和裸 `mtime` 都不能直接授权删除；cleanup 必须在 SQLite 写事务中按 attachment ID 重新检查 `attachments` 和活动 operation 均不存在，再插入唯一 `ORPHAN_FINAL` PREPARED claim。上传的 STAGING 事务反向拒绝已有 claim；claim 提交后、文件隔离前再次验证 token/phase 与 observation identity，随后隔离并在后续执行 purge。身份变化、未来 observation 时间或时钟回拨必须重置隔离期；无法取得 claim或无法证明无引用时绝不删除。

默认批次为 100；取消时停止领取新 batch，已取得 operation 按其 phase 留下可恢复状态，命令输出 `scanned/prepared/skipped/committed/purged/failed`。重复执行必须收敛且不得把同一附件重复计为业务删除。清理不得删除 `BOUND` 文件，不得跟随 symlink/reparse point，不得删除附件根目录之外的目标。并发 bind/delete/cleanup 至多一个操作取得 CAS；loser 重读后返回稳定结果。

### 7.2 seed/reset

附件 demo 不新增环境变量或 CLI 开关：入口固定为现有 `cmd/admin seed` 的 development path，在 `APP_ENV=development` 且 `LoadDemoSeed` 成功后按 `runtime -> auth_demo -> order_demo -> refund_demo -> attachment_demo` 执行；`attachment_demo` 是 `seed_versions.name`，版本固定为 `1`。production `cmd/admin seed` 只执行 runtime seed，不调用或写入任何 `*_demo` group；若代码试图在 production 执行 attachment demo，必须在打开附件 root 或写文件前拒绝。attachment demo 生成小型、确定性的合法 PDF/PNG/JPEG fixture，至少覆盖一个已绑定附件和一个未绑定附件。首次 seed 必须复用上传的 `temp -> STAGING -> final -> UPLOADED` durable 协议，再通过正常绑定事务产生 `order_attachments/BOUND`，不得直接写 final + metadata；真实子进程测试覆盖 seed 的每个 commit/rename 边界。seed 重放只读验证 `attachment_demo=1`、DB 元数据、映射、文件大小和 SHA-256；发现文件缺失或内容漂移时失败，不静默重建掩盖损坏。seed/reset 均要求 API 停止并取得 `API exclusive -> maintenance exclusive`；reset 删除受控附件目录前还要通过现有 reparse point/path containment 防线扩展验证。

### 7.3 readiness 与启动

API 启动在打开 DB/root 前取得 API lock 和独占 maintenance lock，随后执行 schema gate、创建并验证受控目录、恢复本 binary 支持的全部遗留 `STAGING/PREPARED/COMMITTED`，再按第 30 项的批次与 30 秒预算扫描稳定 `UPLOADED/BOUND` 的映射、final 存在性、普通文件身份和记录 size；启动路径不得全量 hash 或重新探测类型。完成后才允许构造业务 service/handler 和注册路由。`newAPI`、`Handler()` 测试装配和嵌入式入口都必须经过同一门禁，不得把恢复只放在 `Run`。无法安全恢复、遇到当前 5A binary 不支持的 group/kind、仍有 unresolved operation、稳定附件缺失/不安全 final 或扫描超预算时拒绝启动。对外服务前释放 maintenance lock。`/readyz` 加入轻量文件存储 probe，验证根目录存在、可创建并清理零长度 probe 文件且无路径逃逸；运行中 unresolved operation 超过 5 分钟时，按第 29 项由独立安全停服 watchdog 在最多 5 秒内标记 not-ready、触发一次性优雅退出并由进程监督器重启，禁止在线 PREPARED 接管。5A 必须用无 probe 流量的真实子进程测试证明退出码 75、15 秒宽限、重复触发幂等和重启后启动恢复。probe 不读取业务附件、不留下临时文件、不把本地路径写入响应。

`admin verify-attachments --full` 是独立的离线只读入口：API 必须停止，命令按顺序取得 API 与 maintenance 双独占锁，使用 v7 feature capability 打开 DB/root 后直接执行完整性扫描，不运行 migration、journal recovery、cleanup、seed 或业务 service。扫描按 500 行批次、默认 2 worker 使用统一 integrity admission pool，输出机器可解析且不含本地路径的摘要；任何修复只能按恢复手册另行执行，verify 本身不得改写 DB 或文件。

## 8. 实施里程碑

| 里程碑 | 主要交付 | 开发 effort（人日） | 环境等待（日历工作日） | review/修复缓冲（人日） |
|---|---|---:|---:|---:|
| P0 契约与原型 | 八工作包、fresh/v6/v7/5B 矩阵规格、`os.Root`/deadline/文件身份可抛弃 spike、运行契约、锁/进程/规模 fixture 和 Evidence 工具链；不含 release binary 或真实规模结论 | 13–20 | 1–3 | 3–5 |
| M1A 5A 存储与恢复 | migration v7、`order_attachments`、`internal/files`、STAGING/单项 delete/expiry journal、构造期 schema gate/启动恢复、运行期 watchdog | 6–9 | 1–2 | 1–2 |
| M2 HTTP 文件闭环 | upload/download/delete、JWT/CORS、deadline、统一 integrity admission/header 矩阵 | 3–5 | 0–1 | 1 |
| M3A create 与幂等 v2 | create 绑定、`CreateResult`、DTO、snapshot v1/v2、并发 winner 重放 | 4–6 | 0–1 | 1–2 |
| 5A-I 实现门禁 | 上传、未绑定删除、create 绑定、普通 PATCH 无损、鉴权下载、v2 幂等、基础启动恢复与 watchdog 的本机 smoke/crash | 3–5 | 1–2 | 1–2 |
| 5A-D 部署门禁（条件） | v6/schema-only/5A 三产物矩阵、cleanup 调度、进程监督、成套回退、ENOSPC、Windows/Linux 证据与发布说明 | 2–4 | 2–5 | 1–2 |
| M1B 组 journal 与恢复 | REMOVING、group journal、edit-remove/ORDER_DELETE、orphan claim 与完整组恢复 | 4–6 | 1–2 | 1–2 |
| M3B edit 与删除原语 | edit 组移除、窗口语义、并发不变量、受退款历史限制的订单删除 cleanup 原语 | 5–8 | 1–2 | 1–2 |
| M4 运维闭环 | 完整 cleanup/orphan、离线 verify、seed/reset、readiness 扩展和完整 crash smoke | 4–6 | 1–3 | 1–2 |
| 5B 最终集成 | 新目录 app smoke、最终受控环境 deployment approval、回退与平台证据 | 2–4 | 2–4 | 1–2 |
| R 文档发布 | 事实源迁移、恢复手册、全仓/文档/基线 diff Evidence | 1–3 | 1–2 | 1 |

默认按 5A/5B 两批交付，不把拆分作为压缩工期时的可选项。表中开发列统一为 effort 人日，不是 elapsed 排期；环境等待是日历工作日，review/修复为独立人日。单人完整开发范围暂按 47–76 个开发人日管理，不能与等待或 review 合并后宣称为同一日历周期。P0 的 13–20 人日来自八工作包上限汇总；只有至少 3 名实现 owner 加 1 名独立 reviewer/批准人、Windows 与 Linux 环境 owner 已落实且允许并行时，才可把 P0 elapsed 估为 6–10 个工作日。单实现者加独立 reviewer 时，P0 elapsed 下限不得低于 16 个工作日，并需另加环境等待。P0 dependency DAG 的唯一事实源是第 3 节 tracked work-package 表：WP-Facts 先于依赖它的 Schema-Recovery、HTTP-Order、Lock 与 Baseline-Evidence；Files/Runtime 依赖 Lock；Release 等待表中列出的其余七包批准输出。排期工具和 P0-21/requirements validator 只能从该表导出关键路径，不得另写一套固定路径。若 Windows 需要专用打开/rename 层，P0 和 M1A 都必须单独增加缓冲。P0 八个工作包结束后，必须用实测结果替换上表原 ROM，而不是只追加说明；重估覆盖 M1A、M2、M3A、5A-D、M1B、M3B、M4、5B 和 R，并记录 owner、review 带宽、环境可用性、关键路径和资源不足时的降级顺序。重估完成前不得承诺交付日期。降级顺序固定为先取消持续部署批准，再推迟 5B edit/order-delete/orphan，最后才讨论缩小支持规模，恢复、安全、race 和 Evidence 门禁不得降级。并行时由单一 owner 持有 file/journal/recovery 协议，另一 owner 持有 HTTP/order integration；事务协议变更仍须串行 review。5A 只实现 STAGING、单项 delete/expiry、create bind 和对应恢复；5A binary 对 group/REMOVING/ORDER_DELETE/ORPHAN_FINAL 等不支持状态在构造 handler 前 fail-closed。5B 再交付 group journal、edit 组移除、完整 cleanup/orphan、受 DRAFT/version/退款历史和幂等保留约束的订单删除 cleanup 原语、独立 `v7-5b-feature` capability 和完整 crash matrix。

## 9. 验收门禁

### 9.1 5A-I 实现 exit gate

进入 M1B/M3B/M4 开发前，5A feature binary 必须在同一候选 revision 独立满足：

- P0 阻塞原型和 M1A/M2/M3A checklist 全部完成；事实源同步必须先于 schema/API 编码；
- 5A 目标包非缓存 test/race、全仓 test/vet 和文档 validator 通过；
- 真实 app smoke 覆盖上传、未绑定删除、create 绑定、普通 PATCH 保留附件与顺序、鉴权下载、v2 重放与重启；
- STAGING/rename/UPLOADED、下载校验、create bind 的进程退出与恢复矩阵通过；
- v6 gate、schema-only 与 5A feature 的构造期 gate、启动限制和本机 migrate/启动锁测试通过；
- 最小 `cleanup-attachments` 已能处理过期 UPLOADED、5A 单项 COMMITTED 和旧 temp；
- 运行中 unresolved operation 超过 5 分钟会使 `/readyz` not-ready，并由一次性 callback 在无 probe 流量时完成 15 秒宽限和退出码 75；真实子进程重启后由启动恢复收敛；
- 发布说明明确 PATCH 暂不接受 `attachmentIds`、group/edit/orphan/order-delete 能力尚未启用；若 5A 不单独上线，也必须以内部候选产物完成同等 exit gate。

通过 5A-I 后可以继续 5B 代码开发，但任何持续可访问部署还必须通过 5A-D。

### 9.2 5A-D 部署 approval gate

5A 持续部署批准必须额外满足：

- 三类产物的完整 revision/build/file/SHA-256/scan/退出码/部署顺序矩阵和成套备份回退演练已记录；
- 受支持的进程监督器确认退出码 75 后重启，cleanup 调度固定每 15 分钟、超时 10 分钟、禁止重叠且失败告警，均有真实环境证据；
- `ENOSPC` 无残留演练、Windows 打开句柄 rename/reparse/applock 证据和 Linux CI test/race 有记录；
- 发布说明、产物 manifest、CI 产物和部署 README 统一标注 `internal/demo-only`，并明确 5A 只适用于单进程、受控 demo/内部环境，不构成生产或多副本批准，且 ENOSPC 收敛不等价于容量保护。

缺任一项时，5A-I 产物只能用于短期本机/隔离测试，不得部署到持续可访问环境。

若项目没有把 5A 产物部署到持续可访问环境，则 5A-D 是条件 N/A：必须记录批准人、未部署原因和替代 gate，不能伪造监督器或调度证据。此时阶段完成由最终 5B deployment approval 覆盖同等级的监督器退出码 75 重启、cleanup 调度、ENOSPC、平台文件语义、成套回退和 Evidence 留存；5A-I 始终不可 N/A。

### 9.3 5B 与阶段完成 gate

阶段完成必须同时满足：

- [配套 checklist](./PLN-0005-phase-05-attachment-lifecycle-checklist.md) 的所有适用项完成并记录实际 Evidence；未持续部署 5A 时，5A-D 必须按批准的 N/A 流程关闭，并由最终 5B deployment approval 取代；
- `go test -tags phase5_test ./... -count=1`、`go vet -tags phase5_test ./...`、独立 `go test -race -tags phase5_test ./... -count=1` 通过；四个发布 tag 的独立 test/vet/build 矩阵同时通过；
- 冻结阶段五审计 baseline/merge base，记录 baseline 与 HEAD，并同时通过 `git diff --check <baseline>...HEAD` 和工作树 `git diff --check`；`docs/tools/validate.ps1`、`docs/tools/validate.tests.ps1` 同时通过；
- 新数据目录按入口矩阵完成 v6 bootstrap `migrate` → v7 schema-only `migrate` → seed → 5B API 启动 → 登录 → 上传 → 创建绑定 → 下载 → 编辑替换/移除 → 删除未绑定 → cleanup → 重启 smoke；5B 集成前置必须同时满足 5A-I、M1B、M3B、M4，不能以 M3B 单元测试绕过 M1B group journal/REMOVING/ORPHAN_FINAL/ORDER_DELETE 的真实 crash/recovery Evidence；
- 上传失败、DB 提交失败、rename/remove/open/hash 失败以及进程在每个 rename/commit 边界退出后，均无可访问孤儿、BOUND 缺失 final 或路径泄露；
- 5A-I、按实际部署路径适用的 5A-D 或其批准 N/A、最终 5B deployment approval、schema-only 限制、回退恢复、远端 Linux CI、最终 revision 和剩余风险有记录；5B approval 仍只批准单机受控 `internal/demo-only` profile，不构成正式生产、多副本或互联网不受控上传批准；
- 当前 API、目标 API、领域模型、验证矩阵、路线图、README、场景文档和 CHANGELOG 与实现一致；
- 交付 fail-closed 恢复手册，覆盖只读诊断、DB/WAL/SHM/附件目录成套恢复、允许隔离/删除的对象、禁止直接改写的状态以及修复后完整性扫描；
- 只有用户明确确认后才把 plan/checklist 原地改为 `status: archived`，保留稳定路径并更新计划索引；不得移动文件而破坏既有 AUD/REM/IMP 链接。

## 10. 规划基线

- 规划前代码基线为 `main@e28e9ac`，当时已跟踪代码工作树干净；该 revision 不包含本阶段 plan/checklist，不能作为 P0 文档基线。
- Go 版本：`go 1.26`。
- 2026-07-13 实跑 `go test ./... -count=1`：通过，wall 16.6s。
- 2026-07-13 实跑 `go vet ./...`：通过，wall 2.0s。
- P0-14 必须在 plan/checklist 进入版本库后的准确 revision 上重新记录 `git status --short`、Go 版本、全仓 test/vet/race、文档 validator 与 diff check；若工作树不干净，必须列出与基线 revision 的关系，不能把 `e28e9ac` 与尚未进入该 revision 的计划文本混为同一 Evidence。
- 本次规划未把任何第三方审计报告中声称的 race、文档 validator 或 diff 结果视为本计划 Evidence。真实上传 smoke、磁盘失败注入和进程退出恢复仍属于 M1–R 实施门禁，不得写成已通过。

## 11. 主要风险

1. SQLite 与文件系统没有共同事务；必须依赖 STAGING、durable intent/tombstone、temp/final/trash 编排和 CAS，不能假设 rename 与 DB commit 原子或只依赖进程内补偿。
2. order create 幂等 v1/v2 兼容容易误改历史 digest 或把当前附件状态带入重放；必须用固定历史 fixture 回归。
3. Windows reparse point、跨卷 rename 和文件占用语义与 Linux 不同；本地测试和远端 Linux CI 都必须覆盖。
4. multipart 解析器可能在超限、重复 part 或客户端中断时留下临时文件；所有退出路径都要验证清理。
5. edit、delete、cleanup 与订单/refund 写事务竞争可能形成双重绑定或逻辑成功但文件缺失；跨连接并发测试必须覆盖 winner/loser 重试分类。
6. 当前仅做格式与内容头检查，不等价于恶意文件检测；生产化前仍需病毒扫描、配额、对象存储和审计策略。
7. 启动结构扫描虽不再全量 hash，仍可能因损坏或超出 30 秒预算 fail-closed；部署前必须用目标数据规模演练，并保留离线完整校验与成套恢复路径。
8. 三类迁移产物拿错 revision 或跳过备份会破坏 forward-only 边界；部署只认 manifest 中的 revision、SHA-256 和发布矩阵，不认文件名猜测。

## 12. 2026-07-14 自审计结论

本节只记录可由当前 plan/checklist 自身复核的修订结论，不引用或依赖根目录临时报告；该修订进入版本库后的准确 revision、命令输出和 validator 结果仍由 P0-14 记录，不能把本次文档编辑写成阶段五功能 Evidence。自审计结论继续为“需要修改，完成全部 P0 Evidence 后方可进入 M1A”。

本轮冻结的新增契约为：v6 gate/bootstrap、v7 schema-only、v7 5A feature、v7 5B feature 四类发布产物与 `phase5_test` 质量 capability 必须由互斥编译输入产生真实差异，并把 artifact/revision/binary/manifest/run 绑定；P0 只冻结规格与可抛弃 spike，真实产物分别由后续门禁交付；CI 使用确定的 tag 矩阵；ORDER_DELETE 仅允许受信任 maintenance 编排删除 expected-version 匹配的 DRAFT，保留退款/退款幂等历史和 order-create snapshot，删除后同 key 只重放首次响应；temp/orphan 以持久 observation 而非裸 `mtime` 计算隔离期；create/download/cleanup/verify 共享统一 integrity admission；`verify-attachments --full` 成为停服、双独占锁下的只读交付；`os.Root` 只作为 containment 基线并由平台适配接口补全安全语义；P0 ROM 明确区分 effort、elapsed、环境等待和独立 reviewer；tracked work-package 表、`P5-CRITICAL` 与 P0-25 原子验收矩阵分别成为依赖、协议变更与验收事实源；5B 仍只批准单机受控 `internal/demo-only` profile。既有独立 `order_attachments`、durable journal、v1/v2 snapshot、原始 multipart header、stable applock 和 fail-closed 恢复主线继续保留。

未把任何工具声称已经通过的 test、race、validator 或 diff 结果作为本计划 Evidence，也没有把 Windows 与 Linux 同时宣称为支持部署环境；只有完成监督、调度、恢复和 ENOSPC 证据的 profile 才能进入部署批准。对 5A 磁盘风险继续采用“前移最小 cleanup + 无部署证据时禁止持续部署”，不在本阶段引入完整配额系统。5A-I 通过后可继续 5B 开发；若实际持续部署 5A，则 5A-D 必须先通过，否则以批准的 N/A 关闭并由最终 5B deployment approval 替代。P0-1、fresh/legacy migration 执行链、journal 外键语义、startup/exit 协议、启动规模、Evidence validator/留存、完整锁矩阵和平台 harness 完成前保持 No-Go。
