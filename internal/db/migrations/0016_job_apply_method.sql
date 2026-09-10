ALTER TABLE jobs ADD COLUMN apply_method TEXT NOT NULL DEFAULT 'url' CHECK (apply_method IN ('url', 'email'));
ALTER TABLE jobs ADD COLUMN apply_value TEXT NOT NULL DEFAULT '';
