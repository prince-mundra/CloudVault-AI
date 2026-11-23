-- 012_anonymous_users.down.sql
-- Safety guard: refuse rollback if any anonymous accounts exist.
-- Never automatically cascade-delete accounts, files, or audit logs.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM users WHERE role = 'anonymous' OR is_anonymous = TRUE) THEN
        RAISE EXCEPTION 'Cannot roll back migration 012: anonymous accounts still exist in the database. Clean up or archive anonymous accounts first.';
    END IF;
END $$;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('user', 'admin'));
DROP INDEX IF EXISTS idx_users_is_anonymous;
ALTER TABLE users DROP COLUMN IF EXISTS is_anonymous;
