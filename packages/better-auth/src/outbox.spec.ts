import { eq } from "drizzle-orm";
import { describe, expect, it } from "vitest";
import { auth } from "./auth";
import { authDB } from "./drizzle";
import { outbox } from "./drizzle/schema/outbox";
import { TOPIC } from "./topics";

const context = await auth.$context;
const util = context.test;

const rowsFor = (key: string) =>
  authDB.select().from(outbox).where(eq(outbox.key, key)).orderBy(outbox.id);

type Row = Awaited<ReturnType<typeof rowsFor>>[number];
const evt = (rows: Row[], type: string) =>
  rows.find((r) => r.headers?.["event-type"] === type);

describe("Outbox triggers", () => {
  it("emits user created", async () => {
    const created = await util.saveUser(util.createUser({ name: "outbox" }));
    const e = evt(await rowsFor(created.id), "created");

    expect(e?.topic).toBe(TOPIC.user);
    expect(e?.publishedAt).toBeNull();
    expect(e?.payload).toMatchObject({ id: created.id, email: created.email });
  });

  it("emits user updated only when published fields change", async () => {
    const created = await util.saveUser(util.createUser({ name: "outbox-up" }));
    const { headers } = await util.login({ userId: created.id });
    await auth.api.updateUser({ headers, body: { locale: "id" } });

    const e = evt(await rowsFor(created.id), "updated");
    expect(e?.payload).toMatchObject({ id: created.id, locale: "id" });
  });

  it("emits tombstone on user delete", async () => {
    const created = await util.saveUser(
      util.createUser({ name: "outbox-del" }),
    );
    await util.deleteUser(created.id);

    const e = evt(await rowsFor(created.id), "deleted");
    expect(e?.topic).toBe(TOPIC.user);
    expect(e?.payload).toBeNull();
  });

  it("emits team created for the personal workspace", async () => {
    const created = await util.saveUser(util.createUser({ name: "outbox-tm" }));
    const { session } = await util.login({ userId: created.id });
    const teamId = (session as { activeTeamId?: string }).activeTeamId;
    expect(teamId).toBeTruthy();

    const e = evt(await rowsFor(teamId as string), "created");
    expect(e?.topic).toBe(TOPIC.team);
    expect(e?.payload).toMatchObject({ id: teamId, personal: true });
  });
});
