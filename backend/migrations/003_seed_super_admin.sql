-- 003_seed_super_admin.sql
-- Intentionally empty. A hard-coded super admin credential used to be seeded here, which is
-- unsafe for any real deployment. The first super admin is now created at startup by
-- internal/database/bootstrap.go from BOOTSTRAP_ADMIN_EMAIL / BOOTSTRAP_ADMIN_PASSWORD
-- (or a random one-time password in non-production environments).
SELECT 1;
