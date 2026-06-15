-- Gaugd baseline schema

CREATE TABLE accounts (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    supertokens_id VARCHAR     UNIQUE,
    username       VARCHAR     UNIQUE,
    email          TEXT        NOT NULL UNIQUE,
    name           TEXT,
    acct_type      VARCHAR     NOT NULL DEFAULT 'driver',
    visitor_id     TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE account_photos (
    account_id UUID        PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    banner_key VARCHAR,
    avatar_key VARCHAR,
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE garage_layouts (
    account_id      UUID        PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    layout          JSONB       NOT NULL DEFAULT '[]',
    theme           JSONB       NOT NULL DEFAULT '{}',
    cover_photo_key VARCHAR,
    updated_at      TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE hq_layouts (
    account_id      UUID        PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    layout          JSONB       NOT NULL DEFAULT '[]',
    theme           JSONB       NOT NULL DEFAULT '{}',
    cover_photo_key VARCHAR,
    updated_at      TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE club_memberships (
    id         SERIAL      PRIMARY KEY,
    club_id    UUID        REFERENCES accounts(id) ON DELETE CASCADE,
    member_id  UUID        REFERENCES accounts(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE (club_id, member_id)
);

CREATE TABLE shops (
    account_id     UUID        PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    shop_name      VARCHAR     NOT NULL,
    owner_name     VARCHAR     NOT NULL,
    owner_id       UUID        REFERENCES accounts(id) ON DELETE SET NULL,
    address        VARCHAR,
    phone          VARCHAR,
    business_hours VARCHAR,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE cars (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id   UUID        REFERENCES accounts(id) ON DELETE CASCADE,
    year       INT         NOT NULL,
    make       VARCHAR     NOT NULL,
    model      VARCHAR     NOT NULL,
    trim       VARCHAR,
    mileage    INT,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX cars_owner_id_idx ON cars(owner_id);

CREATE TABLE car_photos (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    car_id      UUID        NOT NULL REFERENCES cars(id) ON DELETE CASCADE,
    object_key  VARCHAR     NOT NULL,
    is_primary  BOOLEAN     NOT NULL DEFAULT false,
    uploaded_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX car_photos_car_id_idx ON car_photos(car_id);

CREATE TABLE car_mods (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    car_id             UUID        NOT NULL REFERENCES cars(id) ON DELETE CASCADE,
    name               VARCHAR     NOT NULL,
    category           VARCHAR     NOT NULL DEFAULT 'Other',
    install_date       DATE,
    mileage_at_install INT,
    cost               INT,
    notes              TEXT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE car_maintenance (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    car_id       UUID        NOT NULL REFERENCES cars(id) ON DELETE CASCADE,
    name         VARCHAR     NOT NULL,
    category     VARCHAR     NOT NULL DEFAULT 'Other',
    service_date DATE,
    mileage      INT,
    cost         INT,
    notes        VARCHAR     NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE car_uploads (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    mod_id         UUID        REFERENCES car_mods(id) ON DELETE CASCADE,
    maintenance_id UUID        REFERENCES car_maintenance(id) ON DELETE CASCADE,
    object_key     VARCHAR     NOT NULL,
    name           VARCHAR     NOT NULL,
    upload_type    VARCHAR     NOT NULL DEFAULT 'Receipt',
    content_type   VARCHAR     NOT NULL DEFAULT 'application/octet-stream',
    uploaded_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE car_verifications (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    mod_id         UUID        REFERENCES car_mods(id) ON DELETE CASCADE,
    maintenance_id UUID        REFERENCES car_maintenance(id) ON DELETE CASCADE,
    shop_id        UUID        REFERENCES accounts(id) ON DELETE SET NULL,
    level          VARCHAR     NOT NULL DEFAULT 'reported',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT exactly_one_parent CHECK (
        (mod_id IS NOT NULL AND maintenance_id IS NULL) OR
        (mod_id IS NULL AND maintenance_id IS NOT NULL)
    )
);

CREATE TABLE verification_requests (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    requester_id UUID        NOT NULL REFERENCES accounts(id),
    shop_id      UUID        NOT NULL REFERENCES accounts(id),
    service_type VARCHAR     NOT NULL DEFAULT 'past',
    work_type    VARCHAR     NOT NULL DEFAULT 'own_work',
    status       VARCHAR     NOT NULL DEFAULT 'requested',
    notes        VARCHAR,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE verification_request_items (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id      UUID        NOT NULL REFERENCES verification_requests(id) ON DELETE CASCADE,
    verification_id UUID        NOT NULL REFERENCES car_verifications(id) ON DELETE CASCADE,
    status          VARCHAR     NOT NULL DEFAULT 'pending',
    denial_notes    VARCHAR,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (request_id, verification_id)
);

CREATE TABLE page_events (
    id         BIGSERIAL   PRIMARY KEY,
    visitor_id TEXT        NOT NULL,
    event      TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Grants
GRANT ALL ON TABLE accounts                   TO guagd_app;
GRANT ALL ON TABLE account_photos             TO guagd_app;
GRANT ALL ON TABLE garage_layouts             TO guagd_app;
GRANT ALL ON TABLE hq_layouts                 TO guagd_app;
GRANT ALL ON TABLE club_memberships           TO guagd_app;
GRANT ALL ON TABLE shops                      TO guagd_app;
GRANT ALL ON TABLE cars                       TO guagd_app;
GRANT ALL ON TABLE car_photos                 TO guagd_app;
GRANT ALL ON TABLE car_mods                   TO guagd_app;
GRANT ALL ON TABLE car_maintenance            TO guagd_app;
GRANT ALL ON TABLE car_uploads                TO guagd_app;
GRANT ALL ON TABLE car_verifications          TO guagd_app;
GRANT ALL ON TABLE verification_requests      TO guagd_app;
GRANT ALL ON TABLE verification_request_items TO guagd_app;
GRANT ALL ON TABLE page_events                TO guagd_app;
GRANT USAGE, SELECT ON SEQUENCE club_memberships_id_seq TO guagd_app;
GRANT USAGE, SELECT ON SEQUENCE page_events_id_seq      TO guagd_app;
