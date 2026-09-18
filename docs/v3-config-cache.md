# V3 项目配置缓存

## 提交与生效

管理写接口返回 HTTP 200、`code=0` 表示项目/表格/Scope/Key、配置版本、成功审计和 Outbox 已在同一 MySQL 事务提交。缓存传播是异步的，响应不表示所有实例已经应用。

默认事件传播观测目标为 5 秒；该目标以数据库和 Redis 可用、无积压为前提，不是硬实时承诺。每个实例启动即对账，此后每 30 秒全量对账，单轮预算 10 秒。正常调度、MySQL 可用且对账在预算内完成时，包括仅 Redis 故障的情况，通常最迟约 40 秒发现并应用更新或删除。重叠对账串行运行；排队、超预算、进程暂停和过载会延长该时间，必须结合完整对账时间指标判断。

安全边界独立于事件传播：从最近一次成功数据库确认的**查询开始时间**起最多 60 秒，缓存才能继续用于授权。收到更高版本后旧版本立即不可用于新的配置获取；加载失败不会恢复旧版。超过 60 秒且数据库无法确认时返回 HTTP 503（`V3ConfigUnavailableCode`），不能以慢查询结束时间延长旧快照寿命。

Scope 收紧、表格移除、项目删除遵守上述窗口。窗口内可能继续接受使用旧 Scope 的请求；已经完成授权的在途请求和已经发出的飞书调用不追溯撤销。兑换直接查数据库；Key 轮换阻止旧 Key 的新兑换，已签发 JWT 保持有效，项目删除仍受配置撤销窗口约束。

Redis 故障不再把数据库已提交的管理写入改成 500。但现有业务限流、管理员写入限流和 nonce 校验仍依赖 Redis，可能在事务开始前拒绝请求；本功能不承诺 Redis 故障时全部 API 可用。健康检查、受 BasicAuth 保护的 metrics、管理员配置状态查询绕过 Redis 限流，原有认证权限继续执行。

## 数据与恢复协议

- `feedback_projects.config_version`：注册从 1 开始，项目/表格/Scope 更新、删除、Key 轮换推进版本。同项目事务先锁项目行；不存在的项目不可更新。删除保留软删除墓碑和版本，不复用 ID。
- `feedback_project_config_audits`：操作者来自已验证管理员，记录请求关联 ID、项目、操作类别、字段类别、前后版本及提交结果；与配置一起提交。审计入库失败导致整个事务回滚。
- `feedback_project_config_outbox`：同事务保存事件，实例每秒用条件更新领取租约（默认 15 秒），事务外 XADD，再标记完成。不需要 `SKIP LOCKED` 或全局发布单例。租约使用数据库记录与实例时钟，部署应保持时钟同步；即使租约重叠，重复事件仍安全。
- 发布失败按 200 毫秒到 30 秒指数退避并加抖动重试；进程崩溃后租约到期重领。发布成功但完成标记丢失会再次发布，消费端按版本吸收。失去租约的发布者不能标记另一领取者的记录完成。
- 未发布记录不因重试次数删除。已发布记录默认保留 7 天，每次维护有界清理；审计不自动删除，需要另行制定存储归档策略。

PUT 同内容具有相同最终效果，仍推进版本并增加审计；并发编辑是最后提交生效。DELETE 重试保留已删除状态，不复活项目。注册、轮换不是幂等操作：网络断开导致提交结果未知时，用预先保存的请求 ID 查询审计，不要自动重试。系统不持久化一次性 Key 明文；确认提交但响应丢失时，由管理员明确再次轮换。

更新、删除和 Key 轮换的目标项目不存在时返回 HTTP 404（`V3ProjectNotFoundCode`），不归类为数据库故障。已有删除墓碑的 DELETE 重试仍返回成功；对已删除或停用项目的更新和 Key 轮换返回相同的 404。数据库故障仍返回 500。

## 各实例缓存与消费者

进程身份为 hostname、PID 和启动 UUID。每个实例使用独立消费组 `feedback-v3-config-refresh-v2:<instance_id>`，每次新进程以 `$` 建组并重新对账；Redis 启动不可用时持续退避重试，数据库对账独立运行，Redis 恢复建组后再次对账。不能把所有实例改成共享消费组。

建组后先尝试一次全量对账；某项目加载失败不会阻止随后消费其他项目的事件。失败对账由独立定时任务继续重试，`consumer_up=1` 只表示消费连接可用，完整对账是否成功仍须检查 `last_full_success_timestamp_seconds`。

