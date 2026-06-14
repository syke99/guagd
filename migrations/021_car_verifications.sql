CREATE TABLE car_verifications (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mod_id         UUID REFERENCES car_mods(id) ON DELETE CASCADE,
    maintenance_id UUID REFERENCES car_maintenance(id) ON DELETE CASCADE,
    shop_id        UUID REFERENCES accounts(id) ON DELETE SET NULL,
    level          VARCHAR NOT NULL DEFAULT 'reported',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT exactly_one_parent CHECK (
        (mod_id IS NOT NULL AND maintenance_id IS NULL) OR
        (mod_id IS NULL AND maintenance_id IS NOT NULL)
    )
);

CREATE TABLE verification_requests (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    verification_id UUID NOT NULL REFERENCES car_verifications(id) ON DELETE CASCADE,
    requester_id    UUID NOT NULL REFERENCES accounts(id),
    shop_id         UUID NOT NULL REFERENCES accounts(id),
    request_type    VARCHAR NOT NULL,
    status          VARCHAR NOT NULL DEFAULT 'requested',
    notes           VARCHAR,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Backfill existing mod records
INSERT INTO car_verifications (mod_id, level)
SELECT id, 'reported' FROM car_mods;

-- Backfill existing maintenance records
INSERT INTO car_verifications (maintenance_id, level)
SELECT id, 'reported' FROM car_maintenance;

-- Bump to 'documented' for any that already have uploads
UPDATE car_verifications cv
SET level = 'documented', updated_at = NOW()
WHERE cv.mod_id IS NOT NULL
  AND cv.level = 'reported'
  AND EXISTS (SELECT 1 FROM car_uploads cu WHERE cu.mod_id = cv.mod_id);

UPDATE car_verifications cv
SET level = 'documented', updated_at = NOW()
WHERE cv.maintenance_id IS NOT NULL
  AND cv.level = 'reported'
  AND EXISTS (SELECT 1 FROM car_uploads cu WHERE cu.maintenance_id = cv.maintenance_id);
