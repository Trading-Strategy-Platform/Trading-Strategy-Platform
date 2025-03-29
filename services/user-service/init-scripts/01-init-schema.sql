-- User Service Database Schema
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

-- Indexes
CREATE INDEX IF NOT EXISTS "idx_notifications_user_id" ON "notifications" ("user_id", "is_read");
CREATE INDEX ON "users" ("email");

-- Foreign Keys
ALTER TABLE "user_sessions" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE;
ALTER TABLE "user_preferences" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE;
ALTER TABLE "notifications" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE;

-- Insert default roles
INSERT INTO users (username, email, password_hash, role, is_active, created_at) VALUES 
('admin', 'admin@example.com', '$2a$10$ZYwnmqZPir2/HTIvTl7za.gqbZwAUUyXOLJFOuln9wVcJoEgJNTlm', 'admin', true, NOW()),
('user', 'user@example.com', '$2a$10$ZYwnmqZPir2/HTIvTl7za.gqbZwAUUyXOLJFOuln9wVcJoEgJNTlm', 'user', true, NOW())
ON CONFLICT DO NOTHING;

-- ==========================================
-- USER MANAGEMENT FUNCTIONS
-- ==========================================

-- Get user with active notification count and preferences
CREATE OR REPLACE VIEW v_user_details AS
SELECT
    u.id,
    u.username,
    u.email,
    u.role,
    u.profile_photo_url,
    u.is_active,
    u.last_login,
    u.created_at,
    u.updated_at,
    (
        SELECT COUNT(*) 
        FROM notifications n 
        WHERE n.user_id = u.id AND n.is_read = FALSE
    ) AS unread_notifications_count,
    p.theme,
    p.default_timeframe,
    p.chart_preferences,
    p.notification_settings
FROM
    users u
    LEFT JOIN user_preferences p ON u.id = p.user_id;

-- Get user details
CREATE OR REPLACE FUNCTION get_user_details(p_user_id INT)
RETURNS TABLE (
    id INT,
    username VARCHAR(50),
    email VARCHAR(100),
    role user_role,
    profile_photo_url VARCHAR(255),
    is_active BOOLEAN,
    last_login TIMESTAMP,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    unread_notifications_count BIGINT,
    theme VARCHAR(20),
    default_timeframe VARCHAR(10),
    chart_preferences JSONB,
    notification_settings JSONB
) AS $$
BEGIN
    RETURN QUERY
    SELECT * FROM v_user_details
    WHERE id = p_user_id;
END;
$$ LANGUAGE plpgsql;

-- Update user
CREATE OR REPLACE FUNCTION update_user(
    p_user_id INT,
    p_username VARCHAR(50) DEFAULT NULL,
    p_email VARCHAR(100) DEFAULT NULL,
    p_profile_photo_url VARCHAR(255) DEFAULT NULL,
    p_theme VARCHAR(20) DEFAULT NULL,
    p_default_timeframe VARCHAR(10) DEFAULT NULL,
    p_chart_preferences JSONB DEFAULT NULL,
    p_notification_settings JSONB DEFAULT NULL
)
RETURNS BOOLEAN AS $$
DECLARE
    affected_rows INT;
BEGIN
    -- Update user table
    IF p_username IS NOT NULL OR p_email IS NOT NULL OR p_profile_photo_url IS NOT NULL THEN
        UPDATE users
        SET 
            username = COALESCE(p_username, username),
            email = COALESCE(p_email, email),
            profile_photo_url = COALESCE(p_profile_photo_url, profile_photo_url),
            updated_at = NOW()
        WHERE 
            id = p_user_id;
        
        GET DIAGNOSTICS affected_rows = ROW_COUNT;
        
        IF affected_rows = 0 THEN
            RETURN FALSE;
        END IF;
    END IF;
    
    -- Update preferences
    IF p_theme IS NOT NULL OR p_default_timeframe IS NOT NULL OR p_chart_preferences IS NOT NULL OR p_notification_settings IS NOT NULL THEN
        -- Check if preferences exist
        PERFORM 1 FROM user_preferences WHERE user_id = p_user_id;
        
        IF FOUND THEN
            -- Update existing preferences
            UPDATE user_preferences
            SET 
                theme = COALESCE(p_theme, theme),
                default_timeframe = COALESCE(p_default_timeframe, default_timeframe),
                chart_preferences = COALESCE(p_chart_preferences, chart_preferences),
                notification_settings = COALESCE(p_notification_settings, notification_settings),
                updated_at = NOW()
            WHERE 
                user_id = p_user_id;
        ELSE
            -- Create new preferences
            INSERT INTO user_preferences (
                user_id,
                theme,
                default_timeframe,
                chart_preferences,
                notification_settings,
                created_at,
                updated_at
            )
            VALUES (
                p_user_id,
                p_theme,
                p_default_timeframe,
                p_chart_preferences,
                p_notification_settings,
                NOW(),
                NOW()
            );
        END IF;
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Delete user (mark as inactive)
CREATE OR REPLACE FUNCTION delete_user(p_user_id INT)
RETURNS BOOLEAN AS $$
DECLARE
    affected_rows INT;
BEGIN
    UPDATE users
    SET 
        is_active = FALSE,
        updated_at = NOW()
    WHERE 
        id = p_user_id;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows > 0;
END;
$$ LANGUAGE plpgsql;

-- Create new user
CREATE OR REPLACE FUNCTION create_user(
    p_username VARCHAR(50),
    p_email VARCHAR(100),
    p_password_hash VARCHAR(255),
    p_role user_role DEFAULT 'user'::user_role,
    p_profile_photo_url VARCHAR(255) DEFAULT NULL
)
RETURNS INT AS $$
DECLARE
    new_user_id INT;
