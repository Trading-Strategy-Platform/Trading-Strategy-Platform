-- Strategy Service Tag Functions
-- File: 06-tag-functions.sql
-- Contains functions for tag operations

-- Get strategy tags
CREATE OR REPLACE FUNCTION get_strategy_tags(
    p_search VARCHAR DEFAULT NULL,
    p_limit INT DEFAULT 100,
    p_offset INT DEFAULT 0
)
RETURNS TABLE (
    id INT,
    name VARCHAR(50)
) AS $$
BEGIN
    RETURN QUERY
    SELECT t.id, t.name
    FROM strategy_tags t
    WHERE 
        p_search IS NULL 
        OR t.name ILIKE '%' || p_search || '%'
    ORDER BY t.name
    LIMIT p_limit OFFSET p_offset;
END;
$$ LANGUAGE plpgsql;

-- Add strategy tag
CREATE OR REPLACE FUNCTION add_strategy_tag(p_name VARCHAR(50))
RETURNS INT AS $$
DECLARE
    new_tag_id INT;
BEGIN
    INSERT INTO strategy_tags (name)
    VALUES (p_name)
    RETURNING id INTO new_tag_id;
    
    RETURN new_tag_id;
END;
$$ LANGUAGE plpgsql;

-- Update strategy tag
CREATE OR REPLACE FUNCTION update_strategy_tag(p_id INT, p_name VARCHAR(50))
RETURNS BOOLEAN AS $$
DECLARE
    affected_rows INT;
BEGIN
    UPDATE strategy_tags
    SET name = p_name
    WHERE id = p_id;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows > 0;
END;
$$ LANGUAGE plpgsql;


-- Delete strategy tag
CREATE OR REPLACE FUNCTION delete_strategy_tag(p_id INT)
RETURNS BOOLEAN AS $$
DECLARE
    affected_rows INT;
BEGIN
    -- Check if tag is in use before deleting
    IF EXISTS (SELECT 1 FROM strategy_tag_mappings WHERE tag_id = p_id) THEN
        RETURN FALSE;
    END IF;
    
    DELETE FROM strategy_tags
    WHERE id = p_id;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows > 0;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION count_strategy_tags(
    p_search VARCHAR DEFAULT NULL
)
RETURNS BIGINT AS $$
DECLARE
    tag_count BIGINT;
BEGIN
    SELECT COUNT(*)
    INTO tag_count
    FROM strategy_tags t
    WHERE 
        p_search IS NULL 
        OR t.name ILIKE '%' || p_search || '%';
        
    RETURN tag_count;
END;
$$ LANGUAGE plpgsql;