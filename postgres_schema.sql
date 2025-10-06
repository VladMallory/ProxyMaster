-- PostgreSQL схема для VPN бота
-- Миграция с MongoDB на PostgreSQL

-- Создание базы данных (выполнить отдельно под суперпользователем)
-- CREATE DATABASE vpn_bot;
-- CREATE USER vpn_bot_user WITH ENCRYPTED PASSWORD 'your_secure_password';
-- GRANT ALL PRIVILEGES ON DATABASE vpn_bot TO vpn_bot_user;

-- Подключиться к базе vpn_bot и выполнить следующее:

-- Основная таблица пользователей
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    telegram_id BIGINT UNIQUE NOT NULL,
    username VARCHAR(255),
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    balance DECIMAL(10,2) DEFAULT 0.00,
    total_paid DECIMAL(10,2) DEFAULT 0.00,
    configs_count INTEGER DEFAULT 0,
    has_active_config BOOLEAN DEFAULT FALSE,
    client_id VARCHAR(255),
    sub_id VARCHAR(255),
    email VARCHAR(255),
    config_created_at TIMESTAMP,
    expiry_time BIGINT,
    has_used_trial BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    -- Реферальная система
    referral_code VARCHAR(50) UNIQUE,
    referred_by BIGINT,
    referral_earnings DECIMAL(10,2) DEFAULT 0.00,
    referral_count INTEGER DEFAULT 0,
    -- Дополнительный инбаунд
    secondary_client_id VARCHAR(255),
    secondary_sub_id VARCHAR(255),
    secondary_email VARCHAR(255),
    secondary_config_created_at TIMESTAMP,
    secondary_expiry_time BIGINT,
    has_active_secondary_config BOOLEAN DEFAULT FALSE
);

-- Настройки трафика
CREATE TABLE traffic_configs (
    id VARCHAR(50) PRIMARY KEY DEFAULT 'default',
    enabled BOOLEAN DEFAULT TRUE,
    daily_limit_gb INTEGER,
    weekly_limit_gb INTEGER,
    monthly_limit_gb INTEGER,
    limit_gb INTEGER,
    reset_days INTEGER,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- IP подключения (с автоочисткой)
CREATE TABLE ip_connections (
    id SERIAL PRIMARY KEY,
    telegram_id BIGINT,
    ip_address INET,
    connection_data JSONB, -- Дополнительные данные подключения
    timestamp TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (telegram_id) REFERENCES users(telegram_id) ON DELETE CASCADE
);

-- IP нарушения
CREATE TABLE ip_violations (
    id SERIAL PRIMARY KEY,
    telegram_id BIGINT,
    ip_address INET,
    is_blocked BOOLEAN DEFAULT FALSE,
    violation_count INTEGER DEFAULT 1,
    violation_type VARCHAR(100),
    violation_data JSONB, -- Дополнительные данные о нарушении
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (telegram_id) REFERENCES users(telegram_id) ON DELETE CASCADE
);

-- === РЕФЕРАЛЬНАЯ СИСТЕМА ===

-- Таблица для отслеживания реферальных переходов
CREATE TABLE referral_transitions (
    id SERIAL PRIMARY KEY,
    referrer_telegram_id BIGINT NOT NULL,
    referred_telegram_id BIGINT NOT NULL,
    referral_code VARCHAR(50) NOT NULL,
    transition_date TIMESTAMP DEFAULT NOW(),
    bonus_paid BOOLEAN DEFAULT FALSE,
    bonus_amount DECIMAL(10,2) DEFAULT 0.00,
    created_at TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (referrer_telegram_id) REFERENCES users(telegram_id) ON DELETE CASCADE,
    FOREIGN KEY (referred_telegram_id) REFERENCES users(telegram_id) ON DELETE CASCADE
);

-- Таблица для истории реферальных бонусов
CREATE TABLE referral_bonuses (
    id SERIAL PRIMARY KEY,
    user_telegram_id BIGINT NOT NULL,
    bonus_type VARCHAR(20) NOT NULL, -- 'referrer' или 'referred'
    amount DECIMAL(10,2) NOT NULL,
    referral_code VARCHAR(50),
    related_user_id BIGINT, -- ID пользователя, связанного с бонусом
    description TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (user_telegram_id) REFERENCES users(telegram_id) ON DELETE CASCADE
);

-- Индексы для производительности
CREATE INDEX idx_users_telegram_id ON users(telegram_id);
CREATE INDEX idx_users_created_at ON users(created_at);
CREATE INDEX idx_users_has_active_config ON users(has_active_config);
CREATE INDEX idx_users_has_active_secondary_config ON users(has_active_secondary_config);
CREATE INDEX idx_users_has_used_trial ON users(has_used_trial);
CREATE INDEX idx_users_balance ON users(balance);

CREATE INDEX idx_ip_connections_telegram_timestamp ON ip_connections(telegram_id, timestamp DESC);
CREATE INDEX idx_ip_connections_timestamp ON ip_connections(timestamp);
CREATE INDEX idx_ip_connections_ip ON ip_connections(ip_address);

CREATE INDEX idx_ip_violations_telegram_blocked ON ip_violations(telegram_id, is_blocked);
CREATE INDEX idx_ip_violations_ip ON ip_violations(ip_address);
CREATE INDEX idx_ip_violations_created_at ON ip_violations(created_at);

-- Индексы для реферальной системы
CREATE INDEX idx_users_referral_code ON users(referral_code);
CREATE INDEX idx_users_referred_by ON users(referred_by);
CREATE INDEX idx_referral_transitions_referrer ON referral_transitions(referrer_telegram_id);
CREATE INDEX idx_referral_transitions_referred ON referral_transitions(referred_telegram_id);
CREATE INDEX idx_referral_transitions_code ON referral_transitions(referral_code);
CREATE INDEX idx_referral_bonuses_user ON referral_bonuses(user_telegram_id);
CREATE INDEX idx_referral_bonuses_type ON referral_bonuses(bonus_type);
CREATE INDEX idx_referral_bonuses_created_at ON referral_bonuses(created_at);

-- Функция для автоматического обновления updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Триггеры для автоматического обновления updated_at
CREATE TRIGGER update_users_updated_at 
    BEFORE UPDATE ON users 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_traffic_configs_updated_at 
    BEFORE UPDATE ON traffic_configs 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_ip_violations_updated_at 
    BEFORE UPDATE ON ip_violations 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Вставка конфигурации трафика по умолчанию
INSERT INTO traffic_configs (id, enabled, daily_limit_gb, weekly_limit_gb, monthly_limit_gb, limit_gb, reset_days)
VALUES ('default', true, 0, 0, 0, 0, 30)
ON CONFLICT (id) DO NOTHING;

-- Функция для очистки старых IP подключений (аналог TTL в MongoDB)
CREATE OR REPLACE FUNCTION cleanup_old_ip_connections()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM ip_connections 
    WHERE timestamp < NOW() - INTERVAL '1 hour';
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    IF deleted_count > 0 THEN
        RAISE NOTICE 'Удалено старых IP подключений: %', deleted_count;
    END IF;
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- === ФУНКЦИИ РЕФЕРАЛЬНОЙ СИСТЕМЫ ===

-- Функция для генерации уникального реферального кода
CREATE OR REPLACE FUNCTION generate_referral_code(telegram_id BIGINT)
RETURNS VARCHAR(50) AS $$
DECLARE
    code VARCHAR(50);
    exists_count INTEGER;
BEGIN
    -- Генерируем код на основе telegram_id + случайные символы
    code := 'REF' || telegram_id || LPAD(FLOOR(RANDOM() * 1000)::TEXT, 3, '0');
    
    -- Проверяем уникальность
    SELECT COUNT(*) INTO exists_count FROM users WHERE referral_code = code;
    
    -- Если код уже существует, генерируем новый
    WHILE exists_count > 0 LOOP
        code := 'REF' || telegram_id || LPAD(FLOOR(RANDOM() * 10000)::TEXT, 4, '0');
        SELECT COUNT(*) INTO exists_count FROM users WHERE referral_code = code;
    END LOOP;
    
    RETURN code;
END;
$$ LANGUAGE plpgsql;

-- Функция для обработки реферального перехода
CREATE OR REPLACE FUNCTION process_referral_transition(
    referrer_id BIGINT,
    referred_id BIGINT,
    referral_code VARCHAR(50)
)
RETURNS BOOLEAN AS $$
DECLARE
    referrer_exists BOOLEAN;
    referred_exists BOOLEAN;
    already_referred BOOLEAN;
    referrer_balance DECIMAL(10,2);
BEGIN
    -- Проверяем существование пользователей
    SELECT EXISTS(SELECT 1 FROM users WHERE telegram_id = referrer_id) INTO referrer_exists;
    SELECT EXISTS(SELECT 1 FROM users WHERE telegram_id = referred_id) INTO referred_exists;
    
    IF NOT referrer_exists OR NOT referred_exists THEN
        RETURN FALSE;
    END IF;
    
    -- Проверяем, не был ли уже приглашен этот пользователь
    SELECT EXISTS(SELECT 1 FROM referral_transitions WHERE referred_telegram_id = referred_id) INTO already_referred;
    
    IF already_referred THEN
        RETURN FALSE;
    END IF;
    
    -- Проверяем, что пользователь не приглашает сам себя
    IF referrer_id = referred_id THEN
        RETURN FALSE;
    END IF;
    
    -- Записываем переход
    INSERT INTO referral_transitions (referrer_telegram_id, referred_telegram_id, referral_code)
    VALUES (referrer_id, referred_id, referral_code);
    
    -- Обновляем счетчик рефералов у пригласившего
    UPDATE users SET referral_count = referral_count + 1 WHERE telegram_id = referrer_id;
    
    -- Устанавливаем связь у приглашенного
    UPDATE users SET referred_by = referrer_id WHERE telegram_id = referred_id;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Функция для начисления реферального бонуса
CREATE OR REPLACE FUNCTION award_referral_bonus(
    user_id BIGINT,
    bonus_type VARCHAR(20),
    amount DECIMAL(10,2),
    referral_code VARCHAR(50) DEFAULT NULL,
    related_user_id BIGINT DEFAULT NULL,
    description TEXT DEFAULT NULL
)
RETURNS BOOLEAN AS $$
DECLARE
    current_balance DECIMAL(10,2);
BEGIN
    -- Получаем текущий баланс
    SELECT balance INTO current_balance FROM users WHERE telegram_id = user_id;
    
    -- Обновляем баланс
    UPDATE users SET balance = balance + amount WHERE telegram_id = user_id;
    
    -- Если это бонус пригласившему, обновляем общую сумму реферальных заработков
    IF bonus_type = 'referrer' THEN
        UPDATE users SET referral_earnings = referral_earnings + amount WHERE telegram_id = user_id;
    END IF;
    
    -- Записываем в историю бонусов
    INSERT INTO referral_bonuses (user_telegram_id, bonus_type, amount, referral_code, related_user_id, description)
    VALUES (user_id, bonus_type, amount, referral_code, related_user_id, description);
    
    -- Обновляем статус выплаты в referral_transitions
    IF bonus_type = 'referrer' THEN
        UPDATE referral_transitions 
        SET bonus_paid = TRUE, bonus_amount = amount 
        WHERE referrer_telegram_id = user_id AND referred_telegram_id = related_user_id;
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Представления для удобства работы
CREATE VIEW active_users AS
SELECT * FROM users WHERE has_active_config = true;

CREATE VIEW trial_available_users AS
SELECT * FROM users WHERE has_used_trial = false AND balance <= 0;

CREATE VIEW paying_users AS
SELECT * FROM users WHERE total_paid > 0;

-- Функция для получения статистики пользователей
CREATE OR REPLACE FUNCTION get_users_statistics()
RETURNS TABLE(
    total_users INTEGER,
    paying_users INTEGER,
    trial_available_users INTEGER,
    trial_used_users INTEGER,
    inactive_users INTEGER,
    active_configs INTEGER,
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
        COUNT(CASE WHEN u.has_active_config = false THEN 1 END)::INTEGER as inactive_users,
        COUNT(CASE WHEN u.has_active_config = true THEN 1 END)::INTEGER as active_configs,
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

-- Выдача прав пользователю базы данных
-- GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO vpn_bot_user;
-- GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO vpn_bot_user;
-- GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO vpn_bot_user;

COMMENT ON TABLE users IS 'Пользователи VPN бота';
COMMENT ON TABLE traffic_configs IS 'Настройки трафика';
COMMENT ON TABLE ip_connections IS 'Временные подключения IP адресов (TTL 1 час)';
COMMENT ON TABLE ip_violations IS 'Нарушения и блокировки IP адресов';
COMMENT ON TABLE referral_transitions IS 'Отслеживание реферальных переходов';
COMMENT ON TABLE referral_bonuses IS 'История реферальных бонусов';

-- Комментарии к полям реферальной системы
COMMENT ON COLUMN users.referral_code IS 'Уникальный реферальный код пользователя';
COMMENT ON COLUMN users.referred_by IS 'Telegram ID пользователя, который пригласил';
COMMENT ON COLUMN users.referral_earnings IS 'Общая сумма заработанных реферальных бонусов';
COMMENT ON COLUMN users.referral_count IS 'Количество приглашенных пользователей';

-- === МУЛЬТИПОДПИСКИ ===

-- Таблица серверов для мультиподписок
CREATE TABLE multi_servers (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    country VARCHAR(100) NOT NULL,
    country_code VARCHAR(3) NOT NULL,
    flag VARCHAR(10) NOT NULL,
    inbound_id INTEGER NOT NULL,
    config_url VARCHAR(500) NOT NULL,
    json_url VARCHAR(500) NOT NULL,
    protocol VARCHAR(20) NOT NULL DEFAULT 'vless',
    transport VARCHAR(20) NOT NULL DEFAULT 'websocket',
    enabled BOOLEAN DEFAULT TRUE,
    priority INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Таблица мультиподписок пользователей
CREATE TABLE multi_subscriptions (
    id VARCHAR(50) PRIMARY KEY,
    user_id BIGINT NOT NULL,
    subscription_url VARCHAR(500) NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    expiry_time BIGINT,
    FOREIGN KEY (user_id) REFERENCES users(telegram_id) ON DELETE CASCADE
);

-- Таблица связи мультиподписок с серверами (many-to-many)
CREATE TABLE multi_subscription_servers (
    id SERIAL PRIMARY KEY,
    subscription_id VARCHAR(50) NOT NULL,
    server_id VARCHAR(50) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (subscription_id) REFERENCES multi_subscriptions(id) ON DELETE CASCADE,
    FOREIGN KEY (server_id) REFERENCES multi_servers(id) ON DELETE CASCADE,
    UNIQUE(subscription_id, server_id)
);

-- Таблица состояний выбора серверов (для временного хранения)
CREATE TABLE server_selection_states (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    selected_servers JSONB NOT NULL DEFAULT '[]',
    max_servers INTEGER DEFAULT 5,
    step VARCHAR(20) DEFAULT 'select',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP DEFAULT (NOW() + INTERVAL '1 hour'),
    FOREIGN KEY (user_id) REFERENCES users(telegram_id) ON DELETE CASCADE
);

-- Индексы для мультиподписок
CREATE INDEX idx_multi_servers_enabled ON multi_servers(enabled);
CREATE INDEX idx_multi_servers_priority ON multi_servers(priority);
CREATE INDEX idx_multi_servers_country ON multi_servers(country);

CREATE INDEX idx_multi_subscriptions_user ON multi_subscriptions(user_id);
CREATE INDEX idx_multi_subscriptions_active ON multi_subscriptions(is_active);
CREATE INDEX idx_multi_subscriptions_created ON multi_subscriptions(created_at);

CREATE INDEX idx_multi_subscription_servers_subscription ON multi_subscription_servers(subscription_id);
CREATE INDEX idx_multi_subscription_servers_server ON multi_subscription_servers(server_id);

CREATE INDEX idx_server_selection_states_user ON server_selection_states(user_id);
CREATE INDEX idx_server_selection_states_expires ON server_selection_states(expires_at);

-- Триггеры для автоматического обновления updated_at
CREATE TRIGGER update_multi_servers_updated_at 
    BEFORE UPDATE ON multi_servers 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_multi_subscriptions_updated_at 
    BEFORE UPDATE ON multi_subscriptions 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_server_selection_states_updated_at 
    BEFORE UPDATE ON server_selection_states 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Функция для очистки истекших состояний выбора серверов
CREATE OR REPLACE FUNCTION cleanup_expired_server_selection_states()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM server_selection_states 
    WHERE expires_at < NOW();
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    IF deleted_count > 0 THEN
        RAISE NOTICE 'Удалено истекших состояний выбора серверов: %', deleted_count;
    END IF;
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Функция для получения мультиподписки пользователя
CREATE OR REPLACE FUNCTION get_user_multi_subscription(user_telegram_id BIGINT)
RETURNS TABLE(
    subscription_id VARCHAR(50),
    subscription_url VARCHAR(500),
    is_active BOOLEAN,
    created_at TIMESTAMP,
    expiry_time BIGINT,
    servers JSONB
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        ms.id as subscription_id,
        ms.subscription_url,
        ms.is_active,
        ms.created_at,
        ms.expiry_time,
        COALESCE(
            jsonb_agg(
                jsonb_build_object(
                    'id', s.id,
                    'name', s.name,
                    'country', s.country,
                    'country_code', s.country_code,
                    'flag', s.flag,
                    'inbound_id', s.inbound_id,
                    'config_url', s.config_url,
                    'json_url', s.json_url,
                    'protocol', s.protocol,
                    'transport', s.transport,
                    'enabled', s.enabled,
                    'priority', s.priority
                ) ORDER BY s.priority DESC, s.name
            ) FILTER (WHERE s.id IS NOT NULL),
            '[]'::jsonb
        ) as servers
    FROM multi_subscriptions ms
    LEFT JOIN multi_subscription_servers mss ON ms.id = mss.subscription_id
    LEFT JOIN multi_servers s ON mss.server_id = s.id
    WHERE ms.user_id = user_telegram_id
    GROUP BY ms.id, ms.subscription_url, ms.is_active, ms.created_at, ms.expiry_time;
END;
$$ LANGUAGE plpgsql;

-- Вставка базовых серверов по умолчанию
INSERT INTO multi_servers (id, name, country, country_code, flag, inbound_id, config_url, json_url, protocol, transport, enabled, priority) VALUES
('server_de_1', 'Германия #1', 'Германия', 'DE', '🇩🇪', 1, 'https://example.com/config/de1', 'https://example.com/json/de1', 'vless', 'websocket', true, 100),
('server_fi_1', 'Финляндия #1', 'Финляндия', 'FI', '🇫🇮', 2, 'https://example.com/config/fi1', 'https://example.com/json/fi1', 'vless', 'websocket', true, 90),
('server_nl_1', 'Нидерланды #1', 'Нидерланды', 'NL', '🇳🇱', 3, 'https://example.com/config/nl1', 'https://example.com/json/nl1', 'vless', 'websocket', true, 80),
('server_us_1', 'США #1', 'США', 'US', '🇺🇸', 4, 'https://example.com/config/us1', 'https://example.com/json/us1', 'vless', 'websocket', true, 70),
('server_sg_1', 'Сингапур #1', 'Сингапур', 'SG', '🇸🇬', 5, 'https://example.com/config/sg1', 'https://example.com/json/sg1', 'vless', 'websocket', true, 60)
ON CONFLICT (id) DO NOTHING;

-- Комментарии к таблицам мультиподписок
COMMENT ON TABLE multi_servers IS 'Серверы для мультиподписок';
COMMENT ON TABLE multi_subscriptions IS 'Мультиподписки пользователей';
COMMENT ON TABLE multi_subscription_servers IS 'Связь мультиподписок с серверами';
COMMENT ON TABLE server_selection_states IS 'Временные состояния выбора серверов';

-- ========================================
-- РЕФЕРАЛЬНАЯ СИСТЕМА И МУЛЬТИПОДПИСКИ ИНТЕГРИРОВАНЫ
-- ========================================
-- Реферальная система и мультиподписки полностью интегрированы в основную схему
-- Все таблицы, функции и индексы созданы выше
