import { getAuthStore } from "@/repository/admin";

/**
 * 判断当前登录者是否为管理员：
 * 超级管理员（_superusers），或 users 集合中 role=admin 的成员。
 */
export const isAdmin = () => {
  const auth = getAuthStore();
  if (!auth.isValid) {
    return false;
  }
  if (auth.isSuperuser) {
    return true;
  }
  return auth.record?.collectionName === "users" && auth.record?.role === "admin";
};
