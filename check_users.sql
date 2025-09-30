SELECT 
    telegram_id,
    first_name,
    email,
    has_active_config,
    balance,
    expiry_time,
    to_timestamp(expiry_time / 1000) as expiry_date,
    CASE 
        WHEN expiry_time > extract(epoch from now()) * 1000 THEN 'АКТИВНА'
        ELSE 'ИСТЕКЛА'
    END as status
FROM users 
WHERE email IS NOT NULL 
ORDER BY telegram_id
LIMIT 20;
