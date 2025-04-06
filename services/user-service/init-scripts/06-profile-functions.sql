-- User Service Database - Profile Functions

-- Get user profile photo URL
CREATE OR REPLACE FUNCTION get_profile_photo_url(p_user_id INT)
RETURNS VARCHAR AS $$
DECLARE
    photo_url VARCHAR;
BEGIN
    SELECT profile_photo_url INTO photo_url
    FROM users
    WHERE id = p_user_id;
    
    RETURN photo_url;
END;
$$ LANGUAGE plpgsql;

-- Update user profile photo URL
CREATE OR REPLACE FUNCTION update_profile_photo(
    p_user_id INT,
    p_photo_url VARCHAR(255)
)
RETURNS BOOLEAN AS $$
DECLARE
    affected_rows INT;
BEGIN
    UPDATE users
    SET 
        profile_photo_url = p_photo_url,
        updated_at = NOW()
    WHERE 
        id = p_user_id;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows > 0;
END;
$$ LANGUAGE plpgsql;

-- Clear user profile photo URL
CREATE OR REPLACE FUNCTION clear_profile_photo(p_user_id INT)
RETURNS BOOLEAN AS $$
DECLARE
    affected_rows INT;
BEGIN
    UPDATE users
    SET 
        profile_photo_url = NULL,
        updated_at = NOW()
    WHERE 
        id = p_user_id;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows > 0;
END;
$$ LANGUAGE plpgsql;