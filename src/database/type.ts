import type { NeonDatabase } from "drizzle-orm/neon-serverless";
import type { relations } from "./relations";

export type OmedDatabase = NeonDatabase<typeof relations>;
