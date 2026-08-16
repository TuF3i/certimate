import { COLLECTION_NAME_USER, getPocketBase } from "./_pocketbase";

const pb = getPocketBase();
const pbco = pb.collection(COLLECTION_NAME_USER);

export type UserRole = "user" | "admin";

export interface UserModel extends BaseModel {
  email: string;
  name?: string;
  role?: UserRole;
  verified?: boolean;
}

export interface UserCreateData {
  email: string;
  password: string;
  passwordConfirm: string;
  name?: string;
}

export interface WorkflowGrantModel extends BaseModel {
  name: string;
  grantedUsers?: string[];
}

export const authWithPassword = (username: string, password: string) => {
  return pbco.authWithPassword(username, password);
};

export const list = async (): Promise<UserModel[]> => {
  return await pbco.getFullList<UserModel>({
    sort: "created",
  });
};

export const create = async (data: UserCreateData): Promise<UserModel> => {
  return await pbco.create<UserModel>(data);
};

export const updateRole = async (id: string, role: UserRole) => {
  return await pbco.update<UserModel>(id, { role });
};

export const updatePassword = async (id: string, password: string) => {
  return await pbco.update(id, {
    password: password,
    passwordConfirm: password,
  });
};

export const remove = async (id: string) => {
  return await pbco.delete(id);
};

export const isCurrentUser = (id: string) => {
  return pb.authStore?.record?.id === id;
};

// 列出全部工作流的授权信息（管理员视角，可见全部工作流）。
export const listWorkflowGrants = async (): Promise<WorkflowGrantModel[]> => {
  return await pb.collection("workflow").getFullList<WorkflowGrantModel>({
    fields: "id,name,grantedUsers",
    sort: "-created",
  });
};

// 批量更新工作流的授权用户列表。
export const saveWorkflowGrants = async (workflowId: string, grantedUsers: string[]) => {
  return await pb.collection("workflow").update(workflowId, { grantedUsers });
};
