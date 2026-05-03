CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    birthday DATE NOT NULL,
    birthday_self_notify BOOLEAN NOT NULL DEFAULT TRUE,
    birthday_member_notify BOOLEAN NOT NULL DEFAULT TRUE,
    post_notify BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);