同进程消费循环先恢复自身 PEL 的一页，再读取一页 `>`，批量 20，处理并发 4。失败项目保留 PEL，ACK 失败允许重复处理；重复、乱序事件不能降低缓存版本。建组/读/处理失败均有可取消退避，`NOGROUP` 重新建组并对账。ACK 的含义是完整项目快照已应用到至少事件版本，或确认墓碑；接收消息和删除缓存都不算成功刷新。

每轮 PEL 扫描开始时通过 XPENDING 固定当前最大消息 ID，分页游标到达该终点后回卷。新增失败消息不会无限延长当前扫描，旧消息可以再次重试；重试间隔仍随本轮页数、加载耗时和退避增长，不承诺固定秒数。

Stream 保留原 `project_id`、`changed_at` 字段，新增 `config_version`、`change_id`、`kind`、`schema_version=1`。旧事件强制数据库核对。无法解析或 PEL 正文被裁剪时，成功全量对账后 ACK 丢弃，不记录原载荷。XADD 默认 `MAXLEN ~ 10000`，保留条数近似，不能换算固定时间。积压达到 1000 或检测到消费位置早于保留首条时触发对账，消费继续有界推进。

每 10 秒续租，租约 120 秒；每分钟用 Lua 原子检查租约并清理登记的失效组。只清理新前缀且属于本 Stream 登记集合的组；续租与删除不会在单次脚本内交错。进程重新启动使用新缓存、新组，旧组到期清理。旧版本遗留组不自动处理。

缓存是项目级不可变快照（项目、所有活动表格及 Scope），运行时 DAO 在一个 MySQL REPEATABLE READ 事务里读取，不读取 Key 摘要。同项目并发加载合并，总数据库加载并发 4。加载前记录 generation，写入时比较 generation、最低要求版本及当前版本；失效之前开始的旧结果不能写回或直接返回。对调用方复制 Scope 切片。

若加载期间 generation 未改变，但数据库快照仍低于要求版本（包括异常硬删除后返回版本 0），本次刷新立即以 `version_behind` 失败，不在加载期限内循环查库，也不会降低缓存版本。外层消费退避和定时对账负责后续重试。仅真正被并发失效取代的加载计为 `superseded`；刷新调用以超时或取消退出时也记录 `failed`。

全量对账分页枚举包含冷项目和墓碑的版本；版本相同才续期，改变则完整加载并逐项目替换。只有扫描完整成功后才核对本地多余项目；单页失败不误删，某项目失败不清空其他缓存。已知旧版拒用，尚未发现变化的缓存只用到 60 秒。所有实例各自对账，没有全局调度锁。直接 SQL 修改配置不在保证范围：维护写入必须同时遵守版本、审计与 Outbox 事务协议。

## 生命周期和配置

构造函数不启动 V3 goroutine。App 显式启动配置运行时，收到退出信号后先停止接受 HTTP 请求并等待在途请求排空，期间配置缓存和加载仍可用；之后取消消费者、发布器、对账和维护任务，停止 timer/ticker，取消查询、关闭专用阻塞读取连接并等待退出。HTTP 优雅关闭预算 10 秒，V3 任务关闭预算 5 秒。V1/V2 飞书刷新和已有业务队列的生命周期没有在此重构。

完整可选配置见 [`config/example_config.yaml`](../config/example_config.yaml) 的 `v3_config_cache`。缺省使用默认值；显式零值不能关闭兜底或授权期限，时间与容量组合非法时启动失败。配置沿用 YAML/Nacos 启动加载方式，变更这些参数需要重启实例。

## 管理接口

现有写响应 JSON 和一次性 Key 返回字段保留；增加响应头：

| 响应头 | 含义 |
| --- | --- |
| `X-Request-ID` | 调用方可传 UUID；缺失/非法则服务生成。网络重试排查应由调用方预先生成并保存 |
| `X-Config-Version` | 此次成功提交后的配置版本 |
| `X-Config-Change-ID` | 对应审计/Outbox 的变更 ID；重复删除无新变更时为空 |
| `X-Config-Propagation` | `asynchronous` |

CORS 允许 `X-Request-ID` 并暴露以上响应头。项目详情和列表的项目对象新增 `config_version`。HTTP 503 新错误码追加在现有枚举尾部，Scope 拒绝仍为 403。

