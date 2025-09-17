--
-- PostgreSQL database dump
--

\restrict cx8ajGJ1S8IEfazqwRQIgedbCBegO3ihRnYKxULsVEYYuBnVnKW8gUyJiovA1YI

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
-- Name: cleanup_old_ip_connections(); Type: FUNCTION; Schema: public; Owner: postgres
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


ALTER FUNCTION public.cleanup_old_ip_connections() OWNER TO postgres;

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
-- Name: get_users_statistics(); Type: FUNCTION; Schema: public; Owner: postgres
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


ALTER FUNCTION public.get_users_statistics() OWNER TO postgres;

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
-- Name: update_updated_at_column(); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.update_updated_at_column() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;


ALTER FUNCTION public.update_updated_at_column() OWNER TO postgres;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: users; Type: TABLE; Schema: public; Owner: postgres
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
    referral_count integer DEFAULT 0
);


ALTER TABLE public.users OWNER TO postgres;

--
-- Name: TABLE users; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON TABLE public.users IS 'Пользователи VPN бота';


--
-- Name: active_users; Type: VIEW; Schema: public; Owner: postgres
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


ALTER VIEW public.active_users OWNER TO postgres;

--
-- Name: ip_connections; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.ip_connections (
    id integer NOT NULL,
    telegram_id bigint,
    ip_address inet,
    connection_data jsonb,
    "timestamp" timestamp without time zone DEFAULT now()
);


ALTER TABLE public.ip_connections OWNER TO postgres;

--
-- Name: TABLE ip_connections; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON TABLE public.ip_connections IS 'Временные подключения IP адресов (TTL 1 час)';


--
-- Name: ip_connections_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.ip_connections_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.ip_connections_id_seq OWNER TO postgres;

--
-- Name: ip_connections_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.ip_connections_id_seq OWNED BY public.ip_connections.id;


--
-- Name: ip_violations; Type: TABLE; Schema: public; Owner: postgres
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


ALTER TABLE public.ip_violations OWNER TO postgres;

--
-- Name: TABLE ip_violations; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON TABLE public.ip_violations IS 'Нарушения и блокировки IP адресов';


--
-- Name: ip_violations_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.ip_violations_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.ip_violations_id_seq OWNER TO postgres;

--
-- Name: ip_violations_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.ip_violations_id_seq OWNED BY public.ip_violations.id;


--
-- Name: paying_users; Type: VIEW; Schema: public; Owner: postgres
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


ALTER VIEW public.paying_users OWNER TO postgres;

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
-- Name: traffic_configs; Type: TABLE; Schema: public; Owner: postgres
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


ALTER TABLE public.traffic_configs OWNER TO postgres;

--
-- Name: TABLE traffic_configs; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON TABLE public.traffic_configs IS 'Настройки трафика';


--
-- Name: trial_available_users; Type: VIEW; Schema: public; Owner: postgres
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


ALTER VIEW public.trial_available_users OWNER TO postgres;

--
-- Name: users_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.users_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.users_id_seq OWNER TO postgres;

--
-- Name: users_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;


--
-- Name: ip_connections id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.ip_connections ALTER COLUMN id SET DEFAULT nextval('public.ip_connections_id_seq'::regclass);


--
-- Name: ip_violations id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.ip_violations ALTER COLUMN id SET DEFAULT nextval('public.ip_violations_id_seq'::regclass);


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
-- Name: users id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users ALTER COLUMN id SET DEFAULT nextval('public.users_id_seq'::regclass);


--
-- Data for Name: ip_connections; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.ip_connections (id, telegram_id, ip_address, connection_data, "timestamp") FROM stdin;
\.


--
-- Data for Name: ip_violations; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.ip_violations (id, telegram_id, ip_address, is_blocked, violation_count, violation_type, violation_data, created_at, updated_at) FROM stdin;
\.


--
-- Data for Name: promo_codes; Type: TABLE DATA; Schema: public; Owner: vpn_bot_user
--

COPY public.promo_codes (id, code, amount, created_by, created_at, expires_at, is_active, used_by, used_at, usage_count, max_uses) FROM stdin;
promo_1757952253_9f823c74	9f823c74	5000.00	873925520	2025-09-15 18:04:13.645742+02	2025-09-29 18:04:13.64574+02	t	873925520	2025-09-15 18:04:20.576971+02	1	1
\.


--
-- Data for Name: promo_usage; Type: TABLE DATA; Schema: public; Owner: vpn_bot_user
--

COPY public.promo_usage (id, promo_id, user_id, amount, used_at) FROM stdin;
1	promo_1757952253_9f823c74	873925520	5000.00	2025-09-15 18:04:20.576971+02
\.


--
-- Data for Name: referral_bonuses; Type: TABLE DATA; Schema: public; Owner: vpn_bot_user
--

