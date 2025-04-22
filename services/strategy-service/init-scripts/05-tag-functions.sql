-- Strategy Service Tag Functions
-- File: 06-tag-functions.sql
-- Contains functions for tag operations

-- Get all strategy tags with enhanced filtering and sorting
CREATE OR REPLACE FUNCTION get_all_tags(
    p_search VARCHAR DEFAULT NULL,
    p_sort_by VARCHAR DEFAULT 'name',
    p_sort_direction VARCHAR DEFAULT 'ASC',
    p_limit INT DEFAULT 100,
    p_offset INT DEFAULT 0
)
RETURNS TABLE (
    id INT,
    name VARCHAR(50),
    strategy_count BIGINT  -- Added count of strategies using this tag
) AS $$
BEGIN
    -- Validate sort field
    IF p_sort_by NOT IN ('name', 'strategy_count', 'id') THEN
        p_sort_by := 'name'; -- Default sort by name
    END IF;
    
    -- Validate sort direction
    IF UPPER(p_sort_direction) NOT IN ('ASC', 'DESC') THEN
        p_sort_direction := 'ASC'; -- Default ascending for tags
    ELSE
        p_sort_direction := UPPER(p_sort_direction);
    END IF;

    RETURN QUERY
    SELECT 
        t.id, 
        t.name,
        COUNT(DISTINCT stm.strategy_id) AS strategy_count
    FROM 
        strategy_tags t
        LEFT JOIN strategy_tag_mappings stm ON t.id = stm.tag_id
    WHERE 
        p_search IS NULL 
        OR t.name ILIKE '%' || p_search || '%'
    GROUP BY t.id, t.name
    ORDER BY
        CASE WHEN p_sort_by = 'name' AND p_sort_direction = 'ASC' THEN t.name END ASC,
        CASE WHEN p_sort_by = 'name' AND p_sort_direction = 'DESC' THEN t.name END DESC,
        CASE WHEN p_sort_by = 'strategy_count' AND p_sort_direction = 'ASC' THEN COUNT(DISTINCT stm.strategy_id) END ASC,
        CASE WHEN p_sort_by = 'strategy_count' AND p_sort_direction = 'DESC' THEN COUNT(DISTINCT stm.strategy_id) END DESC,
        CASE WHEN p_sort_by = 'id' AND p_sort_direction = 'ASC' THEN t.id END ASC,
        CASE WHEN p_sort_by = 'id' AND p_sort_direction = 'DESC' THEN t.id END DESC
    LIMIT p_limit OFFSET p_offset;
END;
$$ LANGUAGE plpgsql;

-- Get tag by ID
CREATE OR REPLACE FUNCTION get_tag_by_id(
    p_id INT
)
RETURNS TABLE (
    id INT,
    name VARCHAR(50),
    strategy_count BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        t.id, 
        t.name,
        COUNT(DISTINCT stm.strategy_id) AS strategy_count
    FROM 
        strategy_tags t
        LEFT JOIN strategy_tag_mappings stm ON t.id = stm.tag_id
    WHERE 
        t.id = p_id
    GROUP BY t.id, t.name;
END;
$$ LANGUAGE plpgsql;

-- Create a new tag
CREATE OR REPLACE FUNCTION create_tag(
    p_name VARCHAR(50)
)
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

-- Update a tag
CREATE OR REPLACE FUNCTION update_tag(
    p_id INT, 
    p_name VARCHAR(50)
)
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

-- Delete a tag
CREATE OR REPLACE FUNCTION delete_tag(
    p_id INT
)
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

-- Count tags with filtering
CREATE OR REPLACE FUNCTION count_tags(
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

-- Get popular tags with usage count
CREATE OR REPLACE FUNCTION get_popular_tags(
    p_limit INT DEFAULT 10
)
RETURNS TABLE (
    id INT,
    name VARCHAR(50),
    strategy_count BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        t.id, 
        t.name,
        COUNT(DISTINCT stm.strategy_id) AS strategy_count
    FROM 
        strategy_tags t
        JOIN strategy_tag_mappings stm ON t.id = stm.tag_id
    GROUP BY t.id, t.name
    ORDER BY 
        strategy_count DESC,
        t.name ASC
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;