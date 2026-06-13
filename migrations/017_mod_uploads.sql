CREATE TABLE mod_uploads (
  id           BIGSERIAL    PRIMARY KEY,
  mod_id       UUID         NOT NULL REFERENCES car_mods(id) ON DELETE CASCADE,
  object_key   VARCHAR      NOT NULL,
  name         VARCHAR      NOT NULL,
  upload_type  VARCHAR      NOT NULL DEFAULT 'Receipt',
  content_type VARCHAR      NOT NULL DEFAULT 'application/octet-stream',
  uploaded_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
