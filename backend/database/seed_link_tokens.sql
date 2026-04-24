-- Seed script: link tokens for ~2/3 of the 129 test users.
-- Token format: 64-char hex string (32 random bytes), matching generateToken() in link.go.
-- Tokens last 24 hours. Existing records are either active or expired-but-unused
-- (used tokens are deleted from the DB by LinkUsersHandler).
--
-- Distribution:
--   rn % 3 = 1 (~43 users): active token, created 2–10 hours ago
--   rn % 3 = 2 (~43 users): expired token, created 2–5 days ago (never used)
--   rn % 3 = 0 (~43 users): no token (user never generated one)

WITH numbered_users AS (
    SELECT
        user_id,
        ROW_NUMBER() OVER (ORDER BY user_id) AS rn
    FROM users
    WHERE login LIKE 'user\_%' ESCAPE '\'
)
INSERT INTO link_tokens (token, user_id, creation_datetime, expiration_datetime)
SELECT
    encode(gen_random_bytes(32), 'hex'),
    nu.user_id,
    created_at,
    created_at + INTERVAL '24 hours'
FROM numbered_users nu
CROSS JOIN LATERAL (
    SELECT
        CASE nu.rn % 3
            -- Active: created 2–10 hours ago
            WHEN 1 THEN NOW() - (2 + (nu.rn * 7) % 9)  * INTERVAL '1 hour'
            -- Expired: created 2–5 days ago
            WHEN 2 THEN NOW() - (2 + (nu.rn * 3) % 4)  * INTERVAL '1 day'
        END AS created_at
) AS gen
WHERE nu.rn % 3 != 0;
