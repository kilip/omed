import { Pool as NeonPool } from "@neondatabase/serverless";
import { drizzle as neonDrizzle } from "drizzle-orm/neon-serverless";
import {
  type NodePgDatabase,
  drizzle as nodeDrizzle,
} from "drizzle-orm/node-postgres";
import { Pool as NodePool } from "pg";
import { authEnv } from "../env";
import { relations } from "./relations";

export type AuthDatabase = NodePgDatabase<typeof relations>;

let cachedDB: AuthDatabase | null = null;

export const getDBInstance = (): AuthDatabase => {
  if (cachedDB) return cachedDB;

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

export const authDB = getDBInstance();
