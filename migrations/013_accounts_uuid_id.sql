ALTER TABLE accounts ADD COLUMN IF NOT EXISTS id UUID NOT NULL DEFAULT gen_random_uuid();

CREATE UNIQUE INDEX IF NOT EXISTS accounts_id_unique ON accounts(id);

GRANT ALL ON TABLE accounts TO guagd_app;
