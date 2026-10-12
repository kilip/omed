import { Pool } from "@neondatabase/serverless";
import { drizzle as drizzleNeon } from "drizzle-orm/neon-serverless";
import {
  drizzle as drizzleNode,
  type NodePgDatabase as NodeDatabase,
} from "drizzle-orm/node-postgres";
import { Pool as NodePool } from "pg";
import { authEnv } from "../authEnv";
import type { relations } from "./relations";

type AuthDB = NodeDatabase<typeof relations>;

let cachedDB: AuthDB | null = null;

const getCachedDB = (): AuthDB => {
  if (cachedDB != null) return cachedDB;

  const connectionString = authEnv.OMED_DB_URL;

  if (authEnv.OMED_DB_DRIVER === "node") {
    const client = new NodePool({
      connectionString,
    });
    cachedDB = drizzleNode({
      client,
    });

    return cachedDB;
  }

  const client = new Pool({
    connectionString,
  });
  cachedDB = drizzleNeon({
    client,
  });

  return cachedDB;
};

export const authDb = getCachedDB();
