import PocketBase from "pocketbase";

import { getBasePath } from "@/utils/url";

let pb: PocketBase;
export const getPocketBase = () => {
  if (pb) return pb;
  pb = new PocketBase(getBasePath());
  pb.afterSend = (res, data) => {
    // 排除 auth-refresh：SSO 回调消费 token 时会主动用 auth-refresh 拉取完整 record，
    // 若此时命中错误的集合会返回 401，不应触发清空会话 + 整页刷新（会造成刷新循环）。
    const isAuthRefresh = res.url.includes("/auth-refresh");
    if ((res.status === 401 || res.status === 403) && pb.authStore?.isValid && !isAuthRefresh) {
      pb.authStore.clear();
      location.reload();
    }
    return data;
  };
  return pb;
};

export const COLLECTION_NAME_ADMIN = "_superusers";
export const COLLECTION_NAME_USER = "users";
export const COLLECTION_NAME_ACCESS = "access";
export const COLLECTION_NAME_CERTIFICATE = "certificate";
export const COLLECTION_NAME_SETTINGS = "settings";
export const COLLECTION_NAME_WORKFLOW = "workflow";
export const COLLECTION_NAME_WORKFLOW_RUN = "workflow_run";
export const COLLECTION_NAME_WORKFLOW_OUTPUT = "workflow_output";
export const COLLECTION_NAME_WORKFLOW_LOG = "workflow_logs";
