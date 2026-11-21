ALTER TABLE files ADD COLUMN folder_id VARCHAR(36) REFERENCES folders (id) ON DELETE RESTRICT;
CREATE INDEX IF NOT EXISTS idx_files_folder_id ON files (folder_id);
