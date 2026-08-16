import { memo, useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Navigate, Outlet, useLocation, useNavigate } from "react-router-dom";
import {
  IconBrandGithub,
  IconCertificate,
  IconCodeDots,
  IconFingerprint,
  IconHelpCircle,
  IconHierarchy3,
  IconHome,
  IconKey,
  IconLayoutSidebarLeftCollapse,
  IconLayoutSidebarRightCollapse,
  IconMenu2,
  IconPower,
  IconSettings,
} from "@tabler/icons-react";
import { Alert, App, Button, Drawer, Form, Input, Layout, Menu, type MenuProps, Modal, theme } from "antd";
import { createSchemaFieldRule } from "antd-zod";
import { z } from "zod";

import AppLocale from "@/components/AppLocale";
import AppTheme from "@/components/AppTheme";
import AppVersion from "@/components/AppVersion";
import Show from "@/components/Show";
import { APP_DOCUMENT_URL, APP_REPO_URL } from "@/domain/app";
import { useAntdForm, useTriggerElement } from "@/hooks";
import { getAuthStore } from "@/repository/admin";
import { authWithPassword, updatePassword } from "@/repository/user";
import { isBrowserHappy } from "@/utils/browser";
import { unwrapErrMsg } from "@/utils/error";
import { withBasePath } from "@/utils/url";

const ConsoleLayout = () => {
  const navigate = useNavigate();

  const { t } = useTranslation();

  const { token: themeToken } = theme.useToken();

  const [siderCollapsed, setSiderCollapsed] = useState(false);
  const [changePasswordOpen, setChangePasswordOpen] = useState(false);

  const handleLogoutClick = () => {
    auth.clear();
    navigate("/login");
  };

  const handleDocumentClick = () => {
    window.open(APP_DOCUMENT_URL, "_blank");
  };

  const handleGitHubClick = () => {
    window.open(APP_REPO_URL, "_blank");
  };

  const auth = getAuthStore();
  // 多用户模式：超级管理员（_superusers）与成员（users）均可进入控制台。
  if (!auth.isValid) {
    return <Navigate to="/login" />;
  }
  const isMember = !auth.isSuperuser;

  return (
    <Layout className="h-screen bg-background text-foreground">
      <Show when={!isBrowserHappy()}>
        <Alert banner closable showIcon title={t("common.text.happy_browser")} type="warning" />
      </Show>

      <Layout className="h-screen" hasSider>
        <Layout.Sider
          className="group/sider z-20 h-full border-r bg-background max-md:static max-md:hidden"
          style={{ borderColor: themeToken.colorBorderSecondary }}
          theme="light"
          width={siderCollapsed ? 81 : 256}
        >
          <div className="flex size-full flex-col items-center justify-between overflow-hidden select-none">
            <div className="w-full px-2">
              <SiderMenu collapsed={siderCollapsed} />
            </div>
            <div className="w-full px-2 pb-2">
              <Menu
                style={{ background: "transparent", borderInlineEnd: "none" }}
                inlineCollapsed={siderCollapsed}
                items={[
                  {
                    key: "document",
                    icon: (
                      <span className="anticon scale-125" role="img">
                        <IconHelpCircle size="1em" />
                      </span>
                    ),
                    label: t("common.menu.gethelp"),
                    onClick: handleDocumentClick,
                  },
                  ...(isMember
                    ? [
                        {
                          key: "change-password",
                          icon: (
                            <span className="anticon scale-125" role="img">
                              <IconKey size="1em" />
                            </span>
                          ),
                          label: t("common.menu.change_password"),
                          onClick: () => setChangePasswordOpen(true),
                        },
                      ]
                    : []),
                  {
                    key: "logout",
                    danger: true,
                    icon: (
                      <span className="anticon scale-125" role="img">
                        <IconPower size="1em" />
                      </span>
                    ),
                    label: t("common.menu.logout"),
                    onClick: handleLogoutClick,
                  },
                ]}
                mode="vertical"
                selectable={false}
              />
            </div>
          </div>
          <div className="absolute top-1/2 right-0 translate-x-1/2 -translate-y-1/2 opacity-0 transition-opacity group-hover/sider:opacity-100">
            <Button
              className="bg-background shadow-sm"
              icon={
                siderCollapsed ? (
                  <IconLayoutSidebarRightCollapse size="1.5em" stroke="1.25" color="#999" />
                ) : (
                  <IconLayoutSidebarLeftCollapse size="1.5em" stroke="1.25" color="#999" />
                )
              }
              shape="circle"
              type="text"
              onClick={() => setSiderCollapsed(!siderCollapsed)}
            />
          </div>
        </Layout.Sider>

        <Layout className="flex flex-col overflow-hidden">
          <Layout.Header
            className="relative border-b shadow-sm md:hidden"
            style={{
              padding: 0,
              borderBottomColor: themeToken.colorBorderSecondary,
            }}
          >
            <div className="absolute inset-0 z-0">
              <div
                className="size-full"
                style={{
                  backgroundImage:
                    "linear-gradient(rgba(255, 255, 255, 0.063) 1px, transparent 1px), linear-gradient(90deg, rgba(255, 255, 255, 0.063) 1px, transparent 1px)",
                  backgroundSize: "20px 20px",
                }}
              >
                <div className="size-full backdrop-blur-[1px]"></div>
              </div>
            </div>
            <div className="flex size-full items-center justify-between overflow-hidden px-4">
              <div className="flex items-center gap-4">
                <SiderMenuDrawer trigger={<Button icon={<IconMenu2 size="1.25em" stroke="1.25" />} />} />
              </div>
              <div className="flex size-full grow items-center justify-end gap-4 overflow-hidden">
                <AppTheme.Dropdown>
                  <Button icon={<AppTheme.Icon size="1.25em" stroke="1.25" />} />
                </AppTheme.Dropdown>
                <AppLocale.Dropdown>
                  <Button icon={<AppLocale.Icon size="1.25em" stroke="1.25" />} />
                </AppLocale.Dropdown>
                <AppVersion.Badge>
                  <Button icon={<IconBrandGithub size="1.25em" stroke="1.25" />} onClick={handleGitHubClick} />
                </AppVersion.Badge>
                <Button danger icon={<IconPower size="1.25em" stroke="1.25" />} onClick={handleLogoutClick} />
              </div>
            </div>
          </Layout.Header>

          <Layout.Content className="relative flex-1 overflow-x-hidden overflow-y-auto">
            <Outlet />
          </Layout.Content>
        </Layout>
      </Layout>

      {isMember ? <ChangePasswordModal open={changePasswordOpen} onClose={() => setChangePasswordOpen(false)} /> : null}
    </Layout>
  );
};

