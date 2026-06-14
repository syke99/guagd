-- Reshape verification_requests: drop single-FK model, support batch requests with per-item status
ALTER TABLE verification_requests
    DROP COLUMN verification_id,
    DROP COLUMN request_type,
    ADD COLUMN service_type VARCHAR NOT NULL DEFAULT 'past',
    ADD COLUMN work_type    VARCHAR NOT NULL DEFAULT 'own_work';

-- Per-item table: one request → many car_verifications, each with its own status
CREATE TABLE verification_request_items (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id      UUID NOT NULL REFERENCES verification_requests(id) ON DELETE CASCADE,
    verification_id UUID NOT NULL REFERENCES car_verifications(id)  ON DELETE CASCADE,
    status          VARCHAR NOT NULL DEFAULT 'pending',
    denial_notes    VARCHAR,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (request_id, verification_id)
);
