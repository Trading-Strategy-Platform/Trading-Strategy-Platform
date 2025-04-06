-- Strategy Service Strategy Functions
-- File: 04-strategy-functions.sql
-- Contains functions for strategy operations

-- Create a view for my strategies (created by me + purchased)
CREATE OR REPLACE VIEW v_my_strategies AS
SELECT 
    s.id, 
    s.name, 
    s.description, 
    s.thumbnail_url,
    s.user_id AS owner_id,
    s.user_id AS owner_user_id, -- Using user_id instead of username
    s.is_public,
    s.is_active,
    s.version,
    s.created_at,
    s.updated_at,
    'owner' AS access_type,
    NULL::INT AS purchase_id,
    NULL::TIMESTAMP AS purchase_date,
    ARRAY(
        SELECT tag_id 
        FROM strategy_tag_mappings 
        WHERE strategy_id = s.id
    ) AS tag_ids
FROM 
    strategies s
WHERE 
    s.is_active = TRUE

UNION ALL

SELECT 
    s.id, 
    s.name, 
    s.description,
    s.thumbnail_url,
    s.user_id AS owner_id,
    s.user_id AS owner_user_id, -- Using user_id instead of username
    s.is_public,
    s.is_active,
    v.version,
    p.created_at AS purchase_date,
    s.updated_at,
    'purchased' AS access_type,
    p.id AS purchase_id,
    p.created_at AS purchase_date,
    ARRAY(
        SELECT tag_id 
        FROM strategy_tag_mappings 
        WHERE strategy_id = s.id
    ) AS tag_ids
FROM 
    strategy_purchases p
    JOIN strategy_marketplace m ON p.marketplace_id = m.id
    JOIN strategies s ON m.strategy_id = s.id
    JOIN strategy_versions v ON s.id = v.strategy_id AND v.version = p.strategy_version
WHERE 
    s.is_active = TRUE 
    AND (
        p.subscription_end IS NULL
        OR p.subscription_end > NOW()
    );

