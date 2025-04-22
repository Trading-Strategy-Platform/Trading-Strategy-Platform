-- ==========================================
-- DOWNLOAD JOB FUNCTIONS
-- ==========================================

-- Create a new market data download job
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
        total_candles,
        processed_candles,
        retries,
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
        0,
        0,
        0,
        NOW(),
        NOW()
    )
    RETURNING id INTO new_job_id;
    
    RETURN new_job_id;
END;
$$ LANGUAGE plpgsql;

-- Get download job by ID
CREATE OR REPLACE FUNCTION get_download_job_by_id(
    p_job_id INT
)
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
    SELECT 
        j.id,
        j.symbol_id,
        j.symbol,
        j.source,
        j.timeframe,
        j.start_date,
        j.end_date,
        j.status,
        j.progress,
        j.total_candles,
        j.processed_candles,
        j.retries,
        j.error,
        j.created_at,
        j.updated_at,
        j.last_processed_time
    FROM market_data_download_jobs j
    WHERE j.id = p_job_id;
END;
$$ LANGUAGE plpgsql;

-- Update download job status
CREATE OR REPLACE FUNCTION update_market_data_download_job_status(
    p_job_id INT,
    p_status VARCHAR(20),
    p_progress NUMERIC(5,2),
    p_processed_candles INT,
    p_total_candles INT,
    p_retries INT,
    p_error TEXT
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
        last_processed_time = CASE WHEN p_status = 'in_progress' THEN NOW() ELSE last_processed_time END
    WHERE 
        id = p_job_id;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows > 0;
END;
$$ LANGUAGE plpgsql;

-- Get active download jobs with filtering, sorting, and pagination
CREATE OR REPLACE FUNCTION get_active_download_jobs(
    p_source VARCHAR DEFAULT NULL,
    p_sort_by VARCHAR DEFAULT 'created_at',
    p_sort_direction VARCHAR DEFAULT 'DESC',
    p_limit INT DEFAULT 100,
    p_offset INT DEFAULT 0
)
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
    -- Validate sort field
    IF p_sort_by NOT IN ('id', 'symbol', 'source', 'status', 'progress', 'created_at', 'updated_at') THEN
        p_sort_by := 'created_at';
    END IF;
    
    -- Normalize sort direction
    p_sort_direction := UPPER(p_sort_direction);
    IF p_sort_direction NOT IN ('ASC', 'DESC') THEN
        p_sort_direction := 'DESC';
    END IF;

    RETURN QUERY
    SELECT 
        j.id,
        j.symbol_id,
        j.symbol,
        j.source,
        j.timeframe,
        j.start_date,
        j.end_date,
        j.status,
        j.progress,
        j.total_candles,
        j.processed_candles,
        j.retries,
        j.error,
        j.created_at,
        j.updated_at,
        j.last_processed_time
    FROM market_data_download_jobs j
    WHERE j.status IN ('pending', 'in_progress')
      AND (p_source IS NULL OR j.source = p_source)
    ORDER BY
        CASE WHEN p_sort_by = 'id' AND p_sort_direction = 'ASC' THEN j.id END ASC,
        CASE WHEN p_sort_by = 'id' AND p_sort_direction = 'DESC' THEN j.id END DESC,
        CASE WHEN p_sort_by = 'symbol' AND p_sort_direction = 'ASC' THEN j.symbol END ASC,
        CASE WHEN p_sort_by = 'symbol' AND p_sort_direction = 'DESC' THEN j.symbol END DESC,
        CASE WHEN p_sort_by = 'source' AND p_sort_direction = 'ASC' THEN j.source END ASC,
        CASE WHEN p_sort_by = 'source' AND p_sort_direction = 'DESC' THEN j.source END DESC,
        CASE WHEN p_sort_by = 'status' AND p_sort_direction = 'ASC' THEN j.status END ASC,
        CASE WHEN p_sort_by = 'status' AND p_sort_direction = 'DESC' THEN j.status END DESC,
        CASE WHEN p_sort_by = 'progress' AND p_sort_direction = 'ASC' THEN j.progress END ASC,
        CASE WHEN p_sort_by = 'progress' AND p_sort_direction = 'DESC' THEN j.progress END DESC,
        CASE WHEN p_sort_by = 'created_at' AND p_sort_direction = 'ASC' THEN j.created_at END ASC,
        CASE WHEN p_sort_by = 'created_at' AND p_sort_direction = 'DESC' THEN j.created_at END DESC,
        CASE WHEN p_sort_by = 'updated_at' AND p_sort_direction = 'ASC' THEN j.updated_at END ASC,
        CASE WHEN p_sort_by = 'updated_at' AND p_sort_direction = 'DESC' THEN j.updated_at END DESC
    LIMIT p_limit OFFSET p_offset;
END;
$$ LANGUAGE plpgsql;

-- Count active download jobs for pagination
CREATE OR REPLACE FUNCTION count_active_download_jobs(
    p_source VARCHAR DEFAULT NULL
)
RETURNS BIGINT AS $$
DECLARE
    job_count BIGINT;
BEGIN
    SELECT COUNT(*)
    INTO job_count
    FROM market_data_download_jobs j
    WHERE j.status IN ('pending', 'in_progress')
      AND (p_source IS NULL OR j.source = p_source);
    
    RETURN job_count;
END;
$$ LANGUAGE plpgsql;

-- Get download jobs summary with counts
CREATE OR REPLACE FUNCTION get_download_jobs_summary()
RETURNS TABLE (
    status VARCHAR(20),
    count BIGINT,
    last_24h BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        j.status, 
        COUNT(*) as count,
        SUM(CASE WHEN j.created_at > NOW() - INTERVAL '24 hours' THEN 1 ELSE 0 END) as last_24h
    FROM market_data_download_jobs j
    GROUP BY j.status
    ORDER BY j.status;
END;
$$ LANGUAGE plpgsql;

-- Get download jobs by status with filtering and pagination
CREATE OR REPLACE FUNCTION get_download_jobs_by_status(
    p_status VARCHAR,
    p_source VARCHAR DEFAULT NULL,
    p_symbol VARCHAR DEFAULT NULL,
    p_sort_by VARCHAR DEFAULT 'created_at',
    p_sort_direction VARCHAR DEFAULT 'DESC',
    p_limit INT DEFAULT 100,
    p_offset INT DEFAULT 0
)
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
    -- Validate sort field
    IF p_sort_by NOT IN ('id', 'symbol', 'source', 'status', 'progress', 'created_at', 'updated_at') THEN
        p_sort_by := 'created_at';
    END IF;
    
    -- Normalize sort direction
    p_sort_direction := UPPER(p_sort_direction);
    IF p_sort_direction NOT IN ('ASC', 'DESC') THEN
        p_sort_direction := 'DESC';
    END IF;

    RETURN QUERY
    SELECT 
        j.id,
        j.symbol_id,
        j.symbol,
        j.source,
        j.timeframe,
        j.start_date,
        j.end_date,
        j.status,
        j.progress,
        j.total_candles,
        j.processed_candles,
        j.retries,
        j.error,
        j.created_at,
        j.updated_at,
        j.last_processed_time
    FROM market_data_download_jobs j
    WHERE j.status = p_status
      AND (p_source IS NULL OR j.source = p_source)
      AND (p_symbol IS NULL OR j.symbol ILIKE '%' || p_symbol || '%')
    ORDER BY
        CASE WHEN p_sort_by = 'id' AND p_sort_direction = 'ASC' THEN j.id END ASC,
        CASE WHEN p_sort_by = 'id' AND p_sort_direction = 'DESC' THEN j.id END DESC,
        CASE WHEN p_sort_by = 'symbol' AND p_sort_direction = 'ASC' THEN j.symbol END ASC,
        CASE WHEN p_sort_by = 'symbol' AND p_sort_direction = 'DESC' THEN j.symbol END DESC,
        CASE WHEN p_sort_by = 'source' AND p_sort_direction = 'ASC' THEN j.source END ASC,
        CASE WHEN p_sort_by = 'source' AND p_sort_direction = 'DESC' THEN j.source END DESC,
        CASE WHEN p_sort_by = 'status' AND p_sort_direction = 'ASC' THEN j.status END ASC,
        CASE WHEN p_sort_by = 'status' AND p_sort_direction = 'DESC' THEN j.status END DESC,
        CASE WHEN p_sort_by = 'progress' AND p_sort_direction = 'ASC' THEN j.progress END ASC,
        CASE WHEN p_sort_by = 'progress' AND p_sort_direction = 'DESC' THEN j.progress END DESC,
        CASE WHEN p_sort_by = 'created_at' AND p_sort_direction = 'ASC' THEN j.created_at END ASC,
        CASE WHEN p_sort_by = 'created_at' AND p_sort_direction = 'DESC' THEN j.created_at END DESC,
        CASE WHEN p_sort_by = 'updated_at' AND p_sort_direction = 'ASC' THEN j.updated_at END ASC,
        CASE WHEN p_sort_by = 'updated_at' AND p_sort_direction = 'DESC' THEN j.updated_at END DESC
    LIMIT p_limit OFFSET p_offset;
END;
$$ LANGUAGE plpgsql;

-- Count download jobs by status for pagination
CREATE OR REPLACE FUNCTION count_download_jobs_by_status(
    p_status VARCHAR,
    p_source VARCHAR DEFAULT NULL,
    p_symbol VARCHAR DEFAULT NULL
)
RETURNS BIGINT AS $$
DECLARE
    job_count BIGINT;
BEGIN
    SELECT COUNT(*)
    INTO job_count
    FROM market_data_download_jobs j
    WHERE j.status = p_status
      AND (p_source IS NULL OR j.source = p_source)
      AND (p_symbol IS NULL OR j.symbol ILIKE '%' || p_symbol || '%');
    
    RETURN job_count;
END;
$$ LANGUAGE plpgsql;

-- Cancel download job
CREATE OR REPLACE FUNCTION cancel_download_job(
    p_job_id INT,
    p_force BOOLEAN DEFAULT FALSE
)
RETURNS BOOLEAN AS $$
DECLARE
    affected_rows INT;
    job_record market_data_download_jobs%ROWTYPE;
BEGIN
    -- Get the current job record
    SELECT * INTO job_record FROM market_data_download_jobs WHERE id = p_job_id;
    
    IF NOT FOUND THEN
        RETURN FALSE;
    END IF;
    
    -- Check if job can be cancelled
    IF NOT p_force AND job_record.status NOT IN ('pending', 'in_progress') AND job_record.processed_candles > 0 THEN
        RETURN FALSE;
    END IF;
    
    -- Update the job to cancelled
    UPDATE market_data_download_jobs
    SET 
        status = 'cancelled',
        updated_at = NOW(),
        error = COALESCE(error, '') || ' Cancelled by user'
    WHERE 
        id = p_job_id;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows > 0;
END;
$$ LANGUAGE plpgsql;