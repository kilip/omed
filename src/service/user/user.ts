import type { User } from "@/auth";
import type { OmedDatabase } from "@/database";
import { UserModel } from "@/database/model/user";

export class UserService {
  private readonly model: UserModel;

  constructor(private readonly db: OmedDatabase) {
    this.model = new UserModel(db);
  }

  async initPersonalWorkspace(user: User) {
    const model = new UserModel(this.db);
    return await model.initPersonalWorkspace(user);
  }

  async findPersonalOrganization() {}

  async findPersonalWorkspace(userId: string) {
    return await this.model.findPersonalWorkspace(userId);
  }
}