COPY public.referral_bonuses (id, user_telegram_id, bonus_type, amount, referral_code, related_user_id, description, created_at) FROM stdin;
3	873925520	referrer	500.00	ref_873925520520	5035512654	Реферальный бонус за приглашение друга	2025-09-16 17:11:29.368349
4	5035512654	referred	500.00	ref_873925520520	873925520	Приветственный бонус за регистрацию по реферальной ссылке	2025-09-16 17:11:29.476189
1	873925520	referrer	40.00	ref_873925520520	5035512654	Реферальный бонус за приглашение друга	2025-09-15 18:03:02.049494
2	5035512654	referred	40.00	ref_873925520520	873925520	Приветственный бонус за регистрацию по реферальной ссылке	2025-09-15 18:03:02.267335
\.


--
-- Data for Name: referral_transitions; Type: TABLE DATA; Schema: public; Owner: vpn_bot_user
--

COPY public.referral_transitions (id, referrer_telegram_id, referred_telegram_id, referral_code, transition_date, bonus_paid, bonus_amount, created_at) FROM stdin;
3	873925520	1039240440	ref_873925520520	2025-09-17 12:00:27.370356	f	0.00	2025-09-17 12:00:27.370356
2	873925520	5035512654	ref_873925520520	2025-09-16 17:11:29.301012	f	0.00	2025-09-16 17:11:29.301012
1	873925520	5035512654	ref_873925520520	2025-09-15 18:03:01.842104	f	0.00	2025-09-15 18:03:01.842104
\.


--
-- Data for Name: traffic_configs; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.traffic_configs (id, enabled, daily_limit_gb, weekly_limit_gb, monthly_limit_gb, limit_gb, reset_days, created_at, updated_at) FROM stdin;
default	t	0	0	0	0	30	2025-09-16 17:10:29.859714	2025-09-16 17:10:29.859714
\.


--
-- Data for Name: users; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.users (id, telegram_id, username, first_name, last_name, balance, total_paid, configs_count, has_active_config, client_id, sub_id, email, config_created_at, expiry_time, has_used_trial, created_at, updated_at, referral_code, referred_by, referral_earnings, referral_count) FROM stdin;
133	1039240440	Kovalenkoelena14021981	Елена	Коваленко	500.00	500.00	1	t	8b05e658-4c2d-4d50-b5ec-cc560c8a8b6e	d9smru0o9jeefavp	1039240440	2025-09-17 12:00:28.512408	1763460028512	f	2025-09-17 12:00:27.362795	2025-09-17 12:00:30.465749	\N	873925520	0.00	0
115	7108317408	aaandrey23	Андрей		50.00	50.00	0	t	c4317ed5-f893-4f30-91a1-70b1767b7ad6	1vumicoe86ez0kau	7108317408	\N	1758625249375	t	2025-09-17 08:12:30.372758	2025-09-17 13:00:50.431903	\N	0	0.00	0
117	5035512654	moment_was	Слава		5750.00	5750.00	1	t	ab706267-3a05-466f-958f-c80c404b8486	hoxl42991bz1ajki	5035512654	2025-09-16 17:11:30.432681	1820142051052	t	2025-09-16 17:11:29.292495	2025-09-17 13:00:51.061883	\N	0	0.00	0
116	873925520	BloknotaNet	Vlad		1149.00	1150.00	0	t	2116d0fd-78a4-47f9-96c3-c10678239ae8	gpud4sq9zottu4s5	873925520	\N	1770462051666	t	2025-09-16 17:10:36.951115	2025-09-17 13:00:52.699073	\N	0	0.00	0
121	5083088553	missMOS6581	miss＾ω＾		50.00	50.00	0	t	0bf7d05d-50cc-4a3f-b4d0-1511449b2e1c	927n1pbdu8m99oma	5083088553	\N	1758625729398	t	2025-09-17 10:06:29.414063	2025-09-17 13:08:50.437217	\N	0	0.00	0
\.


--
-- Name: ip_connections_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.ip_connections_id_seq', 1, false);


--
-- Name: ip_violations_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.ip_violations_id_seq', 1, false);


--
-- Name: promo_usage_id_seq; Type: SEQUENCE SET; Schema: public; Owner: vpn_bot_user
--

SELECT pg_catalog.setval('public.promo_usage_id_seq', 1, true);


--
-- Name: referral_bonuses_id_seq; Type: SEQUENCE SET; Schema: public; Owner: vpn_bot_user
--

SELECT pg_catalog.setval('public.referral_bonuses_id_seq', 4, true);


--
-- Name: referral_transitions_id_seq; Type: SEQUENCE SET; Schema: public; Owner: vpn_bot_user
--

SELECT pg_catalog.setval('public.referral_transitions_id_seq', 3, true);


