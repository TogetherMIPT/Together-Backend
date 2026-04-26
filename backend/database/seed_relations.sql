-- Seed script: relations (links) between users.
-- The unique index idx_unique_relation is on (LEAST(a,b), GREATEST(a,b)),
-- so pairs are normalised to first_user_id < second_user_id to avoid conflicts.
--
-- Each user is connected to neighbours at ring offsets +1, +4, +12 (modular),
-- giving each user ~3–6 connections and ~350 total unique pairs.

WITH numbered_users AS (
    SELECT
        user_id,
        ROW_NUMBER() OVER (ORDER BY user_id) AS rn
    FROM users
    WHERE login LIKE 'user\_%' ESCAPE '\'
),
total AS (
    SELECT COUNT(*)::int AS n FROM numbered_users
),
offsets(d) AS (
    VALUES (1), (4), (12)
),
raw_pairs AS (
    SELECT DISTINCT
        LEAST(a.user_id, b.user_id)    AS first_user_id,
        GREATEST(a.user_id, b.user_id) AS second_user_id
    FROM numbered_users a
    CROSS JOIN offsets
    JOIN numbered_users b
        ON b.rn = ((a.rn - 1 + d) % (SELECT n FROM total)) + 1
    WHERE a.user_id <> b.user_id
)
INSERT INTO relations (first_user_id, second_user_id, creation_datetime)
SELECT
    first_user_id,
    second_user_id,
    -- Spread link creation over the past 60 days
    NOW() - ((first_user_id * second_user_id) % 60) * INTERVAL '1 day'
          - ((first_user_id + second_user_id) % 24) * INTERVAL '1 hour'
FROM raw_pairs
ORDER BY first_user_id, second_user_id;
