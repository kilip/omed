import type { User } from "@/auth";
import {
  members,
  organizations,
  teamMembers,
  teams,
  workspaces,
} from "../schema";
import type { OmedDatabase, OrganizationItem, TeamItem } from "../type";

export class UserModel {
  constructor(private readonly db: OmedDatabase) {}

  async findPersonalOrganization(
    userId: string,
  ): Promise<OrganizationItem | undefined> {
    const membersOf = await this.db.query.members.findFirst({
      where: {
        userId,
      },
      with: {
        organization: {
          where: { isPersonal: true },
        },
      },
    });
    if (membersOf?.organization) return membersOf.organization;
    return undefined;
  }

  async initPersonalOrganization(user: User): Promise<OrganizationItem> {
    let org = await this.findPersonalOrganization(user.id);
    if (!org) {
      const [newOrg] = await this.db
        .insert(organizations)
        .values({
          name: `${user.name} Personal Organization`,
          slug: `${user.name.toLowerCase().replace(" ", "-")}`,
          createdAt: new Date(),
          isPersonal: true,
        })
        .returning();

      // ensure user added to member
      await this.db
        .insert(members)
        .values({
          userId: user.id,
          organizationId: newOrg.id,
          createdAt: new Date(),
          role: "owner",
        })
        .returning();

      org = newOrg;
    }

    return org;
  }

  async findPersonalTeam(userId: string): Promise<TeamItem | undefined> {
    const membersOf = await this.db.query.teamMembers.findFirst({
      where: {
        userId,
      },
      with: {
        team: {
          where: {
            isPersonal: true,
          },
        },
      },
    });

    if (membersOf?.team) return membersOf.team;

    return undefined;
  }

  async initPersonalTeam(user: User) {
    let team = await this.findPersonalTeam(user.id);

    if (!team) {
      const org = await this.initPersonalOrganization(user);
      const [newTeam] = await this.db
        .insert(teams)
        .values({
          organizationId: org.id,
          name: `${user.name} Personal Team`,
          isPersonal: true,
          createdAt: new Date(),
        })
        .returning();

      // ensure user added to team member
      await this.db.insert(teamMembers).values({
        teamId: newTeam.id,
        userId: user.id,
        createdAt: new Date(),
      });
      team = newTeam;
    }
    return team;
  }

  async findPersonalWorkspace(userId: string) {
    return this.db.query.workspaces.findFirst({
      where: {
        isPersonal: true,
        team: {
          teamMembers: {
            userId,
          },
        },
      },
      with: {
        team: {
          with: {
            organization: true,
          },
        },
      },
    });
  }

  async initPersonalWorkspace(user: User) {
    let workspace = await this.findPersonalWorkspace(user.id);
    if (!workspace) {
      const team = await this.initPersonalTeam(user);
      await this.db
        .insert(workspaces)
        .values({
          name: `${user.name} Personal Workspace`,
          createdBy: user.id,
          teamId: team.id,
          isPersonal: true,
        })
        .returning();
      workspace = await this.findPersonalWorkspace(user.id);
    }

    return workspace;
  }
}
