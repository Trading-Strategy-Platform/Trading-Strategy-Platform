-- Get all symbols
CREATE OR REPLACE FUNCTION get_symbols(
    p_search_term VARCHAR DEFAULT NULL,
    p_asset_type VARCHAR DEFAULT NULL,
    p_exchange VARCHAR DEFAULT NULL,
    p_sort_by VARCHAR DEFAULT 'symbol',
    p_sort_direction VARCHAR DEFAULT 'ASC',
    p_limit INT DEFAULT 100,
    p_offset INT DEFAULT 0
)
RETURNS TABLE (
    id INT,
    symbol VARCHAR(20),
    name VARCHAR(100),
    asset_type VARCHAR(20),
    exchange VARCHAR(50),
    is_active BOOLEAN,
    data_available BOOLEAN,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
) AS $$
BEGIN
    -- Validate sort field
    IF p_sort_by NOT IN ('symbol', 'name', 'asset_type', 'exchange', 'created_at', 'data_available') THEN
        p_sort_by := 'symbol';
    END IF;
    
    -- Normalize sort direction
    p_sort_direction := UPPER(p_sort_direction);
    IF p_sort_direction NOT IN ('ASC', 'DESC') THEN
        p_sort_direction := 'ASC';
    END IF;

    RETURN QUERY
    SELECT 
        s.id,
        s.symbol,
        s.name,
        s.asset_type,
        s.exchange,
        s.is_active,
        s.data_available,
        s.created_at,
        s.updated_at
    FROM 
        symbols s
    WHERE 
        s.is_active = TRUE
        AND (
            p_search_term IS NULL 
            OR s.symbol ILIKE '%' || p_search_term || '%' 
            OR s.name ILIKE '%' || p_search_term || '%'
        )
        AND (
            p_asset_type IS NULL 
            OR s.asset_type = p_asset_type
        )
        AND (
            p_exchange IS NULL 
            OR s.exchange = p_exchange
        )
    ORDER BY 
        CASE WHEN p_sort_by = 'symbol' AND p_sort_direction = 'ASC' THEN s.symbol END ASC,
        CASE WHEN p_sort_by = 'symbol' AND p_sort_direction = 'DESC' THEN s.symbol END DESC,
        CASE WHEN p_sort_by = 'name' AND p_sort_direction = 'ASC' THEN s.name END ASC,
        CASE WHEN p_sort_by = 'name' AND p_sort_direction = 'DESC' THEN s.name END DESC,
        CASE WHEN p_sort_by = 'asset_type' AND p_sort_direction = 'ASC' THEN s.asset_type END ASC,
        CASE WHEN p_sort_by = 'asset_type' AND p_sort_direction = 'DESC' THEN s.asset_type END DESC,
        CASE WHEN p_sort_by = 'exchange' AND p_sort_direction = 'ASC' THEN s.exchange END ASC,
        CASE WHEN p_sort_by = 'exchange' AND p_sort_direction = 'DESC' THEN s.exchange END DESC,
        CASE WHEN p_sort_by = 'created_at' AND p_sort_direction = 'ASC' THEN s.created_at END ASC,
        CASE WHEN p_sort_by = 'created_at' AND p_sort_direction = 'DESC' THEN s.created_at END DESC,
        CASE WHEN p_sort_by = 'data_available' AND p_sort_direction = 'ASC' THEN s.data_available END ASC,
        CASE WHEN p_sort_by = 'data_available' AND p_sort_direction = 'DESC' THEN s.data_available END DESC
    LIMIT p_limit OFFSET p_offset;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION count_symbols(
    p_search_term VARCHAR DEFAULT NULL,
    p_asset_type VARCHAR DEFAULT NULL,
    p_exchange VARCHAR DEFAULT NULL
)
RETURNS BIGINT AS $$
DECLARE
    symbol_count BIGINT;
BEGIN
    SELECT COUNT(*)
    INTO symbol_count
    FROM symbols s
    WHERE 
        s.is_active = TRUE
        AND (
            p_search_term IS NULL 
            OR s.symbol ILIKE '%' || p_search_term || '%' 
            OR s.name ILIKE '%' || p_search_term || '%'
        )
        AND (
            p_asset_type IS NULL 
            OR s.asset_type = p_asset_type
        )
        AND (
            p_exchange IS NULL 
            OR s.exchange = p_exchange
        );
    
    RETURN symbol_count;