`GET /api/v3/admin/integrations/projects/:project_id/config-status` 使用管理员 JWT 和 `integration/read` 权限，返回响应实例的 `instance_id`、`project_id`、`target_version`（未知为 null）、`required_version`、`applied_version`、`state`、`last_event_id`、`confirmed_at`、`loaded_at`、`invalidated_at` 和 `error_class`。状态为 `not_loaded`、`behind`、`applied`、`deleted`、`stale` 或 `target_unknown`。查询不会隐式刷新缓存，也不是全实例聚合；经负载均衡查询时必须核对 instance_id，定位指定实例应直连该实例。

`GET /api/v3/admin/integrations/config-audits` 使用相同权限。可选 `project_id`、`request_id`、`before_id`、`limit`（默认 20，1–100），按 ID 倒序返回审计数组，以最后一条 ID 翻页；删除项目仍可查。审计字段为 ID、change_id、project_id、admin_id、request_id、kind、previous_version、version、fields、result、created_at。

排查顺序：按请求 ID 找审计确认事务是否提交 → 用 change_id 找 Outbox 的发布状态/message_id → 直连各实例查目标与应用版本 → 按项目和消息 ID 查日志。`config_change_committed`、`event_received`、`cache_invalidated`、`config_applied`、`refresh_failed`、`event_acked` 是不同阶段；收到事件不能作为配置已生效的证据。

日志、事件、审计不包含 API Key、摘要、JWT、飞书 Token、请求体或完整配置。GORM 使用参数化 SQL 日志。校验拒绝、鉴权拒绝和事务失败写结构化失败日志；数据库故障时不承诺失败审计仍能入库。

Outbox 领取、发布、重试状态写入、完成标记和清理失败分别记录阶段日志；每个实例每个阶段最多每分钟记录一次，发布计数器仍逐次累计。日志包含实例及安全错误分类，单条事件操作还包含项目、变更 ID 和尝试次数。批量领取后续记录失败时，已成功领取的记录仍继续发布。MySQL 启动失败保留初始化/迁移阶段和数据库错误码或网络错误类别，不输出驱动原始消息或连接串。

## 指标和告警

使用已有 Prometheus registry/metrics 端点和认证。所有下列名字都有 `feedback_v3_config_` 前缀；项目/请求/事件 ID 不作为标签，实例区分使用 Prometheus scrape 的 `instance`。

| 指标后缀 | 说明 |
| --- | --- |
| `refresh_total{trigger,result}` | `applied/unchanged/failed/superseded`；trigger 为 request/event/local/reconcile；重复事件不能虚增 applied |
| `reconcile_total{result}` | 完整对账轮次 success/failed，与单项目刷新分别计数 |
| `refresh_duration_seconds{trigger}` | 单项目数据库加载耗时直方图 |
| `apply_age_seconds` | 数据库变更时间到应用的年龄，包含事务时间、时钟偏差，冷启动也会观测 |
| `invalidation_delay_seconds` | 发现失效到成功应用的耗时 |
| `cache_requests_total{result}` | hit/miss/expired/outdated，请求级计数；有效快照确认项目或表格不存在也属于 hit，不等同于授权通过率 |
| `events_total{stage}` | received/invalid/retry/read_failed/ack_failed/acked |
| `publish_total{result}` | published/failed/claim_failed/finish_failed/retry_failed（重试状态写入失败） |
| `consumer_up` | 消费者最近连接/读取状态 |
| `last_full_success_timestamp_seconds` | 最近完整对账成功时间，初始为 0 |
| `stream_pending`、`stream_unread` | 未确认与未投递数量，不能用 Stream 长度代替 |
| `stream_unread_exact`、`stream_stats_up` | Redis 7 可计算 lag 时 exact=1；缺字段/null 时使用有界估计，exact=0。stats_up=0 表示未知，旧值不能当实时数据 |
| `oldest_pending_age_seconds` | 最早 pending 消息自生成后的年龄，不是最后重试 idle |
| `outbox_pending`、`outbox_oldest_age_seconds`、`outbox_stats_up` | 待发布数量、最老待发布年龄及采集状态，维护周期采集 |

告警示例见 [`v3-config-cache-alerts.yml`](v3-config-cache-alerts.yml)。刷新成功率建议 `(applied+unchanged)/(applied+unchanged+failed)`，排除 superseded；缓存命中率为 hit/全部 cache_requests。5 秒事件目标需结合失效延迟和 Outbox 年龄判断，不能仅用冷启动 apply_age 判定传播故障。

