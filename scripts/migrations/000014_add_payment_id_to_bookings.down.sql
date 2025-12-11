-- 000014_add_payment_id_to_bookings.down.sql
-- Remove payment_id column from bookings table

DROP INDEX IF EXISTS idx_bookings_payment_id;
ALTER TABLE bookings DROP COLUMN IF EXISTS payment_id;
