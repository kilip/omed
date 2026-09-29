import { randomBytes } from "node:crypto";
import type { Auth, DBAdapter, InternalAdapter } from "better-auth";
import type { Organization } from "better-auth/plugins";
import type { AuthType, BaseOptions } from "./auth.options";
import type { ActiveTeam, Session, User } from "./type";

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
    private readonly db: DBAdapter<BaseOptions>,
    private readonly internalDb: InternalAdapter<BaseOptions>,
    private readonly api: AuthType["api"],
  ) {}

  async createPersonalWorkspace(u: unknown) {
    const user = u as User;
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

    await this.internalDb.updateUser(user.id, { activeWorkspace: team.id });
  }

  async findActiveTeam(userId: string): Promise<ActiveTeam | null> {
    const isMember = (teamId: string) =>
      this.db.findOne({
        model: "teamMember",
        where: [
          { field: "teamId", value: teamId },
          { field: "userId", value: userId },
        ],
      });

    // 1. last team, kalau masih member
    const user = await this.internalDb.findUserById(userId);
    const activeWorkspace = (user as { activeWorkspace?: string } | null)
      ?.activeWorkspace;

    if (activeWorkspace && (await isMember(activeWorkspace))) {
      const team = await this.db.findOne<ActiveTeam>({
        model: "team",
        where: [{ field: "id", value: activeWorkspace }],
      });
      console.log(team);
      if (team) return team;
    }

    // 2. fallback: personal team
    const personal = await this.db.findOne<ActiveTeam>({
      model: "team",
      where: [{ field: "personal", value: true }],
    });

    return personal && (await isMember(personal.id)) ? personal : null;
  }

  async initSessionData(s: unknown) {
    const session = s as unknown as Session;
    return {
      ...session,
    };
  }

  private async createOrg(user: User, tries = 3) {
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
}
