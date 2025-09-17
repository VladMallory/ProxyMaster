-- Миграция для добавления реферальной системы
-- Добавляем поля в таблицу users для реферальной системы

-- Добавляем поля реферальной системы в таблицу users
ALTER TABLE users ADD COLUMN IF NOT EXISTS referral_code VARCHAR(50) UNIQUE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS referred_by BIGINT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS referral_earnings DECIMAL(10,2) DEFAULT 0.00;
ALTER TABLE users ADD COLUMN IF NOT EXISTS referral_count INTEGER DEFAULT 0;

-- Создаем таблицу для отслеживания реферальных переходов
CREATE TABLE IF NOT EXISTS referral_transitions (
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

-- Создаем таблицу для истории реферальных бонусов
CREATE TABLE IF NOT EXISTS referral_bonuses (
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

-- Создаем индексы для производительности
CREATE INDEX IF NOT EXISTS idx_users_referral_code ON users(referral_code);
CREATE INDEX IF NOT EXISTS idx_users_referred_by ON users(referred_by);
CREATE INDEX IF NOT EXISTS idx_referral_transitions_referrer ON referral_transitions(referrer_telegram_id);
CREATE INDEX IF NOT EXISTS idx_referral_transitions_referred ON referral_transitions(referred_telegram_id);
CREATE INDEX IF NOT EXISTS idx_referral_transitions_code ON referral_transitions(referral_code);
CREATE INDEX IF NOT EXISTS idx_referral_bonuses_user ON referral_bonuses(user_telegram_id);
CREATE INDEX IF NOT EXISTS idx_referral_bonuses_type ON referral_bonuses(bonus_type);
CREATE INDEX IF NOT EXISTS idx_referral_bonuses_created_at ON referral_bonuses(created_at);

-- Добавляем комментарии к таблицам
COMMENT ON COLUMN users.referral_code IS 'Уникальный реферальный код пользователя';
COMMENT ON COLUMN users.referred_by IS 'Telegram ID пользователя, который пригласил';
COMMENT ON COLUMN users.referral_earnings IS 'Общая сумма заработанных реферальных бонусов';
COMMENT ON COLUMN users.referral_count IS 'Количество приглашенных пользователей';

COMMENT ON TABLE referral_transitions IS 'Отслеживание реферальных переходов';
COMMENT ON TABLE referral_bonuses IS 'История реферальных бонусов';

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
