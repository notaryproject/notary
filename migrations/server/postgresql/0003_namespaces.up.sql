ALTER TABLE "tuf_files"
  ADD COLUMN "namespace" VARCHAR(255) NOT NULL DEFAULT ('published'),
  DROP CONSTRAINT "tuf_files_gun_role_version_key",
  ADD UNIQUE ("gun","role","version", "namespace");


ALTER TABLE "changefeed"
  ADD COLUMN "namespace" VARCHAR(255) NOT NULL DEFAULT ('published');

DROP INDEX "idx_changefeed_gun";
CREATE INDEX "idx_changefeed_gun_ns" ON "changefeed" ("gun", "namespace");