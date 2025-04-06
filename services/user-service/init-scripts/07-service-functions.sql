-- User Service Database - Service Communication Functions

-- Log service communication
CREATE OR REPLACE FUNCTION log_service_communication(
    p_source_service VARCHAR,
    p_target_service VARCHAR,
    p_endpoint VARCHAR,
    p_http_method VARCHAR,
    p_status_code INT,
    p_request_id VARCHAR DEFAULT NULL,
    p_user_id INT DEFAULT NULL,
    p_error_message TEXT DEFAULT NULL
)
RETURNS INT AS $$
DECLARE
    log_id INT;
BEGIN
    INSERT INTO service_communication_log (
        source_service,
        target_service,
        endpoint,
        http_method,
        status_code,
        request_id,
        user_id,
        error_message,
        created_at
    )
    VALUES (
        p_source_service,
        p_target_service,
        p_endpoint,
        p_http_method,
        p_status_code,
        p_request_id,
        p_user_id,
        p_error_message,
        NOW()
    )
    RETURNING id INTO log_id;
    
    RETURN log_id;
END;
$$ LANGUAGE plpgsql;

-- Get service communication logs
CREATE OR REPLACE FUNCTION get_service_communication_logs(
    p_source_service VARCHAR DEFAULT NULL,
    p_target_service VARCHAR DEFAULT NULL,
    p_user_id INT DEFAULT NULL,
    p_limit INT DEFAULT 100,
    p_offset INT DEFAULT 0
)
RETURNS TABLE (
    id INT,
    source_service VARCHAR,
    target_service VARCHAR,
    endpoint VARCHAR,
    http_method VARCHAR,
    status_code INT,
    request_id VARCHAR,
    user_id INT,
    error_message TEXT,
    created_at TIMESTAMP
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        scl.id,
        scl.source_service,
        scl.target_service,
        scl.endpoint,
        scl.http_method,
        scl.status_code,
        scl.request_id,
        scl.user_id,
        scl.error_message,
        scl.created_at
    FROM 
        service_communication_log scl
    WHERE
        (p_source_service IS NULL OR scl.source_service = p_source_service) AND
        (p_target_service IS NULL OR scl.target_service = p_target_service) AND
        (p_user_id IS NULL OR scl.user_id = p_user_id)
    ORDER BY 
        scl.created_at DESC
    LIMIT p_limit
    OFFSET p_offset;
END;
$$ LANGUAGE plpgsql;

-- Create service key
CREATE OR REPLACE FUNCTION create_service_key(
    p_service_name VARCHAR(50),
    p_key_hash VARCHAR(255)
)
RETURNS INT AS $$
DECLARE
    key_id INT;
BEGIN
    INSERT INTO service_keys (
        service_name,
        key_hash,
        is_active,
        created_at,
        updated_at
    )
    VALUES (
        p_service_name,
        p_key_hash,
        TRUE,
        NOW(),
        NOW()
    )
    RETURNING id INTO key_id;
    
    RETURN key_id;
END;
$$ LANGUAGE plpgsql;

-- Update service key
CREATE OR REPLACE FUNCTION update_service_key(
    p_service_name VARCHAR(50),
    p_key_hash VARCHAR(255),
    p_is_active BOOLEAN DEFAULT TRUE
)
RETURNS BOOLEAN AS $$
DECLARE
    affected_rows INT;
BEGIN
    UPDATE service_keys
    SET 
        key_hash = p_key_hash,
        is_active = p_is_active,
        updated_at = NOW()
    WHERE 
        service_name = p_service_name;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows > 0;
END;
$$ LANGUAGE plpgsql;

-- Delete service key
CREATE OR REPLACE FUNCTION delete_service_key(p_service_name VARCHAR(50))
RETURNS BOOLEAN AS $$
DECLARE
    affected_rows INT;
BEGIN
    DELETE FROM service_keys
    WHERE service_name = p_service_name;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows > 0;
END;
$$ LANGUAGE plpgsql;