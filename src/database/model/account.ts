import { eq } from "drizzle-orm";
import { type AccountItem, accounts, type NewAccount } from "../schema";
import type { OmedDatabase } from "../type";

export class AccountModel {
  constructor(readonly db: OmedDatabase) {}

  async create(params: NewAccount): Promise<AccountItem> {
    const [row] = await this.db.insert(accounts).values(params).returning();
    return row;
  }

  async update(params: AccountItem) {
    const [row] = await this.db.update(accounts).set(params).returning();

    return row;
  }

  async delete(id: string) {
    await this.db.delete(accounts).where(eq(accounts.id, id));
  }

  async findById(id: string) {
    return this.db.query.accounts.findFirst({ where: { id } });
  }

  async list(tenantId: string): Promise<AccountItem[]> {
    return this.db.query.accounts.findMany({
      where: { tenantId },
    });
  }
}
