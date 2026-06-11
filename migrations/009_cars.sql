CREATE TABLE IF NOT EXISTS cars (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_id   VARCHAR NOT NULL REFERENCES accounts(supertokens_id) ON DELETE CASCADE,
  year       INT NOT NULL,
  make       VARCHAR NOT NULL,
  model      VARCHAR NOT NULL,
  trim       VARCHAR,
  mileage    INT,
  created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS cars_owner_id_idx ON cars(owner_id);

GRANT ALL ON TABLE cars TO guagd_app;
