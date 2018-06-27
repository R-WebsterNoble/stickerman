CREATE EXTENSION "uuid-ossp";

BEGIN;

-- CREATE TABLE "groups" ---------------------------------------
CREATE TABLE "public"."groups" (
  "id"   Bigserial                       NOT NULL,
  "uuid" UUid DEFAULT uuid_generate_v4() NOT NULL,
  CONSTRAINT "unique_groups_id" UNIQUE ("id"),
  CONSTRAINT "unique_groups_uuid" UNIQUE ("uuid")
);

-- Set up a default group
INSERT INTO groups (id) values (1);

-- -------------------------------------------------------------
-- CREATE FIELD "group_id" -------------------------------------
ALTER TABLE "public"."sessions"
  ADD COLUMN "group_id" Bigint DEFAULT 1 NOT NULL;
-- -------------------------------------------------------------
ALTER TABLE "public"."sessions"
  ALTER COLUMN "group_id" DROP DEFAULT;

-- CREATE FIELD "group_id" -------------------------------------
ALTER TABLE "public"."sticker_keywords"
  ADD COLUMN "group_id" Bigint DEFAULT 1 NOT NULL;
-- -------------------------------------------------------------
ALTER TABLE "public"."sticker_keywords"
  ALTER COLUMN "group_id" DROP DEFAULT;

-- CREATE LINK "lnk_groups_sessions" -------------------------
ALTER TABLE "public"."sessions"
  ADD CONSTRAINT "lnk_groups_sessions" FOREIGN KEY ("group_id")
REFERENCES "public"."groups" ("id") MATCH FULL
ON DELETE Cascade
ON UPDATE Cascade;
-- -------------------------------------------------------------

-- CREATE LINK "lnk_groups_sticker_keywords" -------------------
ALTER TABLE "public"."sticker_keywords"
  ADD CONSTRAINT "lnk_groups_sticker_keywords" FOREIGN KEY ("group_id")
REFERENCES "public"."groups" ("id") MATCH FULL
ON DELETE Cascade
ON UPDATE Cascade;
-- -------------------------------------------------------------

-- DROP UNIQUE "stickerkeywords_stickerid_keywordid_groupid" ---
ALTER TABLE "public"."sticker_keywords"
  DROP CONSTRAINT IF EXISTS "stickerkeywords_stickerid_keywordid";
-- -------------------------------------------------------------

-- CREATE UNIQUE "stickerkeywords_stickerid_keywordid_groupid" -
ALTER TABLE "public"."sticker_keywords"
  ADD CONSTRAINT "stickerkeywords_stickerid_keywordid_groupid" UNIQUE ("sticker_id", "keyword_id", "group_id");
-- -------------------------------------------------------------

COMMIT;