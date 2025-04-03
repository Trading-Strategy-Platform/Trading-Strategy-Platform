-- Historical Data Service Database Schema

-- Create timeframe_type enum
CREATE TYPE "timeframe_type" AS ENUM (
  '1m',
  '5m',
  '15m',
  '30m',
  '1h',
  '4h',
  '1d',
  '1w'
);

-- Extension for time series data
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Symbols table
CREATE TABLE IF NOT EXISTS "symbols" (
  "id" SERIAL PRIMARY KEY,
  "symbol" varchar(20) UNIQUE NOT NULL,
  "name" varchar(100) NOT NULL,
  "asset_type" varchar(20) NOT NULL,
  "exchange" varchar(50),
  "is_active" boolean NOT NULL DEFAULT true,
  "data_available" boolean NOT NULL DEFAULT false,
  "created_at" timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  "updated_at" timestamp
);

-- Candles table
CREATE TABLE IF NOT EXISTS "candles" (
  "symbol_id" int NOT NULL,
  "time" timestamp NOT NULL,
  "open" numeric(20,8) NOT NULL,
  "high" numeric(20,8) NOT NULL,
  "low" numeric(20,8) NOT NULL,
  "close" numeric(20,8) NOT NULL,
  "volume" numeric(20,8) NOT NULL,
  PRIMARY KEY ("symbol_id", "time")
);

-- Backtests table
CREATE TABLE IF NOT EXISTS "backtests" (
  "id" SERIAL PRIMARY KEY,
  "user_id" int NOT NULL,
  "strategy_id" int NOT NULL,
  "strategy_version" int NOT NULL,
  "name" varchar(100),
  "description" text,
  "timeframe" timeframe_type NOT NULL,
  "start_date" timestamp NOT NULL,
  "end_date" timestamp NOT NULL,
  "initial_capital" numeric(20,8) NOT NULL,
  "status" varchar(20) NOT NULL DEFAULT 'pending',
  "error_message" text,
  "created_at" timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  "updated_at" timestamp,
  "completed_at" timestamp
);

-- Backtest runs table
CREATE TABLE IF NOT EXISTS "backtest_runs" (
  "id" SERIAL PRIMARY KEY,
  "backtest_id" int NOT NULL,
  "symbol_id" int NOT NULL,
  "status" varchar(20) NOT NULL DEFAULT 'pending',
  "created_at" timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  "completed_at" timestamp
);

-- Backtest results table
CREATE TABLE IF NOT EXISTS "backtest_results" (
  "id" SERIAL PRIMARY KEY,
  "backtest_run_id" int NOT NULL,
  "total_trades" int NOT NULL DEFAULT 0,
  "winning_trades" int NOT NULL DEFAULT 0,
  "losing_trades" int NOT NULL DEFAULT 0,
  "profit_factor" numeric(10,4),
  "sharpe_ratio" numeric(10,4),
  "max_drawdown" numeric(10,4),
  "final_capital" numeric(20,8),
  "total_return" numeric(10,4),
  "annualized_return" numeric(10,4),
  "results_json" jsonb
);

-- Backtest trades table
CREATE TABLE IF NOT EXISTS "backtest_trades" (
  "id" SERIAL PRIMARY KEY,
  "backtest_run_id" int NOT NULL,
  "symbol_id" int NOT NULL,
  "entry_time" timestamp NOT NULL,
  "exit_time" timestamp,
  "position_type" varchar(10) NOT NULL,
  "entry_price" numeric(20,8) NOT NULL,
  "exit_price" numeric(20,8),
  "quantity" numeric(20,8) NOT NULL,
  "profit_loss" numeric(20,8),
  "profit_loss_percent" numeric(10,4),
  "exit_reason" varchar(50)
);

-- Binance download jobs table
CREATE TABLE IF NOT EXISTS "binance_download_jobs" (
  "id" SERIAL PRIMARY KEY,
  "symbol_id" int NOT NULL,
  "symbol" varchar(20) NOT NULL,
  "timeframe" timeframe_type NOT NULL,
  "start_date" timestamp NOT NULL,
  "end_date" timestamp NOT NULL,
  "status" varchar(20) NOT NULL DEFAULT 'pending',
  "progress" numeric(5,2) NOT NULL DEFAULT 0,
  "error" text,
  "created_at" timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  "updated_at" timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP)
);

-- Indexes
CREATE INDEX "idx_backtest_trades_backtest_run_id" ON "backtest_trades" ("backtest_run_id");
CREATE INDEX "idx_backtests_user_id" ON "backtests" ("user_id");
CREATE INDEX "idx_backtests_strategy_id" ON "backtests" ("strategy_id");
CREATE INDEX "idx_backtest_runs_backtest_id" ON "backtest_runs" ("backtest_id");
CREATE UNIQUE INDEX ON "backtest_runs" ("backtest_id", "symbol_id");
CREATE INDEX "idx_binance_download_jobs_status" ON "binance_download_jobs" ("status");
CREATE INDEX "idx_binance_download_jobs_symbol_id" ON "binance_download_jobs" ("symbol_id");
CREATE INDEX "idx_symbols_asset_type" ON "symbols" ("asset_type");
CREATE INDEX "idx_symbols_exchange" ON "symbols" ("exchange");
CREATE INDEX "idx_symbols_symbol" ON "symbols" ("symbol");

-- Foreign Keys
ALTER TABLE "candles" ADD FOREIGN KEY ("symbol_id") REFERENCES "symbols" ("id") ON DELETE CASCADE;
ALTER TABLE "backtest_runs" ADD FOREIGN KEY ("backtest_id") REFERENCES "backtests" ("id") ON DELETE CASCADE;
ALTER TABLE "backtest_runs" ADD FOREIGN KEY ("symbol_id") REFERENCES "symbols" ("id") ON DELETE CASCADE;
ALTER TABLE "backtest_results" ADD FOREIGN KEY ("backtest_run_id") REFERENCES "backtest_runs" ("id") ON DELETE CASCADE;
ALTER TABLE "backtest_trades" ADD FOREIGN KEY ("backtest_run_id") REFERENCES "backtest_runs" ("id") ON DELETE CASCADE;
ALTER TABLE "backtest_trades" ADD FOREIGN KEY ("symbol_id") REFERENCES "symbols" ("id") ON DELETE CASCADE;
ALTER TABLE "binance_download_jobs" ADD FOREIGN KEY ("symbol_id") REFERENCES "symbols" ("id") ON DELETE CASCADE;

-- Convert candles to hypertable for time series optimization
SELECT create_hypertable('candles', 'time', chunk_time_interval => INTERVAL '1 week');

-- Insert default symbols
INSERT INTO symbols (symbol, name, asset_type, exchange, is_active, created_at)
VALUES 
('BTCUSD', 'Bitcoin/US Dollar', 'crypto', 'Coinbase', true, CURRENT_TIMESTAMP),
('ETHUSD', 'Ethereum/US Dollar', 'crypto', 'Coinbase', true, CURRENT_TIMESTAMP),
('AAPL', 'Apple Inc.', 'stock', 'NASDAQ', true, CURRENT_TIMESTAMP),
('MSFT', 'Microsoft Corporation', 'stock', 'NASDAQ', true, CURRENT_TIMESTAMP),
('AMZN', 'Amazon.com, Inc.', 'stock', 'NASDAQ', true, CURRENT_TIMESTAMP),
('BTCUSDT', 'Bitcoin/USDT', 'crypto', 'Binance', true, CURRENT_TIMESTAMP),
('ETHUSDT', 'Ethereum/USDT', 'crypto', 'Binance', true, CURRENT_TIMESTAMP)
ON CONFLICT (symbol) DO NOTHING;

-- ==========================================
-- MARKET DATA (CANDLES AND SYMBOLS)
-- ==========================================

-- Get candle data with specific timeframe
CREATE OR REPLACE FUNCTION get_candles(
    p_symbol_id INT,
    p_timeframe timeframe_type,
    p_start_time TIMESTAMP,
    p_end_time TIMESTAMP,
    p_limit INT DEFAULT NULL
)
RETURNS TABLE (
    symbol_id INT,
    time TIMESTAMP,
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
        SELECT c.symbol_id, c.time, c.open, c.high, c.low, c.close, c.volume
        FROM candles c
        WHERE c.symbol_id = p_symbol_id
          AND c.time BETWEEN p_start_time AND p_end_time
        ORDER BY c.time DESC
        LIMIT p_limit;
    ELSE
        -- Aggregate candles for higher timeframes
        RETURN QUERY
        SELECT 
            c.symbol_id,
            time_bucket(interval_minutes || ' minutes', c.time) AS time,
            FIRST(c.open, c.time) AS open,
            MAX(c.high) AS high,
            MIN(c.low) AS low,
            LAST(c.close, c.time) AS close,
            SUM(c.volume) AS volume
        FROM candles c
        WHERE c.symbol_id = p_symbol_id
          AND c.time BETWEEN p_start_time AND p_end_time
        GROUP BY c.symbol_id, time_bucket(interval_minutes || ' minutes', c.time)
        ORDER BY time DESC
        LIMIT p_limit;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Insert candles in batch
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
        INSERT INTO candles (symbol_id, time, open, high, low, close, volume)
        VALUES (
            (candle_record->>'symbol_id')::INT,
            (candle_record->>'time')::TIMESTAMP,
            (candle_record->>'open')::NUMERIC(20,8),
            (candle_record->>'high')::NUMERIC(20,8),
            (candle_record->>'low')::NUMERIC(20,8),
            (candle_record->>'close')::NUMERIC(20,8),
            (candle_record->>'volume')::NUMERIC(20,8)
        )
        ON CONFLICT (symbol_id, time)
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

-- Get continuous data ranges for a symbol
CREATE OR REPLACE FUNCTION get_symbol_data_ranges(
    p_symbol_id INT,
    p_timeframe timeframe_type
)
RETURNS TABLE (
    start_date TIMESTAMP,
    end_date TIMESTAMP
) AS $$
BEGIN
    RETURN QUERY
    WITH dates AS (
        SELECT 
            time,
            LEAD(time) OVER (ORDER BY time) as next_time
        FROM candles
        WHERE symbol_id = p_symbol_id
        ORDER BY time
    )
    SELECT 
        MIN(time) as start_date,
        MAX(time) as end_date
    FROM (
        SELECT 
            time,
            next_time,
            CASE WHEN next_time IS NULL OR next_time > time + INTERVAL '1 day' THEN 1 ELSE 0 END as is_gap,
            SUM(CASE WHEN next_time IS NULL OR next_time > time + INTERVAL '1 day' THEN 1 ELSE 0 END) OVER (ORDER BY time) as group_id
        FROM dates
    ) t
    GROUP BY group_id
    ORDER BY start_date;
END;
$$ LANGUAGE plpgsql;

-- Get all symbols
CREATE OR REPLACE FUNCTION get_symbols(
    p_search_term VARCHAR DEFAULT NULL,
    p_asset_type VARCHAR DEFAULT NULL,
    p_exchange VARCHAR DEFAULT NULL
)
RETURNS TABLE (
    id INT,
    symbol VARCHAR(20),
    name VARCHAR(100),
    asset_type VARCHAR(20),
    exchange VARCHAR(50),
    is_active BOOLEAN,
    data_available BOOLEAN,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
) AS $$
BEGIN
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
        s.symbol;
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

-- ==========================================
-- BINANCE DATA DOWNLOAD FUNCTIONS
-- ==========================================

-- Create binance download job
CREATE OR REPLACE FUNCTION create_binance_download_job(
    p_symbol_id INT,
    p_symbol VARCHAR(20),
    p_timeframe timeframe_type,
    p_start_date TIMESTAMP,
    p_end_date TIMESTAMP
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
    start_date TIMESTAMP,
    end_date TIMESTAMP,
    status VARCHAR(20),
    progress NUMERIC(5,2),
    error TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
) AS $$
BEGIN
    RETURN QUERY
    SELECT *
    FROM binance_download_jobs
    WHERE status IN ('pending', 'in_progress')
    ORDER BY created_at DESC;
END;
$$ LANGUAGE plpgsql;

-- ==========================================
-- BACKTESTING FUNCTIONS AND VIEWS
-- ==========================================

-- Create a view to display backtest summary per the UI mockup
CREATE OR REPLACE VIEW v_backtest_summary AS
SELECT 
    b.id AS backtest_id,
    b.name,
    b.strategy_id,
    b.created_at AS date,
    b.status,
    (
        SELECT jsonb_agg(jsonb_build_object(
            'symbol_id', br.symbol_id,
            'symbol', sym.symbol,
            'win_rate', 
                CASE 
                    WHEN res.total_trades > 0 
                    THEN (res.winning_trades::FLOAT / res.total_trades::FLOAT) * 100 
                    ELSE 0 
                END,
            'profit', 
                CASE 
                    WHEN res.total_return IS NOT NULL 
                    THEN res.total_return 
                    ELSE 0 
                END
        ))
        FROM backtest_runs br
        JOIN symbols sym ON br.symbol_id = sym.id
        LEFT JOIN backtest_results res ON br.id = res.backtest_run_id
        WHERE br.backtest_id = b.id
    ) AS symbol_results,
    (
        SELECT COUNT(*) 
        FROM backtest_runs br
        WHERE br.backtest_id = b.id AND br.status = 'completed'
    ) AS completed_runs,
    (
        SELECT COUNT(*) 
        FROM backtest_runs br
        WHERE br.backtest_id = b.id
    ) AS total_runs
FROM 
    backtests b
ORDER BY 
    b.created_at DESC;

-- Function to get backtest summary for a user
CREATE OR REPLACE FUNCTION get_backtest_summary(
    p_user_id INT,
    p_limit INT DEFAULT 10,
    p_offset INT DEFAULT 0
)
RETURNS TABLE (
    backtest_id INT,
    name TEXT,
    strategy_id INT,
    date TIMESTAMP,
    status VARCHAR(20),
    symbol_results JSONB,
    completed_runs BIGINT,
    total_runs BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        bs.backtest_id,
        bs.name,
        bs.strategy_id,
        bs.date,
        bs.status,
        bs.symbol_results,
        bs.completed_runs,
        bs.total_runs
    FROM 
        v_backtest_summary bs
        JOIN backtests b ON bs.backtest_id = b.id
    WHERE 
        b.user_id = p_user_id
    ORDER BY 
        bs.date DESC
    LIMIT p_limit
    OFFSET p_offset;
END;
$$ LANGUAGE plpgsql;

-- Function to get backtest details
CREATE OR REPLACE FUNCTION get_backtest_details(p_backtest_id INT)
RETURNS TABLE (
    backtest_id INT,
    name TEXT,
    description TEXT,
    strategy_id INT,
    strategy_version INT,
    timeframe timeframe_type,
    start_date TIMESTAMP,
    end_date TIMESTAMP,
    initial_capital NUMERIC(20,8),
    status VARCHAR(20),
    created_at TIMESTAMP,
    completed_at TIMESTAMP,
    run_results JSONB
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        b.id AS backtest_id,
        b.name,
        b.description,
        b.strategy_id,
        b.strategy_version,
        b.timeframe,
        b.start_date,
        b.end_date,
        b.initial_capital,
        b.status,
        b.created_at,
        b.completed_at,
        (
            SELECT jsonb_agg(jsonb_build_object(
                'run_id', br.id,
                'symbol_id', br.symbol_id,
                'symbol', sym.symbol,
                'status', br.status,
                'completed_at', br.completed_at,
                'results', CASE WHEN res.id IS NOT NULL THEN
                    jsonb_build_object(
                        'total_trades', res.total_trades,
                        'winning_trades', res.winning_trades,
                        'losing_trades', res.losing_trades,
                        'profit_factor', res.profit_factor,
                        'sharpe_ratio', res.sharpe_ratio,
                        'max_drawdown', res.max_drawdown,
                        'final_capital', res.final_capital,
                        'total_return', res.total_return,
                        'annualized_return', res.annualized_return,
                        'detailed_results', res.results_json
                    )
                    ELSE NULL
                END
            ))
            FROM backtest_runs br
            JOIN symbols sym ON br.symbol_id = sym.id
            LEFT JOIN backtest_results res ON br.id = res.backtest_run_id
            WHERE br.backtest_id = b.id
        ) AS run_results
    FROM 
        backtests b
    WHERE 
        b.id = p_backtest_id;
END;
$$ LANGUAGE plpgsql;

-- Create new backtest
CREATE OR REPLACE FUNCTION create_backtest(
    p_user_id INT,
    p_strategy_id INT,
    p_strategy_version INT,
    p_name VARCHAR(100),
    p_description TEXT,
    p_timeframe timeframe_type,
    p_start_date TIMESTAMP,
    p_end_date TIMESTAMP,
    p_initial_capital NUMERIC(20,8),
    p_symbol_ids INT[]
)
RETURNS INT AS $$
DECLARE
    new_backtest_id INT;
    symbol_id INT;
BEGIN
    -- Create backtest record
    INSERT INTO backtests (
        user_id,
        strategy_id,
        strategy_version,
        name,
        description,
        timeframe,
        start_date,
        end_date,
        initial_capital,
        status,
        created_at,
        updated_at
    )
    VALUES (
        p_user_id,
        p_strategy_id,
        p_strategy_version,
        p_name,
        p_description,
        p_timeframe,
        p_start_date,
        p_end_date,
        p_initial_capital,
        'pending',
        NOW(),
        NOW()
    )
    RETURNING id INTO new_backtest_id;
    
    -- Create backtest runs for each symbol
    FOREACH symbol_id IN ARRAY p_symbol_ids LOOP
        INSERT INTO backtest_runs (
            backtest_id,
            symbol_id,
            status,
            created_at
        )
        VALUES (
            new_backtest_id,
            symbol_id,
            'pending',
            NOW()
        );
    END LOOP;
    
    RETURN new_backtest_id;
END;
$$ LANGUAGE plpgsql;

-- Update backtest run status
CREATE OR REPLACE FUNCTION update_backtest_run_status(
    p_run_id INT,
    p_status VARCHAR(20)
)
RETURNS BOOLEAN AS $$
DECLARE
    affected_rows INT;
    backtest_id INT;
BEGIN
    -- Update run status
    UPDATE backtest_runs
    SET 
        status = p_status,
        completed_at = CASE WHEN p_status = 'completed' THEN NOW() ELSE NULL END
    WHERE 
        id = p_run_id
    RETURNING backtest_id INTO backtest_id;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    
    IF affected_rows = 0 THEN
        RETURN FALSE;
    END IF;
    
    -- Check if all runs are completed and update backtest status if needed
    IF (
        SELECT COUNT(*) 
        FROM backtest_runs 
        WHERE backtest_id = backtest_id AND status != 'completed'
    ) = 0 THEN
        UPDATE backtests
        SET 
            status = 'completed',
            completed_at = NOW(),
            updated_at = NOW()
        WHERE 
            id = backtest_id;
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Save backtest result
CREATE OR REPLACE FUNCTION save_backtest_result(
    p_backtest_run_id INT,
    p_total_trades INT,
    p_winning_trades INT,
    p_losing_trades INT,
    p_profit_factor NUMERIC(10,4),
    p_sharpe_ratio NUMERIC(10,4),
    p_max_drawdown NUMERIC(10,4),
    p_final_capital NUMERIC(20,8),
    p_total_return NUMERIC(10,4),
    p_annualized_return NUMERIC(10,4),
    p_results_json JSONB
)
RETURNS INT AS $$
DECLARE
    result_id INT;
BEGIN
    -- Check if result already exists
    SELECT id INTO result_id
    FROM backtest_results
    WHERE backtest_run_id = p_backtest_run_id;
    
    IF FOUND THEN
        -- Update existing result
        UPDATE backtest_results
        SET 
            total_trades = p_total_trades,
            winning_trades = p_winning_trades,
            losing_trades = p_losing_trades,
            profit_factor = p_profit_factor,
            sharpe_ratio = p_sharpe_ratio,
            max_drawdown = p_max_drawdown,
            final_capital = p_final_capital,
            total_return = p_total_return,
            annualized_return = p_annualized_return,
            results_json = p_results_json
        WHERE 
            id = result_id;
    ELSE
        -- Insert new result
        INSERT INTO backtest_results (
            backtest_run_id,
            total_trades,
            winning_trades,
            losing_trades,
            profit_factor,
            sharpe_ratio,
            max_drawdown,
            final_capital,
            total_return,
            annualized_return,
            results_json
        )
        VALUES (
            p_backtest_run_id,
            p_total_trades,
            p_winning_trades,
            p_losing_trades,
            p_profit_factor,
            p_sharpe_ratio,
            p_max_drawdown,
            p_final_capital,
            p_total_return,
            p_annualized_return,
            p_results_json
        )
        RETURNING id INTO result_id;
    END IF;
    
    -- Update run status
    PERFORM update_backtest_run_status(p_backtest_run_id, 'completed');
    
    RETURN result_id;
END;
$$ LANGUAGE plpgsql;

-- Add backtest trade
CREATE OR REPLACE FUNCTION add_backtest_trade(
    p_backtest_run_id INT,
    p_symbol_id INT,
    p_entry_time TIMESTAMP,
    p_exit_time TIMESTAMP,
    p_position_type VARCHAR(10),
    p_entry_price NUMERIC(20,8),
    p_exit_price NUMERIC(20,8),
    p_quantity NUMERIC(20,8),
    p_profit_loss NUMERIC(20,8),
    p_profit_loss_percent NUMERIC(10,4),
    p_exit_reason VARCHAR(50)
)
RETURNS INT AS $$
DECLARE
    new_trade_id INT;
BEGIN
    INSERT INTO backtest_trades (
        backtest_run_id,
        symbol_id,
        entry_time,
        exit_time,
        position_type,
        entry_price,
        exit_price,
        quantity,
        profit_loss,
        profit_loss_percent,
        exit_reason
    )
    VALUES (
        p_backtest_run_id,
        p_symbol_id,
        p_entry_time,
        p_exit_time,
        p_position_type,
        p_entry_price,
        p_exit_price,
        p_quantity,
        p_profit_loss,
        p_profit_loss_percent,
        p_exit_reason
    )
    RETURNING id INTO new_trade_id;
    
    RETURN new_trade_id;
END;
$$ LANGUAGE plpgsql;

-- Get backtest trades
CREATE OR REPLACE FUNCTION get_backtest_trades(
    p_backtest_run_id INT,
    p_limit INT DEFAULT 100,
    p_offset INT DEFAULT 0
)
RETURNS TABLE (
    id INT,
    symbol_id INT,
    symbol VARCHAR(20),
    entry_time TIMESTAMP,
    exit_time TIMESTAMP,
    position_type VARCHAR(10),
    entry_price NUMERIC(20,8),
    exit_price NUMERIC(20,8),
    quantity NUMERIC(20,8),
    profit_loss NUMERIC(20,8),
    profit_loss_percent NUMERIC(10,4),
    exit_reason VARCHAR(50)
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        t.id,
        t.symbol_id,
        s.symbol,
        t.entry_time,
        t.exit_time,
        t.position_type,
        t.entry_price,
        t.exit_price,
        t.quantity,
        t.profit_loss,
        t.profit_loss_percent,
        t.exit_reason
    FROM 
        backtest_trades t
        JOIN symbols s ON t.symbol_id = s.id
    WHERE 
        t.backtest_run_id = p_backtest_run_id
    ORDER BY 
        t.entry_time
    LIMIT p_limit
    OFFSET p_offset;
END;
$$ LANGUAGE plpgsql;

-- Delete backtest
CREATE OR REPLACE FUNCTION delete_backtest(
    p_user_id INT,
    p_backtest_id INT
)
RETURNS BOOLEAN AS $$
DECLARE
    affected_rows INT;
BEGIN
    -- Check ownership
    PERFORM 1 FROM backtests
    WHERE id = p_backtest_id AND user_id = p_user_id;
    
    IF NOT FOUND THEN
        RETURN FALSE;
    END IF;
    
    -- Delete backtest and all related data (cascade will handle related records)
    DELETE FROM backtests
    WHERE id = p_backtest_id;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows > 0;
END;
$$ LANGUAGE plpgsql;