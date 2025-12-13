-- ============================================================================
-- Payment Service Database Schema (Microservice Architecture)
-- ============================================================================
-- This database is isolated for the payment-service only.
-- NO foreign key constraints to tables in other databases (tenants, users, bookings).
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

-- Payment status enum
CREATE TYPE payment_status AS ENUM (
    'pending',         -- Payment initiated
    'processing',      -- Being processed by payment gateway
    'succeeded',       -- Payment successful
    'failed',          -- Payment failed
    'cancelled',       -- Payment cancelled
    'refund_pending',  -- Refund in progress
    'refunded'         -- Full refund completed
);

-- Payment method enum
CREATE TYPE payment_method AS ENUM (
    'credit_card',
    'debit_card',
    'bank_transfer',
    'promptpay',
    'wallet',
    'cash'
);

-- Payments table for tracking payment transactions
CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Cross-database references (NO FK constraints - validated at application level)
    tenant_id UUID NOT NULL,      -- Reference to auth_db.tenants
    booking_id UUID NOT NULL,     -- Reference to booking_db.bookings
    user_id UUID NOT NULL,        -- Reference to auth_db.users

    -- Payment details
    amount DECIMAL(12, 2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'THB',
    method payment_method,

    -- Status
    status payment_status DEFAULT 'pending',

    -- Payment gateway details
    gateway VARCHAR(50), -- e.g., 'stripe', 'omise', '2c2p'
    gateway_payment_id VARCHAR(255),     -- Stripe PaymentIntent ID
    gateway_charge_id VARCHAR(255),      -- Stripe Charge ID
    gateway_customer_id VARCHAR(255),    -- Stripe Customer ID
    gateway_response JSONB,

    -- Idempotency
    idempotency_key VARCHAR(255) UNIQUE,

    -- Card details (masked)
    card_last_four VARCHAR(4),
    card_brand VARCHAR(20),

    -- Processing timestamps
    initiated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE,

    -- Refund tracking
    refund_amount DECIMAL(12, 2),
    refund_reason TEXT,
    refunded_at TIMESTAMP WITH TIME ZONE,

    -- Error handling
    error_code VARCHAR(50),
    error_message TEXT,
    retry_count INT DEFAULT 0,

    -- Metadata
    metadata JSONB DEFAULT '{}',

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_payments_tenant_id ON payments(tenant_id);
CREATE INDEX idx_payments_booking_id ON payments(booking_id);
CREATE INDEX idx_payments_user_id ON payments(user_id);
CREATE INDEX idx_payments_status ON payments(status);
CREATE INDEX idx_payments_gateway_payment_id ON payments(gateway_payment_id);
CREATE INDEX idx_payments_gateway_customer_id ON payments(gateway_customer_id);
CREATE INDEX idx_payments_idempotency_key ON payments(idempotency_key);
CREATE INDEX idx_payments_created_at ON payments(created_at);

-- Index for pending payments (for timeout handling)
CREATE INDEX idx_payments_pending ON payments(created_at)
    WHERE status IN ('pending', 'processing');

-- Index for user's payment history
CREATE INDEX idx_payments_user_history ON payments(user_id, created_at DESC);

-- Trigger for updated_at
CREATE TRIGGER update_payments_updated_at
    BEFORE UPDATE ON payments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
