import type { ElysiaOpenAPIConfig } from "@elysia/openapi";
import { auth } from "@omed/better-auth";

type Documentation = NonNullable<ElysiaOpenAPIConfig["documentation"]>;
type OpenAPIV3_1Doc = Extract<Documentation, { openapi?: `3.1.${number}` }>;

type Paths = NonNullable<OpenAPIV3_1Doc["paths"]>;
type Components = NonNullable<OpenAPIV3_1Doc["components"]>;

let _schema: ReturnType<typeof auth.api.generateOpenAPISchema> | undefined;
const getSchema = async () => (_schema ??= auth.api.generateOpenAPISchema());

export const OpenAPI = {
	getPaths: async (prefix = ""): Promise<Paths> => {
		const { paths } = await getSchema();
		const reference = Object.create(null) as Record<string, unknown>;
		for (const [path, pathItem] of Object.entries(paths)) {
			if (!pathItem) continue;
			const key = prefix + path;
			const clonedItem = { ...pathItem } as Record<string, { tags?: string[] }>;
			for (const operation of Object.values(clonedItem)) {
				if (operation && typeof operation === "object") {
					operation.tags = ["Better Auth"];
				}
			}
			reference[key] = clonedItem;
		}
		return reference as Paths;
	},
	components: getSchema().then(
		({ components }) => components as unknown as Components,
	),
} as const;
