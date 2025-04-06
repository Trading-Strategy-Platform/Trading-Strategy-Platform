-- Strategy Service Version Functions
-- File: 05-version-functions.sql
-- Contains functions for version operations

-- Get all versions of a strategy that the user has access to
CREATE OR REPLACE FUNCTION get_accessible_strategy_versions(
    p_user_id INT,
    p_strategy_id INT
)
RETURNS TABLE (
    version INT,
    change_notes TEXT,
    created_at TIMESTAMP,
    is_active_version BOOLEAN
) AS $$
BEGIN
    -- For strategy owners, return all versions
    IF EXISTS (SELECT 1 FROM strategies WHERE id = p_strategy_id AND user_id = p_user_id) THEN
        RETURN QUERY
        SELECT 
            sv.version,
            sv.change_notes,
            sv.created_at,
            COALESCE(usv.active_version = sv.version, FALSE) AS is_active_version
        FROM 
            strategy_versions sv
            LEFT JOIN user_strategy_versions usv ON 
                usv.user_id = p_user_id AND 
                usv.strategy_id = p_strategy_id
        WHERE 
            sv.strategy_id = p_strategy_id
            AND sv.is_deleted = FALSE
        ORDER BY 
            sv.version DESC;
    ELSE
        -- For purchasers, return only versions they've purchased
        RETURN QUERY
        SELECT 
            sv.version,
            sv.change_notes,
            sv.created_at,
            COALESCE(usv.active_version = sv.version, FALSE) AS is_active_version
        FROM 
            strategy_versions sv
            JOIN strategy_purchases sp ON sp.strategy_version = sv.version
            JOIN strategy_marketplace sm ON 
                sp.marketplace_id = sm.id AND 
                sm.strategy_id = p_strategy_id
            LEFT JOIN user_strategy_versions usv ON 
                usv.user_id = p_user_id AND 
                usv.strategy_id = p_strategy_id
        WHERE 
            sv.strategy_id = p_strategy_id
            AND sv.is_deleted = FALSE
            AND sp.buyer_id = p_user_id
            AND (sp.subscription_end IS NULL OR sp.subscription_end > NOW())
        ORDER BY 
            sv.version DESC;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Update strategy active version for a user
CREATE OR REPLACE FUNCTION update_user_strategy_version(
    p_user_id INT,
    p_strategy_id INT,
    p_version INT
)
RETURNS BOOLEAN AS $$
DECLARE
    affected_rows INT;
BEGIN
    -- Verify user has access to this strategy version
    -- Either they own it or purchased it
    PERFORM 1 
    FROM strategies s
    WHERE s.id = p_strategy_id AND s.user_id = p_user_id
    
    UNION
    
    SELECT 1
    FROM strategy_purchases p
    JOIN strategy_marketplace m ON p.marketplace_id = m.id
    WHERE m.strategy_id = p_strategy_id 
      AND p.buyer_id = p_user_id
      AND p.strategy_version = p_version
      AND (p.subscription_end IS NULL OR p.subscription_end > NOW());
      
    IF NOT FOUND THEN
        RETURN FALSE;
    END IF;
    
    -- Check if record exists
    PERFORM 1 
    FROM user_strategy_versions 
    WHERE user_id = p_user_id AND strategy_id = p_strategy_id;
    
    IF FOUND THEN
        -- Update existing record
        UPDATE user_strategy_versions
        SET 
            active_version = p_version,
            updated_at = NOW()
        WHERE 
            user_id = p_user_id 
            AND strategy_id = p_strategy_id;
    ELSE
        -- Insert new record
        INSERT INTO user_strategy_versions (
            user_id,
            strategy_id,
            active_version,
            updated_at
        )
        VALUES (
            p_user_id,
            p_strategy_id,
            p_version,
            NOW()
        );
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;