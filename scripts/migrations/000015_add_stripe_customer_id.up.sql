-- 000015_add_stripe_customer_id.up.sql
-- Add stripe_customer_id to users table for Stripe Customer Portal integration

ALTER TABLE users ADD COLUMN IF NOT EXISTS stripe_customer_id VARCHAR(255);

-- Index for faster lookup by stripe_customer_id
CREATE INDEX IF NOT EXISTS idx_users_stripe_customer_id ON users(stripe_customer_id) WHERE stripe_customer_id IS NOT NULL;
