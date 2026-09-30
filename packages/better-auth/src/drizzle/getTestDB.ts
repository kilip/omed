import { PGlite } from "@electric-sql/pglite";
import { pushSchema } from "drizzle-kit/api-postgres";
import { drizzle as pgliteDrizzle } from "drizzle-orm/pglite";
import type { AuthDatabase } from "./adapter";
import { relations } from "./relations";
import * as schema from "./schema";

export const getTestDB = async (): Promise<AuthDatabase> => {
  const client = new PGlite();
  const db = pgliteDrizzle({ client, relations });
  const result = await pushSchema(schema, db);
  await result.apply();
  return db as unknown as AuthDatabase;
};