const SiderMenu = memo(({ collapsed, onSelect }: { collapsed?: boolean; onSelect?: (key: string) => void }) => {
  const location = useLocation();
  const navigate = useNavigate();

  const { t } = useTranslation();

  const MENU_KEY_HOME = "/";
  const MENU_KEY_WORKFLOWS = "/workflows";
  const MENU_KEY_CERTIFICATES = "/certificates";
  const MENU_KEY_ACCESSES = "/accesses";
  const MENU_KEY_PRESETS = "/presets";
  const MENU_KEY_SETTINGS = "/settings";
  // 设置菜单仅超级管理员可见。
  const isSuperuser = getAuthStore().isSuperuser;
  const menuItems: Required<MenuProps>["items"] = (
    [
      [MENU_KEY_HOME, "dashboard.page.heading", <IconHome size="1em" />],
      [MENU_KEY_WORKFLOWS, "workflow.page.heading", <IconHierarchy3 size="1em" />],
      [MENU_KEY_CERTIFICATES, "certificate.page.heading", <IconCertificate size="1em" />],
      [MENU_KEY_ACCESSES, "access.page.heading", <IconFingerprint size="1em" />],
      [MENU_KEY_PRESETS, "preset.page.heading", <IconCodeDots size="1em" />],
      // 设置菜单仅超级管理员可见。
      ...(isSuperuser ? [[MENU_KEY_SETTINGS, "settings.page.heading", <IconSettings size="1em" />]] : []),
    ] as Array<[string, string, React.ReactNode]>
  ).map(([key, label, icon]) => {
    return {
      key: key,
      icon: (
        <span className="anticon scale-125" role="img">
          {icon}
        </span>
      ),
      label: t(label),
      onClick: () => {
        navigate(key);
        onSelect?.(key);
      },
    };
  });
  const [menuSelectedKey, setMenuSelectedKey] = useState<string>();

  const getActiveMenuItem = () => {
    const item =
      menuItems.find((item) => item!.key === location.pathname) ??
      menuItems.find((item) => item!.key !== MENU_KEY_HOME && location.pathname.startsWith(item!.key as string));
    return item;
  };

  useEffect(() => {
    const item = getActiveMenuItem();
    if (item) {
      setMenuSelectedKey(item.key as string);
    } else {
      setMenuSelectedKey(void 0);
    }
  }, [location.pathname]);

  useEffect(() => {
    if (menuSelectedKey && menuSelectedKey !== getActiveMenuItem()?.key) {
      navigate(menuSelectedKey);
    }
  }, [menuSelectedKey]);

  return (
    <>
      <div className="h-[64px] w-full overflow-hidden px-4 py-2 max-md:py-0">
        <div className="flex size-full items-center justify-around gap-2">
          <img src={withBasePath("/logo.svg")} className="size-[36px]" />
          <Show when={!collapsed}>
            <span className="w-[81px] truncate text-base leading-[64px] font-semibold">Certimate</span>
            <AppVersion.LinkButton className="text-xs" />
          </Show>
        </div>
      </div>
      <div className="w-full grow overflow-x-hidden overflow-y-auto">
        <Menu
          style={{ background: "transparent", borderInlineEnd: "none" }}
          inlineCollapsed={collapsed}
          items={menuItems}
          mode="vertical"
          selectedKeys={menuSelectedKey ? [menuSelectedKey] : []}
          onSelect={({ key }) => {
            setMenuSelectedKey(key);
          }}
        />
      </div>
    </>
  );
});

