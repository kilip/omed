import type { SQL } from "drizzle-orm";
import { sql } from "drizzle-orm";
import { TOPIC } from "../topics";

// Satu statement per item (body fungsi mengandung ';', jadi jangan di-split manual).
export const OUTBOX_TRIGGER_STATEMENTS: string[] = [
  `CREATE OR REPLACE FUNCTION auth.outbox_ts(ts timestamp) RETURNS text
   LANGUAGE sql IMMUTABLE AS $$
     SELECT to_char(ts, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"')
   $$`,

  `CREATE OR REPLACE FUNCTION auth.outbox_user_snapshot(u auth."user") RETURNS jsonb
   LANGUAGE sql IMMUTABLE AS $$
     SELECT jsonb_build_object(
       'id', u.id,
       'name', u.name,
       'email', u.email,
       'emailVerified', u.email_verified,
       'image', u.image,
       'role', u.role,
       'banned', COALESCE(u.banned, false),
       'locale', u.locale,
       'defaultCurrency', u.default_currency,
       'createdAt', auth.outbox_ts(u.created_at),
       'updatedAt', auth.outbox_ts(u.updated_at)
     )
   $$`,

  `CREATE OR REPLACE FUNCTION auth.outbox_user_changed() RETURNS trigger
   LANGUAGE plpgsql AS $$
   DECLARE
     evt text;
   BEGIN
     IF TG_OP = 'DELETE' THEN
       INSERT INTO auth.outbox (topic, key, payload, headers)
       VALUES (
         '${TOPIC.user}', OLD.id::text, NULL,
         jsonb_build_object(
           'event-type', 'deleted',
           'occurred-at', auth.outbox_ts(clock_timestamp() AT TIME ZONE 'utc')
         )
       );
       RETURN OLD;
     END IF;

     IF TG_OP = 'UPDATE' THEN
       -- skip kalau tidak ada perubahan di field yang dipublish
       IF (auth.outbox_user_snapshot(NEW) - 'updatedAt')
          = (auth.outbox_user_snapshot(OLD) - 'updatedAt') THEN
         RETURN NEW;
       END IF;
       evt := 'updated';
     ELSE
       evt := 'created';
     END IF;

     INSERT INTO auth.outbox (topic, key, payload, headers)
     VALUES (
       '${TOPIC.user}', NEW.id::text, auth.outbox_user_snapshot(NEW),
       jsonb_build_object(
         'event-type', evt,
         'occurred-at', auth.outbox_ts(clock_timestamp() AT TIME ZONE 'utc')
       )
     );
     RETURN NEW;
   END
   $$`,

  `DROP TRIGGER IF EXISTS outbox_user ON auth."user"`,
  `CREATE TRIGGER outbox_user
   AFTER INSERT OR UPDATE OR DELETE ON auth."user"
   FOR EACH ROW EXECUTE FUNCTION auth.outbox_user_changed()`,

  `CREATE OR REPLACE FUNCTION auth.outbox_team_created() RETURNS trigger
   LANGUAGE plpgsql AS $$
   BEGIN
     INSERT INTO auth.outbox (topic, key, payload, headers)
     VALUES (
       '${TOPIC.team}', NEW.id::text,
       jsonb_build_object(
         'id', NEW.id,
         'name', NEW.name,
         'organizationId', NEW.organization_id,
         'personal', COALESCE(NEW.personal, false),
         'createdAt', auth.outbox_ts(NEW.created_at),
         'updatedAt', auth.outbox_ts(NEW.updated_at)
       ),
       jsonb_build_object(
         'event-type', 'created',
         'occurred-at', auth.outbox_ts(clock_timestamp() AT TIME ZONE 'utc')
       )
     );
     RETURN NEW;
   END
   $$`,

  `DROP TRIGGER IF EXISTS outbox_team ON auth.team`,
  `CREATE TRIGGER outbox_team
   AFTER INSERT ON auth.team
   FOR EACH ROW EXECUTE FUNCTION auth.outbox_team_created()`,
];

export async function applyOutboxTriggers(db: {
  execute(query: SQL): PromiseLike<unknown>;
}) {
  for (const stmt of OUTBOX_TRIGGER_STATEMENTS) {
    await db.execute(sql.raw(stmt));
  }
}
