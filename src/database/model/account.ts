import { eq } from "drizzle-orm";
import { type AccountItem, accounts, type NewAccount } from "../schema";
import type { OmedDatabase } from "../type";

export class AccountModel {
  constructor(
    private readonly db: OmedDatabase,
    private readonly userId: string,
    private readonly workspaceId: string,
  ) {}

  async create(
    params: Omit<NewAccount, "workspaceId" | "createdBy">,
  ): Promise<AccountItem> {
    const [row] = await this.db
      .insert(accounts)
      .values({
        ...params,
        createdBy: this.userId,
        workspaceId: this.workspaceId,
      })
      .returning();
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

  async list(workspaceId: string): Promise<AccountItem[]> {
    return this.db.query.accounts.findMany({
      where: { workspaceId },
    });
  }
}
