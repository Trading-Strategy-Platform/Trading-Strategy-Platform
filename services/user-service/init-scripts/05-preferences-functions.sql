-- User Service Database - Preference Functions

-- Get user preferences
CREATE OR REPLACE FUNCTION get_user_preferences(p_user_id INT)
RETURNS TABLE (
    theme VARCHAR(20),
    default_timeframe VARCHAR(10),
    chart_preferences JSONB,
    notification_settings JSONB
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COALESCE(p.theme, 'light') as theme,
        COALESCE(p.default_timeframe, '1h') as default_timeframe,
        COALESCE(p.chart_preferences, '{}'::jsonb) as chart_preferences,
        COALESCE(p.notification_settings, '{}'::jsonb) as notification_settings
    FROM user_preferences p
    WHERE p.user_id = p_user_id;
    
    -- If no row was found, return default values
    IF NOT FOUND THEN
        RETURN QUERY
        SELECT 
            'light'::VARCHAR as theme,
            '1h'::VARCHAR as default_timeframe,
            '{}'::jsonb as chart_preferences,
            '{}'::jsonb as notification_settings;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Update user preferences
CREATE OR REPLACE FUNCTION update_user_preferences(
    p_user_id INT,
    p_theme VARCHAR(20) DEFAULT NULL,
    p_default_timeframe VARCHAR(10) DEFAULT NULL,
    p_chart_preferences JSONB DEFAULT NULL,
    p_notification_settings JSONB DEFAULT NULL
)
RETURNS BOOLEAN AS $$
DECLARE
    preference_exists BOOLEAN;
BEGIN
    -- Check if preferences exist
    SELECT EXISTS(SELECT 1 FROM user_preferences WHERE user_id = p_user_id) INTO preference_exists;
    
    IF preference_exists THEN
        -- Update existing preferences
        UPDATE user_preferences
        SET 
            theme = COALESCE(p_theme, theme),
            default_timeframe = COALESCE(p_default_timeframe, default_timeframe),
            chart_preferences = COALESCE(p_chart_preferences, chart_preferences),
            notification_settings = COALESCE(p_notification_settings, notification_settings),
            updated_at = NOW()
        WHERE 
            user_id = p_user_id;
    ELSE
        -- Insert new preferences
        INSERT INTO user_preferences (
            user_id,
            theme,
            default_timeframe,
            chart_preferences,
            notification_settings,
            created_at,
            updated_at
        )
        VALUES (
            p_user_id,
            COALESCE(p_theme, 'light'),
            COALESCE(p_default_timeframe, '1h'),
            COALESCE(p_chart_preferences, '{}'::jsonb),
            COALESCE(p_notification_settings, '{}'::jsonb),
            NOW(),
            NOW()
        );
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Check if user preferences exist
CREATE OR REPLACE FUNCTION check_user_preferences_exist(p_user_id INT)
RETURNS BOOLEAN AS $$
DECLARE
    preference_exists BOOLEAN;
BEGIN
    SELECT EXISTS(SELECT 1 FROM user_preferences WHERE user_id = p_user_id) INTO preference_exists;
    RETURN preference_exists;
END;
$$ LANGUAGE plpgsql;

-- Delete user preferences
CREATE OR REPLACE FUNCTION delete_user_preferences(p_user_id INT)
RETURNS BOOLEAN AS $$
DECLARE
    affected_rows INT;
BEGIN
    DELETE FROM user_preferences
    WHERE user_id = p_user_id;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows > 0;
END;
$$ LANGUAGE plpgsql;