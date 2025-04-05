-- Strategy Service Database Schema
CREATE TYPE "user_role" AS ENUM (
  'admin',
  'user'
);

CREATE TYPE "timeframe_type" AS ENUM (
  '1m',
  '5m',
  '15m',
  '30m',
  '1h',
  '4h',
  '1d',
  '1w'
);

-- Strategies
CREATE TABLE IF NOT EXISTS "strategies" (
  "id" SERIAL PRIMARY KEY,
  "name" varchar(100) NOT NULL,
  "user_id" int NOT NULL,
  "description" text,
  "thumbnail_url" varchar(255),
  "structure" jsonb NOT NULL,
  "is_public" boolean NOT NULL DEFAULT false,
  "is_active" boolean NOT NULL DEFAULT true,
  "version" int NOT NULL DEFAULT 1,
  "created_at" timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  "updated_at" timestamp
);

-- Strategy Versions
CREATE TABLE IF NOT EXISTS "strategy_versions" (
  "id" SERIAL PRIMARY KEY,
  "strategy_id" int NOT NULL,
  "version" int NOT NULL,
  "structure" jsonb NOT NULL,
  "change_notes" text,
  "is_deleted" boolean NOT NULL DEFAULT false,
  "created_at" timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP)
);

-- Strategy Tags
CREATE TABLE IF NOT EXISTS "strategy_tags" (
  "id" SERIAL PRIMARY KEY,
  "name" varchar(50) UNIQUE NOT NULL
);

-- Strategy to Tag Mappings
CREATE TABLE IF NOT EXISTS "strategy_tag_mappings" (
  "strategy_id" int NOT NULL,
  "tag_id" int NOT NULL,
  PRIMARY KEY ("strategy_id", "tag_id")
);

-- User Strategy Versions
CREATE TABLE IF NOT EXISTS "user_strategy_versions" (
  "id" SERIAL PRIMARY KEY,
  "user_id" int NOT NULL,
  "strategy_id" int NOT NULL,
  "active_version" int NOT NULL,
  "updated_at" timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP)
);

-- Technical Indicators
CREATE TABLE IF NOT EXISTS "indicators" (
  "id" SERIAL PRIMARY KEY,
  "name" varchar(50) UNIQUE NOT NULL,
  "description" text,
  "category" varchar(50),
  "formula" text,
  "created_at" timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  "updated_at" timestamp
);

-- Indicator Parameters
CREATE TABLE IF NOT EXISTS "indicator_parameters" (
  "id" SERIAL PRIMARY KEY,
  "indicator_id" int NOT NULL,
  "parameter_name" varchar(50) NOT NULL,
  "parameter_type" varchar(20) NOT NULL,
  "is_required" boolean NOT NULL DEFAULT true,
  "min_value" float,
  "max_value" float,
  "default_value" varchar(50),
  "description" text
);

-- Parameter Enum Values
CREATE TABLE IF NOT EXISTS "parameter_enum_values" (
  "id" SERIAL PRIMARY KEY,
  "parameter_id" int NOT NULL,
  "enum_value" varchar(50) NOT NULL,
  "display_name" varchar(100)
);

-- Marketplace Listings
CREATE TABLE IF NOT EXISTS "strategy_marketplace" (
  "id" SERIAL PRIMARY KEY,
  "strategy_id" int NOT NULL,
  "version_id" int NOT NULL DEFAULT 1,
  "user_id" int NOT NULL,
  "price" numeric(10,2) NOT NULL DEFAULT 0,
  "is_subscription" boolean NOT NULL DEFAULT false,
  "subscription_period" varchar(20),
  "is_active" boolean NOT NULL DEFAULT true,
  "description" text,
  "created_at" timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  "updated_at" timestamp
);

-- Strategy Purchases
CREATE TABLE IF NOT EXISTS "strategy_purchases" (
  "id" SERIAL PRIMARY KEY,
  "marketplace_id" int NOT NULL,
  "buyer_id" int NOT NULL,
  "strategy_version" int NOT NULL,
  "purchase_price" numeric(10,2) NOT NULL,
  "subscription_end" timestamp,
  "created_at" timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP)
);

-- Strategy Reviews
CREATE TABLE IF NOT EXISTS "strategy_reviews" (
  "id" SERIAL PRIMARY KEY,
  "marketplace_id" int NOT NULL,
  "user_id" int NOT NULL,
  "rating" int NOT NULL,
  "comment" text,
  "created_at" timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  "updated_at" timestamp
);

-- Indexes
CREATE INDEX "idx_strategies_user_id" ON "strategies" ("user_id");
CREATE INDEX ON "strategies" ("is_public");
CREATE INDEX ON "strategies" ("is_active");
CREATE UNIQUE INDEX ON "strategy_versions" ("strategy_id", "version");
CREATE UNIQUE INDEX ON "indicator_parameters" ("indicator_id", "parameter_name");
CREATE INDEX ON "strategy_marketplace" ("is_active");
CREATE UNIQUE INDEX ON "strategy_marketplace" ("strategy_id", "version_id");
CREATE UNIQUE INDEX ON "strategy_reviews" ("marketplace_id", "user_id");
CREATE UNIQUE INDEX ON "user_strategy_versions" ("user_id", "strategy_id");

-- Foreign Keys
ALTER TABLE "strategy_versions" ADD FOREIGN KEY ("strategy_id") REFERENCES "strategies" ("id") ON DELETE CASCADE;
ALTER TABLE "strategy_tag_mappings" ADD FOREIGN KEY ("strategy_id") REFERENCES "strategies" ("id") ON DELETE CASCADE;
ALTER TABLE "strategy_tag_mappings" ADD FOREIGN KEY ("tag_id") REFERENCES "strategy_tags" ("id") ON DELETE CASCADE;
ALTER TABLE "indicator_parameters" ADD FOREIGN KEY ("indicator_id") REFERENCES "indicators" ("id") ON DELETE CASCADE;
ALTER TABLE "parameter_enum_values" ADD FOREIGN KEY ("parameter_id") REFERENCES "indicator_parameters" ("id") ON DELETE CASCADE;
ALTER TABLE "strategy_marketplace" ADD FOREIGN KEY ("strategy_id") REFERENCES "strategies" ("id") ON DELETE CASCADE;
ALTER TABLE "strategy_purchases" ADD FOREIGN KEY ("marketplace_id") REFERENCES "strategy_marketplace" ("id") ON DELETE CASCADE;
ALTER TABLE "strategy_reviews" ADD FOREIGN KEY ("marketplace_id") REFERENCES "strategy_marketplace" ("id") ON DELETE CASCADE;
ALTER TABLE "user_strategy_versions" ADD FOREIGN KEY ("strategy_id") REFERENCES "strategies" ("id") ON DELETE CASCADE;

-- Insert default tags
INSERT INTO strategy_tags (name) VALUES 
('Trend Following'),
('Mean Reversion'),
('Momentum'),
('Breakout'),
('Volatility'),
('Swing Trading'),
('Scalping'),
('Day Trading'),
('Algorithmic'),
('Machine Learning')
ON CONFLICT (name) DO NOTHING;


-- ==========================================
-- STRATEGIES FUNCTIONS AND VIEWS
-- ==========================================

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

-- Get strategy tags
CREATE OR REPLACE FUNCTION get_strategy_tags()
RETURNS TABLE (
    id INT,
    name VARCHAR(50)
) AS $$
BEGIN
    RETURN QUERY
    SELECT t.id, t.name
    FROM strategy_tags t
    ORDER BY t.name;
END;
$$ LANGUAGE plpgsql;

-- Add strategy tag
CREATE OR REPLACE FUNCTION add_strategy_tag(p_name VARCHAR(50))
RETURNS INT AS $$
DECLARE
    new_tag_id INT;
BEGIN
    INSERT INTO strategy_tags (name)
    VALUES (p_name)
    RETURNING id INTO new_tag_id;
    
    RETURN new_tag_id;
END;
$$ LANGUAGE plpgsql;

-- ==========================================
-- MARKETPLACE, PURCHASES, AND REVIEWS
-- ==========================================

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

-- Purchase a strategy
CREATE OR REPLACE FUNCTION purchase_strategy(
    p_user_id INT,
    p_marketplace_id INT
)
RETURNS INT AS $$
DECLARE
    new_purchase_id INT;
    marketplace_record RECORD;
BEGIN
    -- Get marketplace listing details
    SELECT 
        m.*, 
        s.user_id AS seller_id,
        s.name AS strategy_name
    INTO marketplace_record
    FROM 
        strategy_marketplace m
        JOIN strategies s ON m.strategy_id = s.id
    WHERE 
        m.id = p_marketplace_id
        AND m.is_active = TRUE;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Marketplace listing not found or inactive';
    END IF;
    
    -- Check user is not buying their own strategy
    IF marketplace_record.seller_id = p_user_id THEN
        RAISE EXCEPTION 'Cannot purchase your own strategy';
    END IF;
    
    -- Check for existing purchase
    PERFORM 1 FROM strategy_purchases
    WHERE marketplace_id = p_marketplace_id AND buyer_id = p_user_id;
    
    IF FOUND THEN
        RAISE EXCEPTION 'Already purchased this strategy';
    END IF;
    
    -- Insert purchase record
    INSERT INTO strategy_purchases (
        marketplace_id,
        buyer_id,
        strategy_version,
        purchase_price,
        subscription_end,
        created_at
    )
    VALUES (
        p_marketplace_id,
        p_user_id,
        marketplace_record.version_id,
        marketplace_record.price,
        CASE 
            WHEN marketplace_record.is_subscription THEN 
                CASE
                    WHEN marketplace_record.subscription_period = 'monthly' THEN NOW() + INTERVAL '1 month'
                    WHEN marketplace_record.subscription_period = 'quarterly' THEN NOW() + INTERVAL '3 months'
                    WHEN marketplace_record.subscription_period = 'yearly' THEN NOW() + INTERVAL '1 year'
                    ELSE NULL
                END
            ELSE NULL
        END,
        NOW()
    )
    RETURNING id INTO new_purchase_id;
    
    -- Add user_strategy_versions record for initial version tracking
    INSERT INTO user_strategy_versions (
        user_id,
        strategy_id,
        active_version,
        updated_at
    )
    VALUES (
        p_user_id,
        marketplace_record.strategy_id,
        marketplace_record.version_id,
        NOW()
    )
    ON CONFLICT (user_id, strategy_id) DO UPDATE
    SET active_version = marketplace_record.version_id;
    
    -- Return the purchase ID
    RETURN new_purchase_id;
END;
$$ LANGUAGE plpgsql;

-- Cancel subscription
CREATE OR REPLACE FUNCTION cancel_subscription(
    p_user_id INT,
    p_purchase_id INT
)
RETURNS BOOLEAN AS $$
DECLARE
    affected_rows INT;
    marketplace_record RECORD;
BEGIN
    -- Get strategy details
    SELECT 
        m.strategy_id,
        s.user_id AS seller_id,
        s.name AS strategy_name
    INTO marketplace_record
    FROM 
        strategy_purchases p
        JOIN strategy_marketplace m ON p.marketplace_id = m.id
        JOIN strategies s ON m.strategy_id = s.id
    WHERE 
        p.id = p_purchase_id
        AND p.buyer_id = p_user_id;
        
    IF NOT FOUND THEN
        RETURN FALSE;
    END IF;
    
    -- Update subscription end
    UPDATE strategy_purchases
    SET subscription_end = NOW()
    WHERE 
        id = p_purchase_id
        AND buyer_id = p_user_id
        AND subscription_end > NOW();
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    
    RETURN affected_rows > 0;
END;
$$ LANGUAGE plpgsql;

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

-- ==========================================
-- INDICATORS FUNCTIONS AND VIEWS
-- ==========================================

-- Create view for indicators with parameters
CREATE OR REPLACE VIEW v_indicators_with_parameters AS
SELECT 
    i.id,
    i.name,
    i.description,
    i.category,
    i.formula,
    i.created_at,
    i.updated_at,
    ARRAY(
        SELECT jsonb_build_object(
            'id', p.id,
            'name', p.parameter_name,
            'type', p.parameter_type,
            'is_required', p.is_required,
            'min_value', p.min_value,
            'max_value', p.max_value,
            'default_value', p.default_value,
            'description', p.description,
            'enum_values', (
                SELECT jsonb_agg(jsonb_build_object(
                    'id', ev.id,
                    'value', ev.enum_value,
                    'display_name', ev.display_name
                ))
                FROM parameter_enum_values ev
                WHERE ev.parameter_id = p.id
            )
        )
        FROM indicator_parameters p
        WHERE p.indicator_id = i.id
    ) AS parameters
FROM 
    indicators i;

-- Create the get_indicators function
CREATE OR REPLACE FUNCTION get_indicators(p_category VARCHAR)
RETURNS TABLE (
    id INT,
    name VARCHAR(50),
    description TEXT,
    category VARCHAR(50),
    formula TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    parameters JSONB[]
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        i.id,
        i.name,
        i.description,
        i.category,
        i.formula,
        i.created_at,
        i.updated_at,
        ARRAY(
            SELECT jsonb_build_object(
                'id', p.id,
                'name', p.parameter_name,
                'type', p.parameter_type,
                'is_required', p.is_required,
                'min_value', p.min_value,
                'max_value', p.max_value,
                'default_value', p.default_value,
                'description', p.description,
                'enum_values', (
                    SELECT jsonb_agg(jsonb_build_object(
                        'id', ev.id,
                        'value', ev.enum_value,
                        'display_name', ev.display_name
                    ))
                    FROM parameter_enum_values ev
                    WHERE ev.parameter_id = p.id
                )
            )
            FROM indicator_parameters p
            WHERE p.indicator_id = i.id
        ) AS parameters
    FROM 
        indicators i
    WHERE
        p_category IS NULL OR i.category = p_category
    ORDER BY 
        i.name;
END;
$$ LANGUAGE plpgsql;

-- Get indicator details by ID
CREATE OR REPLACE FUNCTION get_indicator_by_id(p_indicator_id INT)
RETURNS TABLE (
    id INT,
    name VARCHAR(50),
    description TEXT,
    category VARCHAR(50),
    formula TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    parameters JSONB[]
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        i.id,
        i.name,
        i.description,
        i.category,
        i.formula,
        i.created_at,
        i.updated_at,
        i.parameters
    FROM 
        v_indicators_with_parameters i
    WHERE
        i.id = p_indicator_id;
END;
$$ LANGUAGE plpgsql;

-- Add new indicator
CREATE OR REPLACE FUNCTION add_indicator(
    p_name VARCHAR(50),
    p_description TEXT,
    p_category VARCHAR(50),
    p_formula TEXT DEFAULT NULL
)
RETURNS INT AS $$
DECLARE
    new_indicator_id INT;
BEGIN
    -- Check uniqueness
    PERFORM 1 FROM indicators 
    WHERE name = p_name;
    
    IF FOUND THEN
        RAISE EXCEPTION 'Indicator name already exists';
    END IF;
    
    INSERT INTO indicators (
        name,
        description,
        category,
        formula,
        created_at,
        updated_at
    )
    VALUES (
        p_name,
        p_description,
        p_category,
        p_formula,
        NOW(),
        NOW()
    )
    RETURNING id INTO new_indicator_id;
    
    RETURN new_indicator_id;
END;
$$ LANGUAGE plpgsql;

-- Add indicator parameter
CREATE OR REPLACE FUNCTION add_indicator_parameter(
    p_indicator_id INT,
    p_parameter_name VARCHAR(50),
    p_parameter_type VARCHAR(20),
    p_is_required BOOLEAN,
    p_min_value FLOAT DEFAULT NULL,
    p_max_value FLOAT DEFAULT NULL,
    p_default_value VARCHAR(50) DEFAULT NULL,
    p_description TEXT DEFAULT NULL
)
RETURNS INT AS $$
DECLARE
    new_parameter_id INT;
BEGIN
    -- Check uniqueness
    PERFORM 1 FROM indicator_parameters 
    WHERE indicator_id = p_indicator_id AND parameter_name = p_parameter_name;
    
    IF FOUND THEN
        RAISE EXCEPTION 'Parameter name already exists for this indicator';
    END IF;
    
    INSERT INTO indicator_parameters (
        indicator_id,
        parameter_name,
        parameter_type,
        is_required,
        min_value,
        max_value,
        default_value,
        description
    )
    VALUES (
        p_indicator_id,
        p_parameter_name,
        p_parameter_type,
        p_is_required,
        p_min_value,
        p_max_value,
        p_default_value,
        p_description
    )
    RETURNING id INTO new_parameter_id;
    
    RETURN new_parameter_id;
END;
$$ LANGUAGE plpgsql;

-- Add parameter enum value
CREATE OR REPLACE FUNCTION add_parameter_enum_value(
    p_parameter_id INT,
    p_enum_value VARCHAR(50),
    p_display_name VARCHAR(100) DEFAULT NULL
)
RETURNS INT AS $$
DECLARE
    new_enum_id INT;
BEGIN
    -- Check uniqueness
    PERFORM 1 FROM parameter_enum_values 
    WHERE parameter_id = p_parameter_id AND enum_value = p_enum_value;
    
    IF FOUND THEN
        RAISE EXCEPTION 'Enum value already exists for this parameter';
    END IF;
    
    INSERT INTO parameter_enum_values (
        parameter_id,
        enum_value,
        display_name
    )
    VALUES (
        p_parameter_id,
        p_enum_value,
        COALESCE(p_display_name, p_enum_value)
    )
    RETURNING id INTO new_enum_id;
    
    RETURN new_enum_id;
END;
$$ LANGUAGE plpgsql;

-- Get indicator categories
CREATE OR REPLACE FUNCTION get_indicator_categories()
RETURNS TABLE (
    category VARCHAR(50),
    count BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        i.category,
        COUNT(*) AS count
    FROM 
        indicators i
    GROUP BY 
        i.category
    ORDER BY 
        i.category;
END;
$$ LANGUAGE plpgsql;