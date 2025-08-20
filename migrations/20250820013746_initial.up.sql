-- create "users" table
CREATE TABLE "public"."users" (
 "id" text NOT NULL DEFAULT uuidv7(),
 "created_at" timestamptz NULL,
 "updated_at" timestamptz NULL,
 "deleted_at" timestamptz NULL,
 "name" text NULL,
 "email" text NULL,
 "avatar" text NULL,
 "password_hash" text NULL,
 PRIMARY KEY ("id"),
 CONSTRAINT "uni_users_email" UNIQUE ("email")
);
-- create index "idx_users_deleted_at" to table: "users"
CREATE INDEX "idx_users_deleted_at" ON "public"."users" ("deleted_at");
-- create index "idx_users_email" to table: "users"
CREATE INDEX "idx_users_email" ON "public"."users" ("email");
-- create index "idx_users_name" to table: "users"
CREATE INDEX "idx_users_name" ON "public"."users" ("name");
