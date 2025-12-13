-- =============================================================================
-- Booking Rush - Local Development Database Initialization
-- =============================================================================
-- This script runs automatically when PostgreSQL container starts
-- Creates all databases needed for microservices architecture
-- =============================================================================

-- Create databases
CREATE DATABASE auth_db;
CREATE DATABASE ticket_db;
CREATE DATABASE booking_db;
CREATE DATABASE payment_db;
CREATE DATABASE booking_rush;

-- Enable uuid-ossp extension for each database
\c auth_db
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

\c ticket_db
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

\c booking_db
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

\c payment_db
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

\c booking_rush
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
