# FeedBack-Backend

# 📝 飞书 OAuth 接口使用说明

## 📌 简介

FeedBack-Backend 是一个基于 Go 的后端服务，用于收集和管理来自飞书（Feishu）多维表格的用户反馈数据。它封装了飞书 Bitable 与开放平台的认证流程（包括租户访问令牌的获取与刷新）、记录的增删查改以及与第三方服务（如图片临时下载链接、消息推送）的交互。该服务适合用于校园、企业或组织内部的反馈收集系统，支持：

- 多维表格记录的创建、查询与更新（包含附件图片处理）
- 自动刷新与管理飞书租户 Token
- 可扩展的服务/中间件架构（日志、限流、Prometheus 指标等）

快速开始提示：请先参考 `config/example_config.yaml` 填写配置，运行 `make swag` 生成 API 文档，使用 `go run main.go` 启动服务。

---

## 🖼️ 架构图

![架构图](https://github.com/muxi-Infra/FeedBack-Backend/blob/main/docs/architecture.png)

---

## ❌ 错误码说明

错误码规范（6 位）：

- 10xxxx：系统 / 基础设施错误
- 20xxxx：业务错误
- 30xxxx：第三方服务错误（如飞书）

注意：
- 错误码必须为 6 位（例如：100001）。

详细错误码请参考 [错误码文档](https://github.com/muxi-Infra/FeedBack-Backend/blob/main/errs/README.md)。

---

## 🛠️ TODO

1. 优化错误处理与日志记录逻辑。
2. 生成 log 追踪 id。
3. 创建 dev 环境
4. 增加重试机制
5. 优化接口速度：mysql 与 飞书（并发） 双写
6. 飞书接口重新封装