-- Get all my strategies with filtering
CREATE OR REPLACE FUNCTION get_my_strategies(
    p_user_id INT,
    p_search_term VARCHAR DEFAULT NULL,
    p_purchased_only BOOLEAN DEFAULT FALSE,
    p_tags INT[] DEFAULT NULL
)
RETURNS TABLE (
    id INT,
    name VARCHAR(100),
    description TEXT,
    thumbnail_url VARCHAR(255),
    owner_id INT,
    owner_user_id INT, -- Changed from owner_username
    is_public BOOLEAN,
    is_active BOOLEAN,
    version INT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    access_type VARCHAR(20),
    purchase_id INT,
    purchase_date TIMESTAMP,
    tag_ids INT[]
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        s.id, 
        s.name, 
        s.description, 
        s.thumbnail_url,
        s.owner_id,
        s.owner_user_id,
        s.is_public,
        s.is_active,
        s.version,
        s.created_at,
        s.updated_at,
        s.access_type,
        s.purchase_id,
        s.purchase_date,
        s.tag_ids
    FROM 
        v_my_strategies s
    WHERE 
        (s.owner_id = p_user_id OR s.access_type = 'purchased') 
        AND (p_purchased_only = FALSE OR s.access_type = 'purchased')
        AND (
            p_search_term IS NULL 
            OR s.name ILIKE '%' || p_search_term || '%' 
            OR s.description ILIKE '%' || p_search_term || '%'
        )
        AND (
            p_tags IS NULL 
            OR s.tag_ids && p_tags
        )
        
    UNION ALL
    
    -- Add expired subscriptions (for strategies that no longer appear in v_my_strategies due to expired subscriptions)
    SELECT 
        s.id, 
        s.name, 
        s.description,
        s.thumbnail_url,
        s.user_id AS owner_id,
        s.user_id AS owner_user_id, -- Using user_id instead of username
        s.is_public,
        s.is_active,
        p.strategy_version AS version,
        p.created_at AS purchase_date,
        s.updated_at,
        'expired' AS access_type,
        p.id AS purchase_id,
        p.created_at AS purchase_date,
        ARRAY(
            SELECT tag_id 
            FROM strategy_tag_mappings 
            WHERE strategy_id = s.id
        ) AS tag_ids
    FROM 
        strategy_purchases p
        JOIN strategy_marketplace m ON p.marketplace_id = m.id
        JOIN strategies s ON m.strategy_id = s.id
    WHERE 
        p.buyer_id = p_user_id
        AND s.is_active = TRUE 
        AND p.subscription_end IS NOT NULL
        AND p.subscription_end <= NOW()
        AND (
            p_search_term IS NULL 
            OR s.name ILIKE '%' || p_search_term || '%' 
            OR s.description ILIKE '%' || p_search_term || '%'
        )
        AND (
            p_tags IS NULL 
            OR EXISTS (
                SELECT 1 
                FROM strategy_tag_mappings 
                WHERE strategy_id = s.id 
                AND tag_id = ANY(p_tags)
            )
        )
        AND (p_purchased_only = FALSE OR TRUE)
    
    ORDER BY 
        created_at DESC;
END;
$$ LANGUAGE plpgsql;

-- Add new strategy
CREATE OR REPLACE FUNCTION add_strategy(
    p_user_id INT,
    p_name VARCHAR(100),
    p_description TEXT,
    p_thumbnail_url VARCHAR(255),
    p_structure JSONB,
    p_is_public BOOLEAN,
    p_tag_ids INT[] DEFAULT NULL
)
RETURNS INT AS $$
DECLARE
    new_strategy_id INT;
    tag_id INT;
BEGIN
    -- Insert strategy
    INSERT INTO strategies (
        name, 
        user_id, 
        description, 
        thumbnail_url,
        structure, 
        is_public, 
        is_active,
        version, 
        created_at, 
        updated_at
    )
    VALUES (
        p_name, 
        p_user_id, 
        p_description, 
        p_thumbnail_url,
        p_structure, 
        p_is_public, 
        TRUE,
        1, 
        NOW(), 
        NOW()
    )
    RETURNING id INTO new_strategy_id;
    
    -- Insert initial version
    INSERT INTO strategy_versions (
        strategy_id, 
        version, 
        structure, 
        change_notes,
        is_deleted,
        created_at
    )
    VALUES (
        new_strategy_id, 
        1, 
        p_structure, 
        'Initial version',
        FALSE,
        NOW()
    );
    
    -- Add tags if provided
    IF p_tag_ids IS NOT NULL THEN
        FOREACH tag_id IN ARRAY p_tag_ids LOOP
            INSERT INTO strategy_tag_mappings (strategy_id, tag_id)
            VALUES (new_strategy_id, tag_id);
        END LOOP;
    END IF;
    
    RETURN new_strategy_id;
END;
$$ LANGUAGE plpgsql;

-- Update strategy (create new version)
CREATE OR REPLACE FUNCTION update_strategy(
    p_strategy_id INT,
    p_user_id INT,
    p_name VARCHAR(100),
    p_description TEXT,
    p_thumbnail_url VARCHAR(255),
    p_structure JSONB,
    p_is_public BOOLEAN,
    p_change_notes TEXT,
    p_tag_ids INT[] DEFAULT NULL
)
RETURNS BOOLEAN AS $$
DECLARE
    current_version INT;
    affected_rows INT;
    tag_id INT;
BEGIN
    -- Check ownership
    PERFORM 1 FROM strategies 
    WHERE id = p_strategy_id AND user_id = p_user_id;
    
    IF NOT FOUND THEN
        RETURN FALSE;
    END IF;
    
    -- Get current version
    SELECT version INTO current_version 
    FROM strategies 
    WHERE id = p_strategy_id;
    
    -- Update strategy
    UPDATE strategies
    SET 
        name = p_name,
        description = p_description,
        thumbnail_url = p_thumbnail_url,
        structure = p_structure,
        is_public = p_is_public,
        version = current_version + 1,
        updated_at = NOW()
    WHERE 
        id = p_strategy_id 
        AND user_id = p_user_id;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    
    IF affected_rows = 0 THEN
        RETURN FALSE;
    END IF;
    
    -- Create new version
    INSERT INTO strategy_versions (
        strategy_id, 
        version, 
        structure, 
        change_notes,
        is_deleted,
        created_at
    )
    VALUES (
        p_strategy_id, 
        current_version + 1, 
        p_structure, 
        p_change_notes,
        FALSE,
        NOW()
    );
    
    -- Update tags if provided
    IF p_tag_ids IS NOT NULL THEN
        -- Delete current tags
        DELETE FROM strategy_tag_mappings
        WHERE strategy_id = p_strategy_id;
        
        -- Add new tags
        FOREACH tag_id IN ARRAY p_tag_ids LOOP
            INSERT INTO strategy_tag_mappings (strategy_id, tag_id)
            VALUES (p_strategy_id, tag_id);
        END LOOP;
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Delete strategy (mark as inactive)
CREATE OR REPLACE FUNCTION delete_strategy(
    p_strategy_id INT,
    p_user_id INT
)
RETURNS BOOLEAN AS $$
DECLARE
    affected_rows INT;
BEGIN
    UPDATE strategies
    SET 
        is_active = FALSE,
        updated_at = NOW()
    WHERE 
        id = p_strategy_id 
        AND user_id = p_user_id;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows > 0;
END;
$$ LANGUAGE plpgsql;