import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { IconArrowRight, IconLock, IconMail, IconUser } from "@tabler/icons-react";
import { App, Button, Card, Divider, Form, Input, Space, Typography } from "antd";
import { createSchemaFieldRule } from "antd-zod";
import { z } from "zod";

import AppDocument from "@/components/AppDocument";
import AppLocale from "@/components/AppLocale";
import AppTheme from "@/components/AppTheme";
import AppVersion from "@/components/AppVersion";
import { useAntdForm, useBrowserTheme } from "@/hooks";
import { loginWithPassword } from "@/repository/auth";
import { consumeSSOCallback, getSSOConfig, ldapLogin, startOIDCLogin } from "@/repository/sso";
import { unwrapErrMsg } from "@/utils/error";
import { withBasePath } from "@/utils/url";

const Login = () => {
  const navigage = useNavigate();

  const { t } = useTranslation();

  const { notification } = App.useApp();
  const { theme: browserTheme } = useBrowserTheme();

  // SSO 状态：oidcEnabled / ldapEnabled 决定登录页额外区块是否展示。
  const [oidcEnabled, setOidcEnabled] = useState(false);
  const [oidcDisplayName, setOidcDisplayName] = useState("OIDC");
  const [ldapEnabled, setLdapEnabled] = useState(false);
  const [ldapDisplayName, setLdapDisplayName] = useState("LDAP");

  useEffect(() => {
    (async () => {
      try {
        const resp = await getSSOConfig();
        setOidcEnabled(!!resp.config?.oidc?.enabled);
        setOidcDisplayName(resp.config?.oidc?.displayName || "OIDC");
        setLdapEnabled(!!resp.config?.ldap?.enabled);
        setLdapDisplayName(resp.config?.ldap?.displayName || "LDAP");
      } catch (err) {
        // 后端未启用 SSO 时忽略，仅保留基础登录。
        console.warn("[certimate] failed to load SSO config:", err);
      }

      // 消费 OIDC 回调带回的 token，完成自动登录。
      const consumed = await consumeSSOCallback();
      if (consumed) {
        navigage("/", { replace: true });
      }
    })();
  }, [navigage]);

  const bgStyle = useMemo<React.CSSProperties>(() => {
    let svg = "";
    let mask = "";
    if (browserTheme === "dark") {
      svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32" width="32" height="32" fill="none" stroke="rgb(202 78 13 / 0.12)"><path d="M0 .5H31.5V32"/></svg>`;
      mask = "white";
    } else {
      svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32" width="32" height="32" fill="none" stroke="rgb(249 115 22 / 0.08)"><path d="M0 .5H31.5V32"/></svg>`;
      mask = "black";
    }

    return {
      backgroundImage: `url('data:image/svg+xml;base64,${btoa(svg)}')`,
      maskImage: `linear-gradient(to bottom right, transparent, ${mask}, transparent)`,
    };
  }, [browserTheme]);

  const formSchema = z.object({
    username: z.email(),
    password: z.string().min(10).max(256),
  });
  const formRule = createSchemaFieldRule(formSchema);
  const {
    form: formInst,
    formPending,
    formProps,
  } = useAntdForm<z.infer<typeof formSchema>>({
    initialValues: {
      username: "",
      password: "",
    },
    onSubmit: async (values) => {
      try {
        await loginWithPassword(values.username, values.password);
        await navigage("/");
      } catch (err) {
        notification.error({ title: t("common.text.request_error"), description: unwrapErrMsg(err) });

        throw err;
      }
    },
  });

  const ssoEnabled = oidcEnabled || ldapEnabled;

  return (
    <>
      <div className="pointer-events-none fixed min-h-screen w-full" style={bgStyle}></div>

      <div className="flex h-screen w-full flex-col items-center justify-center">
        <Card className="w-120 max-w-full rounded-md shadow-md max-sm:size-full max-sm:rounded-none">
          <div className="px-4 py-8">
            <div className="mb-12 flex items-center justify-center">
              <img src={withBasePath("/logo.svg")} className="w-16" />
            </div>

            <Form {...formProps} form={formInst} disabled={formPending} layout="vertical" validateTrigger="onBlur">
              <Form.Item name="username" label={t("login.form.username.label")} rules={[formRule]}>
                <Space.Compact block>
                  <Space.Addon>
                    <IconMail size="1.25em" />
                  </Space.Addon>
                  <Input autoComplete="new-password" autoFocus placeholder={t("login.form.username.placeholder")} size="large" />
                </Space.Compact>
              </Form.Item>

              <Form.Item name="password" label={t("login.form.password.label")} rules={[formRule]}>
                <Space.Compact block>
                  <Space.Addon>
                    <IconLock size="1.25em" />
                  </Space.Addon>
                  <Input.Password autoComplete="new-password" placeholder={t("login.form.password.placeholder")} size="large" />
                </Space.Compact>
              </Form.Item>

              <Form.Item className="mt-8 mb-0">
                <Button block type="primary" htmlType="submit" icon={<IconArrowRight size="1.25em" />} iconPlacement="end" loading={formPending} size="large">
                  {t("login.form.submit.button")}
                </Button>
              </Form.Item>
            </Form>

            {ssoEnabled ? (
              <>
                <div className="my-6">
                  <Divider plain>
                    <Typography.Text type="secondary">{t("login.sso.divider")}</Typography.Text>
                  </Divider>
                </div>

                <Space direction="vertical" className="w-full" size="middle">
                  {oidcEnabled ? (
                    <Button block size="large" icon={<IconUser size="1.1em" />} onClick={() => startOIDCLogin(window.location.pathname)}>
                      {t("login.sso.sign_in_with", { provider: oidcDisplayName })}
                    </Button>
                  ) : null}

                  {ldapEnabled ? <LDAPLoginForm displayName={ldapDisplayName} /> : null}
                </Space>
              </>
            ) : null}

            <div className="mt-12">
              <div className="block max-sm:hidden">
                <div className="flex items-center justify-center">
                  <Space align="center" separator={<Divider orientation="vertical" />} size={4}>
                    <AppLocale.LinkButton />
                    <AppTheme.LinkButton />
                    <AppDocument.LinkButton />
                    <AppVersion.LinkButton />
                  </Space>
                </div>
              </div>
              <div className="hidden max-sm:block">
                <div className="flex items-center justify-center">
                  <Space align="center" separator={<Divider orientation="vertical" />} size={4}>
                    <AppLocale.LinkButton />
                    <AppTheme.LinkButton />
                    <AppDocument.LinkButton />
                  </Space>
                </div>
                <div className="mt-6 flex items-center justify-center">
                  <Space align="center" separator={<Divider orientation="vertical" />} size={4}>
                    <AppVersion.LinkButton />
                  </Space>
                </div>
              </div>
            </div>
          </div>
        </Card>
      </div>
    </>
  );
};

// LDAP 登录表单：用户名 + 密码，走后端绑定认证。
const LDAPLoginForm = ({ displayName }: { displayName: string }) => {
  const navigage = useNavigate();

  const { t } = useTranslation();

  const { notification } = App.useApp();

  const formSchema = z.object({
    username: z.string().min(1).max(256),
    password: z.string().min(1).max(256),
  });
  const formRule = createSchemaFieldRule(formSchema);
  const {
    form: formInst,
    formPending,
    formProps,
  } = useAntdForm<z.infer<typeof formSchema>>({
    initialValues: {
      username: "",
      password: "",
    },
    onSubmit: async (values) => {
      try {
        await ldapLogin(values.username, values.password);
        await navigage("/");
      } catch (err) {
        notification.error({ title: t("common.text.request_error"), description: unwrapErrMsg(err) });

        throw err;
      }
    },
  });

  return (
    <Form {...formProps} form={formInst} disabled={formPending} layout="vertical" validateTrigger="onBlur">
      <Form.Item name="username" label={t("login.sso.ldap.username.label")} rules={[formRule]}>
        <Space.Compact block>
          <Space.Addon>
            <IconUser size="1.1em" />
          </Space.Addon>
          <Input autoComplete="new-password" placeholder={t("login.sso.ldap.username.placeholder")} size="large" />
        </Space.Compact>
      </Form.Item>

      <Form.Item name="password" label={t("login.sso.ldap.password.label")} rules={[formRule]}>
        <Space.Compact block>
          <Space.Addon>
            <IconLock size="1.1em" />
          </Space.Addon>
          <Input.Password autoComplete="new-password" placeholder={t("login.sso.ldap.password.placeholder")} size="large" />
        </Space.Compact>
      </Form.Item>

      <Form.Item className="mb-0">
        <Button block size="large" htmlType="submit" loading={formPending}>
          {t("login.sso.sign_in_with", { provider: displayName })}
        </Button>
      </Form.Item>
    </Form>
  );
};

export default Login;
