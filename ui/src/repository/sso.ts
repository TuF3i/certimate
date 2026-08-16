import type { RecordModel } from "pocketbase";

import { type SSOSettingsContent } from "@/domain/settings";
import { withBasePath } from "@/utils/url";

import { getPocketBase } from "./_pocketbase";

const pb = getPocketBase();

export interface SSOConfigResp {
  config: SSOSettingsContent;
  oidcCallback: string;
}

/**
 * 获取脱敏后的 SSO 配置与统一回调地址。不需要鉴权（登录前可达）。
 */
export async function getSSOConfig(): Promise<SSOConfigResp> {
  const resp = await pb.send<{ code: number; msg: string; data: SSOConfigResp }>("/api/sso/config", {
    method: "GET",
  });
  if (resp.code !== 0) {
    throw new Error(resp.msg || "Failed to load SSO config");
  }
  return resp.data;
}

/**
 * 触发 OIDC 授权跳转（后端 307 到 OIDC 提供商的授权端点）。
 */
export function startOIDCLogin(returnUrl?: string): void {
  const params = new URLSearchParams();
  if (returnUrl) {
    params.set("returnUrl", returnUrl);
  }
  window.location.href = withBasePath(`/api/sso/redirect?${params.toString()}`);
}

/**
 * LDAP 用户名密码登录：后端绑定验证成功后返回 { token, record }。
 */
export async function ldapLogin(username: string, password: string): Promise<void> {
  const resp = await pb.send<{ code: number; msg: string; data: { token: string; record: Record<string, unknown> } }>("/api/sso/ldap/login", {
    method: "POST",
    body: {
      username,
      password,
    },
  });

  if (resp.code !== 0) {
    throw new Error(resp.msg || "LDAP login failed");
  }

  pb.authStore.save(resp.data.token, resp.data.record as RecordModel);
}

/**
 * 在登录页 OnMount 时检查 URL 中的 SSO 回调 token（sso_token），
 * 存在则写入 PocketBase authStore 并刷新对应集合的完整 record，随后清除临时 query。
 *
 * 后端跳转时带 sso_collection（"_superusers" / "users"），直接刷新对应集合，
 * 避免对错误集合发起 auth-refresh 触发 401 刷新循环。
 */
export async function consumeSSOCallback(): Promise<boolean> {
  const url = new URL(window.location.href);
  const token = url.searchParams.get("sso_token");
  if (!token) {
    return false;
  }

  const collection = url.searchParams.get("sso_collection") || undefined;

  const tryRefresh = async (name: string) => {
    try {
      await pb.collection(name).authRefresh();
      return true;
    } catch (_err) {
      return false;
    }
  };

  // 优先按后端标记的集合刷新；缺失时兜底依次尝试。
  const ok = collection ? await tryRefresh(collection) : (await tryRefresh("users")) || (await tryRefresh("_superusers"));

  // 无论成败都清除 URL 上的临时参数，避免刷新后重复消费。
  url.searchParams.delete("sso_token");
  url.searchParams.delete("sso_email");
  url.searchParams.delete("sso_collection");
  window.history.replaceState({}, "", url.toString());

  if (!ok) {
    pb.authStore.clear();
    return false;
  }
  return true;
}
