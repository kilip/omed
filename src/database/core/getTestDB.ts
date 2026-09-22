import { PGlite } from "@electric-sql/pglite";
import { vector } from "@electric-sql/pglite-pgvector";
import { pushSchema } from "drizzle-kit/api-postgres";
import { drizzle as pgliteDrizzle } from "drizzle-orm/pglite";
import { relations } from "../relations";
import * as schema from "../schema";
import type { OmedDatabase } from "../type";

let testDB: OmedDatabase | null = null;

export const getTestDB = async (): Promise<OmedDatabase> => {
  if (testDB) return testDB;

  const client = new PGlite({ extensions: { vector }, dataDir: "./.pglite" });
  testDB = pgliteDrizzle({ client, relations }) as unknown as OmedDatabase;

  const result = await pushSchema(schema, testDB);
  await result.apply();

  return testDB;
};
