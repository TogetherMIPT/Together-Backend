-- Seed script: daily mood surveys for 129 test users.
-- Covers the past 45 days; ~20% of days are skipped per user (realistic compliance).
-- Each user has a personal mood baseline (1–3) that drifts slightly day to day.
-- All answer values are in the valid range 1–3 per app validation.

WITH numbered_users AS (
    SELECT
        user_id,
        ROW_NUMBER() OVER (ORDER BY user_id) AS rn
    FROM users
    WHERE login LIKE 'user\_%' ESCAPE '\'
),
calendar AS (
    SELECT generate_series(0, 44) AS days_ago
),
base AS (
    SELECT
        nu.user_id,
        nu.rn,
        c.days_ago,
        -- Each user has a stable personal baseline shifted by rn
        1 + (nu.rn          % 3) AS mood_base,
        1 + ((nu.rn + 1)    % 3) AS anxiety_base,
        1 + ((nu.rn + 2)    % 3) AS control_base
    FROM numbered_users nu
    CROSS JOIN calendar c
    -- Skip ~20% of days to simulate missed check-ins
    WHERE (nu.rn * c.days_ago * 7 + c.days_ago) % 5 != 0
)
INSERT INTO daily_surveys (user_id, mood_answer, anxiety_answer, control_answer, creation_datetime)
SELECT
    user_id,

    -- Mood drifts ±1 around the baseline, clamped to [1, 3]
    GREATEST(1, LEAST(3, mood_base    + ((rn * days_ago * 11) % 3) - 1)),
    GREATEST(1, LEAST(3, anxiety_base + ((rn * days_ago * 13) % 3) - 1)),
    GREATEST(1, LEAST(3, control_base + ((rn * days_ago * 17) % 3) - 1)),

    -- Surveys are submitted in the morning (8–10 AM), time varies per user
    (CURRENT_DATE - days_ago * INTERVAL '1 day')
        + (8 + (rn % 3)) * INTERVAL '1 hour'
        + ((rn * days_ago * 7) % 60) * INTERVAL '1 minute'

FROM base
ORDER BY user_id, days_ago DESC;