BEGIN
    -- Check username and email uniqueness
    IF EXISTS (SELECT 1 FROM users WHERE username = p_username) THEN
        RAISE EXCEPTION 'Username already exists';
    END IF;
    
    IF EXISTS (SELECT 1 FROM users WHERE email = p_email) THEN
        RAISE EXCEPTION 'Email already exists';
    END IF;
    
    -- Insert user
    INSERT INTO users (
        username,
        email,
        password_hash,
        role,
        profile_photo_url,
        is_active,
        created_at,
        updated_at
    )
    VALUES (
        p_username,
        p_email,
        p_password_hash,
        p_role,
        p_profile_photo_url,
        TRUE,
        NOW(),
        NOW()
    )
    RETURNING id INTO new_user_id;
    
    -- Create default preferences
    INSERT INTO user_preferences (
        user_id,
        theme,
        default_timeframe,
        chart_preferences,
        notification_settings,
        created_at,
        updated_at
    )
    VALUES (
        new_user_id,
        'light',
        '1h',
        '{}'::jsonb,
        '{}'::jsonb,
        NOW(),
        NOW()
    );
    
    RETURN new_user_id;
END;
$$ LANGUAGE plpgsql;

-- Update user password
CREATE OR REPLACE FUNCTION update_user_password(
    p_user_id INT,
    p_new_password_hash VARCHAR(255)
)
RETURNS BOOLEAN AS $$
DECLARE
    affected_rows INT;
BEGIN
    UPDATE users
    SET 
        password_hash = p_new_password_hash,
        updated_at = NOW()
    WHERE 
        id = p_user_id;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows > 0;
END;
$$ LANGUAGE plpgsql;

-- Update last login timestamp
CREATE OR REPLACE FUNCTION update_last_login(p_user_id INT)
RETURNS BOOLEAN AS $$
DECLARE
    affected_rows INT;
BEGIN
    UPDATE users
    SET 
        last_login = NOW()
    WHERE 
        id = p_user_id;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows > 0;
END;
$$ LANGUAGE plpgsql;

-- ==========================================
-- NOTIFICATIONS FUNCTIONS
-- ==========================================

-- Get active notifications for a user
CREATE OR REPLACE FUNCTION get_active_notifications(p_user_id INT)
RETURNS TABLE (
    id INT,
    type notification_type,
    title VARCHAR(100),
    message TEXT,
    is_read BOOLEAN,
    link VARCHAR(255),
    created_at TIMESTAMP
) AS $$
BEGIN
    RETURN QUERY
    SELECT n.id, n.type, n.title, n.message, n.is_read, n.link, n.created_at
    FROM notifications n
    WHERE n.user_id = p_user_id
    ORDER BY n.created_at DESC;
END;
$$ LANGUAGE plpgsql;

-- Get active notification count for a user
CREATE OR REPLACE FUNCTION get_unread_notification_count(p_user_id INT)
RETURNS INTEGER AS $$
DECLARE
    notification_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO notification_count
    FROM notifications
    WHERE user_id = p_user_id AND is_read = FALSE;
    
    RETURN notification_count;
END;
$$ LANGUAGE plpgsql;

-- Get ALL notifications for a user (including read ones)
CREATE OR REPLACE FUNCTION get_all_notifications(
    p_user_id INT,
    p_limit INT DEFAULT 100,
    p_offset INT DEFAULT 0
)
RETURNS TABLE (
    id INT,
    type notification_type,
    title VARCHAR(100),
    message TEXT,
    is_read BOOLEAN,
    link VARCHAR(255),
    created_at TIMESTAMP
) AS $$
BEGIN
    RETURN QUERY
    SELECT n.id, n.type, n.title, n.message, n.is_read, n.link, n.created_at
    FROM notifications n
    WHERE n.user_id = p_user_id
    ORDER BY n.created_at DESC
    LIMIT p_limit
    OFFSET p_offset;
END;
$$ LANGUAGE plpgsql;

-- Mark notification as read
CREATE OR REPLACE FUNCTION mark_notification_as_read(p_notification_id INT)
RETURNS BOOLEAN AS $$
DECLARE
    affected_rows INTEGER;
BEGIN
    UPDATE notifications
    SET is_read = TRUE
    WHERE id = p_notification_id;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows > 0;
END;
$$ LANGUAGE plpgsql;

-- Add notification
CREATE OR REPLACE FUNCTION add_notification(
    p_user_id INT,
    p_type notification_type,
    p_title VARCHAR(100),
    p_message TEXT,
    p_link VARCHAR(255) DEFAULT NULL
)
RETURNS INT AS $$
DECLARE
    new_notification_id INT;
BEGIN
    INSERT INTO notifications (user_id, type, title, message, link, is_read, created_at)
    VALUES (p_user_id, p_type, p_title, p_message, p_link, FALSE, NOW())
    RETURNING id INTO new_notification_id;
    
    RETURN new_notification_id;
END;
$$ LANGUAGE plpgsql;

-- Mark all notifications as read for a user
CREATE OR REPLACE FUNCTION mark_all_notifications_as_read(p_user_id INT)
RETURNS INTEGER AS $$
DECLARE
    affected_rows INTEGER;
BEGIN
    UPDATE notifications
    SET is_read = TRUE
    WHERE user_id = p_user_id AND is_read = FALSE;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows;
END;
$$ LANGUAGE plpgsql;