-- User Service Database - User Functions

-- Get user by ID
CREATE OR REPLACE FUNCTION get_user_by_id(p_user_id INT)
RETURNS TABLE (
    id INT,
    username VARCHAR(50),
    email VARCHAR(100),
    password_hash VARCHAR(255),
    role user_role,
    profile_photo_url VARCHAR(255),
    is_active BOOLEAN,
    last_login TIMESTAMP,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
) AS $$
BEGIN
    RETURN QUERY
    SELECT u.id, u.username, u.email, u.password_hash, u.role, u.profile_photo_url, u.is_active, u.last_login, u.created_at, u.updated_at
    FROM users u
    WHERE u.id = p_user_id;
END;
$$ LANGUAGE plpgsql;

-- Get user by email
CREATE OR REPLACE FUNCTION get_user_by_email(p_email VARCHAR)
RETURNS TABLE (
    id INT,
    username VARCHAR(50),
    email VARCHAR(100),
    password_hash VARCHAR(255),
    role user_role,
    profile_photo_url VARCHAR(255),
    is_active BOOLEAN,
    last_login TIMESTAMP,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
) AS $$
BEGIN
    RETURN QUERY
    SELECT u.id, u.username, u.email, u.password_hash, u.role, u.profile_photo_url, u.is_active, u.last_login, u.created_at, u.updated_at
    FROM users u
    WHERE u.email = p_email;
END;
$$ LANGUAGE plpgsql;

-- Get user with details
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
    WHERE v_user_details.id = p_user_id;
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

-- Update user
CREATE OR REPLACE FUNCTION update_user(
    p_user_id INT,
    p_username VARCHAR(50) DEFAULT NULL,
    p_email VARCHAR(100) DEFAULT NULL,
    p_profile_photo_url VARCHAR(255) DEFAULT NULL,
    p_is_active BOOLEAN DEFAULT NULL
)
RETURNS BOOLEAN AS $$
DECLARE
    affected_rows INT;
BEGIN
    UPDATE users
    SET 
        username = COALESCE(p_username, username),
        email = COALESCE(p_email, email),
        profile_photo_url = COALESCE(p_profile_photo_url, profile_photo_url),
        is_active = COALESCE(p_is_active, is_active),
        updated_at = NOW()
    WHERE 
        id = p_user_id;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows > 0;
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

-- Get user count
CREATE OR REPLACE FUNCTION get_user_count()
RETURNS INT AS $$
DECLARE
    user_count INT;
BEGIN
    SELECT COUNT(*) INTO user_count FROM users;
    RETURN user_count;
END;
$$ LANGUAGE plpgsql;

-- List users with pagination
CREATE OR REPLACE FUNCTION list_users(p_limit INT, p_offset INT)
RETURNS TABLE (
    id INT,
    username VARCHAR(50),
    email VARCHAR(100),
    role user_role,
    profile_photo_url VARCHAR(255),
    is_active BOOLEAN,
    last_login TIMESTAMP,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
) AS $$
BEGIN
    RETURN QUERY
    SELECT u.id, u.username, u.email, u.role, u.profile_photo_url, u.is_active, u.last_login, u.created_at, u.updated_at
    FROM users u
    ORDER BY u.id
    LIMIT p_limit
    OFFSET p_offset;
END;
$$ LANGUAGE plpgsql;

-- Get user's role
CREATE OR REPLACE FUNCTION get_user_role(p_user_id INT)
RETURNS user_role AS $$
DECLARE
    user_role user_role;
BEGIN
    SELECT role INTO user_role
    FROM users
    WHERE id = p_user_id;
    
    RETURN user_role;
END;
$$ LANGUAGE plpgsql;