--
-- Name: users_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.users_id_seq', 135, true);


--
-- Name: ip_connections ip_connections_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.ip_connections
    ADD CONSTRAINT ip_connections_pkey PRIMARY KEY (id);


--
-- Name: ip_violations ip_violations_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.ip_violations
    ADD CONSTRAINT ip_violations_pkey PRIMARY KEY (id);


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
-- Name: traffic_configs traffic_configs_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.traffic_configs
    ADD CONSTRAINT traffic_configs_pkey PRIMARY KEY (id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: users users_referral_code_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_referral_code_key UNIQUE (referral_code);


--
-- Name: users users_telegram_id_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_telegram_id_key UNIQUE (telegram_id);


--
-- Name: idx_ip_connections_ip; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_ip_connections_ip ON public.ip_connections USING btree (ip_address);


--
-- Name: idx_ip_connections_telegram_timestamp; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_ip_connections_telegram_timestamp ON public.ip_connections USING btree (telegram_id, "timestamp" DESC);


--
-- Name: idx_ip_connections_timestamp; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_ip_connections_timestamp ON public.ip_connections USING btree ("timestamp");


--
-- Name: idx_ip_violations_created_at; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_ip_violations_created_at ON public.ip_violations USING btree (created_at);


--
-- Name: idx_ip_violations_ip; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_ip_violations_ip ON public.ip_violations USING btree (ip_address);


--
-- Name: idx_ip_violations_telegram_blocked; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_ip_violations_telegram_blocked ON public.ip_violations USING btree (telegram_id, is_blocked);


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
-- Name: idx_users_balance; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_users_balance ON public.users USING btree (balance);


--
-- Name: idx_users_created_at; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_users_created_at ON public.users USING btree (created_at);


--
-- Name: idx_users_has_active_config; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_users_has_active_config ON public.users USING btree (has_active_config);


--
-- Name: idx_users_has_used_trial; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_users_has_used_trial ON public.users USING btree (has_used_trial);


--
-- Name: idx_users_telegram_id; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX idx_users_telegram_id ON public.users USING btree (telegram_id);


--
-- Name: ip_violations update_ip_violations_updated_at; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER update_ip_violations_updated_at BEFORE UPDATE ON public.ip_violations FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: traffic_configs update_traffic_configs_updated_at; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER update_traffic_configs_updated_at BEFORE UPDATE ON public.traffic_configs FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: users update_users_updated_at; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON public.users FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: ip_connections ip_connections_telegram_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.ip_connections
    ADD CONSTRAINT ip_connections_telegram_id_fkey FOREIGN KEY (telegram_id) REFERENCES public.users(telegram_id) ON DELETE CASCADE;


--
-- Name: ip_violations ip_violations_telegram_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.ip_violations
    ADD CONSTRAINT ip_violations_telegram_id_fkey FOREIGN KEY (telegram_id) REFERENCES public.users(telegram_id) ON DELETE CASCADE;


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
-- Name: FUNCTION cleanup_old_ip_connections(); Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON FUNCTION public.cleanup_old_ip_connections() TO vpn_bot_user;


--
-- Name: FUNCTION get_users_statistics(); Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON FUNCTION public.get_users_statistics() TO vpn_bot_user;


--
-- Name: FUNCTION update_updated_at_column(); Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON FUNCTION public.update_updated_at_column() TO vpn_bot_user;


--
-- Name: TABLE users; Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON TABLE public.users TO vpn_bot_user;


--
-- Name: TABLE active_users; Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON TABLE public.active_users TO vpn_bot_user;


--
-- Name: TABLE ip_connections; Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON TABLE public.ip_connections TO vpn_bot_user;


--
-- Name: SEQUENCE ip_connections_id_seq; Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON SEQUENCE public.ip_connections_id_seq TO vpn_bot_user;


--
-- Name: TABLE ip_violations; Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON TABLE public.ip_violations TO vpn_bot_user;


--
-- Name: SEQUENCE ip_violations_id_seq; Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON SEQUENCE public.ip_violations_id_seq TO vpn_bot_user;


--
-- Name: TABLE paying_users; Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON TABLE public.paying_users TO vpn_bot_user;


--
-- Name: TABLE traffic_configs; Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON TABLE public.traffic_configs TO vpn_bot_user;


--
-- Name: TABLE trial_available_users; Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON TABLE public.trial_available_users TO vpn_bot_user;


--
-- Name: SEQUENCE users_id_seq; Type: ACL; Schema: public; Owner: postgres
--

GRANT ALL ON SEQUENCE public.users_id_seq TO vpn_bot_user;


--
-- PostgreSQL database dump complete
--

\unrestrict cx8ajGJ1S8IEfazqwRQIgedbCBegO3ihRnYKxULsVEYYuBnVnKW8gUyJiovA1YI

