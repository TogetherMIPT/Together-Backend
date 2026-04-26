-- Seed script: traces of 24 payments via last_payment_datetime on users.
-- In the app, last_payment_datetime is set when the free trial ends (message.go).
-- Active subscription: paid within the last 30 days.
-- Expired subscription: paid more than 30 days ago.
--
-- Distribution across the first 24 users (by user_id order):
--   users  1–16: active subscription  (paid 1–28 days ago)
--   users 17–24: expired subscription (paid 31–60 days ago)

WITH ranked AS (
    SELECT
        user_id,
        ROW_NUMBER() OVER (ORDER BY user_id) AS rn
    FROM users
    WHERE login LIKE 'user\_%' ESCAPE '\'
)
UPDATE users
SET last_payment_datetime =
    CASE
        WHEN r.rn <= 16
            THEN NOW() - ((r.rn * 7)       % 28 + 1) * INTERVAL '1 day'
        ELSE
             NOW() - ((r.rn * 5 - 16) % 30 + 31) * INTERVAL '1 day'
    END
FROM ranked r
WHERE users.user_id = r.user_id
  AND r.rn <= 24;
