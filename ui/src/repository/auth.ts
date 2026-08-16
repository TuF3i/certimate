import type { RecordModel } from "pocketbase";
import { getPocketBase } from "./_pocketbase";

const pb = getPocketBase();

/**
 * 统一登录：后端自动识别该邮箱属于超级管理员（_superusers）还是成员（users），
 * 返回与 PocketBase 标准登录一致的 { token, record }，这里直接写入 authStore。
 */
export const loginWithPassword = async (username: string, password: string) => {
  const resp = await pb.send<BaseResponse<{ token: string; record: Record<string, unknown> }>>("/api/auth/login", {
    method: "POST",
    body: {
      username,
      password,
    },
  });

  if (resp.code !== 0) {
    throw new Error(resp.msg || "Failed to login");
  }

  pb.authStore.save(resp.data.token, resp.data.record as RecordModel);
};
