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
    date TIMESTAMPTZ,
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
    start_date TIMESTAMPTZ,
    end_date TIMESTAMPTZ,
    initial_capital NUMERIC(20,8),
    status VARCHAR(20),
    created_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
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
    p_start_date TIMESTAMPTZ,
    p_end_date TIMESTAMPTZ,
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
    p_entry_time TIMESTAMPTZ,
    p_exit_time TIMESTAMPTZ,
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
    entry_time TIMESTAMPTZ,
    exit_time TIMESTAMPTZ,
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