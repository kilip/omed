import { accountingRelations, authRelations } from "./schema";

export const relations = {
  ...authRelations,
  ...accountingRelations,
};
