-- Drop the stale supertokens_id columns from tables that now use account_id as FK anchor.
-- These columns were left behind by migration 014 with a NOT NULL constraint (inherited
-- from the old PRIMARY KEY), which caused insert failures on the new account_id-based queries.

ALTER TABLE account_photos DROP COLUMN IF EXISTS supertokens_id;
ALTER TABLE garage_layouts  DROP COLUMN IF EXISTS supertokens_id;
ALTER TABLE hq_layouts      DROP COLUMN IF EXISTS supertokens_id;
