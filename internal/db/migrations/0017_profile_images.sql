ALTER TABLE candidates ADD COLUMN photo_path TEXT NOT NULL DEFAULT '';
ALTER TABLE candidates ADD COLUMN photo_content_type TEXT NOT NULL DEFAULT '';
ALTER TABLE employers ADD COLUMN logo_path TEXT NOT NULL DEFAULT '';
ALTER TABLE employers ADD COLUMN logo_content_type TEXT NOT NULL DEFAULT '';
