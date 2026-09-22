import { PGlite } from "@electric-sql/pglite";
import { Pool as NeonPool } from "@neondatabase/serverless";
import { drizzle as neonDrizzle } from "drizzle-orm/neon-serverless";
import { drizzle as nodeDrizzle } from "drizzle-orm/node-postgres";
import { drizzle as pgliteDrizzle } from "drizzle-orm/pglite";
import { Pool as NodePool } from "pg";
import { appEnv } from "@/config";
import { relations } from "../relations";
import type { OmedDatabase } from "../type";

export const getDBInstance = (): OmedDatabase => {
  const connectionString = appEnv.DATABASE_URL;

  if (appEnv.DATABASE_DRIVER === "pglite") {
    const client = new PGlite(connectionString);
    return pgliteDrizzle({ client, relations }) as unknown as OmedDatabase;
  }

  if (appEnv.DATABASE_DRIVER === "neon") {
    const client = new NeonPool({ connectionString });
    return neonDrizzle({ client, relations }) as unknown as OmedDatabase;
  }

  const client = new NodePool({ connectionString });
  client.on("error", (err) => {
    console.error(
      "[NodePool] idle client error (swallowed to prevent process crash):",
      {
        code: (err as NodeJS.ErrnoException).code,
        message: err.message,
        stack: err.stack,
      },
    );
  });

  return nodeDrizzle({ client, relations }) as unknown as OmedDatabase;
};
