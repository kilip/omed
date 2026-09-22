import { authRelations, financeRelations, workspaceRelations } from "./schema";

export const relations = {
  ...authRelations,
  ...financeRelations,
  ...workspaceRelations,
};
