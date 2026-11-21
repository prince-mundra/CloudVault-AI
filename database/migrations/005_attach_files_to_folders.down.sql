DROP INDEX IF EXISTS idx_files_folder_id;
ALTER TABLE files DROP COLUMN IF EXISTS folder_id;
