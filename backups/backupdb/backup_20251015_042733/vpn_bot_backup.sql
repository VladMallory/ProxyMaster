--
-- PostgreSQL database dump
--

\restrict o7hYRvFbIkcbmGc0CjZVtXakzywruQGNECedl9bZinVe1VKWVrW9xGfQCp5Kgwx

-- Dumped from database version 16.10 (Ubuntu 16.10-0ubuntu0.24.04.1)
-- Dumped by pg_dump version 16.10 (Ubuntu 16.10-0ubuntu0.24.04.1)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: award_referral_bonus(bigint, character varying, numeric, character varying, bigint, text); Type: FUNCTION; Schema: public; Owner: vpn_bot_user
--

CREATE FUNCTION public.award_referral_bonus(user_id bigint, bonus_type character varying, amount numeric, referral_code character varying DEFAULT NULL::character varying, related_user_id bigint DEFAULT NULL::bigint, description text DEFAULT NULL::text) RETURNS boolean
    LANGUAGE plpgsql
    AS $$
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
$$;


ALTER FUNCTION public.award_referral_bonus(user_id bigint, bonus_type character varying, amount numeric, referral_code character varying, related_user_id bigint, description text) OWNER TO vpn_bot_user;

--
-- Name: cleanup_expired_server_selection_states(); Type: FUNCTION; Schema: public; Owner: vpn_bot_user
--

CREATE FUNCTION public.cleanup_expired_server_selection_states() RETURNS integer
    LANGUAGE plpgsql
    AS $$
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
$$;


ALTER FUNCTION public.cleanup_expired_server_selection_states() OWNER TO vpn_bot_user;

--
-- Name: cleanup_old_ip_connections(); Type: FUNCTION; Schema: public; Owner: vpn_bot_user
--

CREATE FUNCTION public.cleanup_old_ip_connections() RETURNS integer
    LANGUAGE plpgsql
    AS $$
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
$$;


ALTER FUNCTION public.cleanup_old_ip_connections() OWNER TO vpn_bot_user;

--
-- Name: generate_referral_code(bigint); Type: FUNCTION; Schema: public; Owner: vpn_bot_user
--

CREATE FUNCTION public.generate_referral_code(telegram_id bigint) RETURNS character varying
    LANGUAGE plpgsql
    AS $$
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
$$;


ALTER FUNCTION public.generate_referral_code(telegram_id bigint) OWNER TO vpn_bot_user;

--
-- Name: get_user_active_servers(bigint); Type: FUNCTION; Schema: public; Owner: vpn_bot_user
--

CREATE FUNCTION public.get_user_active_servers(user_telegram_id bigint) RETURNS TABLE(server_key text, config jsonb)
    LANGUAGE plpgsql
    AS $$
BEGIN
    RETURN QUERY
    SELECT 
        key::TEXT as server_key,
        value::JSONB as config
    FROM users, 
         jsonb_each(additional_servers)
    WHERE telegram_id = user_telegram_id 
      AND (value->>'has_active_config')::boolean = true
      AND (value->>'expiry_time')::bigint > EXTRACT(EPOCH FROM NOW()) * 1000;
END;
$$;


ALTER FUNCTION public.get_user_active_servers(user_telegram_id bigint) OWNER TO vpn_bot_user;

--
-- Name: FUNCTION get_user_active_servers(user_telegram_id bigint); Type: COMMENT; Schema: public; Owner: vpn_bot_user
--

COMMENT ON FUNCTION public.get_user_active_servers(user_telegram_id bigint) IS 'Возвращает список активных серверов для пользователя';


--
-- Name: get_user_all_servers(bigint); Type: FUNCTION; Schema: public; Owner: vpn_bot_user
--

CREATE FUNCTION public.get_user_all_servers(user_telegram_id bigint) RETURNS TABLE(server_key text, config jsonb)
    LANGUAGE plpgsql
    AS $$
BEGIN
    RETURN QUERY
    SELECT 
        key::TEXT as server_key,
        value::JSONB as config
    FROM users, 
         jsonb_each(additional_servers)
    WHERE telegram_id = user_telegram_id;
END;
$$;


ALTER FUNCTION public.get_user_all_servers(user_telegram_id bigint) OWNER TO vpn_bot_user;

--
-- Name: FUNCTION get_user_all_servers(user_telegram_id bigint); Type: COMMENT; Schema: public; Owner: vpn_bot_user
--

COMMENT ON FUNCTION public.get_user_all_servers(user_telegram_id bigint) IS 'Возвращает список всех серверов (активных и неактивных) для пользователя';


--
-- Name: get_user_multi_subscription(bigint); Type: FUNCTION; Schema: public; Owner: vpn_bot_user
--

CREATE FUNCTION public.get_user_multi_subscription(user_telegram_id bigint) RETURNS TABLE(subscription_id character varying, subscription_url character varying, is_active boolean, created_at timestamp without time zone, expiry_time bigint, servers jsonb)
    LANGUAGE plpgsql
    AS $$
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
$$;


ALTER FUNCTION public.get_user_multi_subscription(user_telegram_id bigint) OWNER TO vpn_bot_user;

--
-- Name: get_users_statistics(); Type: FUNCTION; Schema: public; Owner: vpn_bot_user
--

CREATE FUNCTION public.get_users_statistics() RETURNS TABLE(total_users integer, paying_users integer, trial_available_users integer, trial_used_users integer, inactive_users integer, active_configs integer, total_revenue numeric, new_this_week integer, new_this_month integer, conversion_rate numeric)
    LANGUAGE plpgsql
    AS $$
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
$$;


ALTER FUNCTION public.get_users_statistics() OWNER TO vpn_bot_user;

--
-- Name: process_referral_transition(bigint, bigint, character varying); Type: FUNCTION; Schema: public; Owner: vpn_bot_user
--

CREATE FUNCTION public.process_referral_transition(referrer_id bigint, referred_id bigint, referral_code character varying) RETURNS boolean
    LANGUAGE plpgsql
    AS $$
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
$$;


ALTER FUNCTION public.process_referral_transition(referrer_id bigint, referred_id bigint, referral_code character varying) OWNER TO vpn_bot_user;

--
-- Name: update_updated_at_column(); Type: FUNCTION; Schema: public; Owner: vpn_bot_user
--

CREATE FUNCTION public.update_updated_at_column() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;


ALTER FUNCTION public.update_updated_at_column() OWNER TO vpn_bot_user;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: users; Type: TABLE; Schema: public; Owner: vpn_bot_user
--

CREATE TABLE public.users (
    id integer NOT NULL,
    telegram_id bigint NOT NULL,
    username character varying(255),
    first_name character varying(255),
    last_name character varying(255),
    balance numeric(10,2) DEFAULT 0.00,
    total_paid numeric(10,2) DEFAULT 0.00,
    configs_count integer DEFAULT 0,
    has_active_config boolean DEFAULT false,
    client_id character varying(255),
    sub_id character varying(255),
    email character varying(255),
    config_created_at timestamp without time zone,
    expiry_time bigint,
    has_used_trial boolean DEFAULT false,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now(),
    referral_code character varying(50),
    referred_by bigint,
    referral_earnings numeric(10,2) DEFAULT 0.00,
    referral_count integer DEFAULT 0,
    additional_servers jsonb DEFAULT '{}'::jsonb
);


ALTER TABLE public.users OWNER TO vpn_bot_user;

--
-- Name: TABLE users; Type: COMMENT; Schema: public; Owner: vpn_bot_user
--

COMMENT ON TABLE public.users IS 'Пользователи VPN бота';


--
-- Name: COLUMN users.referral_code; Type: COMMENT; Schema: public; Owner: vpn_bot_user
--

COMMENT ON COLUMN public.users.referral_code IS 'Уникальный реферальный код пользователя';


--
-- Name: COLUMN users.referred_by; Type: COMMENT; Schema: public; Owner: vpn_bot_user
--

COMMENT ON COLUMN public.users.referred_by IS 'Telegram ID пользователя, который пригласил';


--
-- Name: COLUMN users.referral_earnings; Type: COMMENT; Schema: public; Owner: vpn_bot_user
--

COMMENT ON COLUMN public.users.referral_earnings IS 'Общая сумма заработанных реферальных бонусов';


--
-- Name: COLUMN users.referral_count; Type: COMMENT; Schema: public; Owner: vpn_bot_user
--

COMMENT ON COLUMN public.users.referral_count IS 'Количество приглашенных пользователей';


--
-- Name: COLUMN users.additional_servers; Type: COMMENT; Schema: public; Owner: vpn_bot_user
--

COMMENT ON COLUMN public.users.additional_servers IS 'Конфигурации пользователя на дополнительных серверах (формат: {"server_key": {"client_id": "...", "sub_id": "...", ...}})';


--
-- Name: active_users; Type: VIEW; Schema: public; Owner: vpn_bot_user
--

CREATE VIEW public.active_users AS
 SELECT id,
    telegram_id,
    username,
    first_name,
    last_name,
    balance,
    total_paid,
    configs_count,
    has_active_config,
    client_id,
    sub_id,
    email,
    config_created_at,
    expiry_time,
    has_used_trial,
    created_at,
    updated_at
   FROM public.users
  WHERE (has_active_config = true);


ALTER VIEW public.active_users OWNER TO vpn_bot_user;

--
-- Name: ip_connections; Type: TABLE; Schema: public; Owner: vpn_bot_user
--

CREATE TABLE public.ip_connections (
    id integer NOT NULL,
    telegram_id bigint,
    ip_address inet,
    connection_data jsonb,
    "timestamp" timestamp without time zone DEFAULT now()
);


ALTER TABLE public.ip_connections OWNER TO vpn_bot_user;

--
-- Name: TABLE ip_connections; Type: COMMENT; Schema: public; Owner: vpn_bot_user
--

COMMENT ON TABLE public.ip_connections IS 'Временные подключения IP адресов (TTL 1 час)';


--
-- Name: ip_connections_id_seq; Type: SEQUENCE; Schema: public; Owner: vpn_bot_user
--

CREATE SEQUENCE public.ip_connections_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.ip_connections_id_seq OWNER TO vpn_bot_user;

--
-- Name: ip_connections_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: vpn_bot_user
--

ALTER SEQUENCE public.ip_connections_id_seq OWNED BY public.ip_connections.id;


--
-- Name: ip_violations; Type: TABLE; Schema: public; Owner: vpn_bot_user
--

CREATE TABLE public.ip_violations (
    id integer NOT NULL,
    telegram_id bigint,
    ip_address inet,
    is_blocked boolean DEFAULT false,
    violation_count integer DEFAULT 1,
    violation_type character varying(100),
    violation_data jsonb,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now()
);


ALTER TABLE public.ip_violations OWNER TO vpn_bot_user;

--
-- Name: TABLE ip_violations; Type: COMMENT; Schema: public; Owner: vpn_bot_user
--

COMMENT ON TABLE public.ip_violations IS 'Нарушения и блокировки IP адресов';


--
-- Name: ip_violations_id_seq; Type: SEQUENCE; Schema: public; Owner: vpn_bot_user
--

CREATE SEQUENCE public.ip_violations_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.ip_violations_id_seq OWNER TO vpn_bot_user;

--
-- Name: ip_violations_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: vpn_bot_user
--

ALTER SEQUENCE public.ip_violations_id_seq OWNED BY public.ip_violations.id;


--
-- Name: multi_servers; Type: TABLE; Schema: public; Owner: vpn_bot_user
--

CREATE TABLE public.multi_servers (
    id character varying(50) NOT NULL,
    name character varying(255) NOT NULL,
    country character varying(100) NOT NULL,
    country_code character varying(3) NOT NULL,
    flag character varying(10) NOT NULL,
    inbound_id integer NOT NULL,
    config_url character varying(500) NOT NULL,
    json_url character varying(500) NOT NULL,
    protocol character varying(20) DEFAULT 'vless'::character varying NOT NULL,
    transport character varying(20) DEFAULT 'websocket'::character varying NOT NULL,
    enabled boolean DEFAULT true,
    priority integer DEFAULT 0,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now()
);


ALTER TABLE public.multi_servers OWNER TO vpn_bot_user;

--
-- Name: TABLE multi_servers; Type: COMMENT; Schema: public; Owner: vpn_bot_user
--

COMMENT ON TABLE public.multi_servers IS 'Серверы для мультиподписок';


--
-- Name: multi_subscription_servers; Type: TABLE; Schema: public; Owner: vpn_bot_user
--

CREATE TABLE public.multi_subscription_servers (
    id integer NOT NULL,
    subscription_id character varying(50) NOT NULL,
    server_id character varying(50) NOT NULL,
    created_at timestamp without time zone DEFAULT now()
);


ALTER TABLE public.multi_subscription_servers OWNER TO vpn_bot_user;

--
-- Name: TABLE multi_subscription_servers; Type: COMMENT; Schema: public; Owner: vpn_bot_user
--

COMMENT ON TABLE public.multi_subscription_servers IS 'Связь мультиподписок с серверами';


--
-- Name: multi_subscription_servers_id_seq; Type: SEQUENCE; Schema: public; Owner: vpn_bot_user
--

CREATE SEQUENCE public.multi_subscription_servers_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.multi_subscription_servers_id_seq OWNER TO vpn_bot_user;

--
-- Name: multi_subscription_servers_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: vpn_bot_user
--

ALTER SEQUENCE public.multi_subscription_servers_id_seq OWNED BY public.multi_subscription_servers.id;


--
-- Name: multi_subscriptions; Type: TABLE; Schema: public; Owner: vpn_bot_user
--

CREATE TABLE public.multi_subscriptions (
    id character varying(50) NOT NULL,
    user_id bigint NOT NULL,
    subscription_url character varying(500) NOT NULL,
    is_active boolean DEFAULT true,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now(),
    expiry_time bigint
);


ALTER TABLE public.multi_subscriptions OWNER TO vpn_bot_user;

--
-- Name: TABLE multi_subscriptions; Type: COMMENT; Schema: public; Owner: vpn_bot_user
--

COMMENT ON TABLE public.multi_subscriptions IS 'Мультиподписки пользователей';


--
-- Name: paying_users; Type: VIEW; Schema: public; Owner: vpn_bot_user
--

CREATE VIEW public.paying_users AS
 SELECT id,
    telegram_id,
    username,
    first_name,
    last_name,
    balance,
    total_paid,
    configs_count,
    has_active_config,
    client_id,
    sub_id,
    email,
    config_created_at,
    expiry_time,
    has_used_trial,
    created_at,
    updated_at
   FROM public.users
  WHERE (total_paid > (0)::numeric);


