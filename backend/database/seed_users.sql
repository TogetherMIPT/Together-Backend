-- Seed script: generates 29 test users.
--
-- NOTE: name/email/country/city/gender are stored as plain text (not AES-256-GCM encrypted),
-- which is intentional for seed data. The app's Decrypt() function falls back to returning
-- the original string when decryption fails, so these values will be readable through the API.
--
-- Password for all users: Password1!
-- Bcrypt hash (cost=14): $2a$14$oOpmcIzgGmjLfuwlsrRby.Yq3CHjtnF4KFTv9sTS6nz9WTlKycy8q

INSERT INTO users (name, email, login, password, country, city, gender, birthdate, creation_datetime)
SELECT
    'User ' || i,
    'user' || i || '@example.com',
    'user_' || i,
    '$2a$14$oOpmcIzgGmjLfuwlsrRby.Yq3CHjtnF4KFTv9sTS6nz9WTlKycy8q',
    (ARRAY['Russia', 'Germany', 'France', 'USA', 'UK', 'Spain', 'Italy'])[ (i % 7) + 1 ],
    (ARRAY['Moscow', 'Berlin', 'Paris', 'New York', 'London', 'Madrid', 'Rome'])[ (i % 7) + 1 ],
    CASE WHEN i % 3 = 0 THEN 'male' WHEN i % 3 = 1 THEN 'female' ELSE '' END,
    DATE '1990-01-01' + ((i * 97) % 10950) * INTERVAL '1 day',
    NOW()
FROM generate_series(1, 29) AS s(i);
