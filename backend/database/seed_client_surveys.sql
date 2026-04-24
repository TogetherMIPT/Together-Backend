-- Seed script: one client_survey per user (onboarding questionnaire).
-- Covers all 129 users inserted by seed_users.sql.
-- user_ids are resolved by login to avoid hardcoding ID ranges.

INSERT INTO client_surveys (user_id, with_psychologist, therapy_request, therapy_approach, weekly_meetings, creation_datetime)
SELECT
    u.user_id,

    -- ~60% want a psychologist, ~40% prefer self-help
    (ROW_NUMBER() OVER (ORDER BY u.user_id) % 5) != 0,

    -- Therapy request text, 7 variants cycling through users
    (ARRAY[
        'I experience persistent anxiety and panic attacks that interfere with daily life.',
        'I struggle with low mood and lack of motivation for several months.',
        'I have difficulties in relationships and communication with close people.',
        'I want to work through childhood trauma and its impact on my current behaviour.',
        'I face burnout at work and cannot find a work-life balance.',
        'I have low self-esteem and often engage in negative self-talk.',
        'I want to develop emotional resilience and improve stress management.'
    ])[ (ROW_NUMBER() OVER (ORDER BY u.user_id) % 7) + 1 ],

    -- Therapy approach, varies among users; NULL for ~20% (no preference)
    CASE
        WHEN (ROW_NUMBER() OVER (ORDER BY u.user_id) % 5) = 0 THEN NULL
        ELSE (ARRAY[
            'Cognitive Behavioural Therapy',
            'Acceptance and Commitment Therapy',
            'Psychoanalytic',
            'Gestalt'
        ])[ (ROW_NUMBER() OVER (ORDER BY u.user_id) % 4) + 1 ]
    END,

    -- Weekly meetings: 0 for self-help users, 1-3 for those with psychologist
    CASE
        WHEN (ROW_NUMBER() OVER (ORDER BY u.user_id) % 5) = 0 THEN 0
        ELSE (ROW_NUMBER() OVER (ORDER BY u.user_id) % 3) + 1
    END,

    -- Registration-like datetime: spread over the past 6 months, offset per user
    u.creation_datetime + (ROW_NUMBER() OVER (ORDER BY u.user_id) % 10) * INTERVAL '1 minute'

FROM users u
WHERE u.login LIKE 'user\_%' ESCAPE '\'
ORDER BY u.user_id;
