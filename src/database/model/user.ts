import type { User } from "@/auth";
import { organizations, teams, users } from "../schema";
import type { OmedDatabase } from "../type";

export class UserModel {
  constructor(private readonly db: OmedDatabase) {}

  async initUser(user: User) {
    const slug = `${user.name.toLowerCase().replace(" ", "-")}`;
    const [org] = await this.db
      .insert(organizations)
      .values({
        name: `${user.name} Default Organization`,
        slug,
        createdAt: new Date(),
      })
      .returning();

    const [team] = await this.db
      .insert(teams)
      .values({
        name: `${user.name} Default Workspace`,
        organizationId: org.id,
        createdAt: new Date(),
      })
      .returning();

    user.activeWorkspace = team.id;
    await this.db.update(users).set(user);
  }
}
