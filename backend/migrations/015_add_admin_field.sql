-- Add is_admin field to users table
ALTER TABLE users ADD COLUMN is_admin BOOLEAN DEFAULT FALSE;

-- Create index for admin users
CREATE INDEX idx_users_is_admin ON users(is_admin) WHERE is_admin = TRUE;

-- Optional: Make first user admin (or set by email)
-- UPDATE users SET is_admin = TRUE WHERE email = 'your-admin-email@example.com';
