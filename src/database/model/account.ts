import { type AccountItem, accounts, type NewAccount } from "../schema";
import type { OmedDatabase } from "../type";

export class AccountModel {
  constructor(
    private readonly db: OmedDatabase,
    private readonly userId: string,
    private readonly tenantId: string,
  ) {}

  async create(params: Omit<NewAccount, "tenantId">): Promise<AccountItem> {
    params.createdBy = this.userId;
    const [row] = await this.db
      .insert(accounts)
      .values({
        ...params,
        tenantId: this.tenantId,
      })
      .returning();
    return row;
  }

  async findById(id: string) {
    return await this.db.query.accounts.findFirst({
      where: {
        tenantId: this.tenantId,
        id,
      },
    });
  }
}
