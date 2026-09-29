import { PGlite } from "@electric-sql/pglite";
import { Pool as NeonPool } from "@neondatabase/serverless";
import { pushSchema } from "drizzle-kit/api-postgres";
import { drizzle as neonDrizzle } from "drizzle-orm/neon-serverless";
import {
  type NodePgDatabase,
  drizzle as nodeDrizzle,
} from "drizzle-orm/node-postgres";
import { drizzle as pgliteDrizzle } from "drizzle-orm/pglite";
import { Pool as NodePool } from "pg";
import { authEnv } from "../config";
import { relations } from "./relations";
import * as schema from "./schema";
export type AuthDatabase = NodePgDatabase<typeof relations>;

let cachedDB: AuthDatabase | null = null;

export const getDBForTest = async () => {
  const client = new PGlite();
  const db = pgliteDrizzle({ client, relations });
  const result = await pushSchema(schema, db);
  result.apply();
  return db as unknown as AuthDatabase;
};

export const getDBInstance = async (): Promise<AuthDatabase> => {
  if (cachedDB) return cachedDB;

  if (process.env.NODE_ENV === "test") {
    cachedDB = await getDBForTest();
    return cachedDB;
  }

  const connectionString = authEnv.AUTH_DB_URL;
  if (authEnv.AUTH_DB_DRIVER === "node") {
    const client = new NodePool({ connectionString });
    cachedDB = nodeDrizzle({ client, relations });

    return cachedDB;
  }

  const client = new NeonPool({ connectionString });
  cachedDB = neonDrizzle({ client, relations }) as unknown as AuthDatabase;

  return cachedDB;
};

export const authDB = await getDBInstance();
