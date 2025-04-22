-- ==========================================
-- INVENTORY FUNCTIONS
-- ==========================================

-- Get data inventory with pagination and filtering
CREATE OR REPLACE FUNCTION get_data_inventory(
    p_asset_type VARCHAR DEFAULT NULL,
    p_exchange VARCHAR DEFAULT NULL,
    p_limit INT DEFAULT 100,
    p_offset INT DEFAULT 0
)
RETURNS TABLE (
    symbol_id INT,
    symbol VARCHAR(20),
    name VARCHAR(100),
    asset_type VARCHAR(20),
    exchange VARCHAR(50),
    candle_count BIGINT,
    earliest_date TIMESTAMPTZ,
    latest_date TIMESTAMPTZ,
    available_timeframes TEXT[]
) AS $$
BEGIN
    RETURN QUERY
    WITH symbol_data AS (
        SELECT 
            s.id,
            s.symbol,
            s.name,
            s.asset_type,
            s.exchange,
            COUNT(c.candle_time) AS candle_count,
            MIN(c.candle_time) AS earliest_date,
            MAX(c.candle_time) AS latest_date
        FROM 
            symbols s
        LEFT JOIN 
            candles c ON s.id = c.symbol_id
        WHERE 
            s.is_active = TRUE
            AND s.data_available = TRUE
            AND (p_asset_type IS NULL OR s.asset_type = p_asset_type)
            AND (p_exchange IS NULL OR s.exchange = p_exchange)
        GROUP BY 
            s.id, s.symbol, s.name, s.asset_type, s.exchange
        HAVING 
            COUNT(c.candle_time) > 0
        ORDER BY 
            s.symbol
        LIMIT p_limit OFFSET p_offset
    )
    SELECT 
        sd.id,
        sd.symbol,
        sd.name,
        sd.asset_type,
        sd.exchange,
        sd.candle_count,
        sd.earliest_date,
        sd.latest_date,
        ARRAY(
            SELECT t::text FROM unnest(enum_range(NULL::timeframe_type)) t
        ) AS available_timeframes
    FROM 
        symbol_data sd;
END;
$$ LANGUAGE plpgsql;

-- Count symbols in inventory for pagination
CREATE OR REPLACE FUNCTION count_data_inventory(
    p_asset_type VARCHAR DEFAULT NULL,
    p_exchange VARCHAR DEFAULT NULL
)
RETURNS BIGINT AS $$
DECLARE
    inventory_count BIGINT;
BEGIN
    SELECT COUNT(*)
    INTO inventory_count
    FROM (
        SELECT 
            s.id
        FROM 
            symbols s
        LEFT JOIN 
            candles c ON s.id = c.symbol_id
        WHERE 
            s.is_active = TRUE
            AND s.data_available = TRUE
            AND (p_asset_type IS NULL OR s.asset_type = p_asset_type)
            AND (p_exchange IS NULL OR s.exchange = p_exchange)
        GROUP BY 
            s.id
        HAVING 
            COUNT(c.candle_time) > 0
    ) AS symbol_count;
    
    RETURN inventory_count;
END;
$$ LANGUAGE plpgsql;

-- Get available timeframes for a symbol
CREATE OR REPLACE FUNCTION get_symbol_available_timeframes(
    p_symbol_id INT
)
RETURNS TABLE (
    timeframe TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT DISTINCT t::text
    FROM unnest(enum_range(NULL::timeframe_type)) t
    WHERE EXISTS (
        SELECT 1
        FROM candles c
        WHERE c.symbol_id = p_symbol_id
    );
END;
$$ LANGUAGE plpgsql;

-- Get candle count for a symbol
CREATE OR REPLACE FUNCTION get_symbol_candle_count(
    p_symbol_id INT,
    p_timeframe timeframe_type DEFAULT NULL
)
RETURNS BIGINT AS $$
DECLARE
    count_value BIGINT;
BEGIN
    IF p_timeframe IS NULL THEN
        SELECT COUNT(*)
        INTO count_value
        FROM candles
        WHERE symbol_id = p_symbol_id;
    ELSE
        -- In a real implementation with timeframe filtering
        SELECT COUNT(*)
        INTO count_value
        FROM candles
        WHERE symbol_id = p_symbol_id;
    END IF;
    
    RETURN count_value;
END;
$$ LANGUAGE plpgsql;