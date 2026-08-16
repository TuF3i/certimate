import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { IconArrowRight, IconBrandGithub, IconLock, IconMail } from "@tabler/icons-react";
import { App, Button, Card, Divider, Form, Input, Space, Typography } from "antd";
import { createSchemaFieldRule } from "antd-zod";
import { z } from "zod";

import AppDocument from "@/components/AppDocument";
import AppLocale from "@/components/AppLocale";
import AppTheme from "@/components/AppTheme";
import AppVersion from "@/components/AppVersion";
import { useAntdForm, useBrowserTheme } from "@/hooks";
import { loginWithPassword } from "@/repository/auth";
import { type OAuth2Provider, consumeOAuth2Callback, listOAuth2Providers, startOAuth2Login } from "@/repository/oauth2";
import { unwrapErrMsg } from "@/utils/error";
import { withBasePath } from "@/utils/url";

const Login = () => {
  const navigage = useNavigate();

  const { t } = useTranslation();

  const { notification } = App.useApp();
  const { theme: browserTheme } = useBrowserTheme();
  const [oauth2Providers, setOauth2Providers] = useState<OAuth2Provider[]>([]);

  // 1. 初进入时检查后端是否在设置中启用了 OAuth2 提供商。
  // 2. 如果 URL 有 oauth2_token 查询，则消费它并完成登录。
  useEffect(() => {
    (async () => {
      try {
        const list = await listOAuth2Providers();
        setOauth2Providers(list);
      } catch (err) {
        // 忽略明确的 401/404（后端未启用 OAuth2），仅在可观察错误时推入提示。
        console.warn("[certimate] failed to list OAuth2 providers:", err);
      }

      const consumed = await consumeOAuth2Callback();
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

  const renderOAuth2ProviderIcon = (name: string) => {
    if (name === "github") {
      return <IconBrandGithub size="1.25em" />;
    }
    return null;
  };

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

            {oauth2Providers.length > 0 ? (
              <>
                <div className="my-6">
                  <Divider plain>
                    <Typography.Text type="secondary">{t("login.oauth2.divider")}</Typography.Text>
                  </Divider>
                </div>

                <Space direction="vertical" className="w-full" size="middle">
                  {oauth2Providers.map((p) => (
                    <Button
                      key={p.name}
                      block
                      size="large"
                      icon={renderOAuth2ProviderIcon(p.name) || undefined}
                      onClick={() => startOAuth2Login(p.name, window.location.pathname)}
                    >
                      {t("login.oauth2.sign_in_with", { provider: p.displayName || p.name })}
                    </Button>
                  ))}
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

export default Login;
