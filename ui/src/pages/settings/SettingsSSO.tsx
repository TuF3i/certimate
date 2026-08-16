import { useState } from "react";
import { useTranslation } from "react-i18next";
import { CopyOutlined } from "@ant-design/icons";
import { useMount } from "ahooks";
import { App, Button, Card, Divider, Form, Input, Skeleton, Space, Switch, Tag, Typography } from "antd";
import { produce } from "immer";
import { z } from "zod";

import Show from "@/components/Show";
import Tips from "@/components/Tips";
import { type SSOLDAPConfig, type SSOOIDCConfig, type SSOSettingsContent } from "@/domain/settings";
import { useAntdForm, useZustandShallowSelector } from "@/hooks";
import { getSSOConfig } from "@/repository/sso";
import { useSSOSettingsStore } from "@/stores/settings";
import { unwrapErrMsg } from "@/utils/error";

const SettingsSSO = () => {
  const { t } = useTranslation();

  const { message, notification } = App.useApp();

  const { settings, loading, loadSettings, saveSettings } = useSSOSettingsStore(
    useZustandShallowSelector(["settings", "loading", "loadSettings", "saveSettings"])
  );
  useMount(() => loadSettings());

  const [oidcCallback, setOidcCallback] = useState<string>("");

  // 登录页也需要回调地址，这里顺带取一次；取不到不影响设置页功能。
  useMount(() => {
    getSSOConfig()
      .then((resp) => setOidcCallback(resp.oidcCallback))
      .catch(() => void 0);
  });

  const updateContextSettings = async (settings: SSOSettingsContent) => {
    try {
      await saveSettings(settings);
      message.success(t("common.text.operation_succeeded"));
    } catch (err) {
      notification.error({ title: t("common.text.request_error"), description: unwrapErrMsg(err) });
    }
  };

  const handleUpdateOIDC = async (patch: Partial<SSOOIDCConfig>) => {
    const next = produce(settings, (draft) => {
      draft.oidc = { ...(draft.oidc ?? {}), ...patch };
    });
    await updateContextSettings(next);
  };

  const handleUpdateLDAP = async (patch: Partial<SSOLDAPConfig>) => {
    const next = produce(settings, (draft) => {
      draft.ldap = { ...(draft.ldap ?? {}), ...patch };
    });
    await updateContextSettings(next);
  };

  const handleCopyCallback = () => {
    if (!oidcCallback) {
      return;
    }
    navigator.clipboard
      .writeText(oidcCallback)
      .then(() => message.success(t("common.text.copied")))
      .catch(() => void 0);
  };

  return (
    <div>
      <h2>{t("settings.sso.title")}</h2>
      <Tips className="mt-3 md:max-w-5xl" message={t("settings.sso.tips")} />

      <Show when={!loading} fallback={<Skeleton active />}>
        {/* OIDC */}
        <Card className="mt-6 md:max-w-5xl" title={t("settings.sso.oidc.title")}>
          <div className="mb-4">
            <Space>
              <Tag color={settings.oidc?.enabled ? "success" : "default"}>
                {settings.oidc?.enabled ? t("settings.sso.enabled") : t("settings.sso.disabled")}
              </Tag>
              <Button size="small" icon={<CopyOutlined />} disabled={!oidcCallback} onClick={handleCopyCallback}>
                {t("settings.sso.oidc.copy_callback")}
              </Button>
            </Space>
            {oidcCallback ? (
              <div className="mt-2">
                <Typography.Text code>{oidcCallback}</Typography.Text>
              </div>
            ) : null}
          </div>
          <OIDCConfigForm config={settings.oidc} update={handleUpdateOIDC} />
        </Card>

        <Divider className="my-6" />

        {/* LDAP */}
        <Card className="md:max-w-5xl" title={t("settings.sso.ldap.title")}>
          <div className="mb-4">
            <Tag color={settings.ldap?.enabled ? "success" : "default"}>{settings.ldap?.enabled ? t("settings.sso.enabled") : t("settings.sso.disabled")}</Tag>
          </div>
          <LDAPConfigForm config={settings.ldap} update={handleUpdateLDAP} />
        </Card>
      </Show>
    </div>
  );
};

interface OIDCConfigFormProps {
  config: SSOOIDCConfig | null | undefined;
  update: (patch: Partial<SSOOIDCConfig>) => Promise<void>;
}

const OIDCConfigForm = ({ config, update }: OIDCConfigFormProps) => {
  const { t } = useTranslation();

  const _formSchema = z.object({
    enabled: z.boolean().optional(),
    displayName: z.string().max(128).optional(),
    clientId: z.string().min(1).max(256).optional(),
    clientSecret: z.string().max(512).optional(),
    discoveryUrl: z.string().max(1024).optional(),
    scopes: z.string().max(512).optional(),
    autoCreate: z.boolean().optional(),
  });

  const {
    form: formInst,
    formPending,
    formProps,
  } = useAntdForm<z.infer<typeof _formSchema>>({
    initialValues: {
      enabled: config?.enabled,
      displayName: config?.displayName,
      clientId: config?.clientId,
      clientSecret: config?.clientSecret,
      discoveryUrl: config?.discoveryUrl,
      scopes: (config?.scopes ?? []).join(" "),
      autoCreate: config?.autoCreate,
    },
    onSubmit: async (values) => {
      await update({
        enabled: values.enabled,
        displayName: values.displayName || undefined,
        clientId: values.clientId,
        clientSecret: values.clientSecret,
        discoveryUrl: values.discoveryUrl,
        scopes: (values.scopes ?? "")
          .split(/\s+/)
          .map((s: string) => s.trim())
          .filter(Boolean),
        autoCreate: values.autoCreate,
      });
    },
  });

  return (
    <Form {...formProps} form={formInst} disabled={formPending} layout="vertical">
      <Form.Item name="enabled" label={t("settings.sso.form.enabled.label")} valuePropName="checked">
        <Switch />
      </Form.Item>

      <Form.Item name="displayName" label={t("settings.sso.oidc.form.display_name.label")}>
        <Input placeholder="OIDC" />
      </Form.Item>

      <Form.Item name="discoveryUrl" label={t("settings.sso.oidc.form.discovery_url.label")}>
        <Input placeholder="https://auth.example.com/.well-known/openid-configuration" />
      </Form.Item>

      <Form.Item name="clientId" label={t("settings.sso.form.client_id.label")}>
        <Input />
      </Form.Item>

      <Form.Item name="clientSecret" label={t("settings.sso.form.client_secret.label")}>
        <Input.Password />
      </Form.Item>

      <Form.Item name="scopes" label={t("settings.sso.oidc.form.scopes.label")}>
        <Input placeholder="openid email profile" />
      </Form.Item>

      <Divider plain className="my-3" />

      <Form.Item name="autoCreate" label={t("settings.sso.form.auto_create.label")} valuePropName="checked">
        <Switch />
      </Form.Item>

      <Form.Item className="mb-0">
        <Button type="primary" htmlType="submit" loading={formPending}>
          {t("common.button.save")}
        </Button>
      </Form.Item>
    </Form>
  );
};

interface LDAPConfigFormProps {
  config: SSOLDAPConfig | null | undefined;
  update: (patch: Partial<SSOLDAPConfig>) => Promise<void>;
}

const LDAPConfigForm = ({ config, update }: LDAPConfigFormProps) => {
  const { t } = useTranslation();

  const _formSchema = z.object({
    enabled: z.boolean().optional(),
    displayName: z.string().max(128).optional(),
    serverUrl: z.string().min(1).max(512).optional(),
    bindDn: z.string().min(1).max(512).optional(),
    bindPassword: z.string().max(512).optional(),
    searchBase: z.string().min(1).max(512).optional(),
    searchFilter: z.string().max(512).optional(),
    emailAttribute: z.string().max(128).optional(),
    nameAttribute: z.string().max(128).optional(),
    autoCreate: z.boolean().optional(),
  });

  const {
    form: formInst,
    formPending,
    formProps,
  } = useAntdForm<z.infer<typeof _formSchema>>({
    initialValues: {
      enabled: config?.enabled,
      displayName: config?.displayName,
      serverUrl: config?.serverUrl,
      bindDn: config?.bindDn,
      bindPassword: config?.bindPassword,
      searchBase: config?.searchBase,
      searchFilter: config?.searchFilter,
      emailAttribute: config?.emailAttribute,
      nameAttribute: config?.nameAttribute,
      autoCreate: config?.autoCreate,
    },
    onSubmit: async (values) => {
      await update({
        enabled: values.enabled,
        displayName: values.displayName || undefined,
        serverUrl: values.serverUrl,
        bindDn: values.bindDn,
        bindPassword: values.bindPassword,
        searchBase: values.searchBase,
        searchFilter: values.searchFilter,
        emailAttribute: values.emailAttribute || undefined,
        nameAttribute: values.nameAttribute || undefined,
        autoCreate: values.autoCreate,
      });
    },
  });

  return (
    <Form {...formProps} form={formInst} disabled={formPending} layout="vertical">
      <Form.Item name="enabled" label={t("settings.sso.form.enabled.label")} valuePropName="checked">
        <Switch />
      </Form.Item>

      <Form.Item name="displayName" label={t("settings.sso.ldap.form.display_name.label")}>
        <Input placeholder="LDAP" />
      </Form.Item>

      <Form.Item name="serverUrl" label={t("settings.sso.ldap.form.server_url.label")}>
        <Input placeholder="ldap://ldap.example.com:389" />
      </Form.Item>

      <Form.Item name="bindDn" label={t("settings.sso.ldap.form.bind_dn.label")}>
        <Input placeholder="cn=admin,dc=example,dc=com" />
      </Form.Item>

      <Form.Item name="bindPassword" label={t("settings.sso.ldap.form.bind_password.label")}>
        <Input.Password />
      </Form.Item>

      <Form.Item name="searchBase" label={t("settings.sso.ldap.form.search_base.label")}>
        <Input placeholder="ou=people,dc=example,dc=com" />
      </Form.Item>

      <Form.Item name="searchFilter" label={t("settings.sso.ldap.form.search_filter.label")}>
        <Input placeholder="(uid={{username}})" />
      </Form.Item>

      <Space.Compact block>
        <Form.Item name="emailAttribute" label={t("settings.sso.ldap.form.email_attribute.label")} className="flex-1">
          <Input placeholder="mail" />
        </Form.Item>
        <Form.Item name="nameAttribute" label={t("settings.sso.ldap.form.name_attribute.label")} className="flex-1">
          <Input placeholder="displayName" />
        </Form.Item>
      </Space.Compact>

      <Divider plain className="my-3" />

      <Form.Item name="autoCreate" label={t("settings.sso.form.auto_create.label")} valuePropName="checked">
        <Switch />
      </Form.Item>

      <Form.Item className="mb-0">
        <Button type="primary" htmlType="submit" loading={formPending}>
          {t("common.button.save")}
        </Button>
      </Form.Item>
    </Form>
  );
};

export default SettingsSSO;
