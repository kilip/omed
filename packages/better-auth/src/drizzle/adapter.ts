import { neon } from "@neondatabase/serverless";
import { drizzle as neonDrizzle } from "drizzle-orm/neon-http";
import {
  type NodePgDatabase,
  drizzle as nodeDrizzle,
} from "drizzle-orm/node-postgres";
import { Pool as NodePool } from "pg";
import { authEnv } from "../authEnv";
import { relations } from "./relations";

export type AuthDatabase = NodePgDatabase<typeof relations>;

let cachedDB: AuthDatabase | null = null;

export const getDBInstance = async (): Promise<AuthDatabase> => {
  if (cachedDB) return cachedDB;

  if (process.env.NODE_ENV === "test") {
    const { getTestDB } = await import("./getTestDB");
    cachedDB = await getTestDB();
    return cachedDB;
  }

  const connectionString = authEnv.AUTH_DB_URL;
  if (authEnv.AUTH_DB_DRIVER === "node") {
    const client = new NodePool({
      connectionString,
      max: 10,
      maxUses: 1,
      allowExitOnIdle: true,
      idleTimeoutMillis: 1000,
    });
    cachedDB = nodeDrizzle({ client, relations });
    return cachedDB;
  }

  const client = neon(connectionString);
  cachedDB = neonDrizzle({ client, relations }) as unknown as AuthDatabase;
  return cachedDB;
};

export const authDB = await getDBInstance();
