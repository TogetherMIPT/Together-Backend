-- Seed script: session history for 129 test users.
-- Each user gets 2-3 sessions; tokens are UUIDs matching app format (uuid.New().String()).
-- Sessions last 24 hours per app logic. Mix of active and expired sessions.

WITH numbered_users AS (
    SELECT
        user_id,
        ROW_NUMBER() OVER (ORDER BY user_id) AS rn
    FROM users
    WHERE login LIKE 'user\_%' ESCAPE '\'
)
INSERT INTO sessions (token, user_id, creation_datetime, expiration_datetime)
SELECT
    gen_random_uuid()::text,
    nu.user_id,
    gen.created_at,
    gen.created_at + INTERVAL '24 hours'
FROM numbered_users nu
CROSS JOIN LATERAL (
    SELECT
        CASE s.i
            -- Most recent login: 0–19 hours ago → ~80% of users still have an active session
            WHEN 1 THEN NOW() - ((nu.rn * 7) % 20) * INTERVAL '1 hour'
            -- Previous login: 5–19 days ago → always expired
            WHEN 2 THEN NOW() - (5 + (nu.rn * 3) % 15) * INTERVAL '1 day'
            -- Oldest login (only for users where rn % 2 = 0): 20–44 days ago
            WHEN 3 THEN NOW() - (20 + (nu.rn * 5) % 25) * INTERVAL '1 day'
        END AS created_at
    FROM generate_series(1, 2 + (nu.rn % 2)) AS s(i)
) AS gen;
