# ShopMall 后台管理系统 API 文档

基于 Go + Gin 的电商后台管理系统接口文档。

- **Base URL**: `http://localhost:7000`
- **数据格式**: JSON (`Content-Type: application/json`)
- **协议**: HTTP / HTTPS

---

## 目录

- [1. 通用说明](#1-通用说明)
  - [统一响应格式](#统一响应格式)
  - [认证方式](#认证方式)
  - [错误码约定](#错误码约定)
- [2. 健康检查与系统](#2-健康检查与系统)
- [3. 认证接口](#3-认证接口)
- [4. 当前用户](#4-当前用户)
- [5. 员工管理](#5-员工管理)
- [6. 部门管理](#6-部门管理)
- [7. 商品管理](#7-商品管理)
- [8. 角色管理](#8-角色管理)
- [9. 权限管理](#9-权限管理)

---

## 1. 通用说明

### 统一响应格式

所有接口返回统一结构 `R`：

```json
{
  "code": 200,
  "message": "success",
  "data": {},
  "trace_id": "xxxx"    // 请求追踪ID，成功时可能省略
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `code` | int | 业务状态码，`200` 表示成功 |
| `message` | string | 提示信息 |
| `data` | any | 业务数据，失败时省略 |
| `trace_id` | string | 请求追踪ID，由中间件注入（`X-Trace-ID`） |

### 认证方式

本系统采用 **JWT + HttpOnly Cookie + CSRF** 三重机制：

1. **登录成功后**，服务端通过 `Set-Cookie` 下发三个 Cookie：
   - `auth_token`：Access Token（JWT），httpOnly，默认有效期 2 小时，作用于 `/`
   - `refresh_token`：Refresh Token（JWT），httpOnly，默认有效期 7 天，作用于 `/api/auth`
   - `xsrf-token`：CSRF Token，非 httpOnly，前端需读取后在请求头中回传
2. **受保护接口**（`/api` 下除登录/CSRF 外的接口）需同时满足：
   - 携带有效的 `auth_token` Cookie；
   - 请求头带 `X-XSRF-Token`（值来自 `xsrf-token` Cookie）以通过 CSRF 校验。
3. **权限校验**：部分接口额外要求指定权限标识（见各接口标注的 **权限** 列），否则返回 `403`。

> 注：`/api/product/category` 为特殊接口，响应采用 gzip 压缩并支持 ETag 协商缓存，见 [7.2](#72-获取分类级联树)。

### 错误码约定

| HTTP 状态码 | 业务 code | 含义 |
|-------------|-----------|------|
| 200 | 200 | 成功 |
| 201 | 200 | 创建成功 |
| 400 | 400 | 请求参数错误 / 业务校验失败 |
| 401 | 401 | 未登录 / Token 无效或过期 |
| 403 | 403 | 无权限 |
| 404 | 404 | 资源不存在 |
| 409 | 409 | 数据冲突（如手机号/标识已存在） |
| 429 | 429 | 请求过于频繁 / 登录锁定 |
| 500 | 500 | 服务器内部错误 |
| 503 | 503 | 依赖服务（DB/Redis）不可用 |

---

## 2. 健康检查与系统

### 2.1 健康检查

检查数据库与 Redis 连接状态。

- **Method**: `GET`
- **Path**: `/health`
- **权限**: 无需认证

**成功响应 (200)**

```json
{ "code": 200, "message": "success", "data": null }
```

**失败响应 (503)**：数据库或 Redis 任一不可用时返回，`message` 为 `database unavailable` 或 `redis unavailable`。

### 2.2 获取 CSRF Token

获取用于 CSRF 校验的 Token（由中间件写入 `xsrf-token` Cookie）。

- **Method**: `GET`
- **Path**: `/api/csrf/token`
- **权限**: 无需认证

**成功响应 (200)**

```json
{ "code": 200, "message": "success", "data": null }
```

> 调用后需从 `xsrf-token` Cookie 读取值，作为后续请求的 `X-XSRF-Token` 请求头。

---

## 3. 认证接口

### 3.1 员工登录

- **Method**: `POST`
- **Path**: `/api/employee/login`
- **权限**: 无需认证
- **限流**: 登录失败会触发速率限制，超限返回 429

**请求体**

```json
{
  "phone": "13800138000",
  "password": "your-password"
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `phone` | string | 是 | 手机号，需为合法手机号格式 |
| `password` | string | 是 | 密码，长度 8-64 位 |

**成功响应 (200)**

登录成功后自动设置 `auth_token` / `refresh_token` / `xsrf-token` 三个 Cookie。

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "employeeInfo": {
      "employeeId": 1,
      "realName": "张三",
      "role": "admin",
      "age": 30,
      "phone": "138****8000",
      "email": "zhangsan@example.com",
      "avatar": "",
      "gender": 1,
      "department": "技术部",
      "position": "工程师",
      "workStatus": 1,
      "canLogin": 1,
      "createTime": "2026-01-01T10:00:00+08:00"
    }
  }
}
```

> `phone` 返回时会被打码（中间四位替换为 `*`）。

**错误响应**

| 状态码 | 场景 |
|--------|------|
| 400 | 参数错误 / 手机号格式不正确 / 密码长度不合法 |
| 401 | 手机号或密码错误 / 账号被禁用 |
| 429 | 登录尝试次数过多，需等待 |

### 3.2 刷新 Access Token

使用 `refresh_token` Cookie 刷新令牌，实现会话续期。

- **Method**: `POST`
- **Path**: `/api/auth/refresh`
- **权限**: 需携带 `refresh_token` Cookie；已注入 CSRF 校验

**成功响应 (200)**

刷新成功后更新 `auth_token` 与 `refresh_token` Cookie。

```json
{
  "code": 200,
  "message": "success",
  "data": { "message": "token refreshed" }
}
```

**错误响应**

| 状态码 | 场景 |
|--------|------|
| 401 | 缺少 refresh token / token 无效或已过期 / 已被撤销 |
| 429 | 请勿重复刷新（并发刷新锁定） |

### 3.3 员工登出

登出后将当前 Access Token 与 Refresh Token 加入 Redis 黑名单并清除 Cookie。

- **Method**: `POST`
- **Path**: `/api/auth/logout`
- **权限**: 需登录；已注入 CSRF 校验

**成功响应 (200)**

```json
{ "code": 200, "message": "success", "data": "登出成功" }
```

---

## 4. 当前用户

以下接口均需登录，为当前登录用户的白名单接口，无需额外权限。

### 4.1 获取当前用户信息

- **Method**: `GET`
- **Path**: `/api/user/me`
- **权限**: 需登录

**成功响应 (200)**：结构同 [3.1 登录](#31-员工登录) 的 `employeeInfo`。

**错误响应**：`401` 未登录；`404` 用户不存在。

### 4.2 获取当前用户菜单与权限

返回当前用户的菜单树与按钮权限标识列表。

- **Method**: `GET`
- **Path**: `/api/user/menus`
- **权限**: 需登录

**成功响应 (200)**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "menus": [],          // 菜单树（数组）
    "permissions": []     // 按钮权限标识列表，如 ["system:employee:add"]
  }
}
```

### 4.3 修改密码

- **Method**: `PUT`
- **Path**: `/api/user/password`
- **权限**: 需登录

**请求体**

```json
{ "oldPassword": "old-pass", "newPassword": "new-pass" }
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `oldPassword` | string | 是 | 原密码 |
| `newPassword` | string | 是 | 新密码，长度 8-64 位 |

**成功响应 (200)**

```json
{ "code": 200, "message": "success", "data": "密码修改成功" }
```

**错误响应**：`400` 原密码错误 / 参数错误。

---

## 5. 员工管理

> 以下接口除额外标注外，均需登录 + CSRF，并要求对应权限标识。

### 5.1 员工列表（分页）

- **Method**: `GET`
- **Path**: `/api/employee/list`
- **权限**: `system:employee:list`，且要求当前用户为管理员（否则 403）

**查询参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `realName` | string | 否 | 按姓名模糊搜索 |
| `phone` | string | 否 | 按手机号搜索（≤20字符） |
| `page` | int | 否 | 页码，默认 1 |
| `pageSize` | int | 否 | 每页数量，默认 20，上限 100 |

**成功响应 (200)**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "list": [
      {
        "employeeId": 1,
        "realName": "张三",
        "role": "admin",
        "age": 30,
        "phone": "138****8000",
        "email": "zhangsan@example.com",
        "gender": 1,
        "workStatus": 1,
        "canLogin": 1,
        "createTime": "2026-01-01T10:00:00+08:00"
      }
    ],
    "total": 50,
    "page": 1,
    "pageSize": 20
  }
}
```

### 5.2 新增员工

- **Method**: `POST`
- **Path**: `/api/employee`
- **权限**: `system:employee:add`

**请求体**

```json
{
  "realName": "李四",
  "phone": "13900139000",
  "password": "password123",
  "email": "lisi@example.com",
  "gender": 1,
  "age": 28,
  "department": "市场部",
  "position": "专员",
  "role": "admin",
  "workStatus": 1,
  "canLogin": 1,
  "remark": "备注"
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `realName` | string | 是 | 姓名（≤50字） |
| `phone` | string | 否* | 手机号，需唯一、格式合法 |
| `password` | string | 是 | 初始密码，8-64 位 |
| `email` | string | 否 | 邮箱（≤100字，需合法格式） |
| `gender` | int | 否 | 性别 0未知/1男/2女 |
| `age` | int | 否 | 年龄 |
| `department` / `position` | string | 否 | 部门/职位（≤50字） |
| `role` | string | 否 | 角色标识 |
| `workStatus` | int | 否 | 在职状态 0离职/1在职 |
| `canLogin` | int | 否 | 是否允许登录 0否/1是 |
| `remark` | string | 否 | 备注（≤200字） |

**成功响应 (201)**：返回新员工对象，结构同列表项。

**错误响应**：`400` 参数校验失败；`409` 手机号已被使用。

### 5.3 更新员工信息

- **Method**: `PUT`
- **Path**: `/api/employee/:id`
- **权限**: 需登录；仅允许更新**本人**信息（ID 必须等于当前用户，否则 403）

**路径参数**：`id` — 员工ID。

**请求体**（部分字段可为空，仅更新非敏感字段）

```json
{
  "realName": "张三丰",
  "email": "new@example.com",
  "gender": 1,
  "age": 31,
  "remark": "新备注"
}
```

**成功响应 (200)**

```json
{ "code": 200, "message": "success", "data": null }
```

**错误响应**：`400` 参数错误；`403` 无权更新他人信息。

### 5.4 管理员重置员工密码

- **Method**: `PUT`
- **Path**: `/api/employee/:id/password`
- **权限**: `system:employee:reset_pwd`，且需管理员（否则 403）

**路径参数**：`id` — 员工ID。

**请求体**

```json
{ "newPassword": "new-password" }
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `newPassword` | string | 是 | 新密码，8-64 位 |

**成功响应 (200)**

```json
{ "code": 200, "message": "success", "data": "密码重置成功" }
```

**错误响应**：`403` 无权重置 `super_admin` 密码；`404` 目标员工不存在。

---

## 6. 部门管理

> 均需登录 + CSRF + 对应权限。

### 6.1 部门列表（分页）

- **Method**: `GET`
- **Path**: `/api/department`
- **权限**: `system:department:list`

**查询参数**：`page`（默认1）、`pageSize`（默认20）、`deptName`（可选模糊搜索）。

**成功响应 (200)**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "list": [
      {
        "deptId": 1,
        "deptName": "技术部",
        "deptSort": 1,
        "status": 1,
        "createTime": "2026-01-01T10:00:00+08:00",
        "updateTime": "2026-01-01T10:00:00+08:00",
        "remark": ""
      }
    ],
    "total": 10
  }
}
```

### 6.2 获取全部部门（无分页）

- **Method**: `GET`
- **Path**: `/api/department/all`
- **权限**: `system:department:list`

**成功响应 (200)**：`data` 为部门对象数组（无 `list`/`total` 包装）。

### 6.3 获取单个部门

- **Method**: `GET`
- **Path**: `/api/department/:deptId`
- **权限**: `system:department:list`

**成功响应 (200)**：`data` 为单个部门对象。

**错误响应**：`400` ID无效；`404` 部门不存在。

### 6.4 新增部门

- **Method**: `POST`
- **Path**: `/api/department`
- **权限**: `system:department:add`

**请求体**

```json
{ "deptName": "产品部", "deptSort": 2, "remark": "备注" }
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `deptName` | string | 是 | 部门名称（≤50字） |
| `deptSort` | int | 否 | 排序值 0-9999 |
| `remark` | string | 否 | 备注（≤200字） |

**成功响应 (201)**

```json
{ "code": 200, "message": "success", "data": { "deptId": 1 } }
```

### 6.5 更新部门

- **Method**: `PUT`
- **Path**: `/api/department/:deptId`
- **权限**: `system:department:update`

**请求体**：同 [6.4](#64-新增部门)。

**成功响应 (200)**

```json
{ "code": 200, "message": "success", "data": null }
```

**错误响应**：`404` 部门不存在。

### 6.6 删除部门

- **Method**: `DELETE`
- **Path**: `/api/department/:deptId`
- **权限**: `system:department:delete`

**成功响应 (200)**

```json
{ "code": 200, "message": "success", "data": null }
```

---

## 7. 商品管理

> 除分类树外均需登录 + CSRF + 对应权限。分类树为特殊响应。

### 7.1 分类树（特殊接口）

获取商品分类级联树。

- **Method**: `GET`
- **Path**: `/api/product/category`
- **权限**: `product:category:list`
- **响应格式**: **gzip 压缩的 JSON 原始字节**，非标准 `R` 结构

**请求头**：可选 `If-None-Match`（配合 ETag 协商缓存）。

**成功响应 (200)**

```http
Content-Encoding: gzip
Content-Type: application/json; charset=utf-8
ETag: "xxx"
Cache-Control: no-cache
```

`data`（解压后）为分类级联数组：

```json
[
  {
    "catId": 1,
    "name": "手机数码",
    "catLevel": 1,
    "sort": 1,
    "showStatus": 1,
    "children": [
      {
        "catId": 11,
        "name": "手机",
        "catLevel": 2,
        "sort": 1,
        "showStatus": 1,
        "children": []
      }
    ]
  }
]
```

**304 Not Modified**：当请求 `If-None-Match` 与当前 ETag 一致时返回。

### 7.2 新增分类

- **Method**: `POST`
- **Path**: `/api/product/category`
- **权限**: `product:category:add`

**请求体**

```json
{
  "name": "手机数码",
  "parentCid": 0,
  "catLevel": 1,
  "showStatus": 1,
  "sort": 1,
  "icon": "icon-url"
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 分类名称（≤50字） |
| `parentCid` | int64 | 否 | 父分类ID，0 为顶级 |
| `catLevel` | int | 是 | 分类层级 1-3 |
| `showStatus` | int | 是 | 显示状态 0/1 |
| `sort` | int | 否 | 排序值 0-9999 |
| `icon` | string | 否 | 图标（≤500字） |

**成功响应 (201)**

```json
{ "code": 200, "message": "success", "data": { "catId": 1 } }
```

### 7.3 编辑分类

- **Method**: `PUT`
- **Path**: `/api/product/category`
- **权限**: `product:category:update`

**请求体**

```json
{ "catId": 1, "name": "手机", "sort": 2, "showStatus": 1 }
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `catId` | int64 | 是 | 分类ID |
| `name` | string | 是 | 分类名称 |
| `sort` | int | 否 | 排序值 0-9999 |
| `showStatus` | int | 否 | 显示状态 0/1 |

**成功响应 (200)**

```json
{ "code": 200, "message": "success", "data": null }
```

**错误响应**：`404` 分类不存在。

### 7.4 删除分类（逻辑删除）

- **Method**: `DELETE`
- **Path**: `/api/product/category/:catId`
- **权限**: `product:category:delete`

**成功响应 (200)**：返回受影响的关联数量。

```json
{
  "code": 200,
  "message": "success",
  "data": { "categoryCount": 1, "brandCount": 2 }
}
```

**错误响应**：`404` 分类不存在。

### 7.5 获取分类关联品牌列表

- **Method**: `GET`
- **Path**: `/api/product/brands?catId=1`
- **权限**: `product:brand:list`

**查询参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `catId` | int64 | 是 | 分类ID |

**成功响应 (200)**：`data` 为品牌对象数组。

```json
{
  "code": 200,
  "message": "success",
  "data": [
    {
      "brandId": 1,
      "catId": 1,
      "name": "Apple",
      "logo": "",
      "description": "苹果公司",
      "showStatus": 1,
      "firstLetter": "A",
      "sort": 1
    }
  ]
}
```

### 7.6 新增品牌

- **Method**: `POST`
- **Path**: `/api/product/brand`
- **权限**: `product:brand:add`

**请求体**

```json
{
  "catId": 1,
  "name": "小米",
  "logo": "logo-url",
  "description": "小米公司"
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `catId` | int64 | 是 | 关联分类ID |
| `name` | string | 是 | 品牌名称（≤50字） |
| `logo` | string | 否 | Logo（≤500字） |
| `description` | string | 否 | 描述（≤500字） |

**成功响应 (201)**

```json
{ "code": 200, "message": "success", "data": 1 }
```

### 7.7 更新品牌

- **Method**: `PUT`
- **Path**: `/api/product/brand/:brandId`
- **权限**: `product:brand:update`

**请求体**

```json
{ "name": "小米科技", "logo": "new-logo", "description": "新描述" }
```

**成功响应 (200)**

```json
{ "code": 200, "message": "success", "data": null }
```

**错误响应**：`404` 品牌不存在。

### 7.8 删除品牌（物理删除）

- **Method**: `DELETE`
- **Path**: `/api/product/brand/:brandId`
- **权限**: `product:brand:delete`

**成功响应 (200)**

```json
{ "code": 200, "message": "success", "data": null }
```

**错误响应**：`404` 品牌不存在。

---

## 8. 角色管理

> 均需登录 + CSRF + 对应权限。

### 8.1 获取所有角色

- **Method**: `GET`
- **Path**: `/api/admin/role`
- **权限**: `system:role:list`

**成功响应 (200)**：`data` 为角色对象数组。

```json
{
  "code": 200,
  "message": "success",
  "data": [
    {
      "id": 1,
      "roleName": "超级管理员",
      "roleKey": "super_admin",
      "description": "",
      "status": 1,
      "createdAt": "2026-01-01T10:00:00+08:00",
      "updatedAt": "2026-01-01T10:00:00+08:00"
    }
  ]
}
```

### 8.2 获取单个角色

- **Method**: `GET`
- **Path**: `/api/admin/role/:id`
- **权限**: `system:role:list`

**成功响应 (200)**：`data` 为单个角色对象。

**错误响应**：`404` 角色不存在。

### 8.3 创建角色

- **Method**: `POST`
- **Path**: `/api/admin/role`
- **权限**: `system:role:add`

**请求体**

```json
{ "roleName": "运营", "roleKey": "operator", "description": "运营人员" }
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `roleName` | string | 是 | 角色名称（≤50字） |
| `roleKey` | string | 是 | 角色标识（≤50字，需小写字母/数字/下划线） |
| `description` | string | 否 | 描述（≤100字） |

**成功响应 (201)**：`data` 为新角色对象。

**错误响应**：`409` 角色标识已存在。

### 8.4 更新角色

- **Method**: `PUT`
- **Path**: `/api/admin/role/:id`
- **权限**: `system:role:update`

**请求体**（指针字段，仅提供需更新的字段）

```json
{ "roleName": "运营专员", "description": "新描述", "status": 1 }
```

**成功响应 (200)**

```json
{ "code": 200, "message": "success", "data": null }
```

**错误响应**：`404` 角色不存在；`409` 角色标识已存在。

### 8.5 删除角色

- **Method**: `DELETE`
- **Path**: `/api/admin/role/:id`
- **权限**: `system:role:delete`

**成功响应 (200)**

```json
{ "code": 200, "message": "success", "data": "删除成功" }
```

**错误响应**：`400` 系统内置角色（`super_admin`/`admin`）不允许删除，或角色已关联员工。

### 8.6 获取角色权限ID列表

- **Method**: `GET`
- **Path**: `/api/admin/role/:id/permissions`
- **权限**: `system:role:list`

**成功响应 (200)**：`data` 为权限ID数组。

```json
{ "code": 200, "message": "success", "data": [1, 2, 3] }
```

### 8.7 分配权限给角色

- **Method**: `PUT`
- **Path**: `/api/admin/role/:id/permissions`
- **权限**: `system:role:assign_perm`

**请求体**

```json
{ "permissionIds": [1, 2, 3] }
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `permissionIds` | int64[] | 是 | 权限ID列表（≤200个） |

**成功响应 (200)**

```json
{ "code": 200, "message": "success", "data": "分配成功" }
```

---

## 9. 权限管理

> 均需登录 + CSRF + 对应权限。

### 9.1 获取所有权限

- **Method**: `GET`
- **Path**: `/api/admin/permission`
- **权限**: `system:permission:list`

**成功响应 (200)**：`data` 为权限对象数组。

```json
{
  "code": 200,
  "message": "success",
  "data": [
    {
      "id": 1,
      "permissionName": "员工管理",
      "permissionKey": "system:employee",
      "type": 1,
      "parentId": 0,
      "path": "/system/employee",
      "method": "",
      "sortOrder": 1,
      "status": 1,
      "createdAt": "2026-01-01T10:00:00+08:00",
      "updatedAt": "2026-01-01T10:00:00+08:00"
    }
  ]
}
```

> `type`：1=菜单，2=按钮，3=API。

### 9.2 获取单个权限

- **Method**: `GET`
- **Path**: `/api/admin/permission/:id`
- **权限**: `system:permission:list`

**成功响应 (200)**：`data` 为单个权限对象。

**错误响应**：`404` 权限不存在。

### 9.3 创建权限

- **Method**: `POST`
- **Path**: `/api/admin/permission`
- **权限**: `system:permission:add`

**请求体**

```json
{
  "permissionName": "新增员工",
  "permissionKey": "system:employee:add",
  "type": 2,
  "parentId": 1,
  "path": "",
  "method": "POST",
  "sortOrder": 1
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `permissionName` | string | 是 | 权限名称（≤50字） |
| `permissionKey` | string | 是 | 权限标识（≤100字） |
| `type` | int | 是 | 1菜单 / 2按钮 / 3API |
| `parentId` | int64 | 否 | 父权限ID，0为顶级 |
| `path` | string | 否 | 菜单路由（按钮/API留空） |
| `method` | string | 否 | HTTP方法（仅API权限，如 GET/POST） |
| `sortOrder` | int | 否 | 排序值 0-9999 |

**成功响应 (201)**：`data` 为新权限对象。

**错误响应**：`409` 权限标识已存在。

### 9.4 更新权限

- **Method**: `PUT`
- **Path**: `/api/admin/permission/:id`
- **权限**: `system:permission:update`

**请求体**（指针字段，仅提供需更新的字段）

```json
{ "permissionName": "新增员工(新)", "sortOrder": 2 }
```

**成功响应 (200)**

```json
{ "code": 200, "message": "success", "data": null }
```

**错误响应**：`404` 权限不存在；`409` 权限标识已存在。

### 9.5 删除权限

- **Method**: `DELETE`
- **Path**: `/api/admin/permission/:id`
- **权限**: `system:permission:delete`

**成功响应 (200)**

```json
{ "code": 200, "message": "success", "data": "删除成功" }
```

**错误响应**：`400` 存在子权限，无法删除。

---

## 附录：权限标识清单

| 模块 | 权限标识 |
|------|----------|
| 员工 | `system:employee:list`、`system:employee:add`、`system:employee:update`、`system:employee:reset_pwd` |
| 部门 | `system:department:list`、`system:department:add`、`system:department:update`、`system:department:delete` |
| 商品分类 | `product:category:list`、`product:category:add`、`product:category:update`、`product:category:delete` |
| 商品品牌 | `product:brand:list`、`product:brand:add`、`product:brand:update`、`product:brand:delete` |
| 角色 | `system:role:list`、`system:role:add`、`system:role:update`、`system:role:delete`、`system:role:assign_perm` |
| 权限 | `system:permission:list`、`system:permission:add`、`system:permission:update`、`system:permission:delete` |
