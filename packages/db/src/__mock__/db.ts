import { PrismaClient } from "@prisma/client";
import { beforeEach } from "vitest";
import { mockDeep, mockReset } from "vitest-mock-extended";

// 2
beforeEach(() => {
  mockReset(db);
});

// 3
const db = mockDeep<PrismaClient>();
export default db;
