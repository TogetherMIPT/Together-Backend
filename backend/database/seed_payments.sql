-- Seed script: traces of 32 payments via last_payment_datetime on users.
-- In the app, last_payment_datetime is set when the free trial ends (message.go).
-- All 32 payments fall within the last 3 days → all subscriptions are active (< 30 days).
-- Non-uniform spread: quadratic term creates irregular gaps between payments.

WITH ranked AS (
    SELECT
        user_id,
        ROW_NUMBER() OVER (ORDER BY user_id) AS rn
    FROM users
    WHERE login LIKE 'user\_%' ESCAPE '\'
)
UPDATE users
SET last_payment_datetime =
    NOW() - ((r.rn * 83 + r.rn * r.rn * 5) % (3 * 24 * 60)) * INTERVAL '1 minute'
FROM ranked r
WHERE users.user_id = r.user_id
  AND r.rn <= 32;