const SiderMenuDrawer = memo(({ trigger }: { trigger: React.ReactNode }) => {
  const { token: themeToken } = theme.useToken();

  const [siderOpen, setSiderOpen] = useState(false);

  const triggerEl = useTriggerElement(trigger, { onClick: () => setSiderOpen(true) });

  const handleMenuSelect = useCallback(() => {
    setSiderOpen(false);
  }, []);

  return (
    <>
      {triggerEl}

      <Drawer
        closable={false}
        destroyOnHidden
        open={siderOpen}
        placement="left"
        styles={{
          section: { paddingTop: themeToken.paddingSM, paddingBottom: themeToken.paddingSM },
          body: { padding: 0 },
        }}
        onClose={() => setSiderOpen(false)}
      >
        <SiderMenu onSelect={handleMenuSelect} />
      </Drawer>
    </>
  );
});

interface ChangePasswordModalProps {
  open: boolean;
  onClose: () => void;
}

// 成员（users 集合）修改自己的密码：先验证旧密码，再重置新密码。
const ChangePasswordModal = ({ open, onClose }: ChangePasswordModalProps) => {
  const navigate = useNavigate();

  const { t } = useTranslation();

  const { message, notification } = App.useApp();

  const formSchema = z
    .object({
      oldPassword: z.string().min(10).max(256),
      newPassword: z.string().min(10).max(256),
      confirmPassword: z.string().min(10).max(256),
    })
    .refine((values) => values.newPassword === values.confirmPassword, {
      error: t("common.change_password.confirm.errmsg.not_matched"),
      path: ["confirmPassword"],
    });
  const formRule = createSchemaFieldRule(formSchema);
  const {
    form: formInst,
    formPending,
    formProps,
  } = useAntdForm({
    initialValues: {
      oldPassword: "",
      newPassword: "",
      confirmPassword: "",
    },
    onSubmit: async (values) => {
      const auth = getAuthStore();
      if (!auth.record?.email || !auth.record?.id) {
        return;
      }

      try {
        // 先验证旧密码，再更新新密码。
        await authWithPassword(auth.record.email, values.oldPassword);
        await updatePassword(auth.record.id, values.newPassword);

        message.success(t("common.text.operation_succeeded"));

        setTimeout(() => {
          auth.clear();
          navigate("/login");
        }, 500);
      } catch (err) {
        notification.error({ title: t("common.text.request_error"), description: unwrapErrMsg(err) });

        throw err;
      }
    },
  });

  return (
    <Modal
      open={open}
      title={t("common.menu.change_password")}
      footer={null}
      onCancel={() => {
        formInst.resetFields();
        onClose();
      }}
    >
      <Form {...formProps} form={formInst} layout="vertical" validateTrigger="onBlur">
        <Form.Item name="oldPassword" label={t("common.change_password.old_password.label")} rules={[formRule]}>
          <Input.Password autoFocus placeholder={t("common.change_password.old_password.placeholder")} />
        </Form.Item>

        <Form.Item name="newPassword" label={t("common.change_password.new_password.label")} rules={[formRule]}>
          <Input.Password placeholder={t("common.change_password.new_password.placeholder")} />
        </Form.Item>

        <Form.Item name="confirmPassword" label={t("common.change_password.confirm.label")} rules={[formRule]}>
          <Input.Password placeholder={t("common.change_password.confirm.placeholder")} />
        </Form.Item>

        <Form.Item className="mb-0">
          <Button block type="primary" htmlType="submit" loading={formPending}>
            {t("common.change_password.submit")}
          </Button>
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default ConsoleLayout;
