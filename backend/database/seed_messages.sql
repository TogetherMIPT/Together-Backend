-- Seed script: messages for all seeded chats.
-- Messages come in user→assistant pairs per turn, matching real app behaviour.
-- message_text is stored as plain text (not AES-256-GCM encrypted) — the app's
-- Decrypt() fallback returns the original string when decryption fails.
--
-- Turns per chat: 1–10 (based on chat_rn % 10).
-- Turns are spaced 25 minutes apart; assistant replies 3 minutes after the user.

WITH chat_data AS (
    SELECT
        c.chat_id,
        c.creation_datetime                    AS chat_created,
        ROW_NUMBER() OVER (ORDER BY c.chat_id) AS chat_rn
    FROM chats c
    JOIN users u ON c.user_id = u.user_id
    WHERE u.login LIKE 'user\_%' ESCAPE '\'
),
turns AS (
    SELECT
        cd.chat_id,
        cd.chat_created,
        cd.chat_rn,
        s.i AS turn_num
    FROM chat_data cd
    CROSS JOIN LATERAL generate_series(1, 1 + (cd.chat_rn % 10)) AS s(i)
)
INSERT INTO messages (chat_id, message_text, is_from_user, creation_datetime)

-- User messages
SELECT
    t.chat_id,
    (ARRAY[
        'I''ve been feeling really anxious lately and I don''t know how to cope.',
        'I had another argument with my partner and I''m feeling completely lost.',
        'Work has been overwhelming me. I can''t seem to switch off in the evenings.',
        'I woke up feeling hopeless again today. Nothing seems to bring me joy.',
        'I keep overthinking everything and it''s exhausting.',
        'I had a panic attack yesterday for the first time. It was terrifying.',
        'I feel like I''m not good enough no matter what I do.',
        'I''ve been struggling to sleep for weeks now.',
        'I''m finding it hard to connect with people around me lately.',
        'I feel overwhelmed by all my responsibilities and I can''t prioritise.'
    ])[ ((t.chat_rn * t.turn_num * 3) % 10) + 1 ],
    true,
    t.chat_created + (t.turn_num - 1) * INTERVAL '25 minutes'

FROM turns t

UNION ALL

-- Assistant messages
SELECT
    t.chat_id,
    (ARRAY[
        'I hear you, and I''m glad you reached out. Anxiety can feel overwhelming. Let''s explore what''s been triggering these feelings — can you tell me more about when it tends to be strongest?',
        'Arguments in close relationships can leave us feeling deeply unsettled. What do you think was at the heart of the disagreement?',
        'It sounds like work is taking up a lot of mental space. What does your evening routine look like at the moment?',
        'I''m sorry you started the day that way. Feelings of hopelessness can be really heavy to carry. Have there been any small moments of relief recently?',
        'Overthinking often comes from a place of wanting to protect ourselves. What kinds of thoughts tend to loop for you most?',
        'Panic attacks can be really frightening, especially the first time. You''re safe now. Would you like to talk about what was happening just before it started?',
        'That inner critic can be relentless. Where do you think these feelings of not being enough come from?',
        'Sleep difficulties can affect everything else. Are you finding it hard to fall asleep, stay asleep, or both?',
        'Feeling disconnected from others is more common than people realise. Has anything changed in your life recently that might have contributed?',
        'When everything feels urgent at once, it''s hard to know where to start. Let''s try to map out what''s on your plate — would that help?'
    ])[ ((t.chat_rn * t.turn_num * 3) % 10) + 1 ],
    false,
    t.chat_created + (t.turn_num - 1) * INTERVAL '25 minutes' + INTERVAL '3 minutes'

FROM turns t

ORDER BY chat_id, creation_datetime;