ALTER VIEW public.paying_users OWNER TO vpn_bot_user;

--
-- Name: promo_codes; Type: TABLE; Schema: public; Owner: vpn_bot_user
--

CREATE TABLE public.promo_codes (
    id character varying(255) NOT NULL,
    code character varying(255) NOT NULL,
    amount numeric(10,2) NOT NULL,
    created_by bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    used_by bigint,
    used_at timestamp with time zone,
    usage_count integer DEFAULT 0 NOT NULL,
    max_uses integer DEFAULT 1 NOT NULL
);


ALTER TABLE public.promo_codes OWNER TO vpn_bot_user;

--
-- Name: promo_usage; Type: TABLE; Schema: public; Owner: vpn_bot_user
--

CREATE TABLE public.promo_usage (
    id integer NOT NULL,
    promo_id character varying(255) NOT NULL,
    user_id bigint NOT NULL,
    amount numeric(10,2) NOT NULL,
    used_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.promo_usage OWNER TO vpn_bot_user;

--
-- Name: promo_usage_id_seq; Type: SEQUENCE; Schema: public; Owner: vpn_bot_user
--

CREATE SEQUENCE public.promo_usage_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.promo_usage_id_seq OWNER TO vpn_bot_user;

--
-- Name: promo_usage_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: vpn_bot_user
--

ALTER SEQUENCE public.promo_usage_id_seq OWNED BY public.promo_usage.id;


--
-- Name: referral_bonuses; Type: TABLE; Schema: public; Owner: vpn_bot_user
--

CREATE TABLE public.referral_bonuses (
    id integer NOT NULL,
    user_telegram_id bigint NOT NULL,
    bonus_type character varying(20) NOT NULL,
    amount numeric(10,2) NOT NULL,
    referral_code character varying(50),
    related_user_id bigint,
    description text,
    created_at timestamp without time zone DEFAULT now()
);


ALTER TABLE public.referral_bonuses OWNER TO vpn_bot_user;

--
-- Name: TABLE referral_bonuses; Type: COMMENT; Schema: public; Owner: vpn_bot_user
--

COMMENT ON TABLE public.referral_bonuses IS 'История реферальных бонусов';


--
-- Name: referral_bonuses_id_seq; Type: SEQUENCE; Schema: public; Owner: vpn_bot_user
--

CREATE SEQUENCE public.referral_bonuses_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.referral_bonuses_id_seq OWNER TO vpn_bot_user;

--
-- Name: referral_bonuses_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: vpn_bot_user
--

ALTER SEQUENCE public.referral_bonuses_id_seq OWNED BY public.referral_bonuses.id;


--
-- Name: referral_transitions; Type: TABLE; Schema: public; Owner: vpn_bot_user
--

CREATE TABLE public.referral_transitions (
    id integer NOT NULL,
    referrer_telegram_id bigint NOT NULL,
    referred_telegram_id bigint NOT NULL,
    referral_code character varying(50) NOT NULL,
    transition_date timestamp without time zone DEFAULT now(),
    bonus_paid boolean DEFAULT false,
    bonus_amount numeric(10,2) DEFAULT 0.00,
    created_at timestamp without time zone DEFAULT now()
);


ALTER TABLE public.referral_transitions OWNER TO vpn_bot_user;

--
-- Name: TABLE referral_transitions; Type: COMMENT; Schema: public; Owner: vpn_bot_user
--

COMMENT ON TABLE public.referral_transitions IS 'Отслеживание реферальных переходов';


--
-- Name: referral_transitions_id_seq; Type: SEQUENCE; Schema: public; Owner: vpn_bot_user
--

CREATE SEQUENCE public.referral_transitions_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.referral_transitions_id_seq OWNER TO vpn_bot_user;

--
-- Name: referral_transitions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: vpn_bot_user
--

ALTER SEQUENCE public.referral_transitions_id_seq OWNED BY public.referral_transitions.id;


--
-- Name: server_selection_states; Type: TABLE; Schema: public; Owner: vpn_bot_user
--

CREATE TABLE public.server_selection_states (
    id integer NOT NULL,
    user_id bigint NOT NULL,
    selected_servers jsonb DEFAULT '[]'::jsonb NOT NULL,
    max_servers integer DEFAULT 5,
    step character varying(20) DEFAULT 'select'::character varying,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now(),
    expires_at timestamp without time zone DEFAULT (now() + '01:00:00'::interval)
);


ALTER TABLE public.server_selection_states OWNER TO vpn_bot_user;

--
-- Name: TABLE server_selection_states; Type: COMMENT; Schema: public; Owner: vpn_bot_user
--

COMMENT ON TABLE public.server_selection_states IS 'Временные состояния выбора серверов';


--
-- Name: server_selection_states_id_seq; Type: SEQUENCE; Schema: public; Owner: vpn_bot_user
--

CREATE SEQUENCE public.server_selection_states_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.server_selection_states_id_seq OWNER TO vpn_bot_user;

--
-- Name: server_selection_states_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: vpn_bot_user
--

ALTER SEQUENCE public.server_selection_states_id_seq OWNED BY public.server_selection_states.id;


--
-- Name: traffic_configs; Type: TABLE; Schema: public; Owner: vpn_bot_user
--

CREATE TABLE public.traffic_configs (
    id character varying(50) DEFAULT 'default'::character varying NOT NULL,
    enabled boolean DEFAULT true,
    daily_limit_gb integer,
    weekly_limit_gb integer,
    monthly_limit_gb integer,
    limit_gb integer,
    reset_days integer,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now()
);


ALTER TABLE public.traffic_configs OWNER TO vpn_bot_user;

--
-- Name: TABLE traffic_configs; Type: COMMENT; Schema: public; Owner: vpn_bot_user
--

COMMENT ON TABLE public.traffic_configs IS 'Настройки трафика';


--
-- Name: trial_available_users; Type: VIEW; Schema: public; Owner: vpn_bot_user
--

CREATE VIEW public.trial_available_users AS
 SELECT id,
    telegram_id,
    username,
    first_name,
    last_name,
    balance,
    total_paid,
    configs_count,
    has_active_config,
    client_id,
    sub_id,
    email,
    config_created_at,
    expiry_time,
    has_used_trial,
    created_at,
    updated_at
   FROM public.users
  WHERE ((has_used_trial = false) AND (balance <= (0)::numeric));


ALTER VIEW public.trial_available_users OWNER TO vpn_bot_user;

--
-- Name: users_id_seq; Type: SEQUENCE; Schema: public; Owner: vpn_bot_user
--

CREATE SEQUENCE public.users_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.users_id_seq OWNER TO vpn_bot_user;

--
-- Name: users_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: vpn_bot_user
--

ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;


--
-- Name: ip_connections id; Type: DEFAULT; Schema: public; Owner: vpn_bot_user
--

ALTER TABLE ONLY public.ip_connections ALTER COLUMN id SET DEFAULT nextval('public.ip_connections_id_seq'::regclass);


--
-- Name: ip_violations id; Type: DEFAULT; Schema: public; Owner: vpn_bot_user
--

ALTER TABLE ONLY public.ip_violations ALTER COLUMN id SET DEFAULT nextval('public.ip_violations_id_seq'::regclass);


--
-- Name: multi_subscription_servers id; Type: DEFAULT; Schema: public; Owner: vpn_bot_user
--

ALTER TABLE ONLY public.multi_subscription_servers ALTER COLUMN id SET DEFAULT nextval('public.multi_subscription_servers_id_seq'::regclass);


--
-- Name: promo_usage id; Type: DEFAULT; Schema: public; Owner: vpn_bot_user
--

ALTER TABLE ONLY public.promo_usage ALTER COLUMN id SET DEFAULT nextval('public.promo_usage_id_seq'::regclass);


--
-- Name: referral_bonuses id; Type: DEFAULT; Schema: public; Owner: vpn_bot_user
--

ALTER TABLE ONLY public.referral_bonuses ALTER COLUMN id SET DEFAULT nextval('public.referral_bonuses_id_seq'::regclass);


--
-- Name: referral_transitions id; Type: DEFAULT; Schema: public; Owner: vpn_bot_user
--

ALTER TABLE ONLY public.referral_transitions ALTER COLUMN id SET DEFAULT nextval('public.referral_transitions_id_seq'::regclass);


--
-- Name: server_selection_states id; Type: DEFAULT; Schema: public; Owner: vpn_bot_user
--

ALTER TABLE ONLY public.server_selection_states ALTER COLUMN id SET DEFAULT nextval('public.server_selection_states_id_seq'::regclass);


--
-- Name: users id; Type: DEFAULT; Schema: public; Owner: vpn_bot_user
--

ALTER TABLE ONLY public.users ALTER COLUMN id SET DEFAULT nextval('public.users_id_seq'::regclass);


--
-- Data for Name: ip_connections; Type: TABLE DATA; Schema: public; Owner: vpn_bot_user
--

COPY public.ip_connections (id, telegram_id, ip_address, connection_data, "timestamp") FROM stdin;
\.


--
-- Data for Name: ip_violations; Type: TABLE DATA; Schema: public; Owner: vpn_bot_user
--

COPY public.ip_violations (id, telegram_id, ip_address, is_blocked, violation_count, violation_type, violation_data, created_at, updated_at) FROM stdin;
\.


--
-- Data for Name: multi_servers; Type: TABLE DATA; Schema: public; Owner: vpn_bot_user
--

COPY public.multi_servers (id, name, country, country_code, flag, inbound_id, config_url, json_url, protocol, transport, enabled, priority, created_at, updated_at) FROM stdin;
germany	Основной сервер	Germany	DE	🇩🇪	19	https://shadowfade.ru:6187/E4x7DnWKY8QnRdDoc3/dbda1357-2f13-4b60-86e2-fa61d8e8c404	https://example.com/json	vless	xhttp	t	100	2025-10-06 13:48:02.372185	2025-10-06 13:58:42.149719
server_de_1	Сервер #19	Германия	DE	🇩🇪	1	https://shadowfade.ru:6187/E4x7DnWKY8QnRdDoc3/dbda1357-2f13-4b60-86e2-fa61d8e8c404	https://example.com/json/de1	vless	websocket	t	90	2025-10-06 13:47:01.227894	2025-10-06 13:58:48.689756
\.


--
-- Data for Name: multi_subscription_servers; Type: TABLE DATA; Schema: public; Owner: vpn_bot_user
--

COPY public.multi_subscription_servers (id, subscription_id, server_id, created_at) FROM stdin;
1	fe4f84e9-01f2-4662-8eb3-7eb00d4ace5e	germany	2025-10-06 13:48:48.63988
2	fe4f84e9-01f2-4662-8eb3-7eb00d4ace5e	server_de_1	2025-10-06 13:48:48.63988
\.


--
-- Data for Name: multi_subscriptions; Type: TABLE DATA; Schema: public; Owner: vpn_bot_user
--

COPY public.multi_subscriptions (id, user_id, subscription_url, is_active, created_at, updated_at, expiry_time) FROM stdin;
fe4f84e9-01f2-4662-8eb3-7eb00d4ace5e	873925520	https://im.shadowfade.ru:8443/api/multi-subscription?id=fe4f84e9-01f2-4662-8eb3-7eb00d4ace5e	t	2025-10-06 13:48:48.63988	2025-10-06 13:48:48.63988	1762343328640
\.


--
-- Data for Name: promo_codes; Type: TABLE DATA; Schema: public; Owner: vpn_bot_user
--

COPY public.promo_codes (id, code, amount, created_by, created_at, expires_at, is_active, used_by, used_at, usage_count, max_uses) FROM stdin;
promo_1758623292_3a3664cc	3a3664cc	2000.00	873925520	2025-09-23 13:28:12.471743+03	2025-10-07 13:28:12.471743+03	f	873925520	2025-09-23 13:36:01.065767+03	1	1
promo_1757952253_9f823c74	9f823c74	5000.00	873925520	2025-09-15 19:04:13.645742+03	2025-09-29 19:04:13.64574+03	f	873925520	2025-09-15 19:04:20.576971+03	1	1
promo_1758162078_3ed26e13	3ed26e13	5000.00	873925520	2025-09-18 05:21:18.96085+03	2025-10-02 05:21:18.96085+03	f	1105758739	2025-09-18 05:28:48.043722+03	1	1
promo_1758173826_b4a0bf68	b4a0bf68	1000.00	873925520	2025-09-18 08:37:06.738802+03	2025-10-02 08:37:06.738802+03	f	\N	\N	0	1
promo_1758173919_17b90b19	17b90b19	100.00	873925520	2025-09-18 08:38:39.914843+03	2025-10-02 08:38:39.914843+03	f	873925520	2025-09-18 08:38:42.130155+03	1	1
promo_1758179970_42a7eb2f	42a7eb2f	1000.00	873925520	2025-09-18 10:19:30.007354+03	2025-10-02 10:19:30.007354+03	f	\N	\N	0	1
promo_1758179998_e8404ac7	e8404ac7	2000.00	873925520	2025-09-18 10:19:58.294911+03	2025-10-02 10:19:58.294911+03	f	\N	\N	0	1
promo_1758185488_74be4b0d	74be4b0d	2000.00	873925520	2025-09-18 11:51:28.511622+03	2025-10-02 11:51:28.511622+03	f	504886626	2025-09-18 12:11:31.331031+03	1	1
promo_1758248264_ed0a8067	ed0a8067	1000.00	873925520	2025-09-19 05:17:44.849716+03	2025-10-03 05:17:44.849716+03	f	5083088553	2025-09-19 05:18:01.778807+03	1	1
promo_1758284663_7ee176f2	7ee176f2	1000.00	873925520	2025-09-19 15:24:23.933938+03	2025-10-03 15:24:23.933935+03	f	590968416	2025-09-19 15:25:18.877154+03	1	1
promo_1759151563_cbf72521	cbf72521	16.00	873925520	2025-09-29 16:12:43.103462+03	2025-10-13 16:12:43.103461+03	t	873925520	2025-09-29 16:12:46.152763+03	1	1
promo_1759676200_fce23681	fce23681	16.00	873925520	2025-10-05 17:56:40.970025+03	2025-10-19 17:56:40.970025+03	t	873925520	2025-10-05 17:56:45.551272+03	1	1
\.


--
-- Data for Name: promo_usage; Type: TABLE DATA; Schema: public; Owner: vpn_bot_user
--

COPY public.promo_usage (id, promo_id, user_id, amount, used_at) FROM stdin;
1	promo_1757952253_9f823c74	873925520	5000.00	2025-09-15 19:04:20.576971+03
9	promo_1759676200_fce23681	873925520	16.00	2025-10-05 17:56:45.551272+03
2	promo_1758162078_3ed26e13	1105758739	5000.00	2025-09-18 05:28:48.043722+03
7	promo_1758623292_3a3664cc	873925520	2000.00	2025-09-23 13:36:01.065767+03
8	promo_1759151563_cbf72521	873925520	16.00	2025-09-29 16:12:46.152763+03
3	promo_1758173919_17b90b19	873925520	100.00	2025-09-18 08:38:42.130155+03
5	promo_1758248264_ed0a8067	5083088553	1000.00	2025-09-19 05:18:01.778807+03
6	promo_1758284663_7ee176f2	590968416	1000.00	2025-09-19 15:25:18.877154+03
4	promo_1758185488_74be4b0d	504886626	2000.00	2025-09-18 12:11:31.331031+03
\.


--
-- Data for Name: referral_bonuses; Type: TABLE DATA; Schema: public; Owner: vpn_bot_user
--

COPY public.referral_bonuses (id, user_telegram_id, bonus_type, amount, referral_code, related_user_id, description, created_at) FROM stdin;
43	816399085	referrer	450.00	ref_816399085085	948197634	Реферальный бонус за приглашение друга	2025-10-09 06:25:52.28534
44	948197634	referred	90.00	ref_816399085085	816399085	Приветственный бонус за регистрацию по реферальной ссылке	2025-10-09 06:25:52.34049
47	948197634	referrer	450.00	ref_948197634634	2093708356	Реферальный бонус за приглашение друга	2025-10-09 13:14:26.820597
48	2093708356	referred	90.00	ref_948197634634	948197634	Приветственный бонус за регистрацию по реферальной ссылке	2025-10-09 13:14:26.878927
27	1631142357	referrer	1000.00	ref_1631142357357	1906009572	Реферальный бонус за приглашение друга	2025-09-20 20:45:45.433543
28	1906009572	referred	500.00	ref_1631142357357	1631142357	Приветственный бонус за регистрацию по реферальной ссылке	2025-09-20 20:45:45.488842
29	1631142357	referrer	1000.00	ref_1631142357357	7039903298	Реферальный бонус за приглашение друга	2025-09-20 20:47:57.518578
30	7039903298	referred	500.00	ref_1631142357357	1631142357	Приветственный бонус за регистрацию по реферальной ссылке	2025-09-20 20:47:57.581295
12	504886626	referred	500.00	ref_873925520520	873925520	Приветственный бонус за регистрацию по реферальной ссылке	2025-09-18 12:07:28.596704
13	504886626	referrer	1000.00	ref_504886626626	873925520	Реферальный бонус за приглашение друга	2025-09-18 12:13:36.450941
15	504886626	referrer	1000.00	ref_504886626626	6019790478	Реферальный бонус за приглашение друга	2025-09-18 12:18:23.765735
16	6019790478	referred	500.00	ref_504886626626	504886626	Приветственный бонус за регистрацию по реферальной ссылке	2025-09-18 12:18:23.843182
8	1524508927	referred	500.00	ref_5035512654654	5035512654	Приветственный бонус за регистрацию по реферальной ссылке	2025-09-18 08:53:11.576612
10	431539621	referred	500.00	ref_873925520520	873925520	Приветственный бонус за регистрацию по реферальной ссылке	2025-09-18 10:09:44.804838
20	5083088553	referred	500.00	ref_873925520520	873925520	Приветственный бонус за регистрацию по реферальной ссылке	2025-09-19 14:03:45.674406
39	7801545772	referrer	240.00	ref_7801545772772	844750211	Реферальный бонус за приглашение друга	2025-09-30 13:35:26.49722
31	1631142357	referrer	1000.00	1631142357357	5389115327	Реферальный бонус за приглашение друга	2025-09-22 10:06:31.852903
32	5389115327	referred	500.00	1631142357357	1631142357	Приветственный бонус за регистрацию по реферальной ссылке	2025-09-22 10:06:31.854741
40	844750211	referred	240.00	ref_7801545772772	7801545772	Приветственный бонус за регистрацию по реферальной ссылке	2025-09-30 13:35:26.662218
23	1631142357	referrer	1000.00	ref_1631142357357	1834205754	Реферальный бонус за приглашение друга	2025-09-20 15:33:58.128414
24	1834205754	referred	500.00	ref_1631142357357	1631142357	Приветственный бонус за регистрацию по реферальной ссылке	2025-09-20 15:33:58.196208
33	1631142357	referrer	1000.00	ref_1631142357357	455471831	Реферальный бонус за приглашение друга	2025-09-22 16:03:20.108821
22	590968416	referred	500.00	ref_5035512654654	5035512654	Приветственный бонус за регистрацию по реферальной ссылке	2025-09-19 15:24:08.528148
25	1631142357	referrer	1000.00	ref_1631142357357	6913806796	Реферальный бонус за приглашение друга	2025-09-20 20:37:27.721597
26	6913806796	referred	500.00	ref_1631142357357	1631142357	Приветственный бонус за регистрацию по реферальной ссылке	2025-09-20 20:37:27.858768
34	455471831	referred	500.00	ref_1631142357357	1631142357	Приветственный бонус за регистрацию по реферальной ссылке	2025-09-22 16:03:20.192886
35	1631142357	referrer	1000.00	ref_1631142357357	6475411765	Реферальный бонус за приглашение друга	2025-09-22 16:07:17.684503
36	6475411765	referred	500.00	ref_1631142357357	1631142357	Приветственный бонус за регистрацию по реферальной ссылке	2025-09-22 16:07:17.755667
37	1631142357	referrer	240.00	ref_1631142357357	611396753	Реферальный бонус за приглашение друга	2025-09-30 08:17:11.928393
38	611396753	referred	240.00	ref_1631142357357	1631142357	Приветственный бонус за регистрацию по реферальной ссылке	2025-09-30 08:17:11.990802
45	816399085	referrer	450.00	ref_816399085085	468632794	Реферальный бонус за приглашение друга	2025-10-09 10:32:44.903586
46	468632794	referred	90.00	ref_816399085085	816399085	Приветственный бонус за регистрацию по реферальной ссылке	2025-10-09 10:32:44.964146
18	1039240440	referred	500.00	ref_873925520520	873925520	Приветственный бонус за регистрацию по реферальной ссылке	2025-09-18 14:38:23.703021
49	1719171729	referrer	450.00	ref_1719171729729	8365253818	Реферальный бонус за приглашение друга	2025-10-12 13:26:51.377414
50	8365253818	referred	90.00	ref_1719171729729	1719171729	Приветственный бонус за регистрацию по реферальной ссылке	2025-10-12 13:26:51.433889
41	844750211	referrer	240.00	ref_844750211211	509748878	Реферальный бонус за приглашение друга	2025-09-30 13:36:05.883227
42	509748878	referred	240.00	ref_844750211211	844750211	Приветственный бонус за регистрацию по реферальной ссылке	2025-09-30 13:36:05.98745
\.


--
-- Data for Name: referral_transitions; Type: TABLE DATA; Schema: public; Owner: vpn_bot_user
--

COPY public.referral_transitions (id, referrer_telegram_id, referred_telegram_id, referral_code, transition_date, bonus_paid, bonus_amount, created_at) FROM stdin;
23	1631142357	455471831	ref_1631142357357	2025-09-22 16:03:19.983897	f	0.00	2025-09-22 16:03:19.983897
17	1631142357	1834205754	ref_1631142357357	2025-09-20 15:33:57.994548	f	0.00	2025-09-20 15:33:57.994548
24	1631142357	6475411765	ref_1631142357357	2025-09-22 16:07:17.564581	f	0.00	2025-09-22 16:07:17.564581
25	773604014	5143437202	ref_773604014014	2025-09-24 03:15:09.726749	f	0.00	2025-09-24 03:15:09.726749
12	1631142357	6385679953	ref_1631142357357	2025-09-18 18:46:17.364133	f	0.00	2025-09-18 18:46:17.364133
10	504886626	6019790478	ref_504886626626	2025-09-18 12:18:23.708926	f	0.00	2025-09-18 12:18:23.708926
29	7801545772	844750211	ref_7801545772772	2025-09-30 13:35:26.286631	f	0.00	2025-09-30 13:35:26.286631
30	844750211	509748878	ref_844750211211	2025-09-30 13:36:05.719542	f	0.00	2025-09-30 13:36:05.719542
28	1631142357	611396753	ref_1631142357357	2025-09-30 08:17:11.837347	f	0.00	2025-09-30 08:17:11.837347
34	1719171729	8365253818	ref_1719171729729	2025-10-12 13:26:51.260039	f	0.00	2025-10-12 13:26:51.260039
27	1631142357	1430649948	ref_1631142357357	2025-09-30 06:50:49.283467	f	0.00	2025-09-30 06:50:49.283467
18	1631142357	629544996	ref_1631142357357	2025-09-20 17:15:24.898346	f	0.00	2025-09-20 17:15:24.898346
19	1631142357	6913806796	ref_1631142357357	2025-09-20 20:37:27.512336	f	0.00	2025-09-20 20:37:27.512336
20	1631142357	1906009572	ref_1631142357357	2025-09-20 20:45:45.340223	f	0.00	2025-09-20 20:45:45.340223
21	1631142357	7039903298	ref_1631142357357	2025-09-20 20:47:57.405715	f	0.00	2025-09-20 20:47:57.405715
31	816399085	948197634	ref_816399085085	2025-10-09 06:25:52.144506	f	0.00	2025-10-09 06:25:52.144506
26	1631142357	1237517884	ref_1631142357357	2025-09-24 13:01:56.226816	f	0.00	2025-09-24 13:01:56.226816
32	816399085	468632794	ref_816399085085	2025-10-09 10:32:44.794354	f	0.00	2025-10-09 10:32:44.794354
33	948197634	2093708356	ref_948197634634	2025-10-09 13:14:26.709384	f	0.00	2025-10-09 13:14:26.709384
\.


--
-- Data for Name: server_selection_states; Type: TABLE DATA; Schema: public; Owner: vpn_bot_user
--

COPY public.server_selection_states (id, user_id, selected_servers, max_servers, step, created_at, updated_at, expires_at) FROM stdin;
\.


--
-- Data for Name: traffic_configs; Type: TABLE DATA; Schema: public; Owner: vpn_bot_user
--

COPY public.traffic_configs (id, enabled, daily_limit_gb, weekly_limit_gb, monthly_limit_gb, limit_gb, reset_days, created_at, updated_at) FROM stdin;
default	t	0	0	0	0	30	2025-09-16 17:10:29.859714	2025-09-16 17:10:29.859714
\.


--
-- Data for Name: users; Type: TABLE DATA; Schema: public; Owner: vpn_bot_user
--

COPY public.users (id, telegram_id, username, first_name, last_name, balance, total_paid, configs_count, has_active_config, client_id, sub_id, email, config_created_at, expiry_time, has_used_trial, created_at, updated_at, referral_code, referred_by, referral_earnings, referral_count, additional_servers) FROM stdin;
715	123456789	test_user	Test	User	0.00	0.00	0	f	\N	\N	123456789_1	2025-10-06 11:59:46.014726	1762336786014	f	2025-10-06 11:46:59.890477	2025-10-06 14:13:12.084264	123456789789	0	0.00	0	{}
662	5591700754	CyberWolf2022	CyberWolf		0.00	0.00	0	f	\N	\N	\N	\N	\N	f	2025-10-02 07:31:19.791432	2025-10-02 07:34:10.557911	5591700754754	\N	0.00	0	{}
698	7032438837	Clover_alt	️Clover️ | Anolyte VPN		0.00	0.00	0	f	\N	\N	\N	\N	\N	f	2025-10-05 16:28:18.171803	2025-10-05 16:42:24.63217	7032438837837	\N	0.00	0	{}
702	5725968646		Иван Е.		0.00	12.00	1	f	5e6977cd-e297-49ee-a3ce-59c4a574209e	gwhu2xipt0fav943	5725968646	2025-10-05 17:31:38.489825	1760040519904	t	2025-10-05 17:30:43.040218	2025-10-10 08:34:12.466585	5725968646646	0	0.00	0	{}
795	8011677683	isma_ismaaa	•		6.00	12.00	1	t	24488dbf-6dab-480e-a633-e392e58e2c47	rncnzq4z48h25jqf	8011677683	2025-10-13 22:05:22.347824	1760659058726	t	2025-10-13 22:04:51.363249	2025-10-15 02:57:38.842712	\N	0	0.00	0	{}
730	589750092	sunshine_angelia	Angelina		0.00	0.00	0	f	\N	\N	\N	\N	\N	f	2025-10-08 12:33:31.066007	2025-10-08 14:38:34.774302	589750092092	\N	0.00	0	{}
767	7778040290		наталья		97.00	112.00	1	t	c9c6a659-9ae1-4792-b151-d26340f78a92	26xbk185vahte8ks	7778040290	2025-10-10 09:48:12.595286	1763251058726	t	2025-10-10 09:48:09.199554	2025-10-15 02:57:38.990476	\N	0	0.00	0	{}
787	6933062292		527052		1003.00	2012.00	1	t	5c6a0862-3cca-418e-a1c5-e9fc66c15179	5l2g0pe29c4buhrx	6933062292	2025-10-12 10:36:16.367574	1789343858726	t	2025-10-12 10:36:13.755136	2025-10-15 02:57:39.011835	\N	0	0.00	0	{}
585	1050967279	Pashtiga1979	Павел		0.00	0.00	0	f	\N	\N	\N	\N	\N	f	2025-09-26 17:47:44.797516	2025-10-08 09:26:12.87146	1050967279279	\N	0.00	0	{}
768	1028456011	Tessbrink	Tess	Brink	197.00	212.00	1	t	76451274-4efe-47dc-8105-bb66892099d5	u6zp37rkqtaf0aj0	1028456011	2025-10-10 19:31:05.161969	1766102258726	t	2025-10-10 19:27:08.363341	2025-10-15 02:57:39.031233	\N	0	0.00	0	{}
567	867401187	Alx_happy	Александр		0.00	32.00	1	f	97a9a2eb-af30-40f2-bb9c-e8772833274f	cqt2nrwmne5syy1d	867401187	2025-09-25 22:15:25.100541	1759174780853	t	2025-09-25 22:15:11.446363	2025-09-30 06:16:35.99923	867401187187	0	0.00	0	{}
672	763329195	e1tech	👑_𝓔𝓭𝓾𝓪𝓻𝓭_👑		0.00	0.00	0	f	\N	\N	\N	\N	\N	f	2025-10-02 17:43:48.592696	2025-10-03 07:41:13.186992	763329195195	\N	0.00	0	{}
633	5520524111		كميلة		0.00	32.00	1	f	7f3fd3ab-5b9e-465d-b625-f24787dae202	665p93pjz633qedm	5520524111	2025-09-29 22:22:24.291111	1759524079787	t	2025-09-29 22:21:30.81815	2025-10-03 23:51:18.105556	5520524111111	0	0.00	0	{}
731	1817506847	wqxxvuu	.		0.00	0.00	0	f	\N	\N	\N	\N	\N	f	2025-10-08 13:33:19.699691	2025-10-08 14:38:34.911867	1817506847847	\N	0.00	0	{}
383	309230796	Magomed500	vartu	500	0.00	0.00	0	f	\N	\N	\N	\N	\N	f	2025-09-20 15:50:12.891187	2025-09-22 10:11:15.696863	309230796796	\N	0.00	0	{}
657	7850385480		Руслан		183.00	232.00	1	t	b8a272a3-9678-437d-b248-cf1cc6f85b13	eeg7yw3rw60jx0h6	7850385480	2025-10-02 06:00:19.277211	1765756658726	t	2025-10-02 06:00:09.912338	2025-10-15 02:57:39.051114	\N	0	0.00	0	{}
724	871091373		Евгений		188.00	212.00	1	t	b367f863-7359-4698-bfb7-43959b04b467	rfqnryo1t6mg963r	871091373	2025-10-07 20:01:49.528414	1765843058726	t	2025-10-07 20:01:34.571476	2025-10-15 02:57:39.058127	\N	0	0.00	0	{}
607	5143989030	eldarkhalilouv	Eldar	"Kabarxx" Khalilov	0.00	32.00	1	f	26022994-8a34-47c0-83e1-1ba8f9aefd81	ex607ioo6jguhvjz	5143989030	2025-09-28 14:42:34.291836	1759436373314	t	2025-09-28 14:42:32.636128	2025-10-02 23:21:18.17482	5143989030030	0	0.00	0	{}
721	7571942095	Sesshhll	Арсений		188.00	212.00	1	t	1ca75978-3519-4319-80ff-59786040b1c5	iaff24vjwn1g3etq	7571942095	2025-10-07 18:18:37.155954	1765843058726	t	2025-10-07 18:16:15.937564	2025-10-15 02:57:39.065112	\N	0	0.00	0	{}
587	7358253468		Буба		0.00	32.00	1	f	38401c52-d183-40bd-a6c1-c16123d940f5	ycgxoz80u4n3ldvq	7358253468	2025-09-26 20:58:09.240788	1759265106449	t	2025-09-26 20:58:07.339567	2025-09-30 23:51:09.13535	7358253468468	0	0.00	0	{}
761	1651279958		Елена		94.00	112.00	1	t	87db0e63-6ec4-46be-9499-fd1ce6d74b4d	wtbf7y96vjb0i1ci	1651279958	2025-10-09 19:56:01.328105	1763164658726	t	2025-10-09 19:55:58.102819	2025-10-15 02:57:38.951506	\N	0	0.00	0	{}
799	7722021832		Poole	Poole	0.00	0.00	0	f	\N	\N	\N	\N	\N	f	2025-10-14 11:32:31.659695	2025-10-14 11:32:31.659695	\N	\N	0.00	0	{}
736	948197634		Жанарбек	Сагиев	522.00	540.00	2	t	1d00f79d-e458-44c9-b0dc-dd07d8733993	m9uj7tep5lyk051a	948197634	2025-10-09 07:08:39.848752	1775519858726	f	2025-10-09 06:25:52.138797	2025-10-15 02:57:38.846751	\N	0	0.00	0	{}
790	8365253818		mahamoud		1081.00	2090.00	2	t	e26e67c7-80d1-4724-8a44-04ef2d710145	get8nzojlsqohtin	8365253818	2025-10-12 13:49:49.789314	1791590258726	f	2025-10-12 13:26:51.245301	2025-10-15 02:57:38.933419	\N	0	0.00	0	{}
622	816399085		Evgeniy	Nova	894.00	912.00	1	t	6014ac1a-25d3-4706-947a-9ee5a96ca5ae	mwfa1tlluie3cmju	816399085	2025-10-09 06:15:11.75689	1786233458726	t	2025-09-29 08:02:36.268914	2025-10-15 02:57:38.962878	\N	0	0.00	0	{}
746	6523058216	kamxqqq	.		0.00	0.00	0	f	\N	\N	\N	\N	\N	f	2025-10-09 13:03:09.968374	2025-10-09 14:38:35.010551	6523058216216	\N	0.00	0	{}
631	1495462177	madi991992	Мадина	Амиргамзаева	0.00	0.00	0	f	\N	\N	\N	\N	0	f	2025-09-29 18:55:38.895107	2025-09-30 06:16:30.421709	1495462177177	0	0.00	0	{}
793	735565815	cHw_8	Захар		6.00	12.00	1	t	36875498-e900-4f92-82c9-23f5e7e44a7f	ccgrux83h2958fj6	735565815	2025-10-13 15:30:40.363987	1760659058726	t	2025-10-13 15:30:34.342064	2025-10-15 02:57:38.988243	\N	0	0.00	0	{}
773	6511342522	Anigro56	анигро		0.00	12.00	1	t	05be50c7-1c96-4b8a-93eb-a15774cceb61	8dcv71lf3o8tyilb	6511342522	2025-10-11 15:22:36.184377	1760560058710	t	2025-10-11 15:14:42.771681	2025-10-15 00:00:00.697071	\N	0	0.00	0	{}
740	805232031		Вера		194.00	212.00	1	t	6ccb5aa0-5b96-4064-8356-793a3b935a11	ec7hpfergp4gx0x4	805232031	2025-10-09 09:32:22.802702	1766015858726	t	2025-10-09 09:31:42.28089	2025-10-15 02:57:39.055738	\N	0	0.00	0	{}
798	98067573	mikeilin	Mike	Ilin	9.00	12.00	1	t	713b2e3d-173b-43ec-849d-c18f82b83a0b	yb48vw11jf4gqcxf	98067573	2025-10-14 07:07:26.583714	1760745458726	t	2025-10-14 05:17:26.442903	2025-10-15 04:27:33.706671	98067573573	0	0.00	0	{}
690	6282469256	AVSS66	AAA		0.00	0.00	0	f	\N	\N	\N	\N	\N	f	2025-10-04 11:51:20.793339	2025-10-04 15:41:18.867393	6282469256256	\N	0.00	0	{}
416	1645297974	chepalg243	A	M	0.00	0.00	0	f	\N	\N	\N	\N	\N	f	2025-09-21 12:27:34.473901	2025-09-22 10:11:15.713999	1645297974974	\N	0.00	0	{}
485	6985420768	lulilany	꧁⁣༒𓆩𝕯𝖗𝖊𝖆𝖒_𝖌𝖎𝖗𝖑𓆪༒꧂		0.00	0.00	0	f	\N	\N	\N	\N	\N	f	2025-09-23 19:59:02.934284	2025-09-24 06:13:35.122681	6985420768768	\N	0.00	0	{}
115	7108317408	aaandrey23	Андрей		2.00	50.00	2	f	848bbc06-8e44-4299-9441-2cea04c71fec	zniuyb2rsxv6hxzg	7108317408	2025-09-18 10:23:01.327966	1758744330298	t	2025-09-17 08:12:30.372758	2025-09-25 12:10:15.323594	7108317408408	0	0.00	0	{}
515	1003786364		Дмитрий		127.00	240.00	1	t	6a81be12-791e-49d8-a052-b9d8859ee4b5	hsbigevjsnddd6s0	1003786364	2025-09-24 04:48:29.01461	1764115058726	t	2025-09-24 04:47:17.723281	2025-10-15 02:57:38.965454	\N	0	0.00	0	{}
511	1430649948	neohotaa	💤		367.00	480.00	1	t	76c6c0e2-ce9b-4c3c-a4e7-e9b61546cc18	cult2elyjqxw3mqa	1430649948	2025-09-24 02:46:39.278679	1771027058726	t	2025-09-24 02:45:15.361227	2025-10-15 02:57:39.007444	\N	0	0.00	0	{}
556	1024728347	fika4445	fika🐾		127.00	240.00	1	t	fd5f9c5c-ead7-4056-82a4-b15f02f179c5	1ir03j6y7v26d8bi	1024728347	2025-09-24 21:39:10.710595	1764115058726	t	2025-09-24 21:35:51.044589	2025-10-15 02:57:39.036658	\N	0	0.00	0	{}
555	6497669922	cgvbc7	.		0.00	0.00	0	f	\N	\N	\N	\N	\N	f	2025-09-24 19:26:43.881849	2025-09-25 00:36:26.607836	6497669922922	\N	0.00	0	{}
560	951388257	kkkar_k	kar		135.00	240.00	1	t	f36f3951-c937-4b73-9bf6-28eb24c79bec	tkxftx42i2f1eaa3	951388257	2025-09-25 00:27:15.288023	1764374258726	t	2025-09-25 00:16:12.292457	2025-10-15 02:57:39.037967	\N	0	0.00	0	{}
748	2093708356		Vitaly		72.00	90.00	3	t	0b6d3e41-4071-454e-b27a-268bb8390d4a	l20zegnfe3t90sgt	2093708356	2025-10-09 15:08:39.867099	1762559858726	f	2025-10-09 13:11:44.909236	2025-10-15 02:57:39.045788	\N	0	0.00	0	{}
541	1177850558	ABK150	……		127.00	240.00	1	t	1db2a865-52e5-4fc4-9c0a-7553ed97bd97	y2r3ecbih02v972v	1177850558	2025-09-24 12:50:45.726964	1764115058726	t	2025-09-24 12:50:43.155539	2025-10-15 02:57:39.053225	\N	0	0.00	0	{}
569	6933779553	osh1bk	Алексей	К.	0.00	32.00	1	f	6b063250-8695-4d4a-a870-ebf56550a4e2	7wwiq0nr5wb9zynk	6933779553	2025-09-25 22:52:05.912761	1759174780778	t	2025-09-25 22:51:57.842538	2025-09-30 06:16:35.793201	6933779553553	0	0.00	0	{}
489	1166200395	SADVAL05ru	Леки		119.00	240.00	1	t	73d620ec-e8fa-4eaf-9ed3-d471473e90cc	ozqvb6cc8b3xv9g5	1166200395	2025-09-23 20:58:44.614402	1763855858726	t	2025-09-23 20:58:41.580426	2025-10-15 02:57:39.054299	\N	0	0.00	0	{}
599	7805152264		Mmm	Mmm	0.00	32.00	1	f	8ae73e4d-ea61-47e1-bbe8-1882b9463904	qq0qdf3zfpm7ab20	7805152264	2025-09-27 20:32:44.005673	1759349470718	t	2025-09-27 20:32:42.727173	2025-10-01 23:19:32.031408	7805152264264	0	0.00	0	{}
273	1782444605	ismaylloo	Исмаил		79.00	240.00	1	t	0e6a1c45-ee35-4b26-af03-ebe6e07762ef	dbx6c3ki1l2bwki4	1782444605	2025-09-18 18:14:53.541693	1762732658726	t	2025-09-18 18:13:43.902464	2025-10-15 02:57:39.056258	\N	0	0.00	0	{}
603	7556764034	brnvpnsup	Поддержка	Barnaul V	0.00	32.00	1	f	0bc336a5-784f-427b-8b56-fec74e6570fb	idbhf8nqp9zp2wmi	7556764034	2025-09-28 13:16:25.684138	1759436373577	t	2025-09-28 13:16:23.402923	2025-10-02 23:21:18.326317	7556764034034	0	0.00	0	{}
363	608974309		Pioner		0.00	0.00	0	f	\N	\N	\N	\N	\N	f	2025-09-20 07:16:51.599209	2025-09-22 10:11:15.701585	608974309309	\N	0.00	0	{}
546	5016942404	TestAttacks	жехован		0.00	0.00	0	f	\N	\N	\N	\N	\N	f	2025-09-24 16:03:30.661922	2025-09-25 00:36:26.586748	5016942404404	\N	0.00	0	{}
609	1149276168	cugadese	Кирилл		0.00	12.00	1	f	7d8d76e4-a90a-4285-bcaf-5e3bb1781236	1wb2tdspe9l8vcn3	1149276168	2025-10-04 20:42:17.557926	1759955919801	t	2025-09-28 20:16:48.746228	2025-10-09 14:38:39.977609	1149276168168	0	0.00	0	{}
147	1752807128	starli02	Андрей		0.00	0.00	0	t	\N	\N	\N	\N	0	f	2025-09-18 05:09:48.094961	2025-09-29 07:32:36.139149	1752807128128	0	0.00	0	{}
346	1649368038	mtb_mtb3d	no	name	0.00	0.00	0	f	\N	\N	\N	\N	\N	f	2025-09-19 22:03:58.718202	2025-09-22 10:11:15.714775	1649368038038	\N	0.00	0	{}
680	1398581857	Aaasya_17	Асич		0.00	32.00	1	f	067afd27-e10b-4bdb-97e4-c7caa367e37e	qi16lr3qpr3rhblq	1398581857	2025-10-03 08:39:16.464821	1760298367963	t	2025-10-03 08:38:08.037268	2025-10-13 07:56:54.475878	1398581857857	0	0.00	0	{}
581	5981842666	Amballord	Maks	Door	0.00	32.00	1	f	4f918653-788a-4e05-abeb-74dd08fd454f	s3rfjwjnonurpxbj	5981842666	2025-09-26 13:13:05.87642	1759265106685	t	2025-09-26 13:12:55.701171	2025-09-30 23:51:09.335727	5981842666666	0	0.00	0	{}
398	8079913864	D_Otabek02	Otabek		0.00	0.00	1	f	619b3f9a-fb39-47ea-b49b-ccea5e186923	vgfoqwbdjxi8lu8d	8079913864	2025-09-25 00:23:41.511099	1758834000068	f	2025-09-20 19:35:30.247652	2025-09-26 00:00:00.068382	8079913864864	0	0.00	0	{}
598	5084008773		Али		0.00	32.00	1	f	772f9b20-2ee1-4dbe-aaeb-e61027badee7	cnir0vp7gh1b169f	5084008773	2025-09-27 20:16:19.420005	1759349470975	t	2025-09-27 20:16:18.081089	2025-10-01 23:19:32.716837	5084008773773	0	0.00	0	{}
594	6187994886	spermatozold	Алексей	Дмитриевич	0.00	32.00	1	f	f5bed96c-58b6-44e5-803c-9658ccc74c28	688xv5qdj12ndy6d	6187994886	2025-09-27 11:51:17.97143	1759349471364	t	2025-09-27 11:50:58.394377	2025-10-01 23:19:33.414255	6187994886886	0	0.00	0	{}
593	5176049217	uUVyKoBoS	денис	пдк	0.00	32.00	1	f	00c7ea2c-7adc-4d2b-acc8-6d8ee739d8c4	ifrsffcezqxbz5vy	5176049217	2025-09-27 10:57:34.969648	1759349471576	t	2025-09-27 10:56:08.12356	2025-10-01 23:19:33.673132	5176049217217	0	0.00	0	{}
571	1251428581		Михаил		0.00	0.00	0	f	\N	\N	\N	\N	\N	f	2025-09-25 23:41:26.739038	2025-09-26 13:59:41.016808	1251428581581	\N	0.00	0	{}
370	7124147700	MXLZxMiRjF	Сергей		0.00	0.00	0	f	\N	\N	\N	\N	\N	f	2025-09-20 09:16:17.405387	2025-09-22 10:11:15.740771	7124147700700	\N	0.00	0	{}
371	1286095849	by_ellyy	elly		0.00	0.00	0	f	\N	\N	\N	\N	\N	f	2025-09-20 11:03:46.816682	2025-09-22 10:11:15.711005	1286095849849	\N	0.00	0	{}
380	6807910627		M		0.00	0.00	0	f	\N	\N	\N	\N	\N	f	2025-09-20 14:23:58.365763	2025-09-22 10:11:15.737731	6807910627627	\N	0.00	0	{}
565	6832282440	djamalll001	Юлий Цезарь		0.00	32.00	1	f	26b5efdd-0a78-4d44-9224-fe014dd097c6	aqb7c33y5clyfiek	6832282440	2025-09-25 15:59:48.560096	1759174780936	t	2025-09-25 15:59:44.243666	2025-09-30 06:16:36.140075	6832282440440	0	0.00	0	{}
671	7693620333	caster0909	Михаил	Гришман	1.00	32.00	1	f	26710253-dd6b-4508-9de7-bc8225868acf	2tx9zy1lv47ems3l	7693620333	2025-10-02 16:59:29.837204	1760040519904	t	2025-10-02 12:52:17.246786	2025-10-10 08:34:12.483916	7693620333333	0	0.00	0	{}
470	1863078631		Игорь	Крылов	119.00	240.00	1	t	be6c5971-8477-4cb6-892e-2f699f800554	bywcas5362yspvso	1863078631	2025-09-23 11:21:25.487169	1763855858726	t	2025-09-23 11:21:19.014951	2025-10-15 02:57:38.827318	\N	0	0.00	0	{}
300	7622419491	kar0sy	.		87.00	240.00	0	t	650cd15e-7876-49a8-9fbe-189a91867c8b	xenkvnw4os4hqzin	7622419491	\N	1762991858726	t	2025-09-19 09:40:20.516591	2025-10-15 02:57:38.886428	\N	0	0.00	0	{}
381	7338654649		Юсуф	Пирсаидов	95.00	240.00	1	t	76544d50-50b1-488c-ac5a-63972f21c717	xdbdmpmqs60lu2su	7338654649	2025-09-20 16:18:46.61714	1763164658726	t	2025-09-20 16:17:35.899309	2025-10-15 02:57:38.88761	\N	0	0.00	0	{}
471	1876831224	Sb_loft	Element		119.00	240.00	1	t	bca679d4-cf00-48d1-97c3-cb7253b526cb	5rfnv26eshhqhibw	1876831224	2025-09-23 12:28:21.669669	1763855858726	t	2025-09-23 12:28:10.757642	2025-10-15 02:57:38.901485	\N	0	0.00	0	{}
490	8219135807		Artur	Artur	119.00	240.00	1	t	8ef1a9b2-5cc4-4bc5-8e3a-2225381a76cd	5mkmn48xetjlr87c	8219135807	2025-09-23 21:08:53.474674	1763855858726	t	2025-09-23 21:08:51.934514	2025-10-15 02:57:38.919542	\N	0	0.00	0	{}
501	1624243941	paatuii	P		127.00	240.00	1	t	cd2c0f02-4e73-466c-875e-febee97b0bf5	0twahxg0izhpc663	1624243941	2025-09-24 00:47:12.364193	1764115058726	t	2025-09-24 00:46:52.679711	2025-10-15 02:57:38.947165	\N	0	0.00	0	{}
638	2052953508	MrAavtor	Саша👁️‍🗨️		2897.00	2962.00	1	t	180fb0d8-9d4a-4fad-8c63-06064c91ace5	vdyv0ey930cgroge	2052953508	2025-09-30 09:10:33.223633	1843862258726	t	2025-09-30 09:10:30.560503	2025-10-15 02:57:38.997773	\N	0	0.00	0	{}
497	7211109340	Sinlwjjr	hitaki		119.00	240.00	1	t	e7558577-ead7-42e1-90e9-60dd703906ff	2yeig30uv72dq9oj	7211109340	2025-09-23 23:23:40.062987	1763855858726	t	2025-09-23 23:23:37.658853	2025-10-15 02:57:38.998452	\N	0	0.00	0	{}
314	6214279343	tantynverde	_latte☕️_		5347.00	5500.00	2	t	63341a11-5083-49e3-8985-89b5719dc2bd	i581srpfmtnr64pv	6214279343	2025-09-19 14:23:24.222824	1914451058726	f	2025-09-19 13:58:36.502921	2025-10-15 02:57:39.021018	\N	0	0.00	0	{}
564	7584437162		Андрей		0.00	32.00	1	f	fd070a9e-1d18-4c73-ba90-cdebca777934	rqqzrc4oxdt2mtin	7584437162	2025-09-25 13:48:54.302873	1759174781051	t	2025-09-25 13:45:57.401582	2025-09-30 06:16:36.257489	7584437162162	0	0.00	0	{}
488	8262873821		Кардан	Задний	119.00	240.00	1	t	bc0659ca-5199-48b5-bc8f-997dbfa4e79b	k5boglq49y2o40gm	8262873821	2025-09-23 20:50:43.592426	1763855858726	t	2025-09-23 20:50:41.55578	2025-10-15 02:57:39.052697	\N	0	0.00	0	{}
675	1230689301		Лариса Мурадова	Мурадова	1.00	32.00	1	f	020d83cd-3c93-4b63-9ade-7765c8f262e2	4iik0nxkge09l42b	1230689301	2025-10-02 21:35:59.738128	1760040519904	t	2025-10-02 21:35:55.818111	2025-10-10 08:34:12.465132	1230689301301	0	0.00	0	{}
665	1698310483	zimuyu	codeinetears		1.00	32.00	1	f	a75a6e4f-e7c4-42bf-9977-faa9b1759694	yt96s31iv4unakko	1698310483	2025-10-02 08:20:01.757568	1760040519904	t	2025-10-02 08:18:06.907708	2025-10-10 08:34:12.466253	1698310483483	0	0.00	0	{}
596	1719171729	Badour_ali	Badour	Ali	641.00	882.00	2	t	618d0919-a5c4-4bb8-a344-a671964b3443	2szt17bm9n5s16wt	1719171729	2025-10-12 10:49:50.878474	1778889458726	t	2025-09-27 14:33:16.208959	2025-10-15 02:57:39.049979	\N	0	0.00	0	{}
551	5005146078		ДЯДЯ	ФИРИДИН	127.00	240.00	1	t	8142af09-88ad-47eb-bf4f-4ec957f9179e	l5ge7smae6zb24l6	5005146078	2025-09-24 18:41:27.257138	1764115058726	t	2025-09-24 18:41:25.640605	2025-10-15 02:57:38.848955	\N	0	0.00	0	{}
427	1996575801	alllievva	R		103.00	240.00	1	t	2b15f3a3-4ab2-459e-acaa-042ab06623c5	avn7vuypq6ldhkg0	1996575801	2025-09-21 13:51:17.171987	1763423858726	t	2025-09-21 13:51:07.128483	2025-10-15 02:57:38.852456	\N	0	0.00	0	{}
549	5155238301		Вадим		127.00	240.00	1	t	7f679404-0791-4d85-b35a-6dd01c8a4b62	bj56lawlqlapbmuc	5155238301	2025-09-24 17:41:02.427076	1764115058726	t	2025-09-24 17:40:46.61127	2025-10-15 02:57:38.855793	\N	0	0.00	0	{}
301	8483091985	fucksq1	xyz		87.00	240.00	1	t	fe00a07e-e9f8-43a5-97af-c681f361f2da	6qzipi9vv50gpv89	8483091985	2025-09-29 18:45:04.371904	1762991858726	t	2025-09-19 09:47:33.696199	2025-10-15 02:57:38.859361	\N	0	0.00	0	{}
529	672766552	WEplay0002	WEplay/инвест		127.00	240.00	1	t	33b04ef2-3dcc-43b3-af42-63e99db5e1b3	8rlw0ubjyvrbdaid	672766552	2025-09-24 10:26:53.344706	1764115058726	t	2025-09-24 10:26:43.701348	2025-10-15 02:57:38.902795	\N	0	0.00	0	{}
512	5143437202		Пшолты		387.00	500.00	1	t	4412c77d-47eb-4931-b443-2d8172799eff	8wvphx3hsbah1tmz	5143437202	2025-09-24 03:15:10.033089	1771631858726	f	2025-09-24 03:14:51.953366	2025-10-15 02:57:38.9203	\N	0	0.00	0	{}
494	714026609	krbnv111	1		119.00	240.00	1	t	233569f4-1dbe-4ee9-bee5-9c7fb05b2e80	e1xjjleq0wq1wrs3	714026609	2025-09-23 22:02:46.83064	1763855858726	t	2025-09-23 22:02:45.458259	2025-10-15 02:57:38.936481	\N	0	0.00	0	{}
716	5301473673	biravad	Ирина		0.00	12.00	1	f	78b8a588-bf05-4567-8230-c3bdc1d38bbb	ztegvpxdn830tp4z	5301473673	2025-10-06 23:54:38.361663	1760126919803	t	2025-10-06 23:54:32.968676	2025-10-11 05:12:07.807961	5301473673673	0	0.00	0	{}
545	7015972519	comnnee	Adillvnaa		127.00	240.00	1	t	3cdd487c-16e5-4c18-9f1c-f7aa2d18fd17	cqzcxsbjf795kexu	7015972519	2025-09-24 15:19:21.373693	1764115058726	t	2025-09-24 15:19:19.9801	2025-10-15 02:57:38.945438	\N	0	0.00	0	{}
542	8250876261		Islam	Islam	127.00	240.00	1	t	1f831cd0-9f2b-422c-9b99-7ac1351e132c	7gkmlqg5bory0j94	8250876261	2025-09-24 14:14:11.585031	1764115058726	t	2025-09-24 14:14:09.624425	2025-10-15 02:57:38.973911	\N	0	0.00	0	{}
623	611396753	GetoMeron	Арслан		199.00	272.00	1	t	e5745ee7-1963-42c7-b17a-aadf8a56d4ef	3sj9ceql4q4hfan5	611396753	2025-09-29 13:45:52.484299	1766188658726	t	2025-09-29 13:45:34.102712	2025-10-15 02:57:38.974723	\N	0	0.00	0	{}
385	629544996		𝓜𝓪𝓵𝓲𝓴𝓪🫶🏻		355.00	500.00	1	t	01ca2947-6b4d-44d8-b23b-b883eec61312	3laedttpeohudgk7	629544996	2025-09-20 17:15:25.223059	1770681458726	f	2025-09-20 17:15:24.893914	2025-10-15 02:57:38.978439	\N	0	0.00	0	{}
358	773604014	Kotletca_pure	Костя		11355.00	11500.00	1	t	67961e25-040d-481e-ba55-459daf75c616	s80bukjmesdal7lq	773604014	2025-09-20 06:54:30.819833	2087510258726	f	2025-09-20 06:54:17.19655	2025-10-15 02:57:39.020531	\N	0	0.00	0	{}
543	5258791781		Мурад		127.00	240.00	1	t	4b748d7e-deda-4925-b394-ba444b09ed1e	8h49n8zaj8wkcq4k	5258791781	2025-09-24 14:46:54.014026	1764115058726	t	2025-09-24 14:46:34.631292	2025-10-15 02:57:39.033358	\N	0	0.00	0	{}
554	8307143788	n1234509876n	follow for me		127.00	240.00	1	t	1534a35c-de01-47b3-93ab-9e71364ff0be	9vkkg564n4980alg	8307143788	2025-09-24 19:21:35.725265	1764115058726	t	2025-09-24 19:21:30.736107	2025-10-15 02:57:39.048321	\N	0	0.00	0	{}
491	8340095488		Adil	Adil	119.00	240.00	1	t	6845f8dc-e3f7-4929-9d47-107ffcb65634	8ohenrym6z16jsfd	8340095488	2025-09-23 21:34:20.143886	1763855858726	t	2025-09-23 21:34:18.192565	2025-10-15 02:57:39.049439	\N	0	0.00	0	{}
688	6385851218		Нариман	Ismailov	2.00	32.00	1	t	7c24530d-0613-496c-8ed7-b36b7ddec970	gpkdxu9lda6ucdit	6385851218	2025-10-04 01:05:28.513062	1760471814297	t	2025-10-04 01:04:54.330606	2025-10-14 04:27:33.325029	6385851218218	0	0.00	0	{}
639	513700647	vitalyNik1997	Vitaly 🤙🏻		0.00	32.00	1	f	65ce81d5-b06a-477f-af79-91eff8c47653	ircxmgdvdgjvdzv5	513700647	2025-09-30 12:23:37.210879	1759606880081	t	2025-09-30 12:22:44.396461	2025-10-04 22:47:06.288989	513700647647	0	0.00	0	{}
637	728207319	tigra_juli	Юлия	Ханмамедова	0.00	32.00	1	f	0eb55800-e05c-4c55-9f78-74fe434de4c3	1oghrwvoth2nhapn	728207319	2025-09-30 08:26:22.623389	1759606880493	t	2025-09-30 08:26:06.123293	2025-10-04 22:47:06.481896	728207319319	0	0.00	0	{}
276	5944879079		77		79.00	240.00	1	t	71977407-7615-44c0-a578-904cf7a3a2a2	vohftuzggsg75z3r	5944879079	2025-09-18 18:24:32.929083	1762732658726	t	2025-09-18 18:24:31.111192	2025-10-15 02:57:38.84145	\N	0	0.00	0	{}
404	1906009572		.		355.00	500.00	1	t	a6255994-8201-415b-b05b-5420520838ca	sa2s0yf9h5ka0rku	1906009572	2025-09-20 20:45:45.64425	1770681458726	f	2025-09-20 20:45:45.33637	2025-10-15 02:57:38.856522	\N	0	0.00	0	{}
547	6049086437	aus_matris	Индогеймер41 | Новая Эра		127.00	240.00	1	t	26c9bde5-de7f-4435-aa54-535303b074be	0n3bo891gljcno7a	6049086437	2025-09-24 16:03:52.532024	1764115058726	t	2025-09-24 16:03:44.992381	2025-10-15 02:57:38.860616	\N	0	0.00	0	{}
760	787866628		Хозяин		94.00	112.00	1	t	cbdc8de5-63e5-4fb9-a3b5-c266d35e4fac	2krlzl750vh3t2l9	787866628	2025-10-09 19:58:04.850815	1763164658726	t	2025-10-09 19:54:40.282822	2025-10-15 02:57:38.876627	\N	0	0.00	0	{}
265	580689319	AlderMoon	Igor	Khananuev	87.00	240.00	1	t	7c91dc79-3f15-4b96-84a1-67850d9a1067	z6wvjzdmx8kr7y4v	580689319	2025-09-19 23:44:35.31442	1762991858726	t	2025-09-18 17:15:39.142381	2025-10-15 02:57:38.898506	\N	0	0.00	0	{}
344	5948416227	zybeika	ᅠ ᅠᅠ ᅠᅠ ᅠ		87.00	240.00	1	t	be16620b-f8f1-47d3-bc6c-fe7efafe650d	dpq5etyylk68bfl8	5948416227	2025-09-19 20:22:52.683516	1762991858726	t	2025-09-19 20:21:48.987642	2025-10-15 02:57:38.976607	\N	0	0.00	0	{}
552	1612187494		мурик	мурик	127.00	240.00	1	t	6b972d5d-220f-4812-8ca8-54d2ac0e7677	mjku875gusz5elow	1612187494	2025-09-24 18:47:13.981704	1764115058726	t	2025-09-24 18:47:12.190525	2025-10-15 02:57:38.977679	\N	0	0.00	0	{}
525	1237517884	ivvivvav	aml^		627.00	740.00	1	t	753a25a9-c8ec-4386-a073-fc2f13b86e3b	wgcu79ir1icm60oj	1237517884	2025-09-24 08:00:10.20379	1778543858726	t	2025-09-24 07:59:01.53267	2025-10-15 02:57:38.984201	\N	0	0.00	0	{}
495	1296779579	YaTvoyaDepressiYa	Understand me?		119.00	240.00	1	t	e50d2f35-7bb6-48c7-98e7-e609dc9f6ab5	tomos8juwbyvslfq	1296779579	2025-09-23 22:13:34.763042	1763855858726	t	2025-09-23 22:13:23.310727	2025-10-15 02:57:39.002282	\N	0	0.00	0	{}
576	493407464	ab4505	Lynch	Time	0.00	32.00	1	f	08aa9b50-c210-482f-88b7-154d504badf4	d5gk9mjqkj8e5bgk	493407464	2025-09-26 06:36:33.212741	1759265106885	t	2025-09-26 06:36:18.716127	2025-09-30 23:51:09.5243	493407464464	0	0.00	0	{}
548	7863638671	yotomosi	bash.<:𝘺𝘰𝘵𝘰𝘮𝘰𝘴𝘪:>		127.00	240.00	1	t	0622800b-c15b-439c-b8ce-eb4206cacd83	rkqg25hc020zjtuz	7863638671	2025-09-24 22:05:40.475031	1764115058726	t	2025-09-24 16:04:23.38995	2025-10-15 02:57:39.030219	\N	0	0.00	0	{}
286	5083088553	missMOS6581	miss＾ω＾		1587.00	740.00	1	t	48e4ebd1-1b8d-4ec9-9713-d3d6641a9355	q67stb6y26zjz6cz	5083088553	2025-09-19 05:15:40.495237	1806191858726	t	2025-09-19 05:15:38.60048	2025-10-15 02:57:39.053774	\N	0	0.00	0	{}
343	7351472428	crraap	e		87.00	240.00	1	t	5340d7e9-140a-485d-b5c0-91fb22384139	rvna28z7o23l8bhp	7351472428	2025-09-19 20:21:07.231215	1762991858726	t	2025-09-19 20:20:57.389461	2025-10-15 02:57:39.057055	\N	0	0.00	0	{}
540	901023994	s_000_l	️️️		127.00	240.00	1	t	4a123c9d-49b5-48e8-8a0c-ff12edfed569	k8eu7twx94x2wbjj	901023994	2025-09-24 12:25:07.783026	1764115058726	t	2025-09-24 12:19:50.472783	2025-10-15 02:57:39.060967	\N	0	0.00	0	{}
436	6648149382	tvoyanaveki1	🔅		111.00	240.00	0	t	ee621e25-5a75-46d5-a296-67c81513a1ad	ouidcja40o0xy1w7	6648149382	\N	1763683058726	t	2025-09-21 18:28:43.309057	2025-10-15 02:57:39.026595	\N	0	0.00	0	{}
469	6475411765	Alannkki	_T_		371.00	500.00	1	t	99771248-ceb3-4897-ac60-0b22c9f0e1de	8mrh719m6l8hveq0	6475411765	2025-09-22 16:07:17.922081	1771113458726	f	2025-09-22 16:07:17.56075	2025-10-15 02:57:39.048878	\N	0	0.00	0	{}
725	721123829		Artadocks		2441.00	2462.00	1	t	275c65a1-55e1-40c2-8039-f858a3d9cdd3	2k6x871xp30z5p2s	721123829	2025-10-08 09:24:08.521498	1830729458726	t	2025-10-08 09:23:00.224468	2025-10-15 02:57:39.054761	\N	0	0.00	0	{}
732	729997809	Pokrova56	VLADIMIR		191.00	212.00	1	t	2cf8d3fd-6b38-432d-9f5d-4244396397e5	xef8gfmn23rrnja5	729997809	2025-10-08 18:40:23.557064	1765929458726	t	2025-10-08 18:40:11.519205	2025-10-15 02:57:39.062006	\N	0	0.00	0	{}
433	5422340853	I_am_Vitya	Витя	Тоже Витя	5103.00	5240.00	1	t	6e1e8846-d152-4f92-86aa-be43e4cc2611	i20dwfuriolsg5nf	5422340853	2025-09-21 16:07:23.937056	1907452658726	t	2025-09-21 16:07:16.134325	2025-10-15 02:57:39.062495	\N	0	0.00	0	{}
459	1351174013		a		111.00	240.00	1	t	6d027298-945f-45cf-8314-fd190b68626e	qahjngioogigldkp	1351174013	2025-09-22 12:41:28.340181	1763683058726	t	2025-09-22 12:41:25.441286	2025-10-15 02:57:38.766215	\N	0	0.00	0	{}
483	533429919	Artemisto	Я		119.00	240.00	1	t	6e40b3f4-324c-4dce-9e33-01f4ff6f9b21	5b42i9gjec4t5x0p	533429919	2025-09-23 15:11:59.959218	1763855858726	t	2025-09-23 15:11:19.332157	2025-10-15 02:57:38.858549	\N	0	0.00	0	{}
496	844750211		…		415.00	480.00	1	t	71d816da-1747-4b3c-bef8-5ca89f681122	edidoirjgoobwfb8	844750211	2025-09-30 13:35:26.818235	1772409458726	f	2025-09-23 22:41:20.275416	2025-10-15 02:57:38.877705	\N	0	0.00	0	{}
419	5048646132		Сергей		203.00	340.00	1	t	9788da90-dc3e-460b-b887-38c606037a12	8t8w2apxklzwerse	5048646132	2025-09-21 13:41:18.490805	1766275058726	t	2025-09-21 13:40:59.040593	2025-10-15 02:57:38.929134	\N	0	0.00	0	{}
456	5035512654	moment_was	Слава		371.00	500.00	1	t	c4bfcd17-2f82-463b-8a53-eef71679f794	j0zwk4lgtwkcor3c	5035512654	2025-09-22 10:31:08.318866	1771113458726	f	2025-09-22 10:31:07.980423	2025-10-15 02:57:38.938752	\N	0	0.00	0	{}
455	5389115327		Муртуз		611.00	740.00	1	t	35f96d3f-f0da-4020-995c-2e0402960084	nkwnulsp6xfouqdm	5389115327	2025-09-22 09:51:32.25079	1778025458726	t	2025-09-22 09:50:24.064997	2025-10-15 02:57:38.957086	\N	0	0.00	0	{}
476	742241792	KishBeanOnTheStreet	Nbv		2119.00	2240.00	1	t	9babdaa9-3ea7-483b-b0e3-085444123182	b33uquq9cd4zvrvj	742241792	2025-09-23 13:28:56.534451	1821484658726	t	2025-09-23 13:28:49.019196	2025-10-15 02:57:38.9717	\N	0	0.00	0	{}
707	588914719	uriidymchenko	Юрий	Дымченко	82.00	112.00	1	t	8d726e8c-58b3-4a42-9e2a-482ad8bfa904	9infdfywkg5yhao1	588914719	2025-10-05 20:08:17.424902	1762819058726	t	2025-10-05 20:08:10.335167	2025-10-15 02:57:39.01928	\N	0	0.00	0	{}
279	828540887	ddv106	И		79.00	240.00	1	t	32c352e7-7f6a-470f-a6b8-e209cbdbdcca	vw3zofyv1vxze4ja	828540887	2025-09-18 21:29:11.152355	1762732658726	t	2025-09-18 21:26:54.943081	2025-10-15 02:57:39.064629	\N	0	0.00	0	{}
269	231159603	sthansk1yy	Mura		103.00	240.00	1	t	a84c0e6b-4813-48cb-aa14-f401be530818	4gn85bbyntdc6inw	231159603	2025-09-18 19:27:22.411665	1763423858726	t	2025-09-18 19:26:52.038767	2025-10-15 04:27:33.712595	231159603603	0	0.00	0	{}
523	248897919	Galinavylchu	Галина	Вылчу	127.00	240.00	1	t	167f6715-603f-496b-a1c3-2f74a4391007	gofr5ha6we1heinv	248897919	2025-09-24 07:28:17.368832	1764115058726	t	2025-09-24 07:27:57.824751	2025-10-15 04:27:33.716639	248897919919	0	0.00	0	{}
156	267178379	Wellstone	MM		339.00	500.00	3	t	3fd1ba10-3fd4-4c08-8b91-6ef4076dd9c5	oa4vqxlesgslrujn	267178379	2025-09-18 10:23:00.99125	1770249458726	f	2025-09-18 08:46:06.415106	2025-10-15 04:27:33.719543	267178379379	0	0.00	0	{}
683	426222311	Bauto4ka	Татьяна		491.00	532.00	1	t	801eea08-a035-4dde-a3db-51cfdd1453b5	o3gz6qmnpl9hwqis	426222311	2025-10-03 11:52:03.288106	1774569458726	t	2025-10-03 11:51:56.465964	2025-10-15 04:27:33.723987	426222311311	0	0.00	0	{}
468	455471831	XxixixixixixX	XxxxxxxxxX		371.00	500.00	1	t	fc6b4341-24a5-421c-8d5b-d1c5d2ec4526	tjw7e9e580xqh3so	455471831	2025-09-22 16:03:20.354331	1771113458726	f	2025-09-22 16:03:19.979111	2025-10-15 04:27:33.731439	455471831831	0	0.00	0	{}
743	468632794		Sedinkin	Valera	84.00	102.00	2	t	092e7530-3722-466f-be7b-0e797222148a	e93fo95jqbyvrkqu	468632794	2025-10-09 11:03:12.358893	1762905458726	t	2025-10-09 10:32:44.789376	2025-10-15 04:27:33.73583	468632794794	0	0.00	0	{}
376	8139276223		G		95.00	240.00	1	t	b841e480-b16a-48b7-bf9f-ad64a3446988	pdhoc7wqrtz0f75i	8139276223	2025-09-20 23:15:49.838914	1763164658726	t	2025-09-20 13:05:28.885589	2025-10-15 02:57:38.806955	\N	0	0.00	0	{}
341	6302569082	aliski_q72	lil.ali🙏🏻		87.00	240.00	1	t	fdcbb97b-cb8f-417b-9266-68aeddab12de	9q6mdqcu9899uchy	6302569082	2025-09-19 20:16:03.398186	1762991858726	t	2025-09-19 20:16:01.164459	2025-10-15 02:57:38.850397	\N	0	0.00	0	{}
377	1183613976	sallievva	ًًًًًًًٰ		95.00	240.00	1	t	be0ba3bf-f3a8-4c26-9c81-d683c5187291	gxxcc0g7jf5pmvd8	1183613976	2025-09-20 13:59:43.471415	1763164658726	t	2025-09-20 13:59:38.835698	2025-10-15 02:57:38.900487	\N	0	0.00	0	{}
402	6913806796		Ариф		355.00	500.00	1	t	5ceb2a77-ed21-4596-aa06-ce931df8a5b6	pdofzwtqv02uz02v	6913806796	2025-09-20 20:37:28.012817	1770681458726	f	2025-09-20 20:37:27.50884	2025-10-15 02:57:38.903992	\N	0	0.00	0	{}
337	5937302476	khamza_97	حمزة		87.00	240.00	1	t	308c3618-11db-48d7-a139-8ff72ce6d31f	wyagom3ha32qrejg	5937302476	2025-09-19 19:37:27.28034	1762991858726	t	2025-09-19 19:37:25.515066	2025-10-15 02:57:38.93125	\N	0	0.00	0	{}
382	1834205754	miliansis	ML		355.00	500.00	1	t	a2fa2a68-8642-4da8-ac18-b1d2974db6c0	th7ladk3o24ctdv9	1834205754	2025-09-20 15:33:58.352931	1770681458726	f	2025-09-20 15:33:57.986337	2025-10-15 02:57:38.948115	\N	0	0.00	0	{}
340	1862826139		Mmm		87.00	240.00	1	t	12e876e9-1a17-4e27-8324-023182db28eb	yynrg1mf8n9061lf	1862826139	2025-09-19 20:11:28.88668	1762991858726	t	2025-09-19 20:11:26.176051	2025-10-15 02:57:39.050598	\N	0	0.00	0	{}
372	2116824257	yaramiii	ám		95.00	240.00	1	t	501dcf25-0d34-4df5-bd54-103983a389f2	rszpho4sg2v3dor8	2116824257	2025-09-20 11:35:43.18288	1763164658726	t	2025-09-20 11:25:08.83725	2025-10-15 02:57:39.06356	\N	0	0.00	0	{}
384	497299406		т002тт777		95.00	240.00	1	t	3a97b992-1858-460b-a143-d9dadd0bd0b1	7qmxotlp2x02sojb	497299406	2025-09-20 17:11:56.749091	1763164658726	t	2025-09-20 17:11:51.983754	2025-10-15 04:27:33.738649	497299406406	0	0.00	0	{}
336	1763713451	GrigorevaEA_86	Екатерина	Григорьева	119.00	240.00	1	t	b452d4f9-a5d0-4dbb-8c41-1f3e5bd3f695	ejg7ttk05ru7jcj2	1763713451	2025-09-23 13:09:22.894393	1763855858726	t	2025-09-19 18:10:35.654731	2025-10-15 02:57:38.891635	\N	0	0.00	0	{}
277	897966823	IIIIIIIIIIIIIIIlllllll	M		87.00	240.00	0	t	5e4fc2fb-d17a-4e28-abb8-ae6c453b00f5	60e7qqiyzepk55gw	897966823	\N	1762991858726	t	2025-09-18 21:09:20.760519	2025-10-15 02:57:38.915463	\N	0	0.00	0	{}
274	1529679778	weaarrxxx	20:41 🫨		79.00	240.00	1	t	c98adbfb-f411-499c-8833-1d3c7b4fceb6	z0dno9kpuketzta3	1529679778	2025-09-18 20:54:23.15199	1762732658726	t	2025-09-18 20:54:19.6582	2025-10-15 02:57:38.923566	\N	0	0.00	0	{}
335	7801545772	godmytribunal	mcu geek		2327.00	2480.00	1	t	03c90175-310e-4a0f-84db-ad159c730bff	q3a0u2yl7tv8v3os	7801545772	2025-09-19 18:06:13.410341	1827446258726	t	2025-09-19 18:06:03.212831	2025-10-15 02:57:38.926932	\N	0	0.00	0	{}
281	1174043172		М		79.00	240.00	1	t	788f5026-cbfb-4e8c-8775-29a920d1083c	kd093nwcczx9gdh1	1174043172	2025-09-18 21:34:57.041296	1762732658726	t	2025-09-18 21:33:40.759666	2025-10-15 02:57:39.006609	\N	0	0.00	0	{}
330	590968416	Alekseika52	Алексей		1347.00	500.00	2	t	61fa6bfc-4dbb-4b7d-a6d0-4051c578667e	en3rqm157xxmzdq8	590968416	2025-09-19 15:27:01.931185	1799279858726	f	2025-09-19 15:24:08.230878	2025-10-15 02:57:39.03736	\N	0	0.00	0	{}
342	6588405397	narikttt	ZoV		87.00	240.00	1	t	548fdc1a-df15-444b-9409-878f5b2e2072	bad4i3ghgg3j5d6c	6588405397	2025-09-19 18:54:14.127487	1762991858726	t	2025-09-19 18:54:04.174523	2025-10-15 02:57:39.047672	\N	0	0.00	0	{}
275	2089098368		Нуриян Асадулаева	Асадулаева	79.00	240.00	1	t	b67826e4-a7b4-4c66-8924-da5cb550d758	eul17drcgyvgujqt	2089098368	2025-09-18 21:00:40.927579	1762732658726	t	2025-09-18 20:59:22.461324	2025-10-15 02:57:39.057623	\N	0	0.00	0	{}
271	6712475185	sulemanv	Гаджимурад		79.00	240.00	1	t	2a3540ca-e26e-4b62-ba73-2d99fa7c87b0	nk39yvsl71esaqei	6712475185	2025-09-18 20:23:14.598193	1762732658726	t	2025-09-18 20:23:12.757044	2025-10-15 02:57:39.064059	\N	0	0.00	0	{}
264	7372351297	Elen132373	Елена	Исаева	579.00	740.00	1	t	fe32b51a-7b68-4cb2-ae62-71b2c7450112	khqikdn4gpwgjkqb	7372351297	2025-09-18 14:11:38.459972	1777161458726	t	2025-09-18 14:11:18.083143	2025-10-15 02:57:38.845998	\N	0	0.00	0	{}
259	720608118	danyagang	Данил		79.00	240.00	1	t	09072ed8-5649-4ff9-af7f-3c8c0c7924d3	fru5bwmxiawyy4ov	720608118	2025-09-18 14:04:07.65535	1762732658726	t	2025-09-18 13:54:35.369395	2025-10-15 02:57:38.8553	\N	0	0.00	0	{}
188	5464557342	Unsolved_crime	Julie		10339.00	10500.00	2	t	bee0501c-3567-460a-8569-7d6f9b846845	m7elgvvt9htlavzo	5464557342	2025-09-18 12:04:50.88546	2058220658726	f	2025-09-18 12:04:23.547399	2025-10-15 02:57:38.932031	\N	0	0.00	0	{}
160	1524508927	Daughter_of_Apollon	Полина		339.00	500.00	2	t	dd519a3c-6030-4f23-a321-dd56baaf2b3d	em9ffpulnypiuquz	1524508927	2025-09-18 10:23:00.929679	1770249458726	f	2025-09-18 08:53:11.463217	2025-10-15 02:57:38.956067	\N	0	0.00	0	{}
178	7545402103	public_usern	Влад		79.00	240.00	1	t	051c1404-792d-415f-a987-6b386f67bed1	sfb0ykw08ladsro0	7545402103	2025-09-18 10:42:40.112068	1762732658726	t	2025-09-18 10:42:34.999285	2025-10-15 02:57:39.029643	\N	0	0.00	0	{}
270	1631142357	Kasp0501	888		10559.00	10720.00	1	t	5ecd079c-94a9-4320-a771-6057cb439aaa	e9sb9z44pj2v79ni	1631142357	2025-09-18 18:05:12.027076	2064527858726	t	2025-09-18 18:04:04.531355	2025-10-15 02:57:39.031943	\N	0	0.00	0	{}
221	6019790478	AIlizard	Artem		339.00	500.00	2	t	562a4aa4-26ac-49c1-8b57-f7c527cf9761	detnybcruleozb7x	6019790478	2025-09-18 12:18:50.87982	1770249458726	f	2025-09-18 12:18:23.705097	2025-10-15 02:57:39.06152	\N	0	0.00	0	{}
183	431539621	killer_of_soul	Nikolay	Gubarev	339.00	500.00	3	t	a10da1ee-5014-4627-a76d-cd8b0e26e1b4	6pyxr9q43l2gqedy	431539621	2025-09-18 10:23:00.871523	1770249458726	f	2025-09-18 10:09:44.667307	2025-10-15 04:27:33.726696	431539621621	0	0.00	0	{}
179	504886626	Artem_Liz	Артем		4579.00	2740.00	1	t	4a2c8867-ba75-40c0-b71a-36f17db02e7f	kxxdh8onknbgbn3y	504886626	2025-09-18 11:49:35.90722	1892332658726	t	2025-09-18 11:49:27.611567	2025-10-15 04:27:33.740161	504886626626	0	0.00	0	{}
148	1744081769	ZDK_TG	✌️✌️		839.00	240.00	2	t	de28c914-cbd5-4ac7-b261-b03889ce4e49	rhrpfuculqb51doe	1744081769	2025-09-18 10:23:01.044425	1784591858726	t	2025-09-18 08:32:46.843206	2025-10-15 02:57:38.970804	\N	0	0.00	0	{}
146	1105758739	KR7STAL	Kr7stal ?		5079.00	240.00	3	t	a1d2c134-a204-4007-8850-dd1b9d4336c7	nkovfmaxr1ckg793	1105758739	2025-09-18 10:23:01.207073	1906761458726	t	2025-09-18 05:08:57.420913	2025-10-15 02:57:38.989101	\N	0	0.00	0	{}
145	940270721	gospodiprostl	ᅠАлександра)🪱		79.00	240.00	2	t	b9e09b41-c164-473e-b626-d51a883f2af7	r2qwz232nzf8fxoj	940270721	2025-09-18 10:23:01.152006	1762732658726	t	2025-09-18 07:45:56.809907	2025-10-15 02:57:39.014893	\N	0	0.00	0	{}
133	1039240440	Kovalenkoelena14021981	Елена	Коваленко	839.00	1000.00	3	t	c4c77957-f91b-495e-9764-940b45f5376c	jg139epflbwh6ob1	1039240440	2025-09-18 10:23:01.262274	1784591858726	f	2025-09-17 12:00:27.362795	2025-10-15 02:57:39.052202	\N	0	0.00	0	{}
143	7517377017	Jekaxgod13	Almaaazik		79.00	240.00	2	t	a6db89ee-1334-4ad4-b8b2-a8823beb2bf2	a7vftc85bpd25ucx	7517377017	2025-09-18 10:23:01.097797	1762732658726	t	2025-09-18 08:31:52.719864	2025-10-15 02:57:39.055275	\N	0	0.00	0	{}
624	6027269405		Д	А	0.00	32.00	1	f	b8c2af80-4770-494c-afe5-e614be0e70af	zy4xnfzbsq6cij41	6027269405	2025-09-29 13:52:21.9977	1759524080013	t	2025-09-29 13:51:42.794597	2025-10-03 23:51:18.47683	6027269405405	0	0.00	0	{}
635	873925520	BloknotaNet	Vlad		48.00	115.00	4	t	673c2b52-8147-4ed0-93bd-01c8feb8c12c	nxhrp7wu1zyz5v36	873925520	2025-10-06 13:09:58.428467	1761868658726	t	2025-09-30 06:20:25.178557	2025-10-15 02:57:38.786052	\N	0	0.00	0	{}
559	7777442804	trash920	×beast×		135.00	240.00	1	t	4b3e68a9-c11c-4840-8bf9-3fffa673428c	ul562en072h3u2uq	7777442804	2025-09-25 10:34:59.885727	1764374258726	t	2025-09-25 10:34:57.171219	2025-10-15 02:57:38.865096	\N	0	0.00	0	{}
530	1200227132	yulichita_vi	yulichita		127.00	240.00	1	t	159dc3ab-675f-4460-a4ba-c7d313efe908	n7haej3moquro5do	1200227132	2025-09-24 10:39:12.728738	1764115058726	t	2025-09-24 10:39:02.784643	2025-10-15 02:57:38.980624	\N	0	0.00	0	{}
652	8418305024		Родион		2.00	32.00	1	f	48ac774a-32e3-4fff-9fec-a04d59e87fa5	9iimr6qzsf3kmxhr	8418305024	2025-10-01 06:13:29.429935	1759782032704	t	2025-10-01 06:13:27.192822	2025-10-07 13:30:31.453407	8418305024024	0	0.00	0	{}
553	2002968813	amii_na17	Ами💎		127.00	240.00	1	t	af82daac-32c5-451f-ad9c-4f670bf15fd5	dtgf8i2oviy5cbuc	2002968813	2025-09-24 18:47:56.793708	1764115058726	t	2025-09-24 18:47:55.549414	2025-10-15 02:57:38.992055	\N	0	0.00	0	{}
575	5734517523	vdavydof94	Владимир	Давыдов	135.00	232.00	2	t	89a6e966-9cb1-4331-b2c4-846346624c88	smie12amrk3yvq40	5734517523	2025-09-30 10:41:09.858862	1764374258726	t	2025-09-26 04:33:30.125679	2025-10-15 02:57:39.018347	\N	0	0.00	0	{}
558	1574158355		Ваня		627.00	740.00	1	t	d311ae6c-dd2d-48bc-99ab-98f6097c0137	wfldvg6p23a3mrze	1574158355	2025-09-24 23:32:41.773511	1778543858726	t	2025-09-24 23:32:20.721371	2025-10-15 02:57:39.042289	\N	0	0.00	0	{}
646	509748878	Mirmilion	Дракоша 🐉		175.00	240.00	1	t	519f13c0-b950-446c-b460-2525a367d2bd	pcadbcy8ugsf526e	509748878	2025-09-30 13:36:06.142131	1765497458726	f	2025-09-30 13:36:05.714493	2025-10-15 02:57:39.051644	\N	0	0.00	0	{}
508	34354048	andr8ka	Andre		127.00	240.00	1	t	2f1edaf9-5c47-40fb-a840-da41755ca939	hjd40kipw70jj9w2	34354048	2025-09-24 02:03:55.602291	1764115058726	t	2025-09-24 02:03:47.736682	2025-10-15 04:27:33.695308	34354048048	0	0.00	0	{}
406	7039903298	netshtoli	• • •		355.00	500.00	1	t	2566013b-13af-40e4-9f83-5c952cbfbedf	ed8wpte2opxaaqx0	7039903298	2025-09-20 20:47:57.734095	1770681458726	f	2025-09-20 20:47:57.402036	2025-10-15 02:57:38.918808	\N	0	0.00	0	{}
272	6385679953		Adam		339.00	500.00	2	t	8e869990-9b42-4707-b86e-b06d4fc8b411	fl66cyjyejo3muxr	6385679953	2025-09-18 18:47:28.125603	1770249458726	f	2025-09-18 18:46:17.360451	2025-10-15 02:57:38.93421	\N	0	0.00	0	{}
339	6342881708	Eliza_Reina	Eliza		587.00	740.00	1	t	0a967847-3729-449b-8536-e64afb46608b	e94298twcpx982cl	6342881708	2025-09-19 19:09:31.465345	1777334258726	t	2025-09-19 19:07:38.365252	2025-10-15 02:57:39.021662	\N	0	0.00	0	{}
\.


