import { describe, expect, it } from "vitest";
import { auth } from "./auth";
import type { Session } from "./type";

const context = await auth.$context;
const util = context.test;

describe("Auth", () => {
	it("should create default workspace for user", async () => {
		const user = util.createUser({
			name: "testing",
		});
		const created = await util.saveUser(user);
		expect(created).toBeDefined();

		const data = await util.login({
			userId: created.id,
		});

		const session = data.session as Session;
		expect(session).toBeDefined();
		expect(session?.activeTeamId).not.toBeNull();
	});
});

describe("User locale", () => {
	async function loginNewUser() {
		const created = await util.saveUser(util.createUser({ name: "locale" }));
		const { headers } = await util.login({ userId: created.id });
		return { created, headers };
	}

	it("defaults to null for a new user", async () => {
		const { headers } = await loginNewUser();
		const res = await auth.api.getSession({ headers });
		expect(res?.user.locale ?? null).toBeNull();
	});

	it("can be updated to a supported locale", async () => {
		const { headers } = await loginNewUser();
		await auth.api.updateUser({ headers, body: { locale: "id" } });
		const res = await auth.api.getSession({
			headers,
			query: { disableCookieCache: true },
		});
		expect(res?.user.locale).toBe("id");
	});

	it("rejects an unsupported locale", async () => {
		const { headers } = await loginNewUser();
		await expect(
			auth.api.updateUser({ headers, body: { locale: "fr" } }),
		).rejects.toMatchObject({ status: "BAD_REQUEST" });
	});
});

describe("User defaultCurrency", () => {
	async function loginNewUser() {
		const created = await util.saveUser(util.createUser({ name: "currency" }));
		const { headers } = await util.login({ userId: created.id });
		return { created, headers };
	}

	it("can be updated to a supported currency", async () => {
		const { headers } = await loginNewUser();
		await auth.api.updateUser({ headers, body: { defaultCurrency: "USD" } });
		const res = await auth.api.getSession({
			headers,
			query: { disableCookieCache: true },
		});
		expect(res?.user.defaultCurrency).toBe("USD");
	});

	it("rejects an unsupported currency", async () => {
		const { headers } = await loginNewUser();
		await expect(
			auth.api.updateUser({ headers, body: { defaultCurrency: "INVALID" } }),
		).rejects.toMatchObject({ status: "BAD_REQUEST" });
	});
});
