-- Admins are rows in the existing users table with role='admin', reusing
-- the same password hashing, session, and auth-middleware infrastructure
-- as candidates/employers rather than a parallel account system. Admin
-- accounts are never created through public signup (there is no
-- POST /signup/admin route) - see cmd/createadmin.
ALTER TABLE users DROP CONSTRAINT users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('candidate', 'employer', 'admin'));
