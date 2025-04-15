-- ==========================================
-- MARKET DATA DOWNLOAD FUNCTIONS
-- ==========================================

-- Create market data download job
CREATE OR REPLACE FUNCTION create_market_data_download_job(
    p_symbol_id INT,
    p_symbol VARCHAR(20),
    p_source VARCHAR(50),
    p_timeframe timeframe_type,
    p_start_date TIMESTAMPTZ,
    p_end_date TIMESTAMPTZ
)
RETURNS INT AS $$
DECLARE
    new_job_id INT;
BEGIN
    INSERT INTO market_data_download_jobs (
        symbol_id,
        symbol,
        source,
        timeframe,
        start_date,
        end_date,
        status,
        progress,
        created_at,
        updated_at
    )
    VALUES (
        p_symbol_id,
        p_symbol,
        p_source,
        p_timeframe,
        p_start_date,
        p_end_date,
        'pending',
        0,
        NOW(),
        NOW()
    )
    RETURNING id INTO new_job_id;
    
    RETURN new_job_id;
END;
$$ LANGUAGE plpgsql;

-- Update market data download job status
CREATE OR REPLACE FUNCTION update_market_data_download_job_status(
    p_job_id INT,
    p_status VARCHAR(20),
    p_progress NUMERIC(5,2),
    p_processed_candles INT DEFAULT 0,
    p_total_candles INT DEFAULT 0,
    p_retries INT DEFAULT 0,
    p_error TEXT DEFAULT NULL
)
RETURNS BOOLEAN AS $$
DECLARE
    affected_rows INT;
BEGIN
    UPDATE market_data_download_jobs
    SET 
        status = p_status,
        progress = p_progress,
        processed_candles = p_processed_candles,
        total_candles = p_total_candles,
        retries = p_retries,
        error = p_error,
        updated_at = NOW(),
        last_processed_time = 
            CASE WHEN p_status = 'in_progress' THEN NOW() ELSE last_processed_time END
    WHERE 
        id = p_job_id;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows > 0;
END;
$$ LANGUAGE plpgsql;

-- Get active market data download jobs
CREATE OR REPLACE FUNCTION get_active_market_data_download_jobs(p_source VARCHAR DEFAULT NULL)
RETURNS TABLE (
    id INT,
    symbol_id INT,
    symbol VARCHAR(20),
    source VARCHAR(50),
    timeframe timeframe_type,
    start_date TIMESTAMPTZ,
    end_date TIMESTAMPTZ,
    status VARCHAR(20),
    progress NUMERIC(5,2),
    total_candles INT,
    processed_candles INT,
    retries INT,
    error TEXT,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    last_processed_time TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT *
    FROM market_data_download_jobs
    WHERE status IN ('pending', 'in_progress')
      AND (p_source IS NULL OR source = p_source)
    ORDER BY created_at DESC;
END;
$$ LANGUAGE plpgsql;