--
-- Name: ip_connections_id_seq; Type: SEQUENCE SET; Schema: public; Owner: vpn_bot_user
--

SELECT pg_catalog.setval('public.ip_connections_id_seq', 1, false);


--
-- Name: ip_violations_id_seq; Type: SEQUENCE SET; Schema: public; Owner: vpn_bot_user
--

SELECT pg_catalog.setval('public.ip_violations_id_seq', 1, false);


--
-- Name: multi_subscription_servers_id_seq; Type: SEQUENCE SET; Schema: public; Owner: vpn_bot_user
--

SELECT pg_catalog.setval('public.multi_subscription_servers_id_seq', 5, true);


--
-- Name: promo_usage_id_seq; Type: SEQUENCE SET; Schema: public; Owner: vpn_bot_user
--

SELECT pg_catalog.setval('public.promo_usage_id_seq', 9, true);


--
-- Name: referral_bonuses_id_seq; Type: SEQUENCE SET; Schema: public; Owner: vpn_bot_user
--

SELECT pg_catalog.setval('public.referral_bonuses_id_seq', 48, true);


--
-- Name: referral_transitions_id_seq; Type: SEQUENCE SET; Schema: public; Owner: vpn_bot_user
--

SELECT pg_catalog.setval('public.referral_transitions_id_seq', 33, true);


