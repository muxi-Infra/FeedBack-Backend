# FeedBack-Backend

# 📝 飞书 OAuth 接口使用说明

## 📌 简介

本系统使用飞书 OAuth 实现用户认证。首次部署需手动完成授权流程，之后可通过 `refresh_token` 自动刷新 `access_token` 并封装为 JWT，供后续接口使用。

---

## 🚀 首次部署流程（管理员必做）

> ⚠️ 必须先执行此步骤，完成飞书授权并获取初始 token。

1. 打开浏览器访问：
   ```
   http://localhost:8080
   ```
2. 点击页面上的“使用飞书登录”按钮。
3. 成功授权后，页面将自动调用 `/callback` 接口，获取 `access_token` 与 `refresh_token` 并保存在服务端 session 中。
4. 后端会返回飞书授权的 token 信息（包括 `access_token` 和 `refresh_token`）。

---

## 🔁 后续每次调用接口前

前端需按以下顺序调用接口：

### Step 1️⃣ - 调用刷新接口 `/refresh_token`

**说明**：使用之前保存的 `refresh_token` 获取新的 `access_token`

- **接口地址**：`POST /refresh_token`
- **请求体**：
```json
{
  "refresh_token": "xxx"
}
```
- **响应数据**：
```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "access_token": "new_access_token",
    "refresh_token": "new_refresh_token"
  }
}
```

> ✅ 成功后，请务必保存新的 `refresh_token` 用于下次刷新。

---

### Step 2️⃣ - 调用封装接口 `/generate_token`

**说明**：将飞书的 `access_token` 封装为后端 JWT，用于接口身份认证

- **接口地址**：`POST /generate_token`
- **请求体**：
```json
{
  "token": "new_access_token"
}
```
- **响应数据**：
```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "token": "Bearer xxxxxx"
  }
}
```

---

## 📦 示例：前端调用业务接口流程

```ts
async function prepareAuthHeaders() {
  const refreshRes = await fetch('/refresh_token', {
    method: 'POST',
    body: JSON.stringify({ refresh_token: localStorage.getItem('refresh_token') }),
  });
  const { access_token, refresh_token } = await refreshRes.json().data;
  localStorage.setItem('refresh_token', refresh_token);

  const tokenRes = await fetch('/generate_token', {
    method: 'POST',
    body: JSON.stringify({ token: access_token }),
  });
  const jwt = await tokenRes.json().data.token;

  return {
    Authorization: jwt,
  };
}

const headers = await prepareAuthHeaders();
fetch('/sheet/createapp', {
  method: 'POST',
  headers,
  body: JSON.stringify({ ... }),
});
```

---

## 🔐 所有业务接口需携带 Authorization 头

- 示例：
```http
POST /sheet/createapp
Authorization: Bearer xxxx
```

---

## ✅ 已支持的接口列表（需带 JWT）

| 接口路径                   | 说明           |
|----------------------------|----------------|
| `POST /sheet/createapp`    | 创建多维表格   |
| `POST /sheet/copyapp`      | 从模板复制表格 |
| `POST /sheet/createrecord` | 添加记录到表格 |
