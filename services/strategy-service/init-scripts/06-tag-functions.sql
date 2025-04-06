-- Strategy Service Tag Functions
-- File: 06-tag-functions.sql
-- Contains functions for tag operations

-- Get strategy tags
CREATE OR REPLACE FUNCTION get_strategy_tags()
RETURNS TABLE (
    id INT,
    name VARCHAR(50)
) AS $$
BEGIN
    RETURN QUERY
    SELECT t.id, t.name
    FROM strategy_tags t
    ORDER BY t.name;
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