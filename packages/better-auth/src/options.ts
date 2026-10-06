import { drizzleAdapter } from "@better-auth/drizzle-adapter/relations-v2";
import type { BetterAuthOptions } from "better-auth";
import {
	admin,
	jwt,
	openAPI,
	organization,
	testUtils,
} from "better-auth/plugins";
import { authEnv } from "./authEnv";
import { authDB, schema } from "./drizzle";
import { currencySchema, localeSchema } from "./schema";

export const authOptions = {
	baseURL: authEnv.AUTH_URL,
	basePath: "",
	secret: authEnv.AUTH_SECRET,
	trustedOrigins: authEnv.AUTH_TRUSTED_ORIGINS,
	database: drizzleAdapter(authDB, {
		provider: "pg",
		schema,
		schemaName: "auth",
	}),
	plugins: [
		admin(),
		organization({
			teams: {
				enabled: true,
			},
			schema: {
				organization: {
					additionalFields: {
						personal: {
							type: "boolean",
							default: false,
							input: true,
						},
					},
				},
				team: {
					additionalFields: {
						personal: {
							type: "boolean",
							default: false,
							input: true,
						},
					},
				},
			},
		}),
		jwt({
			jwt: {
				definePayload({ user, session }) {
					const sx = session as unknown as Record<string, unknown>;
					return {
						id: user.id,
						name: user.name,
						avatar: user.image,
						activeWorkspaceId: sx.activeWorkspaceId,
						activeWorkspaceName: sx.activeWorkspaceName,
						activeWorkspaceRoles: sx.activeWorkspaceRoles,
					};
				},
			},
		}),
		openAPI(),
		...(authEnv.DEVELOPMENT ? [testUtils()] : []),
	],
	user: {
		additionalFields: {
			locale: {
				type: "string",
				required: false,
				input: true,
				returned: true,
				validator: { input: localeSchema },
			},
			defaultCurrency: {
				type: "string",
				required: false,
				input: true,
				returned: true,
				defaultValue: "IDR",
				validator: { input: currencySchema },
			},
		},
	},
	session: {
		additionalFields: {
			activeWorkspaceId: { type: "string", required: false, input: false },
			activeWorkspaceName: { type: "string", required: false, input: false },
			activeWorkspaceRoles: { type: "string[]", required: false, input: false },
		},
	},
	advanced: {
		database: {
			generateId: "uuid",
		},
	},
} satisfies BetterAuthOptions;
