import { authRelations, financeRelations } from "./schema";

export const relations = {
  ...authRelations,
  ...financeRelations,
};
