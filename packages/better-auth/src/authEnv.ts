import { createEnv } from "@t3-oss/env-core";
import z from "zod";

export const authEnvConfig = () => {
	return createEnv({
		server: {
			AUTH_URL: z.url().default("http://localhost:8001"),
			AUTH_DB_DRIVER: z.enum(["node", "neon"]).default("node"),
			AUTH_DB_URL: z.string().default("postgresql://omed:omed@db:5432/omed"),
      AUTH_TRUSTED_ORIGINS: z.string().default("http://localhost:3001 http://localhost:8001").transform((value) => value.split(" ")),
			AUTH_SECRET: z.string().min(1, "AUTH_SECRET is required"),
			AUTH_GOOGLE_ID: z.string().optional(),
			AUTH_GOOGLE_SECRET: z.string().optional(),
		},
		runtimeEnv: process.env,
	});
};

export const authEnv = authEnvConfig();
