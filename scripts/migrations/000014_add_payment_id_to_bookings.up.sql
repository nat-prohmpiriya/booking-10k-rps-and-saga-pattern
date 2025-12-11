-- 000014_add_payment_id_to_bookings.up.sql
-- Add payment_id column to bookings table for linking with payments
-- Note: No FK constraint since payments table may have different id type

ALTER TABLE bookings
ADD COLUMN payment_id VARCHAR(255);

-- Index for looking up bookings by payment
CREATE INDEX idx_bookings_payment_id ON bookings(payment_id);
