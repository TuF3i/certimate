# OAuth2 Single Sign-On (SSO) Integration

<div align="center">

English ｜ [简体中文](./oauth2_zh.md)

</div>

Starting with v0.4.31, Certimate supports single sign-on (SSO) through any OAuth2-compatible identity provider. Administrators configure one or more providers under **Settings → OAuth2**; corresponding **"Sign in with XXX"** buttons then appear on the login page, while identity lifecycle and tokens stay consistent with the existing password-based account flow.

---

## Table of Contents

- [Overview](#overview)
- [Built-in Providers](#built-in-providers)
- [Quick Start: GitHub Example](#quick-start-github-example)
- [Advanced Configuration](#advanced-configuration)
- [API and Callback Flow](#api-and-callback-flow)
- [Database Schema Changes](#database-schema-changes)
- [Architecture Notes](#architecture-notes)
- [Security Considerations](#security-considerations)
- [Troubleshooting](#troubleshooting)

---

## Overview

- **Non-invasive**: Reuses PocketBase's existing `_superusers` superuser accounts and JWT issuance; the issued token is exactly equivalent to the one obtained via password login. The front-end `authStore` / `authRefresh` flow is unchanged.
- **Extensible**: 6 built-in presets for the most common OAuth2 providers; any other OAuth2 / OIDC provider can be onboarded by simply filling endpoints and field mappings.
- **No internal collection restructure**: We do not rely on PocketBase's built-in auth/OAuth2 settings table, so existing `_superusers` schema stays untouched.
- **Safe defaults**: When unconfigured, the login page is identical to the original behavior. `autoCreate` is off by default to prevent unauthorized users from gaining superuser access.

## Built-in Providers

The `name` column is the provider identifier you use as the `name` field in Settings. Any empty endpoint / field is automatically filled from the preset below.

| name        | Display Name          | Default Scopes        | Subject field |
| ----------- | --------------------- | --------------------- | ------------- |
| `github`    | GitHub                | `read:user`           | `id`          |
| `gitlab`    | GitLab                | `read_user`           | `id`          |
| `gitee`     | Gitee                 | `user_info`           | `id`          |
| `google`    | Google                | `openid email profile`| `sub`         |
| `azuread`   | Microsoft (Azure AD)  | `openid email profile`| `sub`         |
| `dingtalk`  | DingTalk              | _(empty)_             | `openId`      |

> Other providers (custom GitLab, Casdoor, Keycloak, Auth0, generic OIDC, etc.) can be onboarded via the **Custom endpoints** section below — no code change required.

## Quick Start: GitHub Example

> Suppose your Certimate runs at `https://cert.example.com`.

### 1. Create an OAuth App on GitHub

Open <https://github.com/settings/developers> → **New OAuth App**:

- **Homepage URL**: `https://cert.example.com`
- **Authorization callback URL**: `https://cert.example.com/api/oauth2/callback?provider=github`

Record the resulting **Client ID** and **Client Secret**.

> The callback URL must end with `?provider=github` (or another provider name). Use this exact string both on GitHub and in Certimate's **Redirect URL** field.

### 2. Configure in Certimate

Log in with password, then visit **Settings → OAuth2**:

1. Click **Add github** — a GitHub collapse panel appears.
2. Open the panel and fill:

| Field          | Value                                                          |
| -------------- | -------------------------------------------------------------- |
| Enabled        | ✓                                                              |
| Display Name   | `GitHub` (or any label to show on the login button)            |
| Client ID      | from step 1                                                   |
| Client Secret  | from step 1                                                   |
| Redirect URL   | `https://cert.example.com/api/oauth2/callback?provider=github` |

Save.

### 3. Verify

Log out and return to `/login`:

- A divider **"or"** and a **Sign in with GitHub** button appear below the password form.
- Click → authorize at GitHub → automatically redirected to `/login?oauth2_token=...` → automatically logged in and navigated to `/`.

---

## Advanced Configuration

### Auto-create Superuser

By default, any OAuth2 identity **without a pre-linked account** is rejected with `no superuser linked to oauth2 provider`. To allow first-time login to create a `_superusers` row automatically, in the configuration panel:

| Field                 | Effect                                                                                  |
| --------------------- | --------------------------------------------------------------------------------------- |
| Auto-create superuser | When enabled, unlinked identities will create a new superuser record on first login      |
| Email prefix           | When the OAuth2 userinfo lacks an email, a placeholder email is generated as `<prefix>+<provider>+<subject>@certimate.local` |

The auto-created account is given a random strong password (the user does not need it; they can later set one from **Settings → Account**).

### Custom Endpoints / Field Mapping

For self-hosted IdPs (GitLab self-hosted, Casdoor, Keycloak, Auth0, generic OIDC) you must fill these explicitly:

| Field             | Purpose                                              | Example                                                       |
| ------------------ | --------------------------------------------------- | ------------------------------------------------------------- |
| Authorization URL  | OAuth2 authorization endpoint                       | `https://auth.mycompany.com/oauth2/authorize`                |
| Token URL          | OAuth2 token endpoint                               | `https://auth.mycompany.com/oauth2/token`                     |
| UserInfo URL       | UserInfo endpoint                                   | `https://auth.mycompany.com/oauth2/userinfo`                  |
| Subject field      | Unique ID read from UserInfo JSON (≤256 chars)      | `sub` / `id` / `openId` / `preferred_username`                |
| Scopes (space séparé) | Requested scopes                                  | `openid profile email`                                        |
| Scopes Join        | Nonstandard scope separator (default: space)        | Some WeChat-style flows may require `,`                       |

> Only `subject` field mapping is exposed in the UI (it is the only one required for safe linking); `email / name / avatar` use preset/fixed mappings to superuser context.

### Multiple Tenants of the Same Provider

You can add multiple panels with the same or different `name`. Each is keyed by `(provider, subjectId)`, so multiple Client IDs of the same upstream provider work side by side.

---

## API and Callback Flow

Public endpoints (not protected by the `/api` `RequireSuperuserAuth` middleware):

| Method | Path                          | Description                                                                 |
| ------ | ----------------------------- | -------------------------------------------------------------------------- |
| GET    | `/api/oauth2/providers`       | List enabled providers (reveals `name`, `displayName`, `scopes`, `authUrl`) |
| GET    | `/api/oauth2/redirect`        | Issue authorize redirect with `?provider=...&returnUrl=...`               |
| GET    | `/api/oauth2/callback`        | Handle provider callback and issue token                                  |

### Sequence

```
1. User        GET /api/oauth2/redirect?provider=github&returnUrl=/
2. Certimate   Validate provider enabled state; issue one-time state (PocketBase KV, 5min TTL)
               307 → https://github.com/login/oauth/authorize?...&state=...
3. User        Authorizes
4. GitHub      302 → /api/oauth2/callback?provider=github&code=...&state=...
5. Certimate   Verify state; exchange code for access_token; fetch UserInfo
               Look up link, or reuse by email, or autoCreate (if enabled)
               record.NewAuthToken() → superuser JWT
               Set-Cookie: certimate_oauth2_token (HttpOnly, 60s)
               307 → returnUrl?oauth2_token=...
6. Frontend    Login.tsx detects oauth2_token → pb.authStore.save(token)
               pb.collection('_superusers').authRefresh() retrieves full record
               Removes temporary query parameters; navigate('/')
```

---

## Database Schema Changes

Migration `migrations/1790568000_upgrade_v0.4.31.go` runs automatically on upgrade and:

1. Creates the `oauth2_link` collection:

   | Field               | Type     | Description                                             |
   | ------------------- | -------- | ------------------------------------------------------- |
   | `provider`          | text     | OAuth2 provider identifier (e.g. `github`)              |
   | `subjectId`         | text     | User ID at the provider                                 |
   | `superuserId`       | text     | Linked `_superusers.id`                                |
   | `userProfileEmail`  | text     | Latest email snapshot                                   |
   | `userProfileName`   | text     | Latest display name snapshot                            |
   | `userProfileAvatar` | text     | Latest avatar URL snapshot                              |
   | `created`/`updated` | autodate | Auto-managed                                            |

   Unique index on `(provider, subjectId)`; secondary index on `(provider, superuserId)`.

2. Pre-creates an empty `settings` row with `name = "oauth2"`.

> The migration is idempotent. The ahead-of-migration `pkg/sdk3rd-forked/` warning is unrelated and resolves with `git submodule update --init --recursive`.

---

## Architecture Notes

### Key Source Layout

| Package / File                                | Role                                                                        |
| --------------------------------------------- | --------------------------------------------------------------------------- |
| `internal/domain/oauth2_link.go`              | `OAuth2Link` domain model and collection name constant                      |
| `internal/domain/settings.go`                 | `SettingsContentForOAuth2` + `AsOAuth2()`                                   |
| `internal/oauth2/providers.go`                | Built-in presets (endpoints + field mappings)                               |
| `internal/oauth2/oauth2.go`                   | Core service: URL generation, state, token exchange, userinfo, linking     |
| `internal/repository/oauth2_link.go`          | `oauth2_link` CRUD                                                          |
| `internal/rest/handlers/oauth2.go`            | Three public HTTP endpoints                                                |
| `internal/rest/routes/routes.go`              | Registers OAuth2 routes on the **root router** (out of superuser auth scope) |
| `internal/settings/{settings.go,pbstore.go}`  | Registers `oauth2` settings store; `GetGlobalSettingsForOAuth2()`          |
| `internal/domain/dtos/oauth2.go`              | In/out DTOs                                                                |
| `migrations/1790568000_upgrade_v0.4.31.go`   | Auto-migration                                                             |
| `ui/src/repository/oauth2.ts`                 | List providers, trigger login, consume callback token                      |
| `ui/src/stores/settings/oauth2/index.ts`      | `useOAuth2SettingsStore`                                                    |
| `ui/src/pages/settings/SettingsOAuth2.tsx`    | Admin configuration page                                                   |
| `ui/src/pages/login/Login.tsx`                | Render login buttons and auto-consume token                                |

### Design Highlights

1. **Routes registered on the root router group**, bypassing `/api`'s `RequireSuperuserAuth` middleware.
2. **Reuses PocketBase JWT** by calling `record.NewAuthToken()` directly; the issued token is byte-compatible with the password login flow.
3. **State is one-shot and provider-bound**, preventing CSRF reuse across providers.
4. **Graceful degradation**: an empty `/providers` response hides the OAuth2 block on the login page, with no behavior change.

---

## Security Considerations

- `clientSecret` is write-only and never exposed via `GET /api/oauth2/providers`.
- `state` is 24 random bytes + base64url, stored in PocketBase KV with a 5-minute TTL. Provider match is verified on callback; state is deleted immediately after consumption.
- `returnUrl` is validated to be same-origin (relative or same `Host`) to prevent open redirects.
- `autoCreate` defaults to **off**, so unlinked identities cannot become superusers without explicit admin opt-in.
- An `HttpOnly`, 60-second `certimate_oauth2_token` cookie supplements the query token for headless browser scenarios.
- Always deploy Certimate over HTTPS in production (as your OAuth2 provider will require).

---

## Troubleshooting

**No "Sign in with XXX" buttons on the login page?**

- Confirm at least one provider is enabled in **Settings → OAuth2**.
- Make sure `Client ID` and `Client Secret` are filled (the backend `GetProvider` requires both).
- Inspect `GET /api/oauth2/providers` in your browser's DevTools — an empty `providers` array means no enabled/credentialed provider.

**`oauth2 state does not match provider` on callback?**

State is bound at issuance to the specific provider. Make sure the `provider` used at `/redirect` and `/callback` are identical. Manual URL editing or >5 minutes between issuance and callback also fails.

**`no superuser linked to oauth2 provider ...` on callback?**

Default, safe behavior when `(provider, subjectId)` is not pre-linked and `autoCreate` is off. Either:

- Enable **Auto-create superuser** for that provider in Settings; or
- Log in once with password to create a superuser whose email matches the OAuth2 userinfo email — the backend will auto-link by email on the next OAuth2 login.

**Will existing data be wiped on upgrade?**

No. The migration only **adds** a collection and an empty settings record, and is idempotent. `workflow / certificate / access` and other existing data are untouched.

**Can I connect an OIDC provider?**

Yes. OIDC is OAuth2 with a standard userinfo endpoint. Fill `authUrl/tokenUrl/userInfoUrl`, set `Subject field = sub`, and include `openid` in scopes.

---

For deeper customization (PKCE, `id_token` verification, email domain allowlisting, etc.), extend `internal/oauth2/oauth2.go`. A common pattern is to branch `HandleCallback` per provider name into provider-specific implementations without touching the public route layer.