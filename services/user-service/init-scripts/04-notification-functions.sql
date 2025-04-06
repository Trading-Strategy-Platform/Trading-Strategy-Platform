-- User Service Database - Notification Functions

-- Get active notifications for a user
CREATE OR REPLACE FUNCTION get_active_notifications(p_user_id INT)
RETURNS TABLE (
    id INT,
    type notification_type,
    title VARCHAR(100),
    message TEXT,
    is_read BOOLEAN,
    link VARCHAR(255),
    created_at TIMESTAMP
) AS $$
BEGIN
    RETURN QUERY
    SELECT n.id, n.type, n.title, n.message, n.is_read, n.link, n.created_at
    FROM notifications n
    WHERE n.user_id = p_user_id
    ORDER BY n.created_at DESC;
END;
$$ LANGUAGE plpgsql;

-- Get unread notification count for a user
CREATE OR REPLACE FUNCTION get_unread_notification_count(p_user_id INT)
RETURNS INTEGER AS $$
DECLARE
    notification_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO notification_count
    FROM notifications
    WHERE user_id = p_user_id AND is_read = FALSE;
    
    RETURN notification_count;
END;
$$ LANGUAGE plpgsql;

-- Get all notifications for a user with pagination
CREATE OR REPLACE FUNCTION get_all_notifications(
    p_user_id INT,
    p_limit INT DEFAULT 100,
    p_offset INT DEFAULT 0
)
RETURNS TABLE (
    id INT,
    type notification_type,
    title VARCHAR(100),
    message TEXT,
    is_read BOOLEAN,
    link VARCHAR(255),
    created_at TIMESTAMP
) AS $$
BEGIN
    RETURN QUERY
    SELECT n.id, n.type, n.title, n.message, n.is_read, n.link, n.created_at
    FROM notifications n
    WHERE n.user_id = p_user_id
    ORDER BY n.created_at DESC
    LIMIT p_limit
    OFFSET p_offset;
END;
$$ LANGUAGE plpgsql;

-- Mark notification as read
CREATE OR REPLACE FUNCTION mark_notification_as_read(p_notification_id INT)
RETURNS BOOLEAN AS $$
DECLARE
    affected_rows INTEGER;
BEGIN
    UPDATE notifications
    SET is_read = TRUE
    WHERE id = p_notification_id;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows > 0;
END;
$$ LANGUAGE plpgsql;

-- Add notification
CREATE OR REPLACE FUNCTION add_notification(
    p_user_id INT,
    p_type notification_type,
    p_title VARCHAR(100),
    p_message TEXT,
    p_link VARCHAR(255) DEFAULT NULL
)
RETURNS INT AS $$
DECLARE
    new_notification_id INT;
BEGIN
    INSERT INTO notifications (user_id, type, title, message, link, is_read, created_at)
    VALUES (p_user_id, p_type, p_title, p_message, p_link, FALSE, NOW())
    RETURNING id INTO new_notification_id;
    
    RETURN new_notification_id;
END;
$$ LANGUAGE plpgsql;

-- Mark all notifications as read for a user
CREATE OR REPLACE FUNCTION mark_all_notifications_as_read(p_user_id INT)
RETURNS INTEGER AS $$
DECLARE
    affected_rows INTEGER;
BEGIN
    UPDATE notifications
    SET is_read = TRUE
    WHERE user_id = p_user_id AND is_read = FALSE;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows;
END;
$$ LANGUAGE plpgsql;

-- Delete notifications for a user
CREATE OR REPLACE FUNCTION delete_user_notifications(p_user_id INT)
RETURNS INTEGER AS $$
DECLARE
    affected_rows INTEGER;
BEGIN
    DELETE FROM notifications
    WHERE user_id = p_user_id;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows;
END;
$$ LANGUAGE plpgsql;

-- Get notification by ID
CREATE OR REPLACE FUNCTION get_notification_by_id(p_notification_id INT)
RETURNS TABLE (
    id INT,
    user_id INT,
    type notification_type,
    title VARCHAR(100),
    message TEXT,
    is_read BOOLEAN,
    link VARCHAR(255),
    created_at TIMESTAMP
) AS $$
BEGIN
    RETURN QUERY
    SELECT n.id, n.user_id, n.type, n.title, n.message, n.is_read, n.link, n.created_at
    FROM notifications n
    WHERE n.id = p_notification_id;
END;
$$ LANGUAGE plpgsql;