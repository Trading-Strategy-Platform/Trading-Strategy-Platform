-- User Service Database - Authentication Functions

-- Get user password by ID
CREATE OR REPLACE FUNCTION get_user_password_by_id(p_user_id INT)
RETURNS VARCHAR AS $$
DECLARE
    pw_hash VARCHAR;
BEGIN
    SELECT password_hash INTO pw_hash
    FROM users
    WHERE id = p_user_id;
    
    RETURN pw_hash;
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

-- Create user session
CREATE OR REPLACE FUNCTION create_user_session(
    p_user_id INT,
    p_token VARCHAR(255),
    p_expires_at TIMESTAMP,
    p_ip_address VARCHAR(45) DEFAULT NULL,
    p_user_agent VARCHAR(255) DEFAULT NULL
)
RETURNS INT AS $$
DECLARE
    session_id INT;
BEGIN
    INSERT INTO user_sessions (
        user_id,
        token,
        expires_at,
        ip_address,
        user_agent,
        created_at
    )
    VALUES (
        p_user_id,
        p_token,
        p_expires_at,
        p_ip_address,
        p_user_agent,
        NOW()
    )
    RETURNING id INTO session_id;
    
    RETURN session_id;
END;
$$ LANGUAGE plpgsql;

-- Get user session by token
CREATE OR REPLACE FUNCTION get_user_session_by_token(p_token VARCHAR)
RETURNS TABLE (
    id INT,
    user_id INT,
    token VARCHAR(255),
    expires_at TIMESTAMP,
    ip_address VARCHAR(45),
    user_agent VARCHAR(255),
    created_at TIMESTAMP
) AS $$
BEGIN
    RETURN QUERY
    SELECT s.id, s.user_id, s.token, s.expires_at, s.ip_address, s.user_agent, s.created_at
    FROM user_sessions s
    WHERE s.token = p_token AND s.expires_at > NOW();
END;
$$ LANGUAGE plpgsql;

-- Delete user session by token
CREATE OR REPLACE FUNCTION delete_user_session(p_token VARCHAR)
RETURNS BOOLEAN AS $$
DECLARE
    affected_rows INT;
BEGIN
    DELETE FROM user_sessions
    WHERE token = p_token;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows > 0;
END;
$$ LANGUAGE plpgsql;

-- Delete all user sessions
CREATE OR REPLACE FUNCTION delete_user_sessions(p_user_id INT)
RETURNS INT AS $$
DECLARE
    affected_rows INT;
BEGIN
    DELETE FROM user_sessions
    WHERE user_id = p_user_id;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows;
END;
$$ LANGUAGE plpgsql;

-- Validate service key
CREATE OR REPLACE FUNCTION validate_service_key(p_service_name VARCHAR, p_key_hash VARCHAR)
RETURNS BOOLEAN AS $$
DECLARE
    stored_key_hash VARCHAR;
    is_key_active BOOLEAN;
BEGIN
    -- Get the key hash and active status for the service
    SELECT key_hash, is_active INTO stored_key_hash, is_key_active
    FROM service_keys
    WHERE service_name = p_service_name;
    
    -- Update last used timestamp
    IF FOUND THEN
        UPDATE service_keys
        SET last_used = NOW()
        WHERE service_name = p_service_name;
    END IF;
    
    -- Check if key exists, is active, and matches
    RETURN FOUND AND is_key_active AND stored_key_hash = p_key_hash;
END;
$$ LANGUAGE plpgsql;