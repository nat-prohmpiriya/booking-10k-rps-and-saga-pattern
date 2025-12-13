-- ============================================================================
-- Booking Service Database Schema (Microservice Architecture)
-- ============================================================================
-- This database is isolated for the booking-service only.
-- NO foreign key constraints to tables in other databases (tenants, users, events, shows, seat_zones).
-- Cross-database references are stored as UUIDs and validated at application level.
-- ============================================================================

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Function for updating updated_at column
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Booking status enum
CREATE TYPE booking_status AS ENUM (
    'pending',      -- Just created, waiting for payment
    'reserved',     -- Seats reserved in Redis, payment in progress
    'confirmed',    -- Payment successful, booking complete
    'cancelled',    -- Cancelled by user or system
    'expired',      -- Reservation TTL expired
    'refunded'      -- Refund completed
);

-- Bookings table - core transaction for ticket reservations
CREATE TABLE IF NOT EXISTS bookings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Cross-database references (NO FK constraints - validated at application level)
    tenant_id UUID NOT NULL,      -- Reference to auth_db.tenants
    user_id UUID NOT NULL,        -- Reference to auth_db.users
    event_id UUID NOT NULL,       -- Reference to ticket_db.events
    show_id UUID NOT NULL,        -- Reference to ticket_db.shows
    zone_id UUID NOT NULL,        -- Reference to ticket_db.seat_zones

    -- Booking details
    quantity INT NOT NULL CHECK (quantity > 0),
    unit_price DECIMAL(12, 2) NOT NULL,
    total_amount DECIMAL(12, 2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'THB',

    -- Status tracking
    status booking_status DEFAULT 'pending',
    status_reason TEXT, -- Reason for cancellation/expiry

    -- Idempotency
    idempotency_key VARCHAR(255) UNIQUE,

    -- Reservation tracking
    reserved_at TIMESTAMP WITH TIME ZONE,
    reservation_expires_at TIMESTAMP WITH TIME ZONE,

    -- Confirmation
    confirmed_at TIMESTAMP WITH TIME ZONE,
    confirmation_code VARCHAR(20), -- Human readable code like "BK-ABC123"

    -- Cancellation
    cancelled_at TIMESTAMP WITH TIME ZONE,
    cancelled_by UUID, -- Reference to auth_db.users (NO FK)

    -- Payment reference (cross-database reference to payment_db)
    payment_id VARCHAR(255),

    -- Metadata
    metadata JSONB DEFAULT '{}',

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for common queries
CREATE INDEX idx_bookings_tenant_id ON bookings(tenant_id);
CREATE INDEX idx_bookings_user_id ON bookings(user_id);
CREATE INDEX idx_bookings_event_id ON bookings(event_id);
CREATE INDEX idx_bookings_show_id ON bookings(show_id);
CREATE INDEX idx_bookings_zone_id ON bookings(zone_id);
CREATE INDEX idx_bookings_status ON bookings(status);
CREATE INDEX idx_bookings_idempotency_key ON bookings(idempotency_key);
CREATE INDEX idx_bookings_confirmation_code ON bookings(confirmation_code);
CREATE INDEX idx_bookings_payment_id ON bookings(payment_id);
CREATE INDEX idx_bookings_created_at ON bookings(created_at);

-- Index for finding expired reservations (for cleanup worker)
CREATE INDEX idx_bookings_pending_expired ON bookings(reservation_expires_at)
    WHERE status = 'reserved' AND reservation_expires_at IS NOT NULL;

-- Index for user's booking history
CREATE INDEX idx_bookings_user_history ON bookings(user_id, created_at DESC);

-- Trigger for updated_at
CREATE TRIGGER update_bookings_updated_at
    BEFORE UPDATE ON bookings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Function to generate confirmation code
CREATE OR REPLACE FUNCTION generate_confirmation_code()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'confirmed' AND NEW.confirmation_code IS NULL THEN
        NEW.confirmation_code = 'BK-' || UPPER(SUBSTRING(MD5(NEW.id::TEXT || NOW()::TEXT) FROM 1 FOR 6));
    END IF;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER generate_booking_confirmation_code
    BEFORE UPDATE ON bookings
    FOR EACH ROW
    WHEN (NEW.status = 'confirmed' AND OLD.status != 'confirmed')
    EXECUTE FUNCTION generate_confirmation_code();
