import { serverDB } from "@/database";
import { AccountModel } from "@/database/model/account";
import type { NewAccount } from "@/database/schema";

export class AccountService {
  async create(params: NewAccount) {
    const model = new AccountModel(serverDB);

    return model.create(params);
  }
}
