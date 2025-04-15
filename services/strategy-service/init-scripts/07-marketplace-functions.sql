-- Strategy Service Marketplace Functions
-- File: 07-marketplace-functions.sql
-- Contains functions for marketplace operations

-- Create marketplace listing view with ratings 
-- (Keeping this the same from the original file for reference)
CREATE OR REPLACE VIEW v_marketplace_strategy AS
SELECT 
    m.id AS marketplace_id,
    s.id AS strategy_id,
    s.name,
    m.description_public AS description,
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

-- Update the get_marketplace_strategies function to ensure it returns name and thumbnail
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
    id INT,
    strategy_id INT,
    name VARCHAR,
    description_public TEXT,
    thumbnail_url VARCHAR,
    user_id INT,
    price NUMERIC,
    is_subscription BOOLEAN,
    subscription_period VARCHAR,
    is_active BOOLEAN,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    average_rating FLOAT,
    reviews_count BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        m.id,
        m.strategy_id,
        s.name,
        m.description_public,
        s.thumbnail_url,
        m.user_id,
        m.price,
        m.is_subscription,
        m.subscription_period,
        m.is_active,
        m.created_at,
        m.updated_at,
        COALESCE(AVG(r.rating), 0) AS average_rating,
        COUNT(DISTINCT r.id) AS reviews_count
    FROM 
        strategy_marketplace m
        JOIN strategies s ON m.strategy_id = s.id
        LEFT JOIN strategy_reviews r ON m.id = r.marketplace_id
    WHERE 
        m.is_active = TRUE
        AND s.is_active = TRUE
        AND (p_search_term IS NULL OR 
             s.name ILIKE '%' || p_search_term || '%' OR 
             m.description_public ILIKE '%' || p_search_term || '%')
        AND (p_min_price IS NULL OR m.price >= p_min_price)
        AND (p_max_price IS NULL OR m.price <= p_max_price)
        AND (p_is_free IS NULL OR (p_is_free = TRUE AND m.price = 0) OR (p_is_free = FALSE AND m.price > 0))
        AND (p_tags IS NULL OR p_tags = '{}' OR EXISTS (
            SELECT 1 FROM strategy_tag_mappings tm
            WHERE tm.strategy_id = s.id AND tm.tag_id = ANY(p_tags)
        ))
        AND (p_min_rating IS NULL OR COALESCE(AVG(r.rating), 0) >= p_min_rating)
    GROUP BY
        m.id, m.strategy_id, s.name, m.description_public, s.thumbnail_url, m.user_id,
        m.price, m.is_subscription, m.subscription_period, m.is_active, m.created_at, m.updated_at
    ORDER BY
        CASE WHEN p_sort_by = 'popularity' OR p_sort_by IS NULL THEN COUNT(DISTINCT r.id) END DESC,
        CASE WHEN p_sort_by = 'rating' THEN COALESCE(AVG(r.rating), 0) END DESC,
        CASE WHEN p_sort_by = 'price_asc' THEN m.price END ASC,
        CASE WHEN p_sort_by = 'price_desc' THEN m.price END DESC,
        CASE WHEN p_sort_by = 'newest' THEN m.created_at END DESC
    LIMIT p_limit
    OFFSET p_offset;
END;
$$ LANGUAGE plpgsql;

-- Create a separate function for counting total results
CREATE OR REPLACE FUNCTION count_marketplace_strategies(
    p_search_term VARCHAR DEFAULT NULL,
    p_min_price NUMERIC DEFAULT NULL,
    p_max_price NUMERIC DEFAULT NULL,
    p_is_free BOOLEAN DEFAULT NULL,
    p_tags INT[] DEFAULT NULL,
    p_min_rating NUMERIC DEFAULT NULL
)
RETURNS BIGINT AS $$
DECLARE
    total_count BIGINT;
BEGIN
    SELECT COUNT(*)
    INTO total_count
    FROM (
        SELECT 
            m.id
        FROM 
            strategy_marketplace m
            JOIN strategies s ON m.strategy_id = s.id
            LEFT JOIN strategy_reviews r ON m.id = r.marketplace_id
        WHERE 
            m.is_active = TRUE
            AND s.is_active = TRUE
            AND (p_search_term IS NULL OR 
                s.name ILIKE '%' || p_search_term || '%' OR 
                m.description_public ILIKE '%' || p_search_term || '%')
            AND (p_min_price IS NULL OR m.price >= p_min_price)
            AND (p_max_price IS NULL OR m.price <= p_max_price)
            AND (p_is_free IS NULL OR (p_is_free = TRUE AND m.price = 0) OR (p_is_free = FALSE AND m.price > 0))
            AND (p_tags IS NULL OR p_tags = '{}' OR EXISTS (
                SELECT 1 FROM strategy_tag_mappings tm
                WHERE tm.strategy_id = s.id AND tm.tag_id = ANY(p_tags)
            ))
        GROUP BY m.id
        HAVING
            (p_min_rating IS NULL OR COALESCE(AVG(r.rating), 0) >= p_min_rating)
    ) subquery;
    
    RETURN total_count;
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