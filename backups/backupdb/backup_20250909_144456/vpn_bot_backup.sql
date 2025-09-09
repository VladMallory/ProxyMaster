--
-- PostgreSQL database dump
--

\restrict xVLVACUtu5LoemXWIoisBQ2wUo69Qq1o02nHvB76k991HTFcHyyeKpRvQv4fIPj

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
    updated_at timestamp without time zone DEFAULT now()
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
-- Data for Name: traffic_configs; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.traffic_configs (id, enabled, daily_limit_gb, weekly_limit_gb, monthly_limit_gb, limit_gb, reset_days, created_at, updated_at) FROM stdin;
default	t	0	0	0	0	30	2025-09-08 09:20:18.49654	2025-09-08 09:20:18.49654
\.


--
-- Data for Name: users; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.users (id, telegram_id, username, first_name, last_name, balance, total_paid, configs_count, has_active_config, client_id, sub_id, email, config_created_at, expiry_time, has_used_trial, created_at, updated_at) FROM stdin;
3	7517377017	Jekaxgod13	Almaaazik		0.00	0.00	1	t	8c04ebe3-c790-48d0-88fa-fe0016ad454d	hq9fk60uq4gugp8n	\N	\N	1757488863376	t	2025-09-08 13:20:13.35654	2025-09-08 13:20:14.754557
4	5035512654	moment_was	Слава		0.00	0.00	1	t	c7bd19b7-e00d-4a26-99f3-bc632f38acff	mihptl6pdwhyfko6	5035512654	2025-09-08 17:36:17.71233	1757432177712	t	2025-09-08 17:36:16.103374	2025-09-08 17:36:18.225822
2	873925520	BloknotaNet	Vlad		9690.00	10000.00	3	t	0f62f859-e9d8-4e10-b0fd-c5bd0f3c1c79	h9dqnutb7yqp7yui	873925520	2025-09-08 17:31:40.943738	1757431900943	t	2025-09-08 10:48:16.623695	2025-09-08 17:55:20.184299
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
-- Name: users_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.users_id_seq', 4, true);


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

\unrestrict xVLVACUtu5LoemXWIoisBQ2wUo69Qq1o02nHvB76k991HTFcHyyeKpRvQv4fIPj

