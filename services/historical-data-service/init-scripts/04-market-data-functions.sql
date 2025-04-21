-- ==========================================
-- MARKET DATA (CANDLES) FUNCTIONS
-- ==========================================

-- Fixed get_candles function with proper interval casting
CREATE OR REPLACE FUNCTION get_candles(
    p_symbol_id INT,
    p_timeframe timeframe_type,
    p_start_time TIMESTAMPTZ,
    p_end_time TIMESTAMPTZ,
    p_limit INT DEFAULT NULL
)
RETURNS TABLE (
    symbol_id INT,
    candle_time TIMESTAMPTZ,
    open NUMERIC(20,8),
    high NUMERIC(20,8),
    low NUMERIC(20,8),
    close NUMERIC(20,8),
    volume NUMERIC(20,8)
) AS $$
DECLARE
    interval_minutes INT;
BEGIN
    -- Map timeframe to minutes
    CASE p_timeframe
        WHEN '1m' THEN interval_minutes := 1;
        WHEN '5m' THEN interval_minutes := 5;
        WHEN '15m' THEN interval_minutes := 15;
        WHEN '30m' THEN interval_minutes := 30;
        WHEN '1h' THEN interval_minutes := 60;
        WHEN '4h' THEN interval_minutes := 240;
        WHEN '1d' THEN interval_minutes := 1440;
        WHEN '1w' THEN interval_minutes := 10080;
        ELSE interval_minutes := 1; -- Default to 1 minute
    END CASE;
    
    -- Return 1m data directly
    IF interval_minutes = 1 THEN
        RETURN QUERY
        SELECT c.symbol_id, c.candle_time, c.open, c.high, c.low, c.close, c.volume
        FROM candles c
        WHERE c.symbol_id = p_symbol_id
          AND c.candle_time BETWEEN p_start_time AND p_end_time
        ORDER BY c.candle_time DESC
        LIMIT p_limit;
    ELSE
        -- Aggregate candles for higher timeframes
        RETURN QUERY
        SELECT 
            c.symbol_id,
            time_bucket((interval_minutes || ' minutes')::interval, c.candle_time) AS candle_time,
            FIRST(c.open, c.candle_time) AS open,
            MAX(c.high) AS high,
            MIN(c.low) AS low,
            LAST(c.close, c.candle_time) AS close,
            SUM(c.volume) AS volume
        FROM candles c
        WHERE c.symbol_id = p_symbol_id
          AND c.candle_time BETWEEN p_start_time AND p_end_time
        GROUP BY c.symbol_id, time_bucket((interval_minutes || ' minutes')::interval, c.candle_time)
        ORDER BY candle_time DESC
        LIMIT p_limit;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Fix for insert_candles function
CREATE OR REPLACE FUNCTION insert_candles(
    p_candles JSONB
)
RETURNS INT AS $$
DECLARE
    candle_record JSONB;
    inserted_count INT := 0;
BEGIN
    FOR candle_record IN SELECT * FROM jsonb_array_elements(p_candles)
    LOOP
        INSERT INTO candles (symbol_id, candle_time, open, high, low, close, volume)
        VALUES (
            (candle_record->>'symbol_id')::INT,
            (candle_record->>'candle_time')::TIMESTAMPTZ,
            (candle_record->>'open')::NUMERIC(20,8),
            (candle_record->>'high')::NUMERIC(20,8),
            (candle_record->>'low')::NUMERIC(20,8),
            (candle_record->>'close')::NUMERIC(20,8),
            (candle_record->>'volume')::NUMERIC(20,8)
        )
        ON CONFLICT (symbol_id, candle_time)
        DO UPDATE SET
            open = EXCLUDED.open,
            high = EXCLUDED.high,
            low = EXCLUDED.low,
            close = EXCLUDED.close,
            volume = EXCLUDED.volume;
        
        inserted_count := inserted_count + 1;
    END LOOP;
    
    RETURN inserted_count;
END;
$$ LANGUAGE plpgsql;

-- Get continuous data ranges for a symbol - Updated to use "candle_time" column
CREATE OR REPLACE FUNCTION get_symbol_data_ranges(
    p_symbol_id INT,
    p_timeframe timeframe_type
)
RETURNS TABLE (
    start_date TIMESTAMPTZ,
    end_date TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    WITH dates AS (
        SELECT 
            candle_time,
            LEAD(candle_time) OVER (ORDER BY candle_time) as next_time
        FROM candles
        WHERE symbol_id = p_symbol_id
        ORDER BY candle_time
    )
    SELECT 
        MIN(candle_time) as start_date,
        MAX(candle_time) as end_date
    FROM (
        SELECT 
            candle_time,
            next_time,
            CASE WHEN next_time IS NULL OR next_time > candle_time + INTERVAL '1 day' THEN 1 ELSE 0 END as is_gap,
            SUM(CASE WHEN next_time IS NULL OR next_time > candle_time + INTERVAL '1 day' THEN 1 ELSE 0 END) OVER (ORDER BY candle_time) as group_id
        FROM dates
    ) t
    GROUP BY group_id
    ORDER BY start_date;
END;
$$ LANGUAGE plpgsql;