import { useTranslation } from "react-i18next";
import { Navigate, Outlet } from "react-router-dom";
import { Alert, Layout } from "antd";

import Show from "@/components/Show";
import { getAuthStore } from "@/repository/admin";
import { isBrowserHappy } from "@/utils/browser";

const AuthLayout = () => {
  const { t } = useTranslation();

  const auth = getAuthStore();
  // 多用户模式：管理员或成员已登录时均重定向到控制台。
  if (auth.isValid) {
    return <Navigate to="/" />;
  }

  return (
    <Layout className="h-screen">
      <Show when={!isBrowserHappy()}>
        <Alert banner closable showIcon title={t("common.text.happy_browser")} type="warning" />
      </Show>

      <div className="relative">
        <Outlet />
      </div>
    </Layout>
  );
};

export default AuthLayout;