--
-- Name: server_selection_states_id_seq; Type: SEQUENCE SET; Schema: public; Owner: vpn_bot_user
--

SELECT pg_catalog.setval('public.server_selection_states_id_seq', 6, true);


--
-- Name: users_id_seq; Type: SEQUENCE SET; Schema: public; Owner: vpn_bot_user
--

SELECT pg_catalog.setval('public.users_id_seq', 802, true);


--
-- Name: ip_connections ip_connections_pkey; Type: CONSTRAINT; Schema: public; Owner: vpn_bot_user
--

ALTER TABLE ONLY public.ip_connections
    ADD CONSTRAINT ip_connections_pkey PRIMARY KEY (id);


--
-- Name: ip_violations ip_violations_pkey; Type: CONSTRAINT; Schema: public; Owner: vpn_bot_user
--

ALTER TABLE ONLY public.ip_violations
    ADD CONSTRAINT ip_violations_pkey PRIMARY KEY (id);


--
-- Name: multi_servers multi_servers_pkey; Type: CONSTRAINT; Schema: public; Owner: vpn_bot_user
--

ALTER TABLE ONLY public.multi_servers
    ADD CONSTRAINT multi_servers_pkey PRIMARY KEY (id);


--
-- Name: multi_subscription_servers multi_subscription_servers_pkey; Type: CONSTRAINT; Schema: public; Owner: vpn_bot_user
--

