-- Strategy Service Indicator Functions
-- File: 10-indicator-functions.sql
-- Contains functions for technical indicators

-- Create view for indicators with parameters
CREATE OR REPLACE VIEW v_indicators_with_parameters AS
SELECT 
    i.id,
    i.name,
    i.description,
    i.category,
    i.formula,
    i.min_value,  
    i.max_value,  
    i.created_at,
    i.updated_at,
    ARRAY(
        SELECT jsonb_build_object(
            'id', p.id,
            'parameter_name', p.parameter_name,
            'parameter_type', p.parameter_type,
            'is_required', p.is_required,
            'min_value', p.min_value,
            'max_value', p.max_value,
            'default_value', p.default_value,
            'description', p.description,
            'enum_values', (
                SELECT jsonb_agg(jsonb_build_object(
                    'id', ev.id,
                    'enum_value', ev.enum_value,
                    'display_name', ev.display_name
                ))
                FROM parameter_enum_values ev
                WHERE ev.parameter_id = p.id
            )
        )
        FROM indicator_parameters p
        WHERE p.indicator_id = i.id
    ) AS parameters
FROM 
    indicators i;

-- Get indicators with parameters and enum values
CREATE OR REPLACE FUNCTION get_indicators(
    p_search VARCHAR,
    p_categories VARCHAR[],
    p_active BOOLEAN = NULL,
    p_is_admin BOOLEAN = FALSE,
    p_limit INT DEFAULT 20,
    p_offset INT DEFAULT 0
)
RETURNS TABLE (
    id INT,
    name VARCHAR(50),
    description TEXT,
    category VARCHAR(50),
    formula TEXT,
    min_value FLOAT,  
    max_value FLOAT,
    is_active BOOLEAN,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    parameters JSONB
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        i.id,
        i.name,
        i.description,
        i.category,
        i.formula,
        i.min_value,  
        i.max_value,
        i.is_active,
        i.created_at,
        i.updated_at,
        COALESCE(
            (SELECT jsonb_agg(jsonb_build_object(
                'id', p.id,
                'name', p.parameter_name,
                'type', p.parameter_type,
                'is_required', p.is_required,
                'min_value', p.min_value,
                'max_value', p.max_value,
                'default_value', p.default_value,
                'description', p.description,
                'is_public', p.is_public,
                'enum_values', COALESCE(
                    (SELECT jsonb_agg(jsonb_build_object(
                        'id', ev.id,
                        'enum_value', ev.enum_value,
                        'display_name', ev.display_name
                    ))
                    FROM parameter_enum_values ev
                    WHERE ev.parameter_id = p.id), '[]'::jsonb)
            ))
            FROM indicator_parameters p
            WHERE p.indicator_id = i.id
              -- Filter by is_public if not admin
              AND (p_is_admin OR p.is_public)
            ), '[]'::jsonb
        ) AS parameters
    FROM 
        indicators i
    WHERE
        (p_search IS NULL OR i.name ILIKE '%' || p_search || '%' OR i.description ILIKE '%' || p_search || '%')
        AND (p_categories IS NULL OR array_length(p_categories, 1) IS NULL OR i.category = ANY(p_categories))
        AND (p_active IS NULL OR i.is_active = p_active)
    ORDER BY 
        i.name
    LIMIT p_limit OFFSET p_offset;
END;
$$ LANGUAGE plpgsql;

-- Update get_indicator_by_id function
CREATE OR REPLACE FUNCTION get_indicator_by_id(
    p_indicator_id INT,
    p_is_admin BOOLEAN = FALSE -- New parameter to control visibility
)
RETURNS TABLE (
    id INT,
    name VARCHAR(50),
    description TEXT,
    category VARCHAR(50),
    formula TEXT,
    min_value FLOAT, 
    max_value FLOAT,
    is_active BOOLEAN,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    parameters JSONB
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        i.id,
        i.name,
        i.description,
        i.category,
        i.formula,
        i.min_value, 
        i.max_value,
        i.is_active,
        i.created_at,
        i.updated_at,
        COALESCE(
            (SELECT jsonb_agg(jsonb_build_object(
                'id', p.id,
                'name', p.parameter_name,
                'type', p.parameter_type,
                'is_required', p.is_required,
                'min_value', p.min_value,
                'max_value', p.max_value,
                'default_value', p.default_value,
                'description', p.description,
                'is_public', p.is_public,
                'enum_values', COALESCE(
                    (SELECT jsonb_agg(jsonb_build_object(
                        'id', ev.id,
                        'enum_value', ev.enum_value,
                        'display_name', ev.display_name
                    ))
                    FROM parameter_enum_values ev
                    WHERE ev.parameter_id = p.id), '[]'::jsonb)
            ))
            FROM indicator_parameters p
            WHERE p.indicator_id = i.id
              -- Filter by is_public if not admin
              AND (p_is_admin OR p.is_public)
            ), '[]'::jsonb
        ) AS parameters
    FROM 
        indicators i
    WHERE
        i.id = p_indicator_id;
END;
$$ LANGUAGE plpgsql;

-- Get indicator categories
CREATE OR REPLACE FUNCTION get_indicator_categories()
RETURNS TABLE (
    category VARCHAR(50),
    count BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COALESCE(i.category, 'Uncategorized') AS category,
        COUNT(*) AS count
    FROM 
        indicators i
    GROUP BY 
        COALESCE(i.category, 'Uncategorized')
    ORDER BY 
        category;
END;
$$ LANGUAGE plpgsql;

-- Delete indicator
CREATE OR REPLACE FUNCTION delete_indicator(p_id INT) 
RETURNS BOOLEAN AS $$
DECLARE
    affected_rows INT;
BEGIN
    DELETE FROM indicators WHERE id = p_id;
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows > 0;
END;
$$ LANGUAGE plpgsql;

-- Update indicator
CREATE OR REPLACE FUNCTION update_indicator(
    p_id INT,
    p_name VARCHAR(50),
    p_description TEXT,
    p_category VARCHAR(50),
    p_formula TEXT,
    p_min_value FLOAT,  
    p_max_value FLOAT,
    p_is_active BOOLEAN
) 
RETURNS BOOLEAN AS $$
DECLARE
    affected_rows INT;
BEGIN
    UPDATE indicators 
    SET 
        name = p_name,
        description = p_description,
        category = p_category,
        formula = p_formula,
        min_value = p_min_value,  
        max_value = p_max_value,
        is_active = p_is_active,
        updated_at = NOW()
    WHERE id = p_id;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows > 0;
END;
$$ LANGUAGE plpgsql;

-- Delete parameter
CREATE OR REPLACE FUNCTION delete_parameter(p_id INT) 
RETURNS BOOLEAN AS $$
DECLARE
    affected_rows INT;
BEGIN
    DELETE FROM indicator_parameters WHERE id = p_id;
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows > 0;
END;
$$ LANGUAGE plpgsql;

-- Update parameter
CREATE OR REPLACE FUNCTION update_parameter(
    p_id INT,
    p_parameter_name VARCHAR(50),
    p_parameter_type VARCHAR(20),
    p_is_required BOOLEAN,
    p_min_value FLOAT,
    p_max_value FLOAT,
    p_default_value VARCHAR(50),
    p_description TEXT
) 
RETURNS BOOLEAN AS $$
DECLARE
    affected_rows INT;
BEGIN
    UPDATE indicator_parameters 
    SET 
        parameter_name = p_parameter_name,
        parameter_type = p_parameter_type,
        is_required = p_is_required,
        min_value = p_min_value,
        max_value = p_max_value,
        default_value = p_default_value,
        description = p_description
    WHERE id = p_id;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows > 0;
END;
$$ LANGUAGE plpgsql;

-- Delete enum value
CREATE OR REPLACE FUNCTION delete_enum_value(p_id INT) 
RETURNS BOOLEAN AS $$
DECLARE
    affected_rows INT;
BEGIN
    DELETE FROM parameter_enum_values WHERE id = p_id;
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows > 0;
END;
$$ LANGUAGE plpgsql;

-- Update enum value
CREATE OR REPLACE FUNCTION update_enum_value(
    p_id INT,
    p_enum_value VARCHAR(50),
    p_display_name VARCHAR(100)
) 
RETURNS BOOLEAN AS $$
DECLARE
    affected_rows INT;
BEGIN
    UPDATE parameter_enum_values 
    SET 
        enum_value = p_enum_value,
        display_name = p_display_name
    WHERE id = p_id;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows > 0;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION count_indicators(
    p_search VARCHAR,
    p_categories VARCHAR[],
    p_active BOOLEAN = NULL
)
RETURNS BIGINT AS $$
DECLARE
    indicator_count BIGINT;
BEGIN
    SELECT COUNT(*)
    INTO indicator_count
    FROM indicators i
    WHERE
        (p_search IS NULL OR i.name ILIKE '%' || p_search || '%' OR i.description ILIKE '%' || p_search || '%')
        AND (p_categories IS NULL OR array_length(p_categories, 1) IS NULL OR i.category = ANY(p_categories))
        AND (p_active IS NULL OR i.is_active = p_active);
        
    RETURN indicator_count;
END;
$$ LANGUAGE plpgsql;