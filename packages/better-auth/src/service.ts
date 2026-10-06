import { randomBytes } from "node:crypto";
import type {
	ActiveTeam,
	ActiveWorkspace,
	AuthContext,
	AuthType,
	Organization,
	User,
} from "./type";

const suffix = () => randomBytes(3).toString("hex"); // 6 char

function slugify(s: string) {
	return s
		.toLowerCase()
		.normalize("NFKD")
		.replace(/[\u0300-\u036f]/g, "") // buang diakritik
		.replace(/[^a-z0-9]+/g, "-")
		.slice(0, 40)
		.replace(/^-+|-+$/g, ""); // trim setelah slice
}

export function createSlug(name: string | null | undefined, fallback = "ws") {
	const base = slugify(name ?? "") || fallback;
	return `${base}-${suffix()}`;
}

const isSlugTaken = (e: unknown) =>
	/slug/i.test(e instanceof Error ? e.message : String(e));

export class AuthService {
	constructor(
		private readonly db: AuthContext["adapter"],
		private readonly internal: AuthContext["internalAdapter"],
		private readonly api: AuthType["api"],
	) {}

	async createWorkspace(user: User) {
		const org = await this.createOrg(user);
		const team = await this.db.create({
			model: "team",
			data: {
				organizationId: org.id,
				name: "Personal Workspace",
				createdAt: new Date(),
				personal: true,
			},
		});
		await this.db.create({
			model: "teamMember",
			data: {
				teamId: team.id,
				organizationId: org.id,
				userId: user.id,
				createdAt: new Date(),
			},
		});
		if (!team) throw new Error("Default team not created");

		await this.internal.updateUser(user.id, { activeWorkspace: team.id });
	}

	private async createOrg(user: User, tries = 3): Promise<Organization> {
		for (let i = 0; ; i++) {
			try {
				return await this.api.createOrganization({
					body: {
						name: `${user.name}'s Organization`,
						slug: createSlug(user.name, "org"),
						userId: user.id,
						personal: true,
					},
				});
			} catch (e) {
				if (i >= tries - 1 || !isSlugTaken(e)) throw e;
			}
		}
	}

	async findActiveTeam(userId: string): Promise<ActiveWorkspace | null> {
		const team = await this.findTeam(userId);
		if (!team) return null;

		const member = await this.db.findOne<{ role: string }>({
			model: "member",
			where: [
				{ field: "organizationId", value: team.organizationId },
				{ field: "userId", value: userId },
			],
		});
		if (!member) return null;

		return {
			...team,
			activeWorkspaceId: team.id,
			activeWorkspaceName: team.name,
			activeWorkspaceRoles: member.role.split(/[\s,]+/).filter(Boolean),
		};
	}

	private async findTeam(userId: string): Promise<ActiveTeam | null> {
		// 1. last active team, kalau masih member
		const user = await this.internal.findUserById(userId);
		const activeWorkspace = (user as { activeWorkspace?: string } | null)
			?.activeWorkspace;

		if (activeWorkspace) {
			const membership = await this.db.findOne({
				model: "teamMember",
				where: [
					{ field: "teamId", value: activeWorkspace },
					{ field: "userId", value: userId },
				],
			});
			if (membership) {
				const team = await this.db.findOne<ActiveTeam>({
					model: "team",
					where: [{ field: "id", value: activeWorkspace }],
				});
				if (team) return team;
			}
		}

		// 2. fallback: personal team milik user ini
		const memberships = await this.db.findMany<{ teamId: string }>({
			model: "teamMember",
			where: [{ field: "userId", value: userId }],
		});
		if (memberships.length === 0) return null;

		return this.db.findOne<ActiveTeam>({
			model: "team",
			where: [
				{
					field: "id",
					operator: "in",
					value: memberships.map((m) => m.teamId),
				},
				{ field: "personal", value: true },
			],
		});
	}
}