ALTER TABLE ONLY public.multi_subscription_servers
    ADD CONSTRAINT multi_subscription_servers_pkey PRIMARY KEY (id);


--
-- Name: multi_subscription_servers multi_subscription_servers_subscription_id_server_id_key; Type: CONSTRAINT; Schema: public; Owner: vpn_bot_user
--

ALTER TABLE ONLY public.multi_subscription_servers
    ADD CONSTRAINT multi_subscription_servers_subscription_id_server_id_key UNIQUE (subscription_id, server_id);


--
-- Name: multi_subscriptions multi_subscriptions_pkey; Type: CONSTRAINT; Schema: public; Owner: vpn_bot_user
--

ALTER TABLE ONLY public.multi_subscriptions
    ADD CONSTRAINT multi_subscriptions_pkey PRIMARY KEY (id);


--
-- Name: promo_codes promo_codes_code_key; Type: CONSTRAINT; Schema: public; Owner: vpn_bot_user
--

ALTER TABLE ONLY public.promo_codes
    ADD CONSTRAINT promo_codes_code_key UNIQUE (code);


--
-- Name: promo_codes promo_codes_pkey; Type: CONSTRAINT; Schema: public; Owner: vpn_bot_user
--

ALTER TABLE ONLY public.promo_codes
    ADD CONSTRAINT promo_codes_pkey PRIMARY KEY (id);


