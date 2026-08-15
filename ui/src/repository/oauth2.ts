import { getBasePath } from "@/utils/url";
import { getPocketBase } from "./_pocketbase";

const pb = getPocketBase();

export interface OAuth2Provider {
  name: string;
  displayName: string;
  enabled: boolean;
  redirectUrl?: string;
  scopes?: string[];
  authUrl?: string;
}

/**
 * 列出在 settings 中启用且具备凭据的 OAuth2 提供商，用于在登录页展示按钮。
 * 不需要鉴权——后端在 `/api/oauth2/providers` 下未启用 superuser auth。
 */
export async function listOAuth2Providers(): Promise<OAuth2Provider[]> {
  const resp = await pb.send<{ code: number; msg: string; data: { providers: OAuth2Provider[] } }>("/api/oauth2/providers", {
    method: "GET",
  });
  if (resp.code !== 0) {
    throw new Error(resp.msg || "Failed to load OAuth2 providers");
  }
  return resp.data?.providers ?? [];
}

/**
 * 触发 OAuth2 跳转。后端会以 307 跳到 OAuth2 提供商。
 * 在浏览器中调用该方法即可完成整页面跳转。
 */
export function startOAuth2Login(provider: string, returnUrl?: string, redirectUrl?: string): void {
  const params = new URLSearchParams();
  params.set("provider", provider);
  if (returnUrl) {
    params.set("returnUrl", returnUrl);
  }
  if (redirectUrl) {
    params.set("redirectUrl", redirectUrl);
  }
  window.location.href = `${getBasePath()}/api/oauth2/redirect?${params.toString()}`;
}

/**
 * 在登录页 OnMount 时调用，检查 URL 查询参数中的 OAuth2 回调 token，
 * 如果存在则写入 PocketBase authStore，并清除 URL 上的临时 query。
 */
export async function consumeOAuth2Callback(): Promise<boolean> {
  const url = new URL(window.location.href);
  const token = url.searchParams.get("oauth2_token");
  if (!token) {
    return false;
  }

  // PocketBase auth_store 至少需要 token 才能判定为有效会话。
  // 我们没有完整的 record JSON，但只用 token 就能让 PocketBase SDK 通过后续 `GET /api/collections/_superusers/auth-refresh`
  // 来通过校验。
  pb.authStore.save(token, null);
  // 触发一次 auth-refresh 拿回完整 record。
  try {
    await pb.collection("_superusers").authRefresh();
  } catch (_err) {
    pb.authStore.clear();
    return false;
  }

  url.searchParams.delete("oauth2_token");
  url.searchParams.delete("oauth2_email");
  url.searchParams.delete("oauth2_provider");
  window.history.replaceState({}, "", url.toString());
  return true;
}
