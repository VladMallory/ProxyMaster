-- Миграция для добавления полей дополнительного инбаунда
-- Выполнить: psql -d vpn_bot -f migrations/add_secondary_inbound_fields.sql

-- Добавляем поля для дополнительного инбаунда
ALTER TABLE users ADD COLUMN IF NOT EXISTS secondary_client_id VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS secondary_sub_id VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS secondary_email VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS secondary_config_created_at TIMESTAMP;
ALTER TABLE users ADD COLUMN IF NOT EXISTS secondary_expiry_time BIGINT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS has_active_secondary_config BOOLEAN DEFAULT FALSE;

-- Добавляем индекс для нового поля has_active_secondary_config
CREATE INDEX IF NOT EXISTS idx_users_has_active_secondary_config ON users(has_active_secondary_config);

-- Комментарии к новым полям
COMMENT ON COLUMN users.secondary_client_id IS 'ID клиента в дополнительном инбаунде';
COMMENT ON COLUMN users.secondary_sub_id IS 'SubID клиента в дополнительном инбаунде';
COMMENT ON COLUMN users.secondary_email IS 'Email клиента в дополнительном инбаунде';
COMMENT ON COLUMN users.secondary_config_created_at IS 'Время создания конфига в дополнительном инбаунде';
COMMENT ON COLUMN users.secondary_expiry_time IS 'Время истечения конфига в дополнительном инбаунде';
COMMENT ON COLUMN users.has_active_secondary_config IS 'Флаг активности конфига в дополнительном инбаунде';

-- Обновляем представления для учета дополнительного инбаунда
DROP VIEW IF EXISTS active_users;
CREATE VIEW active_users AS
SELECT * FROM users WHERE has_active_config = true OR has_active_secondary_config = true;

-- Обновляем функцию статистики для учета дополнительного инбаунда
CREATE OR REPLACE FUNCTION get_users_statistics()
RETURNS TABLE(
    total_users INTEGER,
    paying_users INTEGER,
    trial_available_users INTEGER,
    trial_used_users INTEGER,
    inactive_users INTEGER,
    active_configs INTEGER,
    active_secondary_configs INTEGER,
    total_revenue DECIMAL(10,2),
    new_this_week INTEGER,
    new_this_month INTEGER,
    conversion_rate DECIMAL(5,2)
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COUNT(*)::INTEGER as total_users,
        COUNT(CASE WHEN u.total_paid > 0 THEN 1 END)::INTEGER as paying_users,
        COUNT(CASE WHEN u.has_used_trial = false AND u.balance <= 0 THEN 1 END)::INTEGER as trial_available_users,
        COUNT(CASE WHEN u.has_used_trial = true AND u.total_paid <= 0 THEN 1 END)::INTEGER as trial_used_users,
        COUNT(CASE WHEN u.has_active_config = false AND u.has_active_secondary_config = false THEN 1 END)::INTEGER as inactive_users,
        COUNT(CASE WHEN u.has_active_config = true THEN 1 END)::INTEGER as active_configs,
        COUNT(CASE WHEN u.has_active_secondary_config = true THEN 1 END)::INTEGER as active_secondary_configs,
        COALESCE(SUM(u.total_paid), 0)::DECIMAL(10,2) as total_revenue,
        COUNT(CASE WHEN u.created_at >= NOW() - INTERVAL '7 days' THEN 1 END)::INTEGER as new_this_week,
        COUNT(CASE WHEN u.created_at >= NOW() - INTERVAL '30 days' THEN 1 END)::INTEGER as new_this_month,
        CASE 
            WHEN COUNT(*) > 0 THEN 
                (COUNT(CASE WHEN u.total_paid > 0 THEN 1 END) * 100.0 / COUNT(*))::DECIMAL(5,2)
            ELSE 0::DECIMAL(5,2)
        END as conversion_rate
    FROM users u;
END;
$$ LANGUAGE plpgsql;
