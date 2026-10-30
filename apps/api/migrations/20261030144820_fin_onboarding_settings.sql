-- Modify "fin_settings" table
ALTER TABLE "fin"."fin_settings" ADD COLUMN "onboarding" boolean NOT NULL DEFAULT false, ADD COLUMN "decimal_style" character varying NOT NULL DEFAULT 'comma';
