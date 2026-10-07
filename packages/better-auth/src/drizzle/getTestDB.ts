import type { AuthDatabase } from "./adapter";
import { relations } from "./relations";
import * as schema from "./schema";

export const getTestDB = async (): Promise<AuthDatabase> => {
	const [{ PGlite }, { pushSchema }, { drizzle }] = await Promise.all([
		import("@electric-sql/pglite"),
		import("drizzle-kit/api-postgres"),
		import("drizzle-orm/pglite"),
	]);

	const client = new PGlite();
	const db = drizzle({ client, relations });
	const result = await pushSchema(schema, db);
	await result.apply();
	return db as unknown as AuthDatabase;
};
