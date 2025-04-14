-- Strategy Service Marketplace Functions
-- File: 07-marketplace-functions.sql
-- Contains functions for marketplace operations

-- Create marketplace listing view with ratings
CREATE OR REPLACE VIEW v_marketplace_strategies AS
SELECT 
    m.id AS marketplace_id,
    s.id AS strategy_id,
    s.name,
    m.description_public AS description, -- Present as 'description' in the view
    s.thumbnail_url,
    s.user_id AS owner_id,
    s.user_id AS owner_user_id,
    NULL AS owner_photo,
    m.version_id,
    m.price,
    m.is_subscription,
    m.subscription_period,
    m.created_at,
    m.updated_at,
    COALESCE(AVG(r.rating), 0) AS avg_rating,
    COUNT(r.id) AS rating_count,
    ARRAY(
        SELECT t.name 
        FROM strategy_tag_mappings tm
        JOIN strategy_tags t ON tm.tag_id = t.id
        WHERE tm.strategy_id = s.id
    ) AS tags,
    ARRAY(
        SELECT t.id 
        FROM strategy_tag_mappings tm
        JOIN strategy_tags t ON tm.tag_id = t.id
        WHERE tm.strategy_id = s.id
    ) AS tag_ids
FROM 
    strategy_marketplace m
    JOIN strategies s ON m.strategy_id = s.id
    LEFT JOIN strategy_reviews r ON m.id = r.marketplace_id
WHERE 
    m.is_active = TRUE
    AND s.is_active = TRUE
GROUP BY 
    m.id, s.id;

CREATE OR REPLACE FUNCTION add_to_marketplace(
    p_user_id INT,
    p_strategy_id INT,
    p_version_id INT,
    p_price NUMERIC(10,2),
    p_is_subscription BOOLEAN,
    p_subscription_period VARCHAR(20) DEFAULT NULL,
    p_description_public TEXT DEFAULT NULL
)
RETURNS INT AS $$
DECLARE
    new_marketplace_id INT;
    strategy_record RECORD;
BEGIN
    -- Get existing strategy data
    SELECT id, name, description INTO strategy_record
    FROM strategies 
    WHERE id = p_strategy_id AND user_id = p_user_id;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Not the owner of this strategy';
    END IF;
    
    -- Insert marketplace listing without modifying the strategy
    INSERT INTO strategy_marketplace (
        strategy_id,
        version_id,
        user_id,
        price,
        is_subscription,
        subscription_period,
        is_active,
        description_public,
        created_at,
        updated_at
    )
    VALUES (
        p_strategy_id,
        p_version_id,
        p_user_id,
        p_price,
        p_is_subscription,
        p_subscription_period,
        TRUE,
        COALESCE(p_description_public, strategy_record.description),
        NOW(),
        NOW()
    )
    RETURNING id INTO new_marketplace_id;
    
    RETURN new_marketplace_id;
END;
$$ LANGUAGE plpgsql;

-- Remove strategy from marketplace
CREATE OR REPLACE FUNCTION remove_from_marketplace(
    p_user_id INT,
    p_marketplace_id INT
)
RETURNS BOOLEAN AS $$
DECLARE
    affected_rows INT;
BEGIN
    -- Check ownership
    PERFORM 1 FROM strategy_marketplace m
    JOIN strategies s ON m.strategy_id = s.id
    WHERE m.id = p_marketplace_id AND s.user_id = p_user_id;
    
    IF NOT FOUND THEN
        RETURN FALSE;
    END IF;
    
    -- Update marketplace listing
    UPDATE strategy_marketplace
    SET 
        is_active = FALSE,
        updated_at = NOW()
    WHERE 
        id = p_marketplace_id;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows > 0;
END;
$$ LANGUAGE plpgsql;