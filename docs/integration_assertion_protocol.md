# 项目身份断言协议

## 目的

校园项目后端使用自己的私钥证明“当前登录用户是谁”，反馈中台使用数据库中登记的公钥验证断言，并签发只绑定当前项目、表格和学生身份的短期反馈 JWT。

前端不参与这一步，也不能接触校园项目私钥。

## Token Exchange 接口

```http
POST /api/v1/integrations/token/exchange
Content-Type: application/json
```

请求体：

```json
{
  "project_id": "forum",
  "key_id": "forum-prod-2026",
  "assertion": "<RS256 JWT>"
}
```

## Assertion JWT

JWT 必须使用 `RS256` 签名，并在 Header 中携带：

```json
{
  "alg": "RS256",
  "typ": "JWT",
  "kid": "forum-prod-2026"
}
```

Payload 必须包含：

| 字段 | 要求 |
| --- | --- |
| `iss` | 必须等于管理后台登记的公钥 issuer |
| `aud` | 必须包含 `feedback-center` |
| `iat` | 签发时间 |
| `exp` | 过期时间，建议有效期不超过 60 秒 |
| `jti` | 每次请求唯一，用于日志追踪和后续防重放扩展 |
| `project_id` | 必须与请求体中的 `project_id` 一致 |
| `student_id` | 已由校园项目后端登录态确认的学号 |
| `table_identity` | 当前要访问的反馈表标识 |

示例：

```json
{
  "iss": "forum",
  "aud": ["feedback-center"],
  "iat": 1780000000,
  "exp": 1780000060,
  "jti": "3d7c...",
  "project_id": "forum",
  "student_id": "2024214815",
  "table_identity": "forum"
}
```

反馈中台会继续校验项目状态、Key 状态、Key 有效期、表状态以及表级 Scope。

## 响应

```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "access_token": "<feedback JWT>",
    "token_type": "Bearer",
    "expires_in": 3600
  }
}
```

返回的反馈 JWT 只应由校园项目后端转交给自己的前端使用。校园项目后端仍需控制 Token 的使用范围，不能把 assertion 或私钥交给前端。
