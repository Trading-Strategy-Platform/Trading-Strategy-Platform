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
    s.user_id AS owner_user_id,
    s.is_public,
    s.is_active,
    s.version,
    s.created_at,
    s.updated_at,
    s.strategy_group_id,
    'owner'::text AS access_type,
    NULL::integer AS purchase_id,
    NULL::timestamp AS purchase_date,
    ARRAY(
        SELECT tag_id 
        FROM strategy_tag_mappings 
        WHERE strategy_id = s.strategy_group_id
    )::integer[] AS tag_ids,
    s.structure
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
    s.user_id AS owner_user_id,
    s.is_public,
    s.is_active,
    s.version,
    p.created_at,
    s.updated_at,
    s.strategy_group_id,
    'purchased'::text AS access_type,
    p.id AS purchase_id,
    p.created_at AS purchase_date,
    ARRAY(
        SELECT tag_id 
        FROM strategy_tag_mappings 
        WHERE strategy_id = s.strategy_group_id
    )::integer[] AS tag_ids,
    s.structure  
FROM 
    strategy_purchases p
    JOIN strategy_marketplace m ON p.marketplace_id = m.id
    JOIN strategies s ON s.id = p.strategy_version_id
WHERE 
    s.is_active = TRUE 
    AND (
        p.subscription_end IS NULL
        OR p.subscription_end > NOW()
    );

-- Get all my strategies with filtering and sorting
CREATE OR REPLACE FUNCTION get_all_strategies(
    p_user_id INTEGER,
    p_search_term CHARACTER VARYING DEFAULT NULL,
    p_purchased_only BOOLEAN DEFAULT FALSE,
    p_tags INTEGER[] DEFAULT NULL,
    p_sort_by VARCHAR DEFAULT 'created_at',
    p_sort_direction VARCHAR DEFAULT 'DESC',
    p_limit INTEGER DEFAULT 10,
    p_offset INTEGER DEFAULT 0
)
RETURNS TABLE (
    id INTEGER,
    name VARCHAR(100),
    description TEXT,
    thumbnail_url VARCHAR(255),
    owner_id INTEGER,
    owner_user_id INTEGER,
    is_public BOOLEAN,
    is_active BOOLEAN,
    version INTEGER,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    strategy_group_id INTEGER,
    access_type TEXT,
    purchase_id INTEGER,
    purchase_date TIMESTAMP,
    tag_ids INTEGER[],
    structure JSONB
) AS $$
DECLARE
    owned_search_condition TEXT := '';
    purchased_search_condition TEXT := '';
    tags_condition TEXT := '';
    sort_clause TEXT := '';
    query TEXT;
BEGIN
    -- Build separate search conditions for owned vs purchased strategies
    IF p_search_term IS NOT NULL THEN
        -- For owned strategies (no marketplace table)
        owned_search_condition := ' AND (s.name ILIKE ''%' || p_search_term || '%'' OR s.description ILIKE ''%' || p_search_term || '%'')';
        
        -- For purchased strategies (includes marketplace table)
        purchased_search_condition := ' AND (s.name ILIKE ''%' || p_search_term || '%'' OR s.description ILIKE ''%' || p_search_term || '%'' OR m.description_public ILIKE ''%' || p_search_term || '%'')';
    END IF;
    
    -- Build tags condition if tags are provided
    IF p_tags IS NOT NULL AND array_length(p_tags, 1) > 0 THEN
        tags_condition := ' AND EXISTS (SELECT 1 FROM strategy_tag_mappings WHERE strategy_id = s.strategy_group_id AND tag_id = ANY(''' || p_tags::text || '''::int[]))';
    END IF;

    -- Validate sort field
    IF p_sort_by NOT IN ('name', 'created_at', 'updated_at', 'version') THEN
        p_sort_by := 'created_at'; -- Default sort by creation date
    END IF;
    
    -- Normalize sort direction
    IF UPPER(p_sort_direction) NOT IN ('ASC', 'DESC') THEN
        p_sort_direction := 'DESC'; -- Default sorting from newest to oldest
    ELSE
        p_sort_direction := UPPER(p_sort_direction);
    END IF;
    
    -- Build sort clause
    sort_clause := ' ORDER BY ';
    
    IF p_sort_by = 'name' THEN
        sort_clause := sort_clause || 'name ' || p_sort_direction;
    ELSIF p_sort_by = 'updated_at' THEN
        sort_clause := sort_clause || 'updated_at ' || p_sort_direction || ' NULLS LAST';
    ELSIF p_sort_by = 'version' THEN
        sort_clause := sort_clause || 'version ' || p_sort_direction;
    ELSE -- Default to created_at
        sort_clause := sort_clause || 'created_at ' || p_sort_direction;
    END IF;

    -- Start building the query
    query := '
        WITH combined_results AS (';
    
    -- Part 1: Include owned strategies if not filtering by purchased only
    IF NOT p_purchased_only THEN
        query := query || '
            SELECT 
                s.id, 
                s.name, 
                s.description AS description,
                s.thumbnail_url,
                s.user_id AS owner_id,
                s.user_id AS owner_user_id,
                s.is_public,
                s.is_active,
                s.version,
                s.created_at,
                s.updated_at,
                s.strategy_group_id,
                ''owner''::text AS access_type,
                NULL::integer AS purchase_id,
                NULL::timestamp AS purchase_date,
                ARRAY(
                    SELECT tag_id 
                    FROM strategy_tag_mappings 
                    WHERE strategy_id = s.strategy_group_id
                )::integer[] AS tag_ids,
                s.structure
            FROM strategies s
            LEFT JOIN user_strategy_versions usv ON s.strategy_group_id = usv.strategy_group_id AND usv.user_id = ' || p_user_id || '
            WHERE 
                s.user_id = ' || p_user_id || '
                AND s.is_active = TRUE
                AND (usv.active_version_id IS NULL OR s.id = usv.active_version_id)' ||
                owned_search_condition || -- Use owned search condition
                tags_condition;

        -- Add UNION if we're going to add more parts
        query := query || '
            UNION ALL';
    END IF;
    
    -- Part 2: Always include active purchased strategies
    query := query || '
        SELECT 
            s.id, 
            s.name, 
            m.description_public AS description,
            s.thumbnail_url,
            s.user_id AS owner_id,
            s.user_id AS owner_user_id,
            s.is_public,
            s.is_active,
            s.version,
            p.created_at,
            s.updated_at,
            s.strategy_group_id,
            ''purchased''::text AS access_type,
            p.id AS purchase_id,
            p.created_at AS purchase_date,
            ARRAY(
                SELECT tag_id 
                FROM strategy_tag_mappings 
                WHERE strategy_id = s.strategy_group_id
            )::integer[] AS tag_ids,
            s.structure
        FROM 
            strategy_purchases p
            JOIN strategy_marketplace m ON p.marketplace_id = m.id
            JOIN strategies s ON p.strategy_version_id = s.id
        WHERE 
            p.buyer_id = ' || p_user_id || '
            AND s.is_active = TRUE 
            AND (
                p.subscription_end IS NULL
                OR p.subscription_end > NOW()
            )' ||
            purchased_search_condition || -- Use purchased search condition
            tags_condition;
            
    -- Part 3: Include expired subscriptions if not filtering by purchased only
    IF NOT p_purchased_only THEN
        query := query || '
            UNION ALL
            SELECT 
                s.id, 
                s.name, 
                m.description_public AS description,
                s.thumbnail_url,
                s.user_id AS owner_id,
                s.user_id AS owner_user_id,
                s.is_public,
                s.is_active,
                s.version,
                p.created_at,
                s.updated_at,
                s.strategy_group_id,
                ''expired''::text AS access_type,
                p.id AS purchase_id,
                p.created_at AS purchase_date,
                ARRAY(
                    SELECT tag_id 
                    FROM strategy_tag_mappings 
                    WHERE strategy_id = s.strategy_group_id
                )::integer[] AS tag_ids,
                s.structure
            FROM 
                strategy_purchases p
                JOIN strategy_marketplace m ON p.marketplace_id = m.id
                JOIN strategies s ON p.strategy_version_id = s.id
            WHERE 
                p.buyer_id = ' || p_user_id || '
                AND s.is_active = TRUE 
                AND p.subscription_end IS NOT NULL
                AND p.subscription_end <= NOW()' ||
                purchased_search_condition || -- Use purchased search condition
                tags_condition;
    END IF;
    
    -- Close the CTE and apply sorting and pagination
    query := query || '
        )
        SELECT * FROM combined_results' ||
        sort_clause || '
        LIMIT ' || p_limit || ' OFFSET ' || p_offset;
        
    -- Execute the dynamically built query
    RETURN QUERY EXECUTE query;
END;
$$ LANGUAGE plpgsql;

-- Create a new strategy
CREATE OR REPLACE FUNCTION create_strategy(
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
        updated_at,
        strategy_group_id
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
        NOW(),
        0
    )
    RETURNING id INTO new_strategy_id;
    
    -- Update strategy_group_id to be the same as ID
    UPDATE strategies 
    SET strategy_group_id = new_strategy_id
    WHERE id = new_strategy_id;
    
    -- Set as user's active version
    INSERT INTO user_strategy_versions (
        user_id,
        strategy_group_id,
        active_version_id,
        updated_at
    )
    VALUES (
        p_user_id,
        new_strategy_id,
        new_strategy_id,
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
RETURNS INT AS $$
DECLARE
    strategy_group_id INT;
    current_version INT;
    affected_rows INT;
    tag_id INT;
    new_version_id INT;
BEGIN
    -- Check ownership
    SELECT s.strategy_group_id, s.version 
    INTO strategy_group_id, current_version
    FROM strategies s
    WHERE s.id = p_strategy_id AND s.user_id = p_user_id;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Strategy not found or you do not have permission to update it';
    END IF;
    
    -- Create new version
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
        updated_at,
        strategy_group_id
    )
    VALUES (
        p_name, 
        p_user_id, 
        p_description, 
        p_thumbnail_url,
        p_structure, 
        p_is_public, 
        TRUE,
        current_version + 1, 
        NOW(), 
        NOW(),
        strategy_group_id
    )
    RETURNING id INTO new_version_id;
    
    -- Update user's active version to the new version
    INSERT INTO user_strategy_versions (
        user_id,
        strategy_group_id,
        active_version_id,
        updated_at
    )
    VALUES (
        p_user_id,
        strategy_group_id,
        new_version_id,
        NOW()
    )
    ON CONFLICT (user_id, strategy_group_id) 
    DO UPDATE SET 
        active_version_id = new_version_id,
        updated_at = NOW();
    
    -- Update tags if provided
    IF p_tag_ids IS NOT NULL THEN
        -- Delete current tags
        DELETE FROM strategy_tag_mappings
        WHERE strategy_id = strategy_group_id;
        
        -- Add new tags
        FOREACH tag_id IN ARRAY p_tag_ids LOOP
            INSERT INTO strategy_tag_mappings (strategy_id, tag_id)
            VALUES (strategy_group_id, tag_id);
        END LOOP;
    END IF;
    
    RETURN new_version_id;
END;
$$ LANGUAGE plpgsql;

-- Delete strategy (mark as inactive)
CREATE OR REPLACE FUNCTION delete_strategy(
    p_strategy_id INT,
    p_user_id INT
)
RETURNS BOOLEAN AS $$
DECLARE
    strategy_group_id INT;
    affected_rows INT;
BEGIN
    -- Get the strategy group ID
    SELECT s.strategy_group_id INTO strategy_group_id
    FROM strategies s
    WHERE s.id = p_strategy_id AND s.user_id = p_user_id;
    
    IF NOT FOUND THEN
        RETURN FALSE;
    END IF;

    -- Mark all versions in this group as inactive
    UPDATE strategies
    SET 
        is_active = FALSE,
        updated_at = NOW()
    WHERE 
        strategy_group_id = strategy_group_id;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows > 0;
END;
$$ LANGUAGE plpgsql;


-- Count strategies with various filters
CREATE OR REPLACE FUNCTION count_strategies(
    p_user_id INTEGER,
    p_search_term CHARACTER VARYING DEFAULT NULL,
    p_purchased_only BOOLEAN DEFAULT FALSE,
    p_tags INTEGER[] DEFAULT NULL
)
RETURNS BIGINT AS $$
DECLARE
    owned_search_condition TEXT := '';
    purchased_search_condition TEXT := '';
    tags_condition TEXT := '';
    query TEXT;
    total_count BIGINT;
BEGIN
    -- Build separate search conditions for owned vs purchased strategies
    IF p_search_term IS NOT NULL THEN
        owned_search_condition := ' AND (s.name ILIKE ''%' || p_search_term || '%'' OR s.description ILIKE ''%' || p_search_term || '%'')';
        purchased_search_condition := ' AND (s.name ILIKE ''%' || p_search_term || '%'' OR s.description ILIKE ''%' || p_search_term || '%'' OR m.description_public ILIKE ''%' || p_search_term || '%'')';
    END IF;
    
    -- Build tags condition if tags are provided
    IF p_tags IS NOT NULL AND array_length(p_tags, 1) > 0 THEN
        tags_condition := ' AND EXISTS (SELECT 1 FROM strategy_tag_mappings WHERE strategy_id = s.strategy_group_id AND tag_id = ANY(''' || p_tags::text || '''::int[]))';
    END IF;

    -- Start building the count query
    query := 'SELECT COUNT(*) FROM (';
    
    -- Part 1: Include owned strategies if not filtering by purchased only
    IF NOT p_purchased_only THEN
        query := query || '
            SELECT s.id 
            FROM strategies s
            LEFT JOIN user_strategy_versions usv ON s.strategy_group_id = usv.strategy_group_id AND usv.user_id = ' || p_user_id || '
            WHERE s.user_id = ' || p_user_id || '
                AND s.is_active = TRUE
                AND (usv.active_version_id IS NULL OR s.id = usv.active_version_id)' ||
                owned_search_condition || 
                tags_condition;

        -- Add UNION if we're going to add more parts
        query := query || ' UNION ALL ';
    END IF;
    
    -- Part 2: Always include active purchased strategies
    query := query || '
        SELECT s.id
        FROM strategy_purchases p
        JOIN strategy_marketplace m ON p.marketplace_id = m.id
        JOIN strategies s ON p.strategy_version_id = s.id
        WHERE p.buyer_id = ' || p_user_id || '
            AND s.is_active = TRUE 
            AND (p.subscription_end IS NULL OR p.subscription_end > NOW())' ||
            purchased_search_condition || 
            tags_condition;
            
    -- Part 3: Include expired subscriptions if not filtering by purchased only
    IF NOT p_purchased_only THEN
        query := query || ' UNION ALL
            SELECT s.id
            FROM strategy_purchases p
            JOIN strategy_marketplace m ON p.marketplace_id = m.id
            JOIN strategies s ON p.strategy_version_id = s.id
            WHERE p.buyer_id = ' || p_user_id || '
                AND s.is_active = TRUE 
                AND p.subscription_end IS NOT NULL
                AND p.subscription_end <= NOW()' ||
                purchased_search_condition || 
                tags_condition;
    END IF;
    
    -- Close the subquery
    query := query || ') AS total_strategies';
        
    -- Execute the count query
    EXECUTE query INTO total_count;
    
    RETURN total_count;
END;
$$ LANGUAGE plpgsql;

-- Get strategy by ID
CREATE OR REPLACE FUNCTION get_strategy_by_id(
    p_strategy_id INT,
    p_user_id INT
)
RETURNS TABLE (
    id INT,
    name VARCHAR(100),
    user_id INT,
    description TEXT,
    thumbnail_url VARCHAR(255),
    structure JSONB,
    is_public BOOLEAN,
    is_active BOOLEAN,
    version INT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    strategy_group_id INT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        s.id,
        s.name,
        s.user_id,
        s.description,
        s.thumbnail_url,
        s.structure,
        s.is_public,
        s.is_active,
        s.version,
        s.created_at,
        s.updated_at,
        s.strategy_group_id
    FROM 
        strategies s
    LEFT JOIN
        user_strategy_versions usv ON s.strategy_group_id = usv.strategy_group_id AND usv.user_id = p_user_id
    WHERE 
        (
            -- Case 1: User owns the strategy, show their active version or the strategy directly requested
            (s.user_id = p_user_id AND (s.id = p_strategy_id OR (s.strategy_group_id = p_strategy_id AND (usv.active_version_id = s.id OR usv.active_version_id IS NULL))))
            
            OR
            
            -- Case 2: User purchased the strategy, show the version they bought
            EXISTS (
                SELECT 1 
                FROM strategy_purchases p
                JOIN strategy_marketplace m ON p.marketplace_id = m.id
                WHERE p.buyer_id = p_user_id
                AND p.strategy_version_id = s.id
                AND (s.id = p_strategy_id OR s.strategy_group_id = p_strategy_id)
                AND (p.subscription_end IS NULL OR p.subscription_end > NOW())
            )
            
            OR
            
            -- Case 3: Strategy is public and the user is accessing by ID directly
            (s.is_public = TRUE AND s.id = p_strategy_id)
        )
        AND s.is_active = TRUE;
END;
$$ LANGUAGE plpgsql;

-- Get all versions of a strategy
CREATE OR REPLACE FUNCTION get_strategy_versions(
    p_strategy_group_id INT,
    p_user_id INT,
    p_sort_by VARCHAR DEFAULT 'version',
    p_sort_direction VARCHAR DEFAULT 'DESC',
    p_limit INT DEFAULT 20,
    p_offset INT DEFAULT 0
)
RETURNS TABLE (
    id INT,
    name VARCHAR(100),
    user_id INT,
    description TEXT,
    thumbnail_url VARCHAR(255),
    structure JSONB,
    is_public BOOLEAN,
    is_active BOOLEAN,
    version INT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    strategy_group_id INT,
    is_current_version BOOLEAN
) AS $$
DECLARE
    owner_id INT;
    is_owner BOOLEAN;
    has_purchase BOOLEAN;
BEGIN
    -- Get the strategy owner
    SELECT s.user_id INTO owner_id
    FROM strategies s
    WHERE s.strategy_group_id = p_strategy_group_id
    LIMIT 1;
    
    -- Determine if user is owner
    is_owner := (owner_id = p_user_id);
    
    -- Determine if user has purchased the strategy
    SELECT EXISTS (
        SELECT 1 
        FROM strategy_purchases p
        JOIN strategy_marketplace m ON p.marketplace_id = m.id
        JOIN strategies s ON p.strategy_version_id = s.id
        WHERE p.buyer_id = p_user_id
        AND s.strategy_group_id = p_strategy_group_id
        AND (p.subscription_end IS NULL OR p.subscription_end > NOW())
    ) INTO has_purchase;
    
    -- Get the user's current active version of this strategy
    RETURN QUERY
    WITH user_active_version AS (
        SELECT active_version_id
        FROM user_strategy_versions
        WHERE user_id = p_user_id AND strategy_group_id = p_strategy_group_id
    )
    SELECT 
        s.id,
        s.name,
        s.user_id,
        s.description,
        s.thumbnail_url,
        s.structure,
        s.is_public,
        s.is_active,
        s.version,
        s.created_at,
        s.updated_at,
        s.strategy_group_id,
        CASE WHEN uav.active_version_id = s.id THEN TRUE ELSE FALSE END AS is_current_version
    FROM strategies s
    LEFT JOIN user_active_version uav ON TRUE
    WHERE 
        s.strategy_group_id = p_strategy_group_id
        AND s.is_active = TRUE
        AND (
            is_owner = TRUE  -- Owner sees all versions
            OR
            (
                has_purchase = TRUE AND  -- Purchaser only sees their purchased version
                EXISTS (
                    SELECT 1 
                    FROM strategy_purchases p
                    JOIN strategy_marketplace m ON p.marketplace_id = m.id
                    WHERE p.buyer_id = p_user_id
                    AND p.strategy_version_id = s.id
                    AND (p.subscription_end IS NULL OR p.subscription_end > NOW())
                )
            )
            OR
            s.is_public = TRUE  -- Anyone can see public versions
        )
    ORDER BY
        CASE WHEN p_sort_by = 'version' AND p_sort_direction = 'DESC' THEN s.version END DESC,
        CASE WHEN p_sort_by = 'version' AND p_sort_direction = 'ASC' THEN s.version END ASC,
        CASE WHEN p_sort_by = 'created_at' AND p_sort_direction = 'DESC' THEN s.created_at END DESC,
        CASE WHEN p_sort_by = 'created_at' AND p_sort_direction = 'ASC' THEN s.created_at END ASC
    LIMIT p_limit OFFSET p_offset;
END;
$$ LANGUAGE plpgsql;

-- Count strategy versions
CREATE OR REPLACE FUNCTION count_strategy_versions(
    p_strategy_group_id INT,
    p_user_id INT
)
RETURNS BIGINT AS $$
DECLARE
    owner_id INT;
    is_owner BOOLEAN;
    has_purchase BOOLEAN;
    version_count BIGINT;
BEGIN
    -- Get the strategy owner
    SELECT s.user_id INTO owner_id
    FROM strategies s
    WHERE s.strategy_group_id = p_strategy_group_id
    LIMIT 1;
    
    -- Determine if user is owner
    is_owner := (owner_id = p_user_id);
    
    -- Determine if user has purchased the strategy
    SELECT EXISTS (
        SELECT 1 
        FROM strategy_purchases p
        JOIN strategy_marketplace m ON p.marketplace_id = m.id
        JOIN strategies s ON p.strategy_version_id = s.id
        WHERE p.buyer_id = p_user_id
        AND s.strategy_group_id = p_strategy_group_id
        AND (p.subscription_end IS NULL OR p.subscription_end > NOW())
    ) INTO has_purchase;
    
    -- Count versions based on access level
    IF is_owner THEN
        -- Owner sees all versions
        SELECT COUNT(*) INTO version_count
        FROM strategies s
        WHERE s.strategy_group_id = p_strategy_group_id AND s.is_active = TRUE;
    ELSIF has_purchase THEN
        -- Purchaser only sees their purchased version
        SELECT COUNT(*) INTO version_count
        FROM strategies s
        JOIN strategy_purchases p ON p.strategy_version_id = s.id
        JOIN strategy_marketplace m ON p.marketplace_id = m.id
        WHERE s.strategy_group_id = p_strategy_group_id
        AND s.is_active = TRUE
        AND p.buyer_id = p_user_id
        AND (p.subscription_end IS NULL OR p.subscription_end > NOW());
    ELSE
        -- Public versions only
        SELECT COUNT(*) INTO version_count
        FROM strategies s
        WHERE s.strategy_group_id = p_strategy_group_id
        AND s.is_active = TRUE
        AND s.is_public = TRUE;
    END IF;
    
    RETURN version_count;
END;
$$ LANGUAGE plpgsql;

-- Set user's active version for a strategy
CREATE OR REPLACE FUNCTION set_user_active_version(
    p_user_id INT,
    p_strategy_group_id INT,
    p_version_id INT
)
RETURNS BOOLEAN AS $$
DECLARE
    strategy_exists BOOLEAN;
BEGIN
    -- Check if version exists and is part of the given strategy group
    SELECT EXISTS (
        SELECT 1 
        FROM strategies s
        WHERE s.id = p_version_id
        AND s.strategy_group_id = p_strategy_group_id
        AND s.is_active = TRUE
    ) INTO strategy_exists;
    
    IF NOT strategy_exists THEN
        RETURN FALSE;
    END IF;
    
    -- Check if user has access to this version
    IF NOT EXISTS (
        -- Owner of the strategy
        SELECT 1 FROM strategies s
        WHERE s.id = p_version_id
        AND s.user_id = p_user_id
        
        UNION ALL
        
        -- Purchased the strategy
        SELECT 1 FROM strategy_purchases p
        JOIN strategy_marketplace m ON p.marketplace_id = m.id
        WHERE p.buyer_id = p_user_id
        AND p.strategy_version_id = p_version_id
        AND (p.subscription_end IS NULL OR p.subscription_end > NOW())
    ) THEN
        RETURN FALSE;
    END IF;
    
    -- Update or insert user preference
    INSERT INTO user_strategy_versions (
        user_id,
        strategy_group_id,
        active_version_id,
        updated_at
    )
    VALUES (
        p_user_id,
        p_strategy_group_id,
        p_version_id,
        NOW()
    )
    ON CONFLICT (user_id, strategy_group_id) DO UPDATE
    SET 
        active_version_id = p_version_id,
        updated_at = NOW();
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;
