CREATE TABLE car_maintenance (
  id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  car_id       UUID         NOT NULL REFERENCES cars(id) ON DELETE CASCADE,
  name         VARCHAR      NOT NULL,
  category     VARCHAR      NOT NULL DEFAULT 'Other',
  service_date DATE,
  mileage      INT,
  cost         INT,
  notes        VARCHAR      NOT NULL DEFAULT '',
  created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

ALTER TABLE car_uploads ALTER COLUMN mod_id DROP NOT NULL;
ALTER TABLE car_uploads ADD COLUMN maintenance_id UUID REFERENCES car_maintenance(id) ON DELETE CASCADE;
