import { COLLECTION_NAME_USER, getPocketBase } from "./_pocketbase";

const pb = getPocketBase();
const pbco = pb.collection(COLLECTION_NAME_USER);

export interface UserModel extends BaseModel {
  email: string;
  name?: string;
  verified?: boolean;
}

export interface UserCreateData {
  email: string;
  password: string;
  passwordConfirm: string;
  name?: string;
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
