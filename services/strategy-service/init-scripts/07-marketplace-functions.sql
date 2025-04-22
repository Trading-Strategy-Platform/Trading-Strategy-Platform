-- Strategy Service Marketplace Functions
-- File: 07-marketplace-functions.sql
-- Contains functions for marketplace operations

-- Create marketplace listing view with ratings 
CREATE OR REPLACE VIEW v_marketplace_listings AS
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

-- Get all marketplace listings with enhanced filtering and sorting
CREATE OR REPLACE FUNCTION get_all_marketplace_listings(
    p_search_term VARCHAR DEFAULT NULL,
    p_min_price NUMERIC DEFAULT NULL,
    p_max_price NUMERIC DEFAULT NULL,
    p_is_free BOOLEAN DEFAULT NULL,
    p_tags INT[] DEFAULT NULL,
    p_min_rating NUMERIC DEFAULT NULL,
    p_sort_by VARCHAR DEFAULT 'popularity',
    p_sort_direction VARCHAR DEFAULT 'DESC',
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
    -- Validate sort field
    IF p_sort_by NOT IN ('popularity', 'rating', 'price', 'newest', 'name') THEN
        p_sort_by := 'popularity'; -- Default sort by popularity
    END IF;
    
    -- Validate sort direction
    IF UPPER(p_sort_direction) NOT IN ('ASC', 'DESC') THEN
        -- For price, we need to handle price_asc and price_desc specially
        IF p_sort_by = 'price' THEN
            p_sort_direction := 'ASC'; -- Default ascending for price
        ELSE
            p_sort_direction := 'DESC'; -- Default descending for other fields
        END IF;
    ELSE
        p_sort_direction := UPPER(p_sort_direction);
    END IF;

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
        -- Apply complex sorting logic
        CASE WHEN p_sort_by = 'popularity' AND p_sort_direction = 'DESC' THEN COUNT(DISTINCT r.id) END DESC,
        CASE WHEN p_sort_by = 'popularity' AND p_sort_direction = 'ASC' THEN COUNT(DISTINCT r.id) END ASC,
        CASE WHEN p_sort_by = 'rating' AND p_sort_direction = 'DESC' THEN COALESCE(AVG(r.rating), 0) END DESC,
        CASE WHEN p_sort_by = 'rating' AND p_sort_direction = 'ASC' THEN COALESCE(AVG(r.rating), 0) END ASC,
        CASE WHEN p_sort_by = 'price' AND p_sort_direction = 'ASC' THEN m.price END ASC,
        CASE WHEN p_sort_by = 'price' AND p_sort_direction = 'DESC' THEN m.price END DESC,
        CASE WHEN p_sort_by = 'newest' AND p_sort_direction = 'DESC' THEN m.created_at END DESC,
        CASE WHEN p_sort_by = 'newest' AND p_sort_direction = 'ASC' THEN m.created_at END ASC,
        CASE WHEN p_sort_by = 'name' AND p_sort_direction = 'ASC' THEN s.name END ASC,
        CASE WHEN p_sort_by = 'name' AND p_sort_direction = 'DESC' THEN s.name END DESC
    LIMIT p_limit
    OFFSET p_offset;
END;
$$ LANGUAGE plpgsql;

-- Create a separate function for counting total results
CREATE OR REPLACE FUNCTION count_marketplace_listings(
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

-- Create new marketplace listing
CREATE OR REPLACE FUNCTION create_marketplace_listing(
    p_user_id INT,
    p_strategy_id INT,
    p_version_id INT,
    p_price NUMERIC,
    p_is_subscription BOOLEAN,
    p_subscription_period VARCHAR,
    p_description_public TEXT
)
RETURNS INT AS $$
DECLARE
    new_listing_id INT;
BEGIN
    -- Check if strategy belongs to user
    PERFORM 1 FROM strategies 
    WHERE id = p_strategy_id AND user_id = p_user_id;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Strategy does not belong to user';
    END IF;
    
    -- Check if listing already exists
    PERFORM 1 FROM strategy_marketplace
    WHERE strategy_id = p_strategy_id AND is_active = TRUE;
    
    IF FOUND THEN
        RAISE EXCEPTION 'Strategy is already listed on marketplace';
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
        p_description_public,
        NOW(),
        NOW()
    )
    RETURNING id INTO new_listing_id;
    
    RETURN new_listing_id;
END;
$$ LANGUAGE plpgsql;

-- Get marketplace listing by ID
CREATE OR REPLACE FUNCTION get_marketplace_listing_by_id(
    p_listing_id INT
)
RETURNS TABLE (
    id INT,
    strategy_id INT,
    version_id INT,
    user_id INT,
    price NUMERIC,
    is_subscription BOOLEAN,
    subscription_period VARCHAR,
    is_active BOOLEAN,
    description_public TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        m.id,
        m.strategy_id,
        m.version_id,
        m.user_id,
        m.price,
        m.is_subscription,
        m.subscription_period,
        m.is_active,
        m.description_public,
        m.created_at,
        m.updated_at
    FROM 
        strategy_marketplace m
    WHERE 
        m.id = p_listing_id;
END;
$$ LANGUAGE plpgsql;

-- Remove strategy from marketplace
CREATE OR REPLACE FUNCTION delete_marketplace_listing(
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