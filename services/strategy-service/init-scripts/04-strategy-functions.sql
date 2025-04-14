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
    'owner'::text AS access_type,
    NULL::integer AS purchase_id,
    NULL::timestamp AS purchase_date,
    ARRAY(
        SELECT tag_id 
        FROM strategy_tag_mappings 
        WHERE strategy_id = s.id
    )::integer[] AS tag_ids,
    s.structure  -- Added structure field
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
    p.strategy_version AS version,
    p.created_at,
    s.updated_at,
    'purchased'::text AS access_type,
    p.id AS purchase_id,
    p.created_at AS purchase_date,
    ARRAY(
        SELECT tag_id 
        FROM strategy_tag_mappings 
        WHERE strategy_id = s.id
    )::integer[] AS tag_ids,
    s.structure  
FROM 
    strategy_purchases p
    JOIN strategy_marketplace m ON p.marketplace_id = m.id
    JOIN strategies s ON m.strategy_id = s.id
WHERE 
    s.is_active = TRUE 
    AND (
        p.subscription_end IS NULL
        OR p.subscription_end > NOW()
    );

-- Get all my strategies with filtering
CREATE OR REPLACE FUNCTION get_my_strategies(
    p_user_id integer,
    p_search_term character varying DEFAULT NULL,
    p_purchased_only boolean DEFAULT false,
    p_tags integer[] DEFAULT NULL
)
RETURNS TABLE (
    id integer,
    name varchar(100),
    description text,
    thumbnail_url varchar(255),
    owner_id integer,
    owner_user_id integer,
    is_public boolean,
    is_active boolean,
    version integer,
    created_at timestamp,
    updated_at timestamp,
    access_type text,
    purchase_id integer,
    purchase_date timestamp,
    tag_ids integer[],
    structure jsonb
) AS $$
DECLARE
    owned_search_condition text := '';
    purchased_search_condition text := '';
    tags_condition text := '';
    query text;
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
        tags_condition := ' AND EXISTS (SELECT 1 FROM strategy_tag_mappings WHERE strategy_id = s.id AND tag_id = ANY(''' || p_tags::text || '''::int[]))';
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
                ''owner''::text AS access_type,
                NULL::integer AS purchase_id,
                NULL::timestamp AS purchase_date,
                ARRAY(
                    SELECT tag_id 
                    FROM strategy_tag_mappings 
                    WHERE strategy_id = s.id
                )::integer[] AS tag_ids,
                s.structure
            FROM 
                strategies s
            WHERE 
                s.user_id = ' || p_user_id || '
                AND s.is_active = TRUE' ||
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
            p.strategy_version AS version,
            p.created_at,
            s.updated_at,
            ''purchased''::text AS access_type,
            p.id AS purchase_id,
            p.created_at AS purchase_date,
            ARRAY(
                SELECT tag_id 
                FROM strategy_tag_mappings 
                WHERE strategy_id = s.id
            )::integer[] AS tag_ids,
            s.structure
        FROM 
            strategy_purchases p
            JOIN strategy_marketplace m ON p.marketplace_id = m.id
            JOIN strategies s ON m.strategy_id = s.id
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
                p.strategy_version AS version,
                p.created_at,
                s.updated_at,
                ''expired''::text AS access_type,
                p.id AS purchase_id,
                p.created_at AS purchase_date,
                ARRAY(
                    SELECT tag_id 
                    FROM strategy_tag_mappings 
                    WHERE strategy_id = s.id
                )::integer[] AS tag_ids,
                s.structure
            FROM 
                strategy_purchases p
                JOIN strategy_marketplace m ON p.marketplace_id = m.id
                JOIN strategies s ON m.strategy_id = s.id
            WHERE 
                p.buyer_id = ' || p_user_id || '
                AND s.is_active = TRUE 
                AND p.subscription_end IS NOT NULL
                AND p.subscription_end <= NOW()' ||
                purchased_search_condition || -- Use purchased search condition
                tags_condition;
    END IF;
    
    -- Close the CTE and order the results
    query := query || '
        )
        SELECT * FROM combined_results
        ORDER BY created_at DESC';
        
    -- Execute the dynamically built query
    RETURN QUERY EXECUTE query;
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