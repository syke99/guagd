-- Backfill account_photos.banner_key from garage_layouts and hq_layouts for accounts
-- where the cover photo was stored in the layout row before account_photos was introduced.
-- Only sets banner_key where account_photos has no row or has a NULL banner_key.

INSERT INTO account_photos (account_id, banner_key)
SELECT gl.account_id, gl.cover_photo_key
FROM garage_layouts gl
WHERE gl.cover_photo_key IS NOT NULL AND gl.cover_photo_key != ''
ON CONFLICT (account_id) DO UPDATE
  SET banner_key = EXCLUDED.banner_key
  WHERE account_photos.banner_key IS NULL;

INSERT INTO account_photos (account_id, banner_key)
SELECT hl.account_id, hl.cover_photo_key
FROM hq_layouts hl
WHERE hl.cover_photo_key IS NOT NULL AND hl.cover_photo_key != ''
ON CONFLICT (account_id) DO UPDATE
  SET banner_key = EXCLUDED.banner_key
  WHERE account_photos.banner_key IS NULL;