--
-- Name: promo_usage promo_usage_pkey; Type: CONSTRAINT; Schema: public; Owner: vpn_bot_user
--

ALTER TABLE ONLY public.promo_usage
    ADD CONSTRAINT promo_usage_pkey PRIMARY KEY (id);


--
-- Name: referral_bonuses referral_bonuses_pkey; Type: CONSTRAINT; Schema: public; Owner: vpn_bot_user
--

ALTER TABLE ONLY public.referral_bonuses
    ADD CONSTRAINT referral_bonuses_pkey PRIMARY KEY (id);


--
-- Name: referral_transitions referral_transitions_pkey; Type: CONSTRAINT; Schema: public; Owner: vpn_bot_user
--

ALTER TABLE ONLY public.referral_transitions
    ADD CONSTRAINT referral_transitions_pkey PRIMARY KEY (id);


--
-- Name: server_selection_states server_selection_states_pkey; Type: CONSTRAINT; Schema: public; Owner: vpn_bot_user
--

ALTER TABLE ONLY public.server_selection_states
    ADD CONSTRAINT server_selection_states_pkey PRIMARY KEY (id);


--
-- Name: traffic_configs traffic_configs_pkey; Type: CONSTRAINT; Schema: public; Owner: vpn_bot_user
--

ALTER TABLE ONLY public.traffic_configs
    ADD CONSTRAINT traffic_configs_pkey PRIMARY KEY (id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: vpn_bot_user
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: users users_referral_code_key; Type: CONSTRAINT; Schema: public; Owner: vpn_bot_user
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_referral_code_key UNIQUE (referral_code);


--
-- Name: users users_telegram_id_key; Type: CONSTRAINT; Schema: public; Owner: vpn_bot_user
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_telegram_id_key UNIQUE (telegram_id);


--
-- Name: idx_ip_connections_ip; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_ip_connections_ip ON public.ip_connections USING btree (ip_address);


--
-- Name: idx_ip_connections_telegram_timestamp; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_ip_connections_telegram_timestamp ON public.ip_connections USING btree (telegram_id, "timestamp" DESC);


--
-- Name: idx_ip_connections_timestamp; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_ip_connections_timestamp ON public.ip_connections USING btree ("timestamp");


--
-- Name: idx_ip_violations_created_at; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_ip_violations_created_at ON public.ip_violations USING btree (created_at);


--
-- Name: idx_ip_violations_ip; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_ip_violations_ip ON public.ip_violations USING btree (ip_address);


--
-- Name: idx_ip_violations_telegram_blocked; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_ip_violations_telegram_blocked ON public.ip_violations USING btree (telegram_id, is_blocked);


--
-- Name: idx_multi_servers_country; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_multi_servers_country ON public.multi_servers USING btree (country);


--
-- Name: idx_multi_servers_enabled; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_multi_servers_enabled ON public.multi_servers USING btree (enabled);


--
-- Name: idx_multi_servers_priority; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_multi_servers_priority ON public.multi_servers USING btree (priority);


--
-- Name: idx_multi_subscription_servers_server; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_multi_subscription_servers_server ON public.multi_subscription_servers USING btree (server_id);


--
-- Name: idx_multi_subscription_servers_subscription; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_multi_subscription_servers_subscription ON public.multi_subscription_servers USING btree (subscription_id);


--
-- Name: idx_multi_subscriptions_active; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_multi_subscriptions_active ON public.multi_subscriptions USING btree (is_active);


--
-- Name: idx_multi_subscriptions_created; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_multi_subscriptions_created ON public.multi_subscriptions USING btree (created_at);


--
-- Name: idx_multi_subscriptions_user; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_multi_subscriptions_user ON public.multi_subscriptions USING btree (user_id);


--
-- Name: idx_promo_codes_code; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_promo_codes_code ON public.promo_codes USING btree (code);


--
-- Name: idx_promo_codes_created_by; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_promo_codes_created_by ON public.promo_codes USING btree (created_by);


--
-- Name: idx_promo_codes_expires_at; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_promo_codes_expires_at ON public.promo_codes USING btree (expires_at);


--
-- Name: idx_promo_usage_promo_id; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_promo_usage_promo_id ON public.promo_usage USING btree (promo_id);


--
-- Name: idx_promo_usage_used_at; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_promo_usage_used_at ON public.promo_usage USING btree (used_at);


--
-- Name: idx_promo_usage_user_id; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_promo_usage_user_id ON public.promo_usage USING btree (user_id);


--
-- Name: idx_referral_bonuses_created_at; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_referral_bonuses_created_at ON public.referral_bonuses USING btree (created_at);


--
-- Name: idx_referral_bonuses_type; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_referral_bonuses_type ON public.referral_bonuses USING btree (bonus_type);


--
-- Name: idx_referral_bonuses_user; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_referral_bonuses_user ON public.referral_bonuses USING btree (user_telegram_id);


--
-- Name: idx_referral_transitions_code; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_referral_transitions_code ON public.referral_transitions USING btree (referral_code);


--
-- Name: idx_referral_transitions_referred; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_referral_transitions_referred ON public.referral_transitions USING btree (referred_telegram_id);


--
-- Name: idx_referral_transitions_referrer; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_referral_transitions_referrer ON public.referral_transitions USING btree (referrer_telegram_id);


--
-- Name: idx_server_selection_states_expires; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_server_selection_states_expires ON public.server_selection_states USING btree (expires_at);


--
-- Name: idx_server_selection_states_user; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_server_selection_states_user ON public.server_selection_states USING btree (user_id);


--
-- Name: idx_users_additional_servers; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_users_additional_servers ON public.users USING gin (additional_servers);


--
-- Name: idx_users_balance; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_users_balance ON public.users USING btree (balance);


--
-- Name: idx_users_created_at; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_users_created_at ON public.users USING btree (created_at);


--
-- Name: idx_users_has_active_config; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_users_has_active_config ON public.users USING btree (has_active_config);


--
-- Name: idx_users_has_used_trial; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_users_has_used_trial ON public.users USING btree (has_used_trial);


--
-- Name: idx_users_referral_code; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_users_referral_code ON public.users USING btree (referral_code);


--
-- Name: idx_users_referred_by; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_users_referred_by ON public.users USING btree (referred_by);


--
-- Name: idx_users_telegram_id; Type: INDEX; Schema: public; Owner: vpn_bot_user
--

CREATE INDEX idx_users_telegram_id ON public.users USING btree (telegram_id);


--
-- Name: ip_violations update_ip_violations_updated_at; Type: TRIGGER; Schema: public; Owner: vpn_bot_user
--

CREATE TRIGGER update_ip_violations_updated_at BEFORE UPDATE ON public.ip_violations FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: multi_servers update_multi_servers_updated_at; Type: TRIGGER; Schema: public; Owner: vpn_bot_user
--

CREATE TRIGGER update_multi_servers_updated_at BEFORE UPDATE ON public.multi_servers FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: multi_subscriptions update_multi_subscriptions_updated_at; Type: TRIGGER; Schema: public; Owner: vpn_bot_user
--

CREATE TRIGGER update_multi_subscriptions_updated_at BEFORE UPDATE ON public.multi_subscriptions FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: server_selection_states update_server_selection_states_updated_at; Type: TRIGGER; Schema: public; Owner: vpn_bot_user
--

CREATE TRIGGER update_server_selection_states_updated_at BEFORE UPDATE ON public.server_selection_states FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: traffic_configs update_traffic_configs_updated_at; Type: TRIGGER; Schema: public; Owner: vpn_bot_user
--

CREATE TRIGGER update_traffic_configs_updated_at BEFORE UPDATE ON public.traffic_configs FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: users update_users_updated_at; Type: TRIGGER; Schema: public; Owner: vpn_bot_user
--

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON public.users FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: ip_connections ip_connections_telegram_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: vpn_bot_user
--

ALTER TABLE ONLY public.ip_connections
    ADD CONSTRAINT ip_connections_telegram_id_fkey FOREIGN KEY (telegram_id) REFERENCES public.users(telegram_id) ON DELETE CASCADE;


--
-- Name: ip_violations ip_violations_telegram_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: vpn_bot_user
--

ALTER TABLE ONLY public.ip_violations
    ADD CONSTRAINT ip_violations_telegram_id_fkey FOREIGN KEY (telegram_id) REFERENCES public.users(telegram_id) ON DELETE CASCADE;


--
-- Name: multi_subscription_servers multi_subscription_servers_server_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: vpn_bot_user
--

ALTER TABLE ONLY public.multi_subscription_servers
    ADD CONSTRAINT multi_subscription_servers_server_id_fkey FOREIGN KEY (server_id) REFERENCES public.multi_servers(id) ON DELETE CASCADE;


--
-- Name: multi_subscription_servers multi_subscription_servers_subscription_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: vpn_bot_user
--

ALTER TABLE ONLY public.multi_subscription_servers
    ADD CONSTRAINT multi_subscription_servers_subscription_id_fkey FOREIGN KEY (subscription_id) REFERENCES public.multi_subscriptions(id) ON DELETE CASCADE;


--
-- Name: multi_subscriptions multi_subscriptions_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: vpn_bot_user
--

ALTER TABLE ONLY public.multi_subscriptions
    ADD CONSTRAINT multi_subscriptions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(telegram_id) ON DELETE CASCADE;


--
-- Name: promo_usage promo_usage_promo_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: vpn_bot_user
--

ALTER TABLE ONLY public.promo_usage
    ADD CONSTRAINT promo_usage_promo_id_fkey FOREIGN KEY (promo_id) REFERENCES public.promo_codes(id) ON DELETE CASCADE;


--
-- Name: referral_bonuses referral_bonuses_user_telegram_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: vpn_bot_user
--

ALTER TABLE ONLY public.referral_bonuses
    ADD CONSTRAINT referral_bonuses_user_telegram_id_fkey FOREIGN KEY (user_telegram_id) REFERENCES public.users(telegram_id) ON DELETE CASCADE;


--
-- Name: referral_transitions referral_transitions_referred_telegram_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: vpn_bot_user
--

ALTER TABLE ONLY public.referral_transitions
    ADD CONSTRAINT referral_transitions_referred_telegram_id_fkey FOREIGN KEY (referred_telegram_id) REFERENCES public.users(telegram_id) ON DELETE CASCADE;


--
-- Name: referral_transitions referral_transitions_referrer_telegram_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: vpn_bot_user
--

ALTER TABLE ONLY public.referral_transitions
    ADD CONSTRAINT referral_transitions_referrer_telegram_id_fkey FOREIGN KEY (referrer_telegram_id) REFERENCES public.users(telegram_id) ON DELETE CASCADE;


--
-- Name: server_selection_states server_selection_states_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: vpn_bot_user
--

ALTER TABLE ONLY public.server_selection_states
    ADD CONSTRAINT server_selection_states_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(telegram_id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

\unrestrict o7hYRvFbIkcbmGc0CjZVtXakzywruQGNECedl9bZinVe1VKWVrW9xGfQCp5Kgwx

