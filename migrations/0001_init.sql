CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key TEXT UNIQUE NOT NULL,
    label TEXT,
    revoked BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT now()
);

INSERT INTO users (id, email, password_hash)
VALUES (
    '11111111-1111-1111-1111-111111111111',
    'demo@puvaan.dev',
    '$2a$10$CwTycUXWue0Thq9StjUM0uJ8PLu/.6KsZTfXYy3jHKzmvVsQt1z8.'
)
ON CONFLICT (email) DO NOTHING;

INSERT INTO api_keys (id, user_id, key, label, revoked)
VALUES (
    '8ccf4307-9d43-4eb7-b9b6-278ade41b1a1',
    '11111111-1111-1111-1111-111111111111',
    'demo-key-123',
    'Demo key',
    false
)
ON CONFLICT (key) DO NOTHING;
