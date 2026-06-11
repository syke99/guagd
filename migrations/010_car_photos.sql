ALTER TABLE garage_layouts ADD COLUMN IF NOT EXISTS cover_photo_key VARCHAR;

CREATE TABLE IF NOT EXISTS car_photos (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  car_id      UUID NOT NULL REFERENCES cars(id) ON DELETE CASCADE,
  object_key  VARCHAR NOT NULL,
  is_primary  BOOLEAN NOT NULL DEFAULT false,
  uploaded_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS car_photos_car_id_idx ON car_photos(car_id);

GRANT ALL ON TABLE car_photos TO guagd_app;
