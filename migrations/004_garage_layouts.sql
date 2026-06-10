CREATE TABLE IF NOT EXISTS garage_layouts (
  supertokens_id VARCHAR PRIMARY KEY REFERENCES users(supertokens_id),
  layout         JSONB NOT NULL DEFAULT '[]',
  theme          JSONB NOT NULL DEFAULT '{}'
);
