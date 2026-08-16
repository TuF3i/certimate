import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useMount } from "ahooks";
import { App, Button, Form, Input, Modal, Popconfirm, Skeleton, Space, Table, Tag, Transfer } from "antd";
import { createSchemaFieldRule } from "antd-zod";
import { z } from "zod";

import Show from "@/components/Show";
import Tips from "@/components/Tips";
import { useAntdForm } from "@/hooks";
import { getAuthStore, save as saveAdmin } from "@/repository/admin";
import { authWithPassword as authAdminWithPassword } from "@/repository/admin";
import {
  type UserModel,
  type UserRole,
  authWithPassword as authUserWithPassword,
  create,
  isCurrentUser,
  list,
  listWorkflowGrants,
  remove,
  saveWorkflowGrants,
  updatePassword,
  updateRole,
} from "@/repository/user";
import { unwrapErrMsg } from "@/utils/error";

const SettingsUsers = () => {
  const { t } = useTranslation();

  const { message, notification } = App.useApp();

  const [loading, setLoading] = useState(false);
  const [users, setUsers] = useState<UserModel[]>([]);
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [changeMyPasswordOpen, setChangeMyPasswordOpen] = useState(false);
  const [resetUser, setResetUser] = useState<UserModel | null>(null);
  const [grantUser, setGrantUser] = useState<UserModel | null>(null);

  // 当前登录管理员（可能为超级管理员，不在 users 集合内）
  const auth = getAuthStore();
  const isSuperuser = auth.isSuperuser;

  const loadUsers = async () => {
    setLoading(true);
    try {
      const memberList = await list();
      // 超级管理员不在 users 集合中，合并进列表并标记当前登录。
      let merged: UserModel[] = memberList;
      if (isSuperuser) {
        const authRecord = auth.record;
        if (authRecord && !memberList.some((u) => u.id === authRecord.id)) {
          merged = [
            {
              id: authRecord.id,
              email: authRecord.email ?? "",
              name: authRecord.name,
              role: "admin",
              verified: true,
            } as UserModel,
            ...memberList,
          ];
        }
      }
      setUsers(merged);
    } catch (err) {
      notification.error({ title: t("common.text.request_error"), description: unwrapErrMsg(err) });
    } finally {
      setLoading(false);
    }
  };
  useMount(() => loadUsers());

  const handleCreateUser = async (data: { email: string; password: string; name?: string }) => {
    try {
      await create({
        email: data.email,
        password: data.password,
        passwordConfirm: data.password,
        name: data.name,
      });
      setCreateModalOpen(false);
      message.success(t("common.text.operation_succeeded"));
      await loadUsers();
    } catch (err) {
      notification.error({ title: t("common.text.request_error"), description: unwrapErrMsg(err) });
    }
  };

  const handleToggleRole = async (user: UserModel) => {
    const newRole: UserRole = user.role === "admin" ? "user" : "admin";
    try {
      await updateRole(user.id, newRole);
      message.success(t("common.text.operation_succeeded"));
      await loadUsers();
    } catch (err) {
      notification.error({ title: t("common.text.request_error"), description: unwrapErrMsg(err) });
    }
  };

  const handleResetPassword = async (user: UserModel, password: string) => {
    try {
      await updatePassword(user.id, password);
      setResetUser(null);
      message.success(t("common.text.operation_succeeded"));
    } catch (err) {
      notification.error({ title: t("common.text.request_error"), description: unwrapErrMsg(err) });
    }
  };

  // 当前登录管理员修改自己的密码：先验证旧密码。
  const handleChangeMyPassword = async (values: { oldPassword: string; newPassword: string }) => {
    try {
      if (isSuperuser) {
        await authAdminWithPassword(auth.record!.email, values.oldPassword);
        await saveAdmin({ password: values.newPassword, passwordConfirm: values.newPassword });
      } else {
        await authUserWithPassword(auth.record!.email, values.oldPassword);
        await updatePassword(auth.record!.id, values.newPassword);
      }
      setChangeMyPasswordOpen(false);
      message.success(t("common.text.operation_succeeded"));
    } catch (err) {
      notification.error({ title: t("common.text.request_error"), description: unwrapErrMsg(err) });
    }
  };

  const handleDeleteUser = async (user: UserModel) => {
    try {
      await remove(user.id);
      message.success(t("common.text.operation_succeeded"));
      await loadUsers();
    } catch (err) {
      notification.error({ title: t("common.text.request_error"), description: unwrapErrMsg(err) });
    }
  };

  const handleSaveGrants = async (userId: string, workflowIds: string[]) => {
    try {
      const workflows = await listWorkflowGrants();
      for (const workflow of workflows) {
        const has = (workflow.grantedUsers ?? []).includes(userId);
        const want = workflowIds.includes(workflow.id);
        if (has !== want) {
          const grantedUsers = (workflow.grantedUsers ?? []).filter((id) => id !== userId);
          if (want) {
            grantedUsers.push(userId);
          }
          await saveWorkflowGrants(workflow.id, grantedUsers);
        }
      }
      message.success(t("common.text.operation_succeeded"));
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
        <h3>{t("settings.users.members.title")}</h3>
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
                render: (value: string, record) => (
                  <Space size="small">
                    <span>{value}</span>
                    {isCurrentUser(record.id) ? <Tag color="blue">{t("settings.users.table.current")}</Tag> : null}
                  </Space>
                ),
              },
              {
                title: t("settings.users.table.name"),
                dataIndex: "name",
                render: (value: string) => value || "-",
              },
              {
                title: t("settings.users.table.role"),
                dataIndex: "role",
                render: (value: UserRole) =>
                  value === "admin" ? <Tag color="gold">{t("settings.users.table.role_admin")}</Tag> : <Tag>{t("settings.users.table.role_user")}</Tag>,
              },
              {
                title: t("settings.users.table.actions"),
                width: 380,
                render: (_value, record) => (
                  <Space size="small">
                    <Button size="small" disabled={isCurrentUser(record.id)} onClick={() => setGrantUser(record)}>
                      {t("settings.users.table.grant")}
                    </Button>
                    {isCurrentUser(record.id) ? (
                      <Button size="small" onClick={() => setChangeMyPasswordOpen(true)}>
                        {t("settings.users.table.change_password")}
                      </Button>
                    ) : (
                      <Button size="small" onClick={() => setResetUser(record)}>
                        {t("settings.users.table.reset_password")}
                      </Button>
                    )}
                    <Button size="small" disabled={isCurrentUser(record.id)} onClick={() => handleToggleRole(record)}>
                      {record.role === "admin" ? t("settings.users.table.demote") : t("settings.users.table.promote")}
                    </Button>
                    <Popconfirm
                      title={t("settings.users.table.delete_confirm")}
                      okText={t("common.button.confirm")}
                      cancelText={t("common.button.cancel")}
                      onConfirm={() => handleDeleteUser(record)}
                    >
                      <Button size="small" danger disabled={isCurrentUser(record.id)}>
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
      <ChangeMyPasswordModal open={changeMyPasswordOpen} onClose={() => setChangeMyPasswordOpen(false)} onSubmit={handleChangeMyPassword} />
      <ResetPasswordModal user={resetUser} onClose={() => setResetUser(null)} onSubmit={handleResetPassword} />
      <GrantWorkflowsModal user={grantUser} onClose={() => setGrantUser(null)} onSave={handleSaveGrants} />
    </>
  );
};

// 当前登录账号修改自己的密码：验证旧密码（superuser 走 _superusers，成员走 users）。
interface ChangeMyPasswordModalProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (values: { oldPassword: string; newPassword: string }) => Promise<void>;
}

const ChangeMyPasswordModal = ({ open, onClose, onSubmit }: ChangeMyPasswordModalProps) => {
  const { t } = useTranslation();

  const formSchema = z
    .object({
      oldPassword: z.string().min(10).max(256),
      newPassword: z.string().min(10).max(256),
      confirmPassword: z.string().min(10).max(256),
    })
    .refine((values) => values.newPassword === values.confirmPassword, {
      error: t("settings.users.my_account.password.form.confirm.errmsg.not_matched"),
      path: ["confirmPassword"],
    });
  const formRule = createSchemaFieldRule(formSchema);
  const {
    form: formInst,
    formPending,
    formProps,
  } = useAntdForm<z.infer<typeof formSchema>>({
    initialValues: {
      oldPassword: "",
      newPassword: "",
      confirmPassword: "",
    },
    onSubmit: async (values) => {
      await onSubmit({
        oldPassword: values.oldPassword,
        newPassword: values.newPassword,
      });
      formInst.resetFields();
    },
  });

  return (
    <Modal
      open={open}
      title={t("settings.users.table.change_password")}
      footer={null}
      onCancel={() => {
        formInst.resetFields();
        onClose();
      }}
    >
      <Form {...formProps} form={formInst} layout="vertical" validateTrigger="onBlur">
        <Form.Item name="oldPassword" label={t("settings.users.my_account.password.form.old_password.label")} rules={[formRule]}>
          <Input.Password autoFocus placeholder={t("settings.users.my_account.password.form.old_password.placeholder")} />
        </Form.Item>
        <Form.Item name="newPassword" label={t("settings.users.my_account.password.form.new_password.label")} rules={[formRule]}>
          <Input.Password placeholder={t("settings.users.my_account.password.form.new_password.placeholder")} />
        </Form.Item>
        <Form.Item name="confirmPassword" label={t("settings.users.my_account.password.form.confirm.label")} rules={[formRule]}>
          <Input.Password placeholder={t("settings.users.my_account.password.form.confirm.placeholder")} />
        </Form.Item>
        <Form.Item className="mb-0">
          <Button block type="primary" htmlType="submit" loading={formPending}>
            {t("common.button.save")}
          </Button>
        </Form.Item>
      </Form>
    </Modal>
  );
};

interface CreateUserModalProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (data: { email: string; password: string; name?: string }) => Promise<void>;
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
  user: UserModel | null;
  onClose: () => void;
  onSubmit: (user: UserModel, password: string) => Promise<void>;
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

interface GrantWorkflowsModalProps {
  user: UserModel | null;
  onClose: () => void;
  onSave: (userId: string, workflowIds: string[]) => Promise<void>;
}

// 工作流授权：勾选该用户可访问的工作流。
const GrantWorkflowsModal = ({ user, onClose, onSave }: GrantWorkflowsModalProps) => {
  const { t } = useTranslation();

  const { notification } = App.useApp();

  const [loading, setLoading] = useState(false);
  const [workflows, setWorkflows] = useState<{ id: string; name: string; grantedUsers?: string[] }[]>([]);
  const [targetKeys, setTargetKeys] = useState<string[]>([]);

  useEffect(() => {
    if (!user) {
      return;
    }
    setLoading(true);
    listWorkflowGrants()
      .then((list) => {
        setWorkflows(list);
        setTargetKeys(list.filter((w) => (w.grantedUsers ?? []).includes(user.id)).map((w) => w.id));
      })
      .catch((err) => {
        notification.error({ title: t("common.text.request_error"), description: unwrapErrMsg(err) });
      })
      .finally(() => setLoading(false));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user]);

  const dataSource = useMemo(() => {
    return workflows.map((w) => ({ key: w.id, title: w.name }));
  }, [workflows]);

  const handleSave = async () => {
    if (!user) {
      return;
    }
    await onSave(user.id, targetKeys);
    onClose();
  };

  return (
    <Modal
      open={!!user}
      title={t("settings.users.grant.title", { email: user?.email ?? "" })}
      footer={
        <Space>
          <Button onClick={onClose}>{t("common.button.cancel")}</Button>
          <Button type="primary" loading={loading} onClick={handleSave}>
            {t("common.button.save")}
          </Button>
        </Space>
      }
      onCancel={onClose}
    >
      <Show when={!loading} fallback={<Skeleton active />}>
        <Transfer
          dataSource={dataSource}
          targetKeys={targetKeys}
          onChange={(next) => setTargetKeys(next as string[])}
          render={(item) => item.title}
          titles={[t("settings.users.grant.unassigned"), t("settings.users.grant.assigned")]}
          listStyle={{ width: "100%", minHeight: 320 }}
        />
      </Show>
    </Modal>
  );
};

export default SettingsUsers;