END;
$$ LANGUAGE plpgsql;

-- Add new symbol
CREATE OR REPLACE FUNCTION add_symbol(
    p_symbol VARCHAR(20),
    p_name VARCHAR(100),
    p_asset_type VARCHAR(20),
    p_exchange VARCHAR(50) DEFAULT NULL
)
RETURNS INT AS $$
DECLARE
    new_symbol_id INT;
BEGIN
    -- Check symbol uniqueness
    PERFORM 1 FROM symbols 
    WHERE symbol = p_symbol;
    
    IF FOUND THEN
        RAISE EXCEPTION 'Symbol already exists';
    END IF;
    
    INSERT INTO symbols (
        symbol,
        name,
        asset_type,
        exchange,
        is_active,
        data_available,
        created_at,
        updated_at
    )
    VALUES (
        p_symbol,
        p_name,
        p_asset_type,
        p_exchange,
        TRUE,
        FALSE,
        NOW(),
        NOW()
    )
    RETURNING id INTO new_symbol_id;
    
    RETURN new_symbol_id;
END;
$$ LANGUAGE plpgsql;

-- Update symbol
CREATE OR REPLACE FUNCTION update_symbol(
    p_symbol_id INT,
    p_symbol VARCHAR(20) DEFAULT NULL,
    p_name VARCHAR(100) DEFAULT NULL,
    p_asset_type VARCHAR(20) DEFAULT NULL,
    p_exchange VARCHAR(50) DEFAULT NULL
)
RETURNS BOOLEAN AS $$
DECLARE
    affected_rows INT;
BEGIN
    -- Check symbol uniqueness if it's being changed
    IF p_symbol IS NOT NULL THEN
        PERFORM 1 FROM symbols 
        WHERE symbol = p_symbol AND id != p_symbol_id;
        
        IF FOUND THEN
            RAISE EXCEPTION 'Symbol already exists';
        END IF;
    END IF;
    
    UPDATE symbols
    SET 
        symbol = COALESCE(p_symbol, symbol),
        name = COALESCE(p_name, name),
        asset_type = COALESCE(p_asset_type, asset_type),
        exchange = COALESCE(p_exchange, exchange),
        updated_at = NOW()
    WHERE 
        id = p_symbol_id;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows > 0;
END;
$$ LANGUAGE plpgsql;

-- Delete symbol (mark as inactive)
CREATE OR REPLACE FUNCTION delete_symbol(p_symbol_id INT)
RETURNS BOOLEAN AS $$
DECLARE
    affected_rows INT;
BEGIN
    UPDATE symbols
    SET 
        is_active = FALSE,
        updated_at = NOW()
    WHERE 
        id = p_symbol_id;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows > 0;
END;
$$ LANGUAGE plpgsql;

-- Get asset types
CREATE OR REPLACE FUNCTION get_asset_types()
RETURNS TABLE (
    asset_type VARCHAR(20),
    count BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        s.asset_type,
        COUNT(*) AS count
    FROM 
        symbols s
    WHERE 
        s.is_active = TRUE
    GROUP BY 
        s.asset_type
    ORDER BY 
        s.asset_type;
END;
$$ LANGUAGE plpgsql;

-- Get exchanges
CREATE OR REPLACE FUNCTION get_exchanges()
RETURNS TABLE (
    exchange VARCHAR(50),
    count BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        s.exchange,
        COUNT(*) AS count
    FROM 
        symbols s
    WHERE 
        s.is_active = TRUE
        AND s.exchange IS NOT NULL
    GROUP BY 
        s.exchange
    ORDER BY 
        s.exchange;
END;
$$ LANGUAGE plpgsql;

-- Update symbol data availability
CREATE OR REPLACE FUNCTION update_symbol_data_availability(
    p_symbol_id INT,
    p_available BOOLEAN
)
RETURNS BOOLEAN AS $$
DECLARE
    affected_rows INT;
BEGIN
    UPDATE symbols
    SET 
        data_available = p_available,
        updated_at = NOW()
    WHERE 
        id = p_symbol_id;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows > 0;
END;
$$ LANGUAGE plpgsql;