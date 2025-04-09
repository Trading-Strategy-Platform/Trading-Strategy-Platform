-- User Service Database Schema - Core Tables and Types

-- Create types
CREATE TYPE "user_role" AS ENUM (
  'admin',
  'user'
);

CREATE TYPE "notification_type" AS ENUM (
  'backtest_completed',
  'strategy_purchased',
  'account_update',
  'system_maintenance',
  'strategy_shared',
  'price_alert'
);

-- Create core tables
CREATE TABLE IF NOT EXISTS "users" (
  "id" SERIAL PRIMARY KEY,
  "username" varchar(50) UNIQUE NOT NULL,
  "email" varchar(100) UNIQUE NOT NULL,
  "password_hash" varchar(255) NOT NULL,
  "role" user_role NOT NULL DEFAULT 'user',
  "profile_photo_url" varchar(255),
  "is_active" boolean NOT NULL DEFAULT true,
  "last_login" timestamp,
  "created_at" timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  "updated_at" timestamp
);

CREATE TABLE IF NOT EXISTS "user_sessions" (
  "id" SERIAL PRIMARY KEY,
  "user_id" int NOT NULL,
  "token" varchar(255) NOT NULL,
  "expires_at" timestamp NOT NULL,
  "ip_address" varchar(45),
  "user_agent" varchar(255),
  "created_at" timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP)
);

CREATE TABLE IF NOT EXISTS "user_preferences" (
  "id" SERIAL PRIMARY KEY,
  "user_id" int UNIQUE NOT NULL,
  "theme" varchar(20) DEFAULT 'light',
  "default_timeframe" varchar(10) DEFAULT '1h',
  "chart_preferences" jsonb,
  "notification_settings" jsonb,
  "created_at" timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  "updated_at" timestamp
);

CREATE TABLE IF NOT EXISTS "notifications" (
  "id" SERIAL PRIMARY KEY,
  "user_id" int NOT NULL,
  "type" notification_type NOT NULL,
  "title" varchar(100) NOT NULL,
  "message" text NOT NULL,
  "is_read" boolean NOT NULL DEFAULT false,
  "link" varchar(255),
  "created_at" timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP)
);

-- Service key table for secure service-to-service communication
CREATE TABLE IF NOT EXISTS "service_keys" (
  "id" SERIAL PRIMARY KEY,
  "service_name" varchar(50) UNIQUE NOT NULL,
  "key_hash" varchar(255) NOT NULL,
  "is_active" boolean NOT NULL DEFAULT true,
  "created_at" timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  "updated_at" timestamp,
  "last_used" timestamp
);

-- Service communication log table for debugging and auditing
CREATE TABLE IF NOT EXISTS "service_communication_log" (
  "id" SERIAL PRIMARY KEY,
  "source_service" varchar(50) NOT NULL,
  "target_service" varchar(50) NOT NULL,
  "endpoint" varchar(255) NOT NULL,
  "http_method" varchar(10) NOT NULL,
  "status_code" int,
  "request_id" varchar(36),
  "user_id" int,
  "error_message" text,
  "created_at" timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP)
);

-- Create indexes
CREATE INDEX IF NOT EXISTS "idx_users_email" ON "users" ("email");
CREATE INDEX IF NOT EXISTS "idx_notifications_user_id" ON "notifications" ("user_id", "is_read");
CREATE INDEX IF NOT EXISTS "idx_service_comm_log_service" ON "service_communication_log" ("source_service", "created_at");
CREATE INDEX IF NOT EXISTS "idx_service_comm_log_user" ON "service_communication_log" ("user_id", "created_at");

-- Add foreign keys
ALTER TABLE "user_sessions" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE;
ALTER TABLE "user_preferences" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE;
ALTER TABLE "notifications" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE;

-- Insert default users
INSERT INTO users (username, email, password_hash, role, is_active, created_at) VALUES 
('admin', 'admin@example.com', '$2a$10$RViVMeSSFklziKBzn8aVyOFGS/nQEfqNSQRg/RAUrgpvmjMc4jtvO', 'admin', true, NOW()),
('user', 'user@example.com', '$2a$10$RViVMeSSFklziKBzn8aVyOFGS/nQEfqNSQRg/RAUrgpvmjMc4jtvO', 'user', true, NOW())
ON CONFLICT DO NOTHING;

-- Insert default service keys
INSERT INTO service_keys (service_name, key_hash, is_active, created_at) VALUES 
('strategy-service', '$2a$10$G/oAFA62ZzBAEjXTDkMudegfXv5Jm5tjH9T/aO2iVrzgup2RpTswy', true, NOW()),
('historical-service', '$2a$10$G/oAFA62ZzBAEjXTDkMudegfXv5Jm5tjH9T/aO2iVrzgup2RpTswy', true, NOW()),
('media-service', '$2a$10$G/oAFA62ZzBAEjXTDkMudegfXv5Jm5tjH9T/aO2iVrzgup2RpTswy', true, NOW())
ON CONFLICT DO NOTHING;

-- Create a user detail view
CREATE OR REPLACE VIEW v_user_details AS
SELECT
    u.id,
    u.username,
    u.email,
    u.role,
    COALESCE(u.profile_photo_url, '') as profile_photo_url,
    u.is_active,
    u.last_login,
    u.created_at,
    u.updated_at,
    (
        SELECT COUNT(*) 
        FROM notifications n 
        WHERE n.user_id = u.id AND n.is_read = FALSE
    ) AS unread_notifications_count,
    COALESCE(p.theme, 'light') as theme,
    COALESCE(p.default_timeframe, '1h') as default_timeframe,
    COALESCE(p.chart_preferences, '{}'::jsonb) as chart_preferences,
    COALESCE(p.notification_settings, '{}'::jsonb) as notification_settings
FROM
    users u
    LEFT JOIN user_preferences p ON u.id = p.user_id;