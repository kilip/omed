import { PrismaClient } from "@prisma/client";

import { singleton } from "@omed/utils";

function getConnectedDb() {
  const db = singleton<PrismaClient>("db.prisma", () => new PrismaClient());
  db.$connect();
  return db;
}
// Hard-code a unique key, so we can look up the client when this module gets re-imported
export const db = getConnectedDb();
