# 项目身份交换协议

校园项目后端使用项目 API Key 生成 HMAC-SHA256 请求签名，反馈中台验证签名后，签发绑定项目、学生和反馈表的短期 JWT。

前端不参与 Token Exchange，也不能接触 API Key。

## Token Exchange 接口

```http
POST /api/v1/integrations/token/exchange
Content-Type: application/json
```

请求体：

```json
{
  "project_id": "chaguan",
  "key_id": "chaguan-key-2026",
  "student_id": "2024214815",
  "table_identity": "forum",
  "timestamp": 1786000000,
  "nonce": "每次请求唯一的随机字符串",
  "signature": "HMAC-SHA256 签名"
}
```

签名原文按以下顺序使用换行符拼接：

```text
project_id
key_id
student_id
table_identity
timestamp
nonce
```

签名密钥为：

```text
HMAC 密钥 = SHA256(api_key)
```

反馈中台会校验项目状态、Key 状态、Key 有效期、签名、时间戳、Nonce、学生身份和表格状态。Nonce 通过 Redis 保存 5 分钟，重复使用会被拒绝。

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

API Key 只应保存在校园项目后端环境变量或密钥管理服务中，不能提交到代码仓库、写入日志或下发到前端。
