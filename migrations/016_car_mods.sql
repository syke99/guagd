CREATE TABLE car_mods (
  id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  car_id             UUID        NOT NULL REFERENCES cars(id) ON DELETE CASCADE,
  name               VARCHAR     NOT NULL,
  category           VARCHAR     NOT NULL DEFAULT 'Other',
  install_date       DATE,
  mileage_at_install INT,
  cost               INT,
  notes              TEXT,
  created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
