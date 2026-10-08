import { applyOutboxTriggers, authDB } from "@omed/better-auth/drizzle";

await applyOutboxTriggers(authDB);
console.log("[db] outbox triggers applied");
process.exit(0);
