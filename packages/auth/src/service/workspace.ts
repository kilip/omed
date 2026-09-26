import type { AuthDatabase } from "../drizzle";

export class Workspace {
  constructor(private readonly db: AuthDatabase) {}
}
