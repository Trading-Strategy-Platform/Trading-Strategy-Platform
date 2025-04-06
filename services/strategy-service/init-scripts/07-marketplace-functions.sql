-- Strategy Service Marketplace Functions
-- File: 07-marketplace-functions.sql
-- Contains functions for marketplace operations

-- Create marketplace listing view with ratings
CREATE OR REPLACE VIEW v_marketplace_strategies AS
SELECT 
    m.id AS marketplace_id,
    s.id AS strategy_id,
    s.name,
    s.description,
    s.thumbnail_url,
    s.user_id AS owner_id,
    s.user_id AS owner_user_id, -- Using user_id instead of username
    NULL AS owner_photo, -- Removed dependency on profile_photo_url
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

-- Get marketplace strategies with filtering and sorting
CREATE OR REPLACE FUNCTION get_marketplace_strategies(
    p_search_term VARCHAR DEFAULT NULL,
    p_min_price NUMERIC DEFAULT NULL,
    p_max_price NUMERIC DEFAULT NULL,
    p_is_free BOOLEAN DEFAULT NULL,
    p_tags INT[] DEFAULT NULL,
    p_min_rating NUMERIC DEFAULT NULL,
    p_sort_by VARCHAR DEFAULT 'popularity',
    p_limit INT DEFAULT 20,
    p_offset INT DEFAULT 0
)
RETURNS TABLE (
    marketplace_id INT,
    strategy_id INT,
    name VARCHAR(100),
    description TEXT,
    thumbnail_url VARCHAR(255),
    owner_id INT,
    owner_user_id INT, -- Changed from owner_username
    owner_photo VARCHAR(255),
    version_id INT,
    price NUMERIC(10,2),
    is_subscription BOOLEAN,
    subscription_period VARCHAR(20),
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    avg_rating NUMERIC,
    rating_count BIGINT,
    tags TEXT[],
    tag_ids INT[]
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        ms.marketplace_id,
        ms.strategy_id,
        ms.name,
        ms.description,
        ms.thumbnail_url,
        ms.owner_id,
        ms.owner_user_id,
        ms.owner_photo,
        ms.version_id,
        ms.price,
        ms.is_subscription,
        ms.subscription_period,
        ms.created_at,
        ms.updated_at,
        ms.avg_rating,
        ms.rating_count,
        ms.tags,
        ms.tag_ids
    FROM 
        v_marketplace_strategies ms
    WHERE 
        (
            p_search_term IS NULL 
            OR ms.name ILIKE '%' || p_search_term || '%' 
            OR ms.description ILIKE '%' || p_search_term || '%'
        )
        AND (p_min_price IS NULL OR ms.price >= p_min_price)
        AND (p_max_price IS NULL OR ms.price <= p_max_price)
        AND (
            p_is_free IS NULL 
            OR (p_is_free = TRUE AND ms.price = 0) 
            OR (p_is_free = FALSE AND ms.price > 0)
        )
        AND (p_tags IS NULL OR ms.tag_ids && p_tags)
        AND (p_min_rating IS NULL OR ms.avg_rating >= p_min_rating)
    ORDER BY
        CASE
            WHEN p_sort_by = 'popularity' THEN ms.rating_count
            ELSE 0
        END DESC,
        CASE
            WHEN p_sort_by = 'rating' THEN ms.avg_rating
            ELSE 0
        END DESC,
        CASE
            WHEN p_sort_by = 'price_asc' THEN ms.price
            ELSE NULL
        END ASC,
        CASE
            WHEN p_sort_by = 'price_desc' THEN ms.price
            ELSE NULL
        END DESC,
        CASE
            WHEN p_sort_by = 'newest' THEN ms.created_at
            ELSE NULL
        END DESC
    LIMIT p_limit
    OFFSET p_offset;
END;
$$ LANGUAGE plpgsql;

-- Add strategy to marketplace
CREATE OR REPLACE FUNCTION add_to_marketplace(
    p_user_id INT,
    p_strategy_id INT,
    p_version_id INT,
    p_price NUMERIC(10,2),
    p_is_subscription BOOLEAN,
    p_subscription_period VARCHAR(20) DEFAULT NULL,
    p_description TEXT DEFAULT NULL
)
RETURNS INT AS $$
DECLARE
    new_marketplace_id INT;
BEGIN
    -- Check ownership
    PERFORM 1 FROM strategies 
    WHERE id = p_strategy_id AND user_id = p_user_id;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Not the owner of this strategy';
    END IF;
    
    -- Insert marketplace listing
    INSERT INTO strategy_marketplace (
        strategy_id,
        version_id,
        user_id,
        price,
        is_subscription,
        subscription_period,
        is_active,
        description,
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
        COALESCE(p_description, (SELECT description FROM strategies WHERE id = p_strategy_id)),
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