# FeedBack-Backend

# 📝 飞书 OAuth 接口使用说明

## 📌 简介

本系统使用飞书 OAuth 实现用户认证。首次部署需完成授权流程并调用初始化接口，之后系统会自动刷新 token，前端可直接从 `/get_token` 获取 `access_token` 并用于后续接口调用。

---

## ❌ 错误码说明

错误码规范（6 位）：

- 10xxxx：系统 / 基础设施错误
- 20xxxx：业务错误
- 30xxxx：第三方服务错误（如飞书）

注意：
- 错误码必须为 6 位（例如：100001）。
- 只能在已有错误码后追加新错误码，禁止在中间插入，否则会因为使用 `iota` 导致历史错误码含义混乱。

详细错误码请参考 [错误码文档](https://github.com/muxi-Infra/FeedBack-Backend/blob/main/errs/README.md)。

---

## 🛠️ TODO

1. 优化错误处理与日志记录逻辑。
2. 生成 log 追踪 id。
3. 创建 dev 环境
4. 增加重试机制
5. 优化接口速度：mysql 与 飞书（并发） 双写