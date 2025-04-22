-- Strategy Service Version Functions
-- File: 05-version-functions.sql
-- Contains functions for version operations

-- Get all versions of a strategy with the active version flag
CREATE OR REPLACE FUNCTION get_all_strategy_versions(
    p_user_id INT,
    p_strategy_id INT,
    p_sort_by VARCHAR DEFAULT 'version',
    p_sort_direction VARCHAR DEFAULT 'DESC',
    p_limit INT DEFAULT 20,
    p_offset INT DEFAULT 0
)
RETURNS TABLE (
    id INT,
    strategy_id INT,
    version INT,
    structure JSONB,
    change_notes TEXT,
    created_at TIMESTAMP,
    is_active BOOLEAN  -- Keep this flag to show which version is active (latest)
) AS $$
DECLARE
    current_version INT;
BEGIN
    -- Get the current version from the strategy
    SELECT version INTO current_version FROM strategies WHERE id = p_strategy_id;
    
    -- Validate sort field
    IF p_sort_by NOT IN ('version', 'created_at') THEN
        p_sort_by := 'version'; -- Default sort by version
    END IF;
    
    -- Normalize sort direction
    IF UPPER(p_sort_direction) NOT IN ('ASC', 'DESC') THEN
        p_sort_direction := 'DESC'; -- Default descending for versions (newest first)
    ELSE
        p_sort_direction := UPPER(p_sort_direction);
    END IF;

    -- For strategy owners, return all versions
    IF EXISTS (SELECT 1 FROM strategies WHERE id = p_strategy_id AND user_id = p_user_id) THEN
        RETURN QUERY
        SELECT 
            sv.id,
            sv.strategy_id,
            sv.version,
            sv.structure,
            sv.change_notes,
            sv.created_at,
            (sv.version = current_version) AS is_active  -- Compare with current version
        FROM 
            strategy_versions sv
        WHERE 
            sv.strategy_id = p_strategy_id
            AND sv.is_deleted = FALSE
        ORDER BY
            CASE WHEN p_sort_by = 'version' AND p_sort_direction = 'DESC' THEN sv.version END DESC,
            CASE WHEN p_sort_by = 'version' AND p_sort_direction = 'ASC' THEN sv.version END ASC,
            CASE WHEN p_sort_by = 'created_at' AND p_sort_direction = 'DESC' THEN sv.created_at END DESC,
            CASE WHEN p_sort_by = 'created_at' AND p_sort_direction = 'ASC' THEN sv.created_at END ASC
        LIMIT p_limit OFFSET p_offset;
    ELSE
        -- For purchasers, return only versions they've purchased
        RETURN QUERY
        SELECT 
            sv.id,
            sv.strategy_id,
            sv.version,
            sv.structure,
            sv.change_notes,
            sv.created_at,
            (sv.version = current_version) AS is_active  -- Compare with current version
        FROM 
            strategy_versions sv
            JOIN strategy_purchases sp ON sp.strategy_version = sv.version
            JOIN strategy_marketplace sm ON 
                sp.marketplace_id = sm.id AND 
                sm.strategy_id = p_strategy_id
        WHERE 
            sv.strategy_id = p_strategy_id
            AND sv.is_deleted = FALSE
            AND sp.buyer_id = p_user_id
            AND (sp.subscription_end IS NULL OR sp.subscription_end > NOW())
        ORDER BY
            CASE WHEN p_sort_by = 'version' AND p_sort_direction = 'DESC' THEN sv.version END DESC,
            CASE WHEN p_sort_by = 'version' AND p_sort_direction = 'ASC' THEN sv.version END ASC,
            CASE WHEN p_sort_by = 'created_at' AND p_sort_direction = 'DESC' THEN sv.created_at END DESC,
            CASE WHEN p_sort_by = 'created_at' AND p_sort_direction = 'ASC' THEN sv.created_at END ASC
        LIMIT p_limit OFFSET p_offset;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Get a specific version by ID
CREATE OR REPLACE FUNCTION get_strategy_version_by_id(
    p_strategy_id INT,
    p_version INT
)
RETURNS TABLE (
    id INT,
    strategy_id INT,
    version INT,
    structure JSONB,
    change_notes TEXT,
    created_at TIMESTAMP,
    is_active BOOLEAN  -- Keep this flag
) AS $$
DECLARE
    current_version INT;
BEGIN
    -- Get the current version from the strategy
    SELECT version INTO current_version FROM strategies WHERE id = p_strategy_id;

    RETURN QUERY
    SELECT 
        sv.id,
        sv.strategy_id,
        sv.version,
        sv.structure,
        sv.change_notes,
        sv.created_at,
        (sv.version = current_version) AS is_active  -- Compare with current version
    FROM 
        strategy_versions sv
    WHERE 
        sv.strategy_id = p_strategy_id
        AND sv.version = p_version
        AND sv.is_deleted = FALSE;
END;
$$ LANGUAGE plpgsql;

-- Count strategy versions
CREATE OR REPLACE FUNCTION count_strategy_versions(
    p_user_id INT,
    p_strategy_id INT
)
RETURNS BIGINT AS $$
DECLARE
    version_count BIGINT;
BEGIN
    -- For strategy owners, count all versions
    IF EXISTS (SELECT 1 FROM strategies WHERE id = p_strategy_id AND user_id = p_user_id) THEN
        SELECT COUNT(*)
        INTO version_count
        FROM strategy_versions sv
        WHERE sv.strategy_id = p_strategy_id
          AND sv.is_deleted = FALSE;
    ELSE
        -- For purchasers, count only versions they've purchased
        SELECT COUNT(*)
        INTO version_count
        FROM strategy_versions sv
        JOIN strategy_purchases sp ON sp.strategy_version = sv.version
        JOIN strategy_marketplace sm ON sp.marketplace_id = sm.id AND sm.strategy_id = p_strategy_id
        WHERE sv.strategy_id = p_strategy_id
          AND sv.is_deleted = FALSE
          AND sp.buyer_id = p_user_id
          AND (sp.subscription_end IS NULL OR sp.subscription_end > NOW());
    END IF;
    
    RETURN version_count;
END;
$$ LANGUAGE plpgsql;