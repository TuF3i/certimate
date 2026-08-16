import { useTranslation } from "react-i18next";
import { useMount } from "ahooks";
import { App, Button, Collapse, Divider, Form, Input, Skeleton, Space, Switch, Tag, Typography } from "antd";
import { produce } from "immer";
import { z } from "zod";

import Show from "@/components/Show";
import Tips from "@/components/Tips";
import { type OAuth2ProviderConfig, type OAuth2SettingsContent } from "@/domain/settings";
import { useAntdForm, useZustandShallowSelector } from "@/hooks";
import { useOAuth2SettingsStore } from "@/stores/settings";
import { unwrapErrMsg } from "@/utils/error";

// Built-in presets matching the backend `internal/oauth2/providers.go`.
// 注意：自托管提供商（如 authentik）端点需用户显式填写。
const PRESET_PROVIDER_NAMES = ["github", "gitlab", "gitee", "google", "azuread", "dingtalk", "authentik"];

const Settings = () => {
  const { t } = useTranslation();

  const { message, notification } = App.useApp();

  const { settings, loading, loadSettings, saveSettings } = useOAuth2SettingsStore(
    useZustandShallowSelector(["settings", "loading", "loadSettings", "saveSettings"])
  );
  useMount(() => loadSettings());

  const updateContextSettings = async (settings: OAuth2SettingsContent) => {
    try {
      await saveSettings(settings);
      message.success(t("common.text.operation_succeeded"));
    } catch (err) {
      notification.error({ title: t("common.text.request_error"), description: unwrapErrMsg(err) });
    }
  };

  const handleAddPresetProvider = (name: string) => {
    const next = produce(settings, (draft) => {
      draft.providers ??= [];
      if (draft.providers.some((p) => p.name === name)) {
        return;
      }
      draft.providers.push({
        name,
        displayName: undefined,
        enabled: false,
        clientId: "",
        clientSecret: "",
        scopes: [],
        redirectUrl: "",
      });
    });
    updateContextSettings(next);
  };

  const handleRemoveProvider = (idx: number) => {
    const next = produce(settings, (draft) => {
      draft.providers?.splice(idx, 1);
    });
    updateContextSettings(next);
  };

  const handleUpdateProvider = (idx: number, patch: Partial<OAuth2ProviderConfig>) => {
    const next = produce(settings, (draft) => {
      const target = draft.providers?.[idx];
      if (!target) {
        return;
      }
      Object.assign(target, patch);
    });
    updateContextSettings(next);
  };

  return (
    <div>
      <h2>{t("settings.oauth2.providers.title")}</h2>
      <div className="md:max-w-5xl">
        <Space wrap size="middle">
          {PRESET_PROVIDER_NAMES.map((name) => (
            <Button key={name} onClick={() => handleAddPresetProvider(name)}>
              {t("settings.oauth2.providers.add_preset", { provider: name })}
            </Button>
          ))}
        </Space>
        <Tips className="mt-3" message={t("settings.oauth2.providers.tips")} />
      </div>

      <Divider className="my-6" />

      <Show when={!loading} fallback={<Skeleton active />}>
        <Collapse
          className="md:max-w-5xl"
          items={(settings.providers ?? []).map((provider, idx) => ({
            key: provider.name + ":" + idx,
            label: (
              <Space size="small" align="center">
                <Typography.Text>{provider.displayName || provider.name}</Typography.Text>
                <Tag>{provider.name}</Tag>
                <Tag color={provider.enabled ? "success" : "default"}>
                  {provider.enabled ? t("settings.oauth2.providers.enabled") : t("settings.oauth2.providers.disabled")}
                </Tag>
              </Space>
            ),
            extra: (
              <Button
                size="small"
                danger
                onClick={(e: React.MouseEvent) => {
                  e.stopPropagation();
                  handleRemoveProvider(idx);
                }}
              >
                {t("common.button.delete")}
              </Button>
            ),
            children: <ProviderConfigForm key={idx} provider={provider as OAuth2ProviderConfig} update={(patch) => handleUpdateProvider(idx, patch)} />,
          }))}
        />
      </Show>
    </div>
  );
};

interface ProviderConfigFormProps {
  provider: OAuth2ProviderConfig;
  update: (patch: Partial<OAuth2ProviderConfig>) => void;
}

const ProviderConfigForm = ({ provider, update }: ProviderConfigFormProps) => {
  const { t } = useTranslation();

  const _formSchema = z.object({
    displayName: z.string().optional(),
    enabled: z.boolean().optional(),
    clientId: z.string().min(1).max(256).optional(),
    clientSecret: z.string().max(512).optional(),
    redirectUrl: z.string().max(1024).optional(),
    scopes: z.string().optional(),
    authUrl: z.string().max(512).optional(),
    tokenUrl: z.string().max(512).optional(),
    userInfoUrl: z.string().max(512).optional(),
    _subjectField: z.string().max(64).optional(),
    scopesJoin: z.string().max(8).optional(),
    autoCreate: z.boolean().optional(),
    autoCreatePrefix: z.string().max(64).optional(),
  });

  const {
    form: formInst,
    formPending,
    formProps,
  } = useAntdForm<z.infer<typeof _formSchema>>({
    initialValues: {
      displayName: provider.displayName,
      enabled: provider.enabled,
      clientId: provider.clientId,
      clientSecret: provider.clientSecret,
      redirectUrl: provider.redirectUrl,
      scopes: (provider.scopes ?? []).join(" "),
      authUrl: provider.authUrl,
      tokenUrl: provider.tokenUrl,
      userInfoUrl: provider.userInfoUrl,
      _subjectField: provider.subjectField,
      scopesJoin: provider.scopesJoin,
      autoCreate: provider.autoCreate,
      autoCreatePrefix: provider.autoCreatePrefix,
    },
    onSubmit: async (values) => {
      update({
        displayName: values.displayName || undefined,
        enabled: values.enabled,
        clientId: values.clientId,
        clientSecret: values.clientSecret,
        redirectUrl: values.redirectUrl,
        scopes: (values.scopes ?? "")
          .split(/\s+/)
          .map((s: string) => s.trim())
          .filter(Boolean),
        authUrl: values.authUrl || undefined,
        tokenUrl: values.tokenUrl || undefined,
        userInfoUrl: values.userInfoUrl || undefined,
        subjectField: values._subjectField || undefined,
        scopesJoin: values.scopesJoin || undefined,
        autoCreate: values.autoCreate,
        autoCreatePrefix: values.autoCreatePrefix || undefined,
      });
    },
  });

  // submission / mutation 通过 update 回调对外父级Store进行增量更新，
  // 顶层 Tags 会随后续重渲染即时刷新。

  return (
    <Form {...formProps} form={formInst} disabled={formPending} layout="vertical">
      <Form.Item name="enabled" label={t("settings.oauth2.providers.form.enabled.label")} valuePropName="checked">
        <Switch onChange={(v: boolean) => update({ enabled: v })} />
      </Form.Item>

      <Form.Item name="displayName" label={t("settings.oauth2.providers.form.display_name.label")}>
        <Input placeholder="GitHub" />
      </Form.Item>

      <Form.Item name="clientId" label={t("settings.oauth2.providers.form.client_id.label")}>
        <Input />
      </Form.Item>

      <Form.Item name="clientSecret" label={t("settings.oauth2.providers.form.client_secret.label")}>
        <Input.Password />
      </Form.Item>

      <Form.Item name="redirectUrl" label={t("settings.oauth2.providers.form.redirect_url.label")}>
        <Input placeholder="https://your-certimate-host/api/oauth2/callback?provider=github" />
      </Form.Item>

      <Form.Item name="scopes" label={t("settings.oauth2.providers.form.scopes.label")}>
        <Input.TextArea rows={2} placeholder="openid email profile" />
      </Form.Item>

      <Divider plain className="my-3" />

      <Form.Item name="authUrl" label={t("settings.oauth2.providers.form.auth_url.label")}>
        <Input placeholder={t("settings.oauth2.providers.form.auth_url.placeholder")} />
      </Form.Item>

      <Form.Item name="tokenUrl" label={t("settings.oauth2.providers.form.token_url.label")}>
        <Input placeholder={t("settings.oauth2.providers.form.token_url.placeholder")} />
      </Form.Item>

      <Form.Item name="userInfoUrl" label={t("settings.oauth2.providers.form.user_info_url.label")}>
        <Input placeholder={t("settings.oauth2.providers.form.user_info_url.placeholder")} />
      </Form.Item>

      <Form.Item name="_subjectField" label={t("settings.oauth2.providers.form.subject_field.label")}>
        <Input placeholder="id / sub / openId" />
      </Form.Item>

      <Divider plain className="my-3" />

      <Space.Compact block>
        <Form.Item name="autoCreate" label={t("settings.oauth2.providers.form.auto_create.label")} className="flex-1">
          <Switch />
        </Form.Item>
        <Form.Item name="autoCreatePrefix" label={t("settings.oauth2.providers.form.auto_create_prefix.label")} className="flex-1">
          <Input placeholder="oauth2" />
        </Form.Item>
      </Space.Compact>

      <Form.Item className="mb-0">
        <Button type="primary" htmlType="submit" loading={formPending}>
          {t("common.button.save")}
        </Button>
      </Form.Item>
    </Form>
  );
};

export default Settings;
