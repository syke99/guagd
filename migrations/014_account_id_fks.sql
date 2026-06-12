-- Switch all account FK columns from supertokens_id VARCHAR to accounts.id UUID

-- ── garage_layouts ────────────────────────────────────────────────────────────
ALTER TABLE garage_layouts ADD COLUMN IF NOT EXISTS account_id UUID;

UPDATE garage_layouts gl
SET account_id = a.id
FROM accounts a
WHERE a.supertokens_id = gl.supertokens_id;

ALTER TABLE garage_layouts DROP CONSTRAINT IF EXISTS garage_layouts_pkey;
ALTER TABLE garage_layouts DROP CONSTRAINT IF EXISTS garage_layouts_supertokens_id_fkey;
ALTER TABLE garage_layouts ADD CONSTRAINT garage_layouts_pkey PRIMARY KEY (account_id);

-- ── hq_layouts ────────────────────────────────────────────────────────────────
ALTER TABLE hq_layouts ADD COLUMN IF NOT EXISTS account_id UUID;

UPDATE hq_layouts hl
SET account_id = a.id
FROM accounts a
WHERE a.supertokens_id = hl.supertokens_id;

ALTER TABLE hq_layouts DROP CONSTRAINT IF EXISTS hq_layouts_pkey;
ALTER TABLE hq_layouts DROP CONSTRAINT IF EXISTS hq_layouts_supertokens_id_fkey;
ALTER TABLE hq_layouts ADD CONSTRAINT hq_layouts_pkey PRIMARY KEY (account_id);

-- ── account_photos ────────────────────────────────────────────────────────────
ALTER TABLE account_photos ADD COLUMN IF NOT EXISTS account_id UUID;

UPDATE account_photos ap
SET account_id = a.id
FROM accounts a
WHERE a.supertokens_id = ap.supertokens_id;

ALTER TABLE account_photos DROP CONSTRAINT IF EXISTS account_photos_pkey;
ALTER TABLE account_photos ADD CONSTRAINT account_photos_pkey PRIMARY KEY (account_id);

-- ── club_memberships ──────────────────────────────────────────────────────────
ALTER TABLE club_memberships ADD COLUMN IF NOT EXISTS new_club_id UUID;
ALTER TABLE club_memberships ADD COLUMN IF NOT EXISTS new_member_id UUID;

UPDATE club_memberships cm
SET new_club_id   = (SELECT id FROM accounts WHERE supertokens_id = cm.club_id),
    new_member_id = (SELECT id FROM accounts WHERE supertokens_id = cm.member_id);

ALTER TABLE club_memberships DROP CONSTRAINT IF EXISTS club_memberships_club_id_fkey;
ALTER TABLE club_memberships DROP CONSTRAINT IF EXISTS club_memberships_member_id_fkey;
ALTER TABLE club_memberships DROP CONSTRAINT IF EXISTS club_memberships_club_id_member_id_key;
ALTER TABLE club_memberships DROP COLUMN club_id;
ALTER TABLE club_memberships DROP COLUMN member_id;
ALTER TABLE club_memberships RENAME COLUMN new_club_id TO club_id;
ALTER TABLE club_memberships RENAME COLUMN new_member_id TO member_id;
ALTER TABLE club_memberships ADD CONSTRAINT club_memberships_club_id_member_id_key UNIQUE (club_id, member_id);

-- ── cars ──────────────────────────────────────────────────────────────────────
ALTER TABLE cars ADD COLUMN IF NOT EXISTS new_owner_id UUID;

UPDATE cars c
SET new_owner_id = a.id
FROM accounts a
WHERE a.supertokens_id = c.owner_id;

ALTER TABLE cars DROP CONSTRAINT IF EXISTS cars_owner_id_fkey;
DROP INDEX IF EXISTS cars_owner_id_idx;
ALTER TABLE cars DROP COLUMN owner_id;
ALTER TABLE cars RENAME COLUMN new_owner_id TO owner_id;
ALTER TABLE cars ADD CONSTRAINT cars_owner_id_fkey
    FOREIGN KEY (owner_id) REFERENCES accounts(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS cars_owner_id_idx ON cars(owner_id);
