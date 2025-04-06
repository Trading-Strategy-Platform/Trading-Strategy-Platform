-- Strategy Service Review Functions
-- File: 09-review-functions.sql
-- Contains functions for reviews

-- Add review
CREATE OR REPLACE FUNCTION add_review(
    p_user_id INT,
    p_marketplace_id INT,
    p_rating INT,
    p_comment TEXT
)
RETURNS INT AS $$
DECLARE
    new_review_id INT;
    marketplace_record RECORD;
BEGIN
    -- Check if user has purchased the strategy
    PERFORM 1 FROM strategy_purchases
    WHERE marketplace_id = p_marketplace_id AND buyer_id = p_user_id;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Must purchase strategy before reviewing';
    END IF;
    
    -- Check if user has already reviewed
    PERFORM 1 FROM strategy_reviews
    WHERE marketplace_id = p_marketplace_id AND user_id = p_user_id;
    
    IF FOUND THEN
        RAISE EXCEPTION 'Already reviewed this strategy';
    END IF;
    
    -- Get marketplace details
    SELECT 
        m.*, 
        s.user_id AS seller_id,
        s.name AS strategy_name
    INTO marketplace_record
    FROM 
        strategy_marketplace m
        JOIN strategies s ON m.strategy_id = s.id
    WHERE 
        m.id = p_marketplace_id;
    
    -- Insert review
    INSERT INTO strategy_reviews (
        marketplace_id,
        user_id,
        rating,
        comment,
        created_at,
        updated_at
    )
    VALUES (
        p_marketplace_id,
        p_user_id,
        p_rating,
        p_comment,
        NOW(),
        NOW()
    )
    RETURNING id INTO new_review_id;
    
    RETURN new_review_id;
END;
$$ LANGUAGE plpgsql;

-- Edit review
CREATE OR REPLACE FUNCTION edit_review(
    p_user_id INT,
    p_review_id INT,
    p_rating INT,
    p_comment TEXT
)
RETURNS BOOLEAN AS $$
DECLARE
    affected_rows INT;
BEGIN
    UPDATE strategy_reviews
    SET 
        rating = p_rating,
        comment = p_comment,
        updated_at = NOW()
    WHERE 
        id = p_review_id
        AND user_id = p_user_id;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows > 0;
END;
$$ LANGUAGE plpgsql;

-- Delete review
CREATE OR REPLACE FUNCTION delete_review(
    p_user_id INT,
    p_review_id INT
)
RETURNS BOOLEAN AS $$
DECLARE
    affected_rows INT;
BEGIN
    DELETE FROM strategy_reviews
    WHERE 
        id = p_review_id
        AND user_id = p_user_id;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows > 0;
END;
$$ LANGUAGE plpgsql;

-- Get review count and average rating
CREATE OR REPLACE FUNCTION get_strategy_rating(p_strategy_id INT)
RETURNS TABLE (
    avg_rating NUMERIC,
    rating_count BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COALESCE(AVG(r.rating), 0) AS avg_rating,
        COUNT(r.id) AS rating_count
    FROM 
        strategy_reviews r
        JOIN strategy_marketplace m ON r.marketplace_id = m.id
    WHERE 
        m.strategy_id = p_strategy_id;
END;
$$ LANGUAGE plpgsql;

-- Get reviews for a strategy
CREATE OR REPLACE FUNCTION get_strategy_reviews(
    p_strategy_id INT,
    p_limit INT DEFAULT 10,
    p_offset INT DEFAULT 0
)
RETURNS TABLE (
    review_id INT,
    user_id INT,
    rating INT,
    comment TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        r.id AS review_id,
        r.user_id,
        r.rating,
        r.comment,
        r.created_at,
        r.updated_at
    FROM 
        strategy_reviews r
        JOIN strategy_marketplace m ON r.marketplace_id = m.id
    WHERE 
        m.strategy_id = p_strategy_id
    ORDER BY 
        r.created_at DESC
    LIMIT p_limit
    OFFSET p_offset;
END;
$$ LANGUAGE plpgsql;