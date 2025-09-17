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
    referral_count INTEGER DEFAULT 0
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

-- ========================================
-- РЕФЕРАЛЬНАЯ СИСТЕМА ИНТЕГРИРОВАНА
-- ========================================
-- Реферальная система полностью интегрирована в основную схему
-- Все таблицы, функции и индексы созданы выше
