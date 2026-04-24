-- Seed script: chats for 129 test users.
-- Each user gets 1–4 chats depending on rn % 4.
-- Most chats keep the default name "New Chat"; ~30% are renamed to topic-based names.
-- updated_datetime reflects when the last message was sent (after creation).

WITH numbered_users AS (
    SELECT
        user_id,
        ROW_NUMBER() OVER (ORDER BY user_id) AS rn
    FROM users
    WHERE login LIKE 'user\_%' ESCAPE '\'
)
INSERT INTO chats (user_id, chat_name, is_active, creation_datetime, updated_datetime)
SELECT
    nu.user_id,

    -- ~30% of chats get a meaningful name (those where (rn + i) % 3 = 0)
    CASE
        WHEN (nu.rn + gen.i) % 3 = 0 THEN
            (ARRAY[
                'Anxiety management',
                'Work-life balance',
                'Relationship advice',
                'Overcoming burnout',
                'Building confidence',
                'Managing stress',
                'Sleep improvement',
                'Daily mood check-in'
            ])[ ((nu.rn * 3 + gen.i * 7) % 8) + 1 ]
        ELSE 'New Chat'
    END,

    -- ~10% of chats are inactive (archived)
    (nu.rn * gen.i) % 10 != 0,

    -- Creation datetime: spread over the past 90 days, older chats first
    NOW() - ((90 - (gen.i - 1) * 20 + (nu.rn * 7) % 15)) * INTERVAL '1 day'
         - ((nu.rn * gen.i * 11) % 23) * INTERVAL '1 hour',

    -- updated_datetime: 0–72 hours after creation (simulates last message activity)
    NOW() - ((90 - (gen.i - 1) * 20 + (nu.rn * 7) % 15)) * INTERVAL '1 day'
         - ((nu.rn * gen.i * 11) % 23) * INTERVAL '1 hour'
         + ((nu.rn * gen.i * 13) % 72) * INTERVAL '1 hour'

FROM numbered_users nu
CROSS JOIN LATERAL (
    SELECT s.i
    FROM generate_series(1, 1 + (nu.rn % 4)) AS s(i)
) AS gen;
