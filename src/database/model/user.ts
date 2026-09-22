import type { User } from "@/auth";
import { organization, user as userSchema } from "../schema";
import type { OmedDatabase } from "../type";

export class UserModel {
  constructor(private readonly db: OmedDatabase) {}

  async initUser(user: User) {
    const slug = `${user.name.toLowerCase().replace(" ", "-")}`;
    const [org] = await this.db
      .insert(organization)
      .values({
        name: `${user.name} Workspace`,
        slug,
        createdAt: new Date(),
      })
      .returning();
    user.activeWorkspace = org.id;
    await this.db.update(userSchema).set(user);
    console.log(user);
    return org;
  }
}
