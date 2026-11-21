DROP TABLE IF EXISTS share_links;
DROP TABLE IF EXISTS file_delete_jobs;
DROP TABLE IF EXISTS audit_events;
DROP TABLE IF EXISTS upload_reservations;

DROP INDEX IF EXISTS idx_file_activity_actor_created;
ALTER TABLE file_activity DROP COLUMN IF EXISTS actor_id;

DROP INDEX IF EXISTS idx_files_owner_sha256;
DROP INDEX IF EXISTS idx_files_state;
ALTER TABLE files DROP CONSTRAINT IF EXISTS files_state_check;
ALTER TABLE files DROP COLUMN IF EXISTS state;
ALTER TABLE files DROP COLUMN IF EXISTS sha256;

ALTER TABLE users DROP COLUMN IF EXISTS auth_version;
