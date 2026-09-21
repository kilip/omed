import { PGlite } from "@electric-sql/pglite";
import { vector } from "@electric-sql/pglite-pgvector";
import { pushSchema } from "drizzle-kit/api-postgres";
import { drizzle as nodeDrizzle } from "drizzle-orm/node-postgres";
import { migrate as nodeMigrate } from "drizzle-orm/node-postgres/migrator";
import { drizzle as pgliteDrizzle } from "drizzle-orm/pglite";
import { schema } from "..";
import { relations } from "../relations";
import type { OmedDatabase } from "../type";

let testClientDB: ReturnType<typeof pgliteDrizzle<typeof relations>> | null =
  null;

export const getTestDB = async (): Promise<OmedDatabase> => {
  if (testClientDB) return testClientDB as unknown as OmedDatabase;

  const pglite = new PGlite({ extensions: { vector } });
  testClientDB = pgliteDrizzle({ client: pglite, relations });

  const { apply } = await pushSchema(schema, testClientDB);
  await apply();

  return testClientDB as unknown as OmedDatabase;
};
