# V3 安全回归测试

对应 [issue #97](https://github.com/muxi-Infra/FeedBack-Backend/issues/97)。

## 执行

```sh
go test ./service ./repository/cache -run TestV3 -count=1
go test ./...
go test -race ./service ./repository/cache -run TestV3 -count=1
```

测试使用仓库已有的 testify、gomock、GORM SQLite 驱动，以及 miniredis v2。
无需业务配置、真实账号、MySQL/Redis 服务、飞书服务或真实密钥。
race 检查需要平台支持及可用的 C 编译器。

`.github/workflows/dev.yml` 和 `prod.yml` 已执行 `go test -v ./...`，上述测试没有额外 build tag，随现有 CI 执行。
`.gitignore` 不再忽略 Go 测试文件。

## 边界与用例

| 边界 | 负责层 | 自动测试 |
| --- | --- | --- |
| A 不能查询 B 的反馈列表 | JWT Claims → 控制器 → SheetService → DAO 的表、学生条件 | `TestV3FeedbackListIdentityIsolation`：交错插入两个学生、两个项目的数据；检查初始页、后续页和伪造身份查询参数；每个身份均能读到自己的两条记录 |
| A 不能通过 record_id 读取 B 的详情 | DAO 同时约束 table_identify、user_id、record_id | `TestV3FeedbackRecordIdentityIsolation`：跨学生、跨项目返回 404；同一学生和同名 record_id 在不同项目中返回各自数据；缺失记录也返回 404 |
| 图片记录归属及每个 file_token 归属 | 同一 DAO 归属查询 + 控制器 `photoTokensBelongToRecord` | `TestV3FeedbackPhotoOwnership`：他人/其他项目记录、同一用户的另一条记录、混合合法与非法 token、空/缺失 token；合法图片成功；被拒请求不得调用飞书 |
| 项目 Token 不能切换到另一项目的数据 | JWT 项目身份 → IntegrationDAO 项目表查询 → 项目缓存 → Sheet/FAQ DAO | 上述列表、详情、图片测试及 `TestV3FAQScopeAndProjectIsolation`：两个项目交替访问，覆盖冷缓存和缓存命中；伪造 project_id/table_identity 无法改变实际访问身份 |
| FAQ 必须具有 feedback:read | IntegrationDAO 查询表的 Scope + 控制器精确匹配 | `TestV3FAQScopeAndProjectIsolation`：无 Scope、read:self、write 均拒绝；其他表/项目的 read 不授权当前 FAQ；read 正常读取 |
| 无效签名、过期时间戳不能兑换 | 请求绑定、AuthService 时间窗、API Key 查询及 HMAC 校验 | `TestV3TokenExchangeValidation`：错误密钥、损坏签名、篡改学生身份、项目与 Key 不匹配、过期/过远未来时间戳、必填参数缺失；拒绝不占用 nonce，随后合法请求可以兑换并读取反馈 |
| 重复 nonce 不能兑换 | AuthService 项目 nonce 命名空间 + Redis SET NX | `TestV3TokenExchangeConcurrentReplay`：16 个并发请求恰好一次成功；同项目换学生仍拒绝；不同项目及新 nonce 成功 |
| nonce 覆盖完整请求有效期 | AuthService 计算 TTL + Redis 过期 | `TestV3NonceCoversEntireTimestampWindow`：可控时间下验证过去边界、当前时间、未来窗口内及未来边界；最后有效秒仍拒绝重放；下一秒按时间戳过期拒绝 |
| Redis 原子性及错误时拒绝兑换 | 生产 go-redis 客户端和 nonce store | `TestV3NonceStoreAtomicityAndExpiration`：两个客户端、32 路并发恰好一次成功，重复写不延长 TTL，过期可重用；`TestV3NonceStoreRejectsInvalidInput`；`TestV3TokenExchangeFailsClosedOnNonceStoreError` |
| API Key 轮换 | 管理员 JWT + Casbin → 控制器 → AdminService 事务 → DAO 的 active Key 条件 | `TestV3APIKeyRotation`：使用全新 nonce 验证旧 Key 失效，新 Key 正常；旧密钥配新 key_id 也失败；其他项目不受影响；轮换前签发的反馈 Token 继续可用 |
| Token 缺少项目/学生身份时拒绝 | V3JWT.Parse + V3AuthMiddleware | `TestV3ProtectedRoutesRejectInvalidIdentity`：自动枚举所有已注册的受保护用户路由；缺失/空身份、无 Token、损坏 Token、错误签名/算法、过期 Token 均为 401；完整身份读取成功 |
| 管理路由拒绝未认证请求 | AdminAuthMiddlewareV3 | `TestV3AdminProtectedRoutesRequireAuthentication`：自动枚举管理路由，无 Token、损坏 Token、反馈 Token 均拒绝；轮换用例覆盖管理员正常访问 |

## 测试实现范围

主测试位于 `service/v3_security_test.go`，使用真实注册函数、请求绑定、JWT、控制器、鉴权服务、数据服务与 DAO。
每个 fixture 使用独立 SQLite 内存数据库、单连接和独立 miniredis 实例，测试结束时关闭连接。
SQLite 执行生产查询条件和轮换事务，而非仅匹配 SQL 字符串或让 DAO mock 返回禁止访问。
本组测试不承担 MySQL 特有迁移、排序规则和方言兼容性验证。

nonce store 使用真实 go-redis 客户端连接 miniredis，覆盖 Redis 协议下的原子 SET NX 和 TTL 语义；不覆盖 Redis 集群或故障转移。
时间窗测试通过包内 `exchangeAt` 入口与 miniredis `FastForward` 同步推进两个时钟，不使用 sleep。

飞书调用使用现有 gomock Client；被拒绝的图片请求没有调用期望，因此一旦触发外部请求测试就失败。
项目配置事件总线使用立即返回的 fake，本组测试不验证分布式配置刷新。
`v3_security_export_test.go` 只在测试构建中提供初始化入口，数据读取仍走生产方法，同时避免启动与本组读测试无关、长期消费全局队列的同步工作协程。

## 修复的重放缺陷

原实现将 nonce 固定保存一个时间偏差窗口（默认 300 秒）。
未来时间戳的请求首次通过后，可能继续有效至多约 600 秒，因此 nonce 过期后仍能重复兑换。
即使请求时间戳等于当前时间，第 300 秒也仍在包含端点的有效窗口内。

先运行 `TestV3NonceCoversEntireTimestampWindow` 确认失败，再将 nonce TTL 改为请求剩余有效秒数加一，覆盖完整的最后有效秒。
修复不改变公开请求/响应、时间戳允许范围或已签发反馈 Token 的有效性。
