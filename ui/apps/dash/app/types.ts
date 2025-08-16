export type AuthenticatedUser = {
  id: number;
  name: string;
  avatar: string;
  token: string;
};

export type LoggedInResponse = {
  userId: number;
  name: string;
  avatar: string;
  token: string;
};

export type Resource<T> = {
  data: T;
  meta: {
    uri?: string;
  };
};

export type RootOutletContext = {
  user: AuthenticatedUser;
};
