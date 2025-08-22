CREATE TABLE IF NOT EXISTS policies (
    id SERIAL PRIMARY KEY,
    subject TEXT NOT NULL,
    object TEXT NOT NULL,
    action TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (subject, object, action)
);