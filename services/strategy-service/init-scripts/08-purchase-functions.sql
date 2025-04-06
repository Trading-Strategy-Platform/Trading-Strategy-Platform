-- Strategy Service Purchase Functions
-- File: 08-purchase-functions.sql
-- Contains functions for purchases and subscriptions

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