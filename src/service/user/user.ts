import type { User } from "@/auth";
import type { OmedDatabase } from "@/database";
import { UserModel } from "@/database/model/user";

export class UserService {
  constructor(private readonly db: OmedDatabase) {}
  async initUser(user: User) {
    const model = new UserModel(this.db);
    await model.initUser(user);
  }
}
