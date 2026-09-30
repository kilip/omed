import { Pool as NeonPool } from "@neondatabase/serverless";
import { drizzle as neonDrizzle } from "drizzle-orm/neon-serverless";
import {
  type NodePgDatabase,
  drizzle as nodeDrizzle,
} from "drizzle-orm/node-postgres";
import { Pool as NodePool } from "pg";
import { authEnv } from "../config";
import { relations } from "./relations";
import * as schema from "./schema";

export type AuthDatabase = NodePgDatabase<typeof relations>;

const cachedDB: AuthDatabase | null = null;

export const getDBForTest = async () => {
  // dynamic import: dev-only deps, jangan ikut ke bundle production
  const [{ PGlite }, { pushSchema }, { drizzle: pgliteDrizzle }] =
    await Promise.all([
      import("@electric-sql/pglite"),
      import("drizzle-kit/api-postgres"),
      import("drizzle-orm/pglite"),
    ]);

  const client = new PGlite();
  const db = pgliteDrizzle({ client, relations });
  const result = await pushSchema(schema, db);
  await result.apply();
  return db as unknown as AuthDatabase;
};

// getDBInstance & authDB tetap sama
