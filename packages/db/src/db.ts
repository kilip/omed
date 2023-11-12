import { PrismaClient } from "@prisma/client";

import { singleton } from "./singleton";

// Hard-code a unique key, so we can look up the client when this module gets re-imported
const db = singleton<PrismaClient>("prisma", () => new PrismaClient());
db.$connect();

export default db;
