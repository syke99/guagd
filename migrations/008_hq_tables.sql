CREATE TABLE IF NOT EXISTS hq_layouts (
  supertokens_id VARCHAR PRIMARY KEY REFERENCES accounts(supertokens_id),
  layout         JSONB NOT NULL DEFAULT '[]',
  theme          JSONB NOT NULL DEFAULT '{}',
  updated_at     TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS club_memberships (
  id         SERIAL PRIMARY KEY,
  club_id    VARCHAR NOT NULL REFERENCES accounts(supertokens_id),
  member_id  VARCHAR NOT NULL REFERENCES accounts(supertokens_id),
  created_at TIMESTAMPTZ DEFAULT now(),
  UNIQUE(club_id, member_id)
);

GRANT ALL ON TABLE hq_layouts TO guagd_app;
GRANT ALL ON TABLE club_memberships TO guagd_app;
GRANT USAGE, SELECT ON SEQUENCE club_memberships_id_seq TO guagd_app;
