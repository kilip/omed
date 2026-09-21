import { Pool as NeonPool, neonConfig } from "@neondatabase/serverless";
import { drizzle as neonDrizzle } from "drizzle-orm/neon-serverless";
import { drizzle as nodeDrizzle } from "drizzle-orm/node-postgres";
import { Pool as NodePool } from "pg";
import ws from "ws";
import { appEnv } from "@/config";
import type { OmedDatabase } from "../type";

export const getDBInstance = (): OmedDatabase => {
  if (process.env.NODE_ENV === "test") return {} as OmedDatabase;

  const connectionString = appEnv.OMED_DB_URL;

  if (!connectionString) {
    throw new Error(
      `You are try to use database, but "OMED_DB_URL" is not set correctly`,
    );
  }

  const statementTimeout = appEnv.OMED_DB_STATEMENT_TIMEOUT;
  const timeoutConfig = statementTimeout
    ? {
        idle_in_transaction_session_timeout: statementTimeout,
        statement_timeout: statementTimeout,
      }
    : {};

  if (appEnv.OMED_DB_DRIVER === "node") {
    const client = new NodePool({ connectionString, ...timeoutConfig });
    // pg.Pool emits 'error' on idle clients when the backend connection drops.
    // Without a listener Node escalates it to uncaughtException and exits the process.
    // See: https://node-postgres.com/apis/pool#error
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
    return nodeDrizzle({ client });
  }

  if (process.env.MIGRATION_DB === "1") {
    neonConfig.webSocketConstructor = ws;
  }
  const client = new NeonPool({ connectionString, ...timeoutConfig });
  // NeonPool runs over WebSocket; transient drops surface as 'error' on the pool.
  // Without a listener Node escalates it to uncaughtException — on Vercel this killed
  // the entire Lambda 1800+ times in 5 minutes (see ).
  client.on("error", (err: Error) => {
    console.error(
      "[NeonPool] idle client error (swallowed to prevent process crash):",
      {
        code: (err as NodeJS.ErrnoException).code,
        message: err.message,
        stack: err.stack,
      },
    );
  });
  return neonDrizzle({ client });
};
