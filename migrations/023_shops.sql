CREATE TABLE IF NOT EXISTS shops (
    account_id     UUID PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    shop_name      VARCHAR NOT NULL,
    owner_name     VARCHAR NOT NULL,
    address        VARCHAR,
    phone          VARCHAR,
    business_hours VARCHAR,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