## 升级与回退

1. 备份并执行兼容 DDL（[`migrations/098-config-cache.sql`](migrations/098-config-cache.sql)），或由启动 `InitTables/AutoMigrate` 执行。原项目版本初始化为 1，不生成虚假历史审计。手工 ALTER 只执行一次，AutoMigrate 可重复执行。
2. 短暂停止配置管理写入，替换全部旧写实例；新旧写程序混用时不能承诺版本/审计保证。新实例初始对账成功、监控可读后再恢复写入。
3. Redis 需要现有认证/SELECT，加上 XADD、XREADGROUP、XACK、XPENDING、XRANGE、XINFO GROUPS、XGROUP CREATE/DESTROY、EVAL/EVALSHA、SET(PX)、EXISTS、SADD、SSCAN、SISMEMBER、SREM。配置相关 key 为 Stream 本身、`<stream>:v2:groups`、`<stream>:v2:lease:<group>`。当前使用普通 Redis client，不新增 Cluster 支持。
4. 只有确认旧程序全部退出，才手工检查并销毁旧前缀消费组；不要批量销毁活实例组。
5. 回退保留新增表/列，记录回退后恢复旧版一致性语义；不要删除未发布 Outbox。V1/V2 API 行为保留。

## 验证

普通测试不访问外部服务：`go test -mod=readonly -count=1 ./...`。Linux 竞态检测：`go test -mod=readonly -race -count=1 ./...`。时间窗口用 `internal/testclock`，加载/提交竞态使用 channel；真实服务异步消费用有界状态轮询，真实超时仅防挂死。

真实服务用 `integration` tag，缺少环境变量直接失败。PowerShell 本地示例（项目名按本次运行取唯一值）：

```powershell
$testProject = 'feedback98-' + [guid]::NewGuid().ToString('N')
$env:REDIS_VERSION = '6.2' # 完成后用新项目名再验证 7.2
docker compose -p $testProject -f test/compose.config-v3.yml up -d --wait
$env:FEEDBACK_INTEGRATION_REDIS_ADDR = '127.0.0.1:6389'
$env:FEEDBACK_INTEGRATION_MYSQL_DSN = 'root:feedback98-test-only@tcp(127.0.0.1:33079)/feedback98_test?parseTime=true'
go test -mod=readonly -tags=integration -count=1 -timeout=180s ./...
docker compose -p $testProject -f test/compose.config-v3.yml down -v
```

并行本地运行需要分别设置 `TEST_REDIS_PORT` 和 `TEST_MYSQL_PORT` 并更新环境地址。Compose MySQL 使用临时文件系统；用例另外创建随机 `feedback98_...` 数据库和 `test:feedback98:<uuid>` key，清理只针对自己创建的资源。不得指向生产服务。CI 的 dev/prod 工作流在独立作业运行 MySQL 8.0 × Redis 6.2/7.2；构建发布依赖这些作业成功。测试不调用 Nacos 或线上飞书，无真实业务 Key。

| 验收 | 测试位置 |
| --- | --- |
| 广播、启动恢复、PEL/ACK、重复乱序、格式错误、裁剪、租约、lag 兼容 | `repository/cache/project_config_event_v3_test.go`、`project_config_integration_test.go`（真实 Redis） |
| 版本/墓碑/逐项目刷新、旧加载失效竞态、60 秒边界、发布/完成失败恢复、审计回滚、任务退出 | `service/config_cache_v3_test.go` |
| MySQL 行锁、并发版本、REPEATABLE READ 快照、租约争抢、旧数据升级、两实例运行时 | `service/config_integration_test.go`（真实 MySQL/Redis） |
| 状态不隐式刷新、审计可信身份与脱敏、热 JWT 撤权和删除窗口 | `service/config_security_v3_test.go` |
| 身份/图片隔离、Scope、Key 轮换、nonce 和在途重放 | 保留 #97 的 `v3_security_test.go`、`auth_v3_inflight_test.go` |
| 限流终止、诊断认证、非法配置拒绝 | `middleware/limit_config_v3_test.go`、`config/config_cache_v3_test.go` |
| HTTP 请求排空后关闭配置运行时、监听失败后回收任务 | `main_test.go` |

miniredis 不用于证明 Redis 7 lag 或真实 PEL/裁剪语义；SQLite 不用于证明 MySQL 行锁和隔离级别。
