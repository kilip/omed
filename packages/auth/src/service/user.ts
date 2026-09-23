import {
  type AuthDatabase,
  members,
  organizations,
  teamMembers,
  teams,
} from "../drizzle";
import type { User } from "../type";

export class UserService {
  constructor(private readonly db: AuthDatabase) {}

  async findPersonalOrganization(userId: string) {
    return await this.db.query.organizations.findFirst({
      where: {
        isPersonal: true,
        members: {
          userId,
        },
      },
    });
  }

  async initPersonalOrganization(user: User) {
    const exists = await this.findPersonalOrganization(user.id);
    if (exists) return exists;

    const [newOrg] = await this.db
      .insert(organizations)
      .values({
        name: `${user.name} Organization`,
        slug: `${user.name.toLowerCase().replace(" ", "-")}`,
        createdAt: new Date(),
        isPersonal: true,
      })
      .returning();

    if (!newOrg) {
      throw new Error("failed to create organization");
    }

    // ensure user owned the organization
    await this.db.insert(members).values({
      userId: user.id,
      createdAt: new Date(),
      organizationId: newOrg.id,
      role: "owner",
    });

    return newOrg;
  }

  async findPersonalTeam(userId: string) {
    return await this.db.query.teams.findFirst({
      where: {
        isPersonal: true,
        teamMembers: {
          userId,
        },
      },
    });
  }

  async initPersonalTeam(user: User) {
    const exists = await this.findPersonalTeam(user.id);
    if (exists) return exists;

    const org = await this.initPersonalOrganization(user);

    const [newTeam] = await this.db
      .insert(teams)
      .values({
        name: `${user.name} Team`,
        isPersonal: true,
        createdAt: new Date(),
        organizationId: org.id,
      })
      .returning();

    if (!newTeam) {
      throw new Error("can't initialize user personal team");
    }

    // ensure user added to team member
    await this.db.insert(teamMembers).values({
      teamId: newTeam.id,
      userId: user.id,
    });

    return newTeam;
  }
}
