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
    i.created_at,
    i.updated_at,
    ARRAY(
        SELECT jsonb_build_object(
            'id', p.id,
            'name', p.parameter_name,
            'type', p.parameter_type,
            'is_required', p.is_required,
            'min_value', p.min_value,
            'max_value', p.max_value,
            'default_value', p.default_value,
            'description', p.description,
            'enum_values', (
                SELECT jsonb_agg(jsonb_build_object(
                    'id', ev.id,
                    'value', ev.enum_value,
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

-- Create the get_indicators function
CREATE OR REPLACE FUNCTION get_indicators(
    p_search VARCHAR,
    p_categories VARCHAR[]
)
RETURNS TABLE (
    id INT,
    name VARCHAR(50),
    description TEXT,
    category VARCHAR(50),
    formula TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    parameters JSONB[]
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        i.id,
        i.name,
        i.description,
        i.category,
        i.formula,
        i.created_at,
        i.updated_at,
        ARRAY(
            SELECT jsonb_build_object(
                'id', p.id,
                'name', p.parameter_name,
                'type', p.parameter_type,
                'is_required', p.is_required,
                'min_value', p.min_value,
                'max_value', p.max_value,
                'default_value', p.default_value,
                'description', p.description,
                'enum_values', (
                    SELECT jsonb_agg(jsonb_build_object(
                        'id', ev.id,
                        'value', ev.enum_value,
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
        indicators i
    WHERE
        (p_search IS NULL OR i.name ILIKE '%' || p_search || '%' OR i.description ILIKE '%' || p_search || '%')
        AND (p_categories IS NULL OR array_length(p_categories, 1) IS NULL OR i.category = ANY(p_categories))
    ORDER BY 
        i.name;
END;
$$ LANGUAGE plpgsql;

-- Update get_indicator_by_id to be compatible with the new get_indicators function
CREATE OR REPLACE FUNCTION get_indicator_by_id(p_indicator_id INT)
RETURNS TABLE (
    id INT,
    name VARCHAR(50),
    description TEXT,
    category VARCHAR(50),
    formula TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    parameters JSONB[]
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        i.id,
        i.name,
        i.description,
        i.category,
        i.formula,
        i.created_at,
        i.updated_at,
        ARRAY(
            SELECT jsonb_build_object(
                'id', p.id,
                'name', p.parameter_name,
                'type', p.parameter_type,
                'is_required', p.is_required,
                'min_value', p.min_value,
                'max_value', p.max_value,
                'default_value', p.default_value,
                'description', p.description,
                'enum_values', (
                    SELECT jsonb_agg(jsonb_build_object(
                        'id', ev.id,
                        'value', ev.enum_value,
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
        i.category,
        COUNT(*) AS count
    FROM 
        indicators i
    GROUP BY 
        i.category
    ORDER BY 
        i.category;
END;
$$ LANGUAGE plpgsql;