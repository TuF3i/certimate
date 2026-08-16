# Single Sign-On (SSO) Integration Guide

<div align="center">

English ｜ [简体中文](./sso_zh.md)

</div>

Since v0.5.2 Certimate supports single sign-on via **standard OIDC** and **LDAP**. Administrators configure one OIDC provider and/or one LDAP provider under **Settings → Single Sign-On**; matching sign-in entries then appear on the login page.

---

## Table of Contents

- [Overview](#overview)
- [OIDC Setup](#oidc-setup)
  - [Callback URL](#callback-url)
  - [Configuration Fields](#configuration-fields)
- [LDAP Setup](#ldap-setup)
- [Accounts and Roles](#accounts-and-roles)
- [API Reference](#api-reference)
- [Database](#database)
- [FAQ](#faq)

---

## Overview

- **Standard OIDC**: fill in the Discovery URL (e.g. `https://auth.example.com/.well-known/openid-configuration`); authorization / token / userinfo endpoints are resolved automatically from the discovery document. Works with Keycloak, Casdoor, Authentik, Azure AD, Google Workspace, Okta and any other standards-compliant OIDC provider.
- **LDAP**: service-account search + user bind verification. Works with OpenLDAP, Active Directory, FreeIPA, etc.
- **Unified callback**: `/api/sso/callback`, no provider parameter; the settings page offers a one-click copy button to paste into the IdP.
- **Auto-create on first sign-in (optional)**: always creates an **ordinary member** (`users` collection, `role=user`). Admins can promote members in **User Management**.

## OIDC Setup

### 1. Create a client in your IdP

Create a confidential OIDC client:

- **Client type / Access type**: Confidential
- **Redirect URI**: the unified callback URL below
- **Scopes**: `openid`, `email`, `profile`

### 2. Configure Certimate

**Settings → Single Sign-On**, in the OIDC card:

| Field | Description |
| ----- | ----------- |
| Enabled | Shows the OIDC button on the login page |
| Display Name | Button label (default `OIDC`) |
| Discovery URL | e.g. `https://auth.example.com/.well-known/openid-configuration` |
| Client ID | from the IdP |
| Client Secret | from the IdP |
| Scopes | default `openid email profile` |
| Auto-create member on first sign-in | creates an ordinary user when no binding exists |

### Callback URL

The callback URL is **fixed and unified**:

```
<your-certimate-url>/api/sso/callback
```

e.g. for `https://cert.example.com`:

```
https://cert.example.com/api/sso/callback
```

The settings page displays the actual callback URL (computed from the current host/protocol, honoring `X-Forwarded-Proto` behind reverse proxies) with a copy button.

> There is no per-provider callback: SSO supports a single OIDC provider.

## LDAP Setup

In the LDAP card:

| Field | Description |
| ----- | ----------- |
| Enabled | Shows the LDAP form on the login page |
| Display Name | Form title (default `LDAP`) |
| Server URL | `ldap://host:389` or `ldaps://host:636` |
| Service Account DN | e.g. `cn=admin,dc=example,dc=com` |
| Service Account Password | bind password of the service account |
| Search Base DN | e.g. `ou=people,dc=example,dc=com` |
| Search Filter | `{{username}}` is replaced by the login name; default `(uid={{username}})` (AD: `(sAMAccountName={{username}})`) |
| Email Attribute | default `mail` |
| Name Attribute | default `displayName` |
| Auto-create member on first sign-in | same as OIDC |

Flow: service-account bind → search user DN → bind with user DN + login password → link or create a Certimate account.

## Accounts and Roles

- Bindings live in `oauth2_link` keyed by `(provider, subject)`: OIDC uses the `sub` claim, LDAP uses the user DN.
- Existing binding → direct login (email/name snapshot refreshed).
- No binding but same email → auto-bound to the existing account (superusers first, then members).
- No binding + auto-create enabled → a new ordinary member (`role=user`) is created.
- Auto-created SSO accounts are **never** admins by default; promote them in User Management.

## API Reference

Public endpoints (reachable before login):

| Method | Path | Description |
| ------ | ---- | ----------- |
| GET | `/api/sso/config` | Sanitized config + unified `oidcCallback` |
| GET | `/api/sso/redirect` | 307 redirect to the OIDC authorization endpoint |
| GET | `/api/sso/callback` | OIDC callback; issues token, 307 back to the front-end |
| POST | `/api/sso/ldap/login` | LDAP username/password login, returns `{ token, record }` |

After a successful OIDC callback a short-lived HttpOnly cookie (`certimate_sso_token`) is set and the browser is redirected to `/login?sso_token=...`; the front-end saves the token into the PocketBase authStore.

## Database

- `settings`: the former `oauth2` record is renamed to `sso` (old multi-provider structure is dropped, content reset).
- `oauth2_link`: reused; `provider` is `oidc` or `ldap`, `subjectId` is `sub` / user DN.
- Old OAuth2 preset bindings (github/gitlab/etc.) are retained but no longer active; they can be cleaned up manually if desired.

## FAQ

**No SSO entry on the login page?**
- Check the provider is enabled and all required fields are filled (the backend validates).
- Inspect `GET /api/sso/config` in DevTools: `config.oidc.enabled` / `config.ldap.enabled`.

**`failed to fetch OIDC discovery`?**
- The Discovery URL must be publicly reachable and return a standard discovery document.

**`redirect_uri` mismatch?**
- The IdP's registered Redirect URI must exactly equal the copyable callback URL shown in settings.

**LDAP `invalid username or password`?**
- The error is intentionally generic. Verify the service account can search for the user (e.g. with `ldapsearch`) and that the filter is correct.

**What can SSO users do?**
- Ordinary members can only see workflows (and their certificates) explicitly granted to them; promoting them to admin grants full access (see User Management).

---

For deeper customization (e.g. `id_token` signature verification, LDAP group → role mapping), extend the `internal/sso/` package without touching the existing flow.