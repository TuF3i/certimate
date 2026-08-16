import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useMount } from "ahooks";
import { App, Button, Form, Input, Modal, Popconfirm, Skeleton, Space, Table, Tag } from "antd";
import { createSchemaFieldRule } from "antd-zod";
import { z } from "zod";

import Show from "@/components/Show";
import Tips from "@/components/Tips";
import { useAntdForm } from "@/hooks";
import * as userRepo from "@/repository/user";
import { unwrapErrMsg } from "@/utils/error";

const SettingsUsers = () => {
  const { t } = useTranslation();

  const { message, notification } = App.useApp();

  const [loading, setLoading] = useState(false);
  const [users, setUsers] = useState<userRepo.UserModel[]>([]);
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [resetUser, setResetUser] = useState<userRepo.UserModel | null>(null);

  const loadUsers = async () => {
    setLoading(true);
    try {
      setUsers(await userRepo.list());
    } catch (err) {
      notification.error({ title: t("common.text.request_error"), description: unwrapErrMsg(err) });
    } finally {
      setLoading(false);
    }
  };
  useMount(() => loadUsers());

  const handleCreateUser = async (data: userRepo.UserCreateData) => {
    try {
      await userRepo.create(data);
      setCreateModalOpen(false);
      message.success(t("common.text.operation_succeeded"));
      await loadUsers();
    } catch (err) {
      notification.error({ title: t("common.text.request_error"), description: unwrapErrMsg(err) });
    }
  };

  const handleResetPassword = async (user: userRepo.UserModel, password: string) => {
    try {
      await userRepo.updatePassword(user.id, password);
      setResetUser(null);
      message.success(t("common.text.operation_succeeded"));
    } catch (err) {
      notification.error({ title: t("common.text.request_error"), description: unwrapErrMsg(err) });
    }
  };

  const handleDeleteUser = async (user: userRepo.UserModel) => {
    try {
      await userRepo.remove(user.id);
      message.success(t("common.text.operation_succeeded"));
      await loadUsers();
    } catch (err) {
      notification.error({ title: t("common.text.request_error"), description: unwrapErrMsg(err) });
    }
  };

  return (
    <>
      <div className="flex items-center justify-between">
        <h2 className="mb-0">{t("settings.users.title")}</h2>
        <Button type="primary" onClick={() => setCreateModalOpen(true)}>
          {t("settings.users.create.button")}
        </Button>
      </div>

      <Tips className="mt-3 md:max-w-5xl" message={t("settings.users.tips")} />

      <div className="mt-6">
        <Show when={!loading} fallback={<Skeleton active />}>
          <Table
            className="md:max-w-5xl"
            rowKey="id"
            dataSource={users}
            pagination={false}
            columns={[
              {
                title: t("settings.users.table.email"),
                dataIndex: "email",
              },
              {
                title: t("settings.users.table.name"),
                dataIndex: "name",
                render: (value: string) => value || "-",
              },
              {
                title: t("settings.users.table.verified"),
                dataIndex: "verified",
                render: (value: boolean) =>
                  value ? <Tag color="success">{t("settings.users.table.verified_yes")}</Tag> : <Tag>{t("settings.users.table.verified_no")}</Tag>,
              },
              {
                title: t("settings.users.table.actions"),
                width: 220,
                render: (_value, record) => (
                  <Space size="small">
                    <Button size="small" onClick={() => setResetUser(record)}>
                      {t("settings.users.table.reset_password")}
                    </Button>
                    <Popconfirm
                      title={t("settings.users.table.delete_confirm")}
                      okText={t("common.button.confirm")}
                      cancelText={t("common.button.cancel")}
                      onConfirm={() => handleDeleteUser(record)}
                    >
                      <Button size="small" danger disabled={userRepo.isCurrentUser(record.id)}>
                        {t("settings.users.table.delete")}
                      </Button>
                    </Popconfirm>
                  </Space>
                ),
              },
            ]}
          />
        </Show>
      </div>

      <CreateUserModal open={createModalOpen} onClose={() => setCreateModalOpen(false)} onSubmit={handleCreateUser} />
      <ResetPasswordModal user={resetUser} onClose={() => setResetUser(null)} onSubmit={handleResetPassword} />
    </>
  );
};

