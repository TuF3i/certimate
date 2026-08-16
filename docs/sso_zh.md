# 单点登录 (SSO) 集成指南

<div align="center">

[English](./sso.md) ｜ 简体中文

</div>

Certimate 自 v0.5.2 起支持通过**标准 OIDC** 与 **LDAP** 实现单点登录 (SSO)。管理员在「设置 → 单点登录」中分别配置（OIDC 与 LDAP 各支持一个提供者）；配置后登录页会出现对应的登录入口。

---

## 目录

- [特性概览](#特性概览)
- [OIDC 接入](#oidc-接入)
  - [回调地址](#回调地址)
  - [配置项](#配置项)
- [LDAP 接入](#ldap-接入)
- [账号与角色](#账号与角色)
- [API 说明](#api-说明)
- [数据库结构](#数据库结构)
- [常见问题](#常见问题)

---

## 特性概览

- **标准 OIDC**：只需填写 Discovery URL（如 `https://auth.example.com/.well-known/openid-configuration`），授权端点、Token 端点、UserInfo 端点全部自动从发现文档解析，兼容 Keycloak、Casdoor、Authentik、Azure AD、Google Workspace、Okta 等任意标准 OIDC 提供者。
- **LDAP**：服务账号搜索 + 用户绑定验证，兼容 OpenLDAP、AD、FreeIPA 等。
- **统一回调地址**：`/api/sso/callback`，无 provider 参数；设置页提供一键复制，方便在 IdP 侧登记。
- **首次登录自动建号（可选）**：默认创建**普通用户**（`users` 集合、`role=user`），管理员可在「用户管理」中提升为管理员。

## OIDC 接入

### 1. 在 IdP 侧创建应用

以 Keycloak / Authentik / Azure AD 等为例，创建 OIDC Client：

- **Client type / Access type**: Confidential（机密）
- **Redirect URI**: 填写 Certimate 的**统一回调地址**（见下）
- **Scopes**: 勾选 `openid`、`email`、`profile`

### 2. 在 Certimate 中配置

进入 **设置 → 单点登录**，在 OIDC 卡片填写：

| 字段 | 说明 |
| ---- | ---- |
| 启用 | 开启后在登录页显示 OIDC 按钮 |
| Display Name | 登录页按钮展示名（默认 `OIDC`） |
| Discovery URL | OIDC 发现端点，如 `https://auth.example.com/.well-known/openid-configuration` |
| Client ID | IdP 侧生成 |
| Client Secret | IdP 侧生成 |
| Scopes | 默认 `openid email profile` |
| 首次登录自动创建成员账号 | 勾选后未绑定账号会以普通用户身份自动建号 |

### 回调地址

回调地址是**统一固定**的：

```
<你的 Certimate 访问地址>/api/sso/callback
```

例如部署在 `https://cert.example.com` 则为：

```
https://cert.example.com/api/sso/callback
```

设置页会展示当前实际回调地址（依据当前访问域名与协议自动计算，兼容反代后的 `X-Forwarded-Proto`），点击「复制回调地址」即可粘贴到 IdP 的 Redirect URI 配置中。

> 注意：无需也不支持为不同 IdP 使用不同回调地址——SSO 只支持一个 OIDC 提供者。

## LDAP 接入

进入 **设置 → 单点登录**，在 LDAP 卡片填写：

| 字段 | 说明 |
| ---- | ---- |
| 启用 | 开启后在登录页显示 LDAP 登录表单 |
| Display Name | 登录页表单标题（默认 `LDAP`） |
| Server URL | `ldap://host:389` 或 `ldaps://host:636` |
| Service Account DN | 用于搜索用户的服务账号，如 `cn=admin,dc=example,dc=com` |
| Service Account Password | 服务账号密码 |
| Search Base DN | 用户搜索基 DN，如 `ou=people,dc=example,dc=com` |
| Search Filter | 搜索过滤器，`{{username}}` 会被替换为登录用户名，默认 `(uid={{username}})`（AD 可改为 `(sAMAccountName={{username}})`） |
| Email Attribute | 邮箱属性名（默认 `mail`） |
| Name Attribute | 显示名属性名（默认 `displayName`） |
| 首次登录自动创建成员账号 | 同上 |

LDAP 登录流程：服务账号绑定 → 按过滤器搜索用户 DN → 用用户 DN + 登录密码绑定验证 → 认证成功后关联/创建 Certimate 账号。

## 账号与角色

- 登录成功后在 `oauth2_link` 表中按 `(provider, subject)` 关联账号：OIDC 以 `sub` claim 为标识，LDAP 以用户 DN 为标识。
- 已有关联 → 直接登录（快照刷新 email/name）。
- 无关联但有同邮箱账号 → 自动绑定该账号（先管理员后成员）。
- 无关联且开启「自动创建」→ 创建普通用户（`role=user`），管理员可在「用户管理」中改角色或重置密码。
- OIDC/LDAP 自动创建的账号默认**普通用户**，不会自动获得管理员权限。

## API 说明

以下端点均为公开（登录前可达）：

| 方法 | 路径 | 说明 |
| ---- | ---- | ---- |
| GET | `/api/sso/config` | 返回脱敏配置 + `oidcCallback` 统一回调地址 |
| GET | `/api/sso/redirect` | 触发 OIDC 授权跳转（307 → IdP） |
| GET | `/api/sso/callback` | OIDC 回调，签发 token 后 307 回前端 |
| POST | `/api/sso/ldap/login` | LDAP 用户名密码登录，返回 `{ token, record }` |

OIDC 回调成功后会写 HttpOnly 短时 cookie（`certimate_sso_token`）并 307 跳回 `/login?sso_token=...`，前端写入 PocketBase authStore 完成登录。

## 数据库结构

- `settings` 集合：原 `oauth2` 记录更名为 `sso`（旧多提供者配置结构废弃，content 重置为空）。
- `oauth2_link` 集合：继续复用，`provider` 取值 `oidc` / `ldap`，`subjectId` 为 `sub` / 用户 DN。
- 旧的 OAuth2 预设（github/gitlab 等）绑定数据保留但不再生效，可在数据库中清理。

## 常见问题

**登录页没有 SSO 入口？**
- 「设置 → 单点登录」中 OIDC/LDAP 的「启用」是否开启、必填项是否填全（后端会校验）。
- 浏览器 DevTools 查看 `GET /api/sso/config` 返回的 `config.oidc.enabled` / `config.ldap.enabled`。

**OIDC 报 `failed to fetch OIDC discovery`？**
- Discovery URL 必须可公开访问且返回标准发现文档；自建 IdP 请确认 `.well-known/openid-configuration` 路径正确。

**OIDC 报 `redirect_uri` 不匹配？**
- IdP 侧登记的 Redirect URI 必须与设置页展示的「复制回调地址」**完全一致**（含大小写与末尾无斜杠）。

**LDAP 报 `invalid username or password`？**
- 用户不存在或密码错误都会返回此信息。先用 `ldapsearch` 等工具验证服务账号能否搜索到目标用户、过滤器是否正确。

**OIDC/LDAP 用户登录后能做什么？**
- 默认普通用户：可查看、编辑并运行被授权的工作流及其证书（不能删除工作流，也不能修改其授权列表）；提升为管理员后拥有全部权限（见「用户管理」）。

---

如需调整（如增加 OIDC `id_token` 签名校验、LDAP 组映射角色等），可扩展 `internal/sso/` 包，不影响现有流程。