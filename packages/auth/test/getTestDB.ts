import { PGlite } from "@electric-sql/pglite";
import { pushSchema } from "drizzle-kit/api-postgres";
import { drizzle } from "drizzle-orm/pglite";
import type { AuthDatabase } from "../src/drizzle";
import { relations } from "../src/drizzle/relations";

import * as schema from "../src/drizzle/schema";

let cachedTestDB: AuthDatabase | null = null;

async function createTestDB(): Promise<AuthDatabase> {
  if (cachedTestDB) return cachedTestDB;

  const client = new PGlite();
  cachedTestDB = drizzle({ client, relations }) as unknown as AuthDatabase;

  const result = await pushSchema(schema, cachedTestDB);
  await result.apply();

  return cachedTestDB;
}

export const testDB = await createTestDB();
