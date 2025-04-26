# FeedBack-Backend

# 📝 飞书 OAuth 接口使用说明

## 📌 简介

本系统使用飞书 OAuth 实现用户认证。首次部署需完成授权流程并调用初始化接口，之后系统会自动刷新 token，前端可直接从 `/get_token` 获取 `access_token` 并用于后续接口调用。

---

## 🚀 首次部署流程（管理员必做）

> ⚠️ 必须先完成此步骤，后续 token 才能自动刷新。

1. 打开浏览器访问：
   ```
   http://localhost:8080
   ```
2. 点击页面上的“使用飞书登录”按钮。
3. 成功授权后，系统将自动调用 `/callback` 接口，获取 `access_token` 与 `refresh_token`。
4. 授权成功后，从响应中获取 `access_token` 和 `refresh_token`。
5. 调用初始化接口 `/init_token`，后台启动自动刷新机制。

### 初始化接口调用示例：

- **接口地址**：`POST /init_token`
- **请求体**：
```json
{
  "access_token": "your_access_token",
  "refresh_token": "your_refresh_token"
}
```
- **响应数据**：
```json
{
  "code": 0,
  "message": "Success",
  "data": null
}
```

---

## 🔁 后续调用流程（推荐前端使用）

### Step ✅ - 调用 `/get_token` 获取当前 `access_token`

- **接口地址**：`POST /get_token`
- **响应数据**：
```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "access_token": "current_access_token"
  }
}
```

---

## 📦 示例：前端调用业务接口流程

```ts
async function prepareAuthHeaders() {
  const res = await fetch('/get_token', {
    method: 'POST',
  });
  const { access_token } = await res.json().data;

  return {
    Authorization: `Bearer ${access_token}`,
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

| 接口路径                       | 说明      |
|----------------------------|---------|
| `POST /sheet/createapp`    | 创建多维表格  |
| `POST /sheet/copyapp`      | 从模板复制表格 |
| `POST /sheet/createrecord` | 添加记录到表格 |
