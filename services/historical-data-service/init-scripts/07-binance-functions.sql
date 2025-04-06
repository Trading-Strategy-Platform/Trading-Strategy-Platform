-- ==========================================
-- BINANCE DATA DOWNLOAD FUNCTIONS
-- ==========================================

-- Create binance download job
CREATE OR REPLACE FUNCTION create_binance_download_job(
    p_symbol_id INT,
    p_symbol VARCHAR(20),
    p_timeframe timeframe_type,
    p_start_date TIMESTAMPTZ,
    p_end_date TIMESTAMPTZ
)
RETURNS INT AS $$
DECLARE
    new_job_id INT;
BEGIN
    INSERT INTO binance_download_jobs (
        symbol_id,
        symbol,
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

-- Update binance download job status
CREATE OR REPLACE FUNCTION update_binance_download_job_status(
    p_job_id INT,
    p_status VARCHAR(20),
    p_progress NUMERIC(5,2),
    p_error TEXT DEFAULT NULL
)
RETURNS BOOLEAN AS $$
DECLARE
    affected_rows INT;
BEGIN
    UPDATE binance_download_jobs
    SET 
        status = p_status,
        progress = p_progress,
        error = p_error,
        updated_at = NOW()
    WHERE 
        id = p_job_id;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows > 0;
END;
$$ LANGUAGE plpgsql;

-- Get active binance download jobs
CREATE OR REPLACE FUNCTION get_active_binance_download_jobs()
RETURNS TABLE (
    id INT,
    symbol_id INT,
    symbol VARCHAR(20),
    timeframe timeframe_type,
    start_date TIMESTAMPTZ,
    end_date TIMESTAMPTZ,
    status VARCHAR(20),
    progress NUMERIC(5,2),
    error TEXT,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT *
    FROM binance_download_jobs
    WHERE status IN ('pending', 'in_progress')
    ORDER BY created_at DESC;
END;
$$ LANGUAGE plpgsql;