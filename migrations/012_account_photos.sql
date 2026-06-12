CREATE TABLE IF NOT EXISTS account_photos (
  supertokens_id VARCHAR PRIMARY KEY,
  banner_key     VARCHAR,
  avatar_key     VARCHAR,
  updated_at     TIMESTAMPTZ DEFAULT now()
);