interface CreateUserModalProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (data: userRepo.UserCreateData) => Promise<void>;
}

const CreateUserModal = ({ open, onClose, onSubmit }: CreateUserModalProps) => {
  const { t } = useTranslation();

  const formSchema = z.object({
    email: z.email(),
    name: z.string().max(128).optional(),
    password: z.string().min(10).max(256),
  });
  const formRule = createSchemaFieldRule(formSchema);
  const {
    form: formInst,
    formPending,
    formProps,
  } = useAntdForm<z.infer<typeof formSchema>>({
    initialValues: {
      email: "",
      name: "",
      password: "",
    },
    onSubmit: async (values) => {
      await onSubmit({
        email: values.email,
        name: values.name,
        password: values.password,
        passwordConfirm: values.password,
      });
    },
  });

  return (
    <Modal
      open={open}
      title={t("settings.users.create.title")}
      footer={null}
      onCancel={() => {
        formInst.resetFields();
        onClose();
      }}
    >
      <Form {...formProps} form={formInst} layout="vertical" validateTrigger="onBlur">
        <Form.Item name="email" label={t("settings.users.form.email.label")} rules={[formRule]}>
          <Input placeholder={t("settings.users.form.email.placeholder")} />
        </Form.Item>

        <Form.Item name="name" label={t("settings.users.form.name.label")} rules={[formRule]}>
          <Input placeholder={t("settings.users.form.name.placeholder")} />
        </Form.Item>

        <Form.Item
          name="password"
          label={t("settings.users.form.password.label")}
          extra={<span dangerouslySetInnerHTML={{ __html: t("settings.users.form.password.help") }}></span>}
          rules={[formRule]}
        >
          <Input.Password autoComplete="new-password" placeholder={t("settings.users.form.password.placeholder")} />
        </Form.Item>

        <Form.Item className="mb-0">
          <Button block type="primary" htmlType="submit" loading={formPending}>
            {t("settings.users.create.submit")}
          </Button>
        </Form.Item>
      </Form>
    </Modal>
  );
};

interface ResetPasswordModalProps {
  user: userRepo.UserModel | null;
  onClose: () => void;
  onSubmit: (user: userRepo.UserModel, password: string) => Promise<void>;
}

const ResetPasswordModal = ({ user, onClose, onSubmit }: ResetPasswordModalProps) => {
  const { t } = useTranslation();

  const formSchema = z.object({
    password: z.string().min(10).max(256),
  });
  const formRule = createSchemaFieldRule(formSchema);
  const {
    form: formInst,
    formPending,
    formProps,
  } = useAntdForm<z.infer<typeof formSchema>>({
    initialValues: { password: "" },
    onSubmit: async (values) => {
      if (user) {
        await onSubmit(user, values.password);
      }
    },
  });

  return (
    <Modal
      open={!!user}
      title={t("settings.users.reset.title", { email: user?.email ?? "" })}
      footer={null}
      onCancel={() => {
        formInst.resetFields();
        onClose();
      }}
    >
      <Form {...formProps} form={formInst} layout="vertical" validateTrigger="onBlur">
        <Form.Item name="password" label={t("settings.users.form.password.label")} rules={[formRule]}>
          <Input.Password autoComplete="new-password" placeholder={t("settings.users.form.password.placeholder")} />
        </Form.Item>

        <Form.Item className="mb-0">
          <Button block type="primary" htmlType="submit" loading={formPending}>
            {t("settings.users.reset.submit")}
          </Button>
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default SettingsUsers;
