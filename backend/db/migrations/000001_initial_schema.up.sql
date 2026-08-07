CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    google_id TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL,
    name TEXT NOT NULL,
    avatar_url TEXT,
    stream_key_hash TEXT NOT NULL,
    stream_title TEXT CHECK (
        stream_title IS NULL OR char_length(stream_title) BETWEEN 1 AND 100
    ),
    stream_category TEXT,
    is_live BOOLEAN NOT NULL DEFAULT false,
    live_since TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_users_google_id ON users(google_id);
CREATE INDEX idx_users_stream_key_hash ON users(stream_key_hash);

CREATE TABLE streams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    title TEXT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ended_at TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'live'
        CHECK (status IN ('live', 'interrupted', 'offline')),
    hls_path TEXT,
    recording_path TEXT,
    recording_status TEXT NOT NULL DEFAULT 'pending'
        CHECK (recording_status IN ('pending', 'processing', 'ready', 'failed')),
    peak_viewers INTEGER NOT NULL DEFAULT 0,
    total_viewers INTEGER NOT NULL DEFAULT 0,
    duration_seconds INTEGER,
    srs_client_id INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_streams_user_id ON streams(user_id);
CREATE INDEX idx_streams_status ON streams(status);

CREATE TABLE chat_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    stream_id UUID NOT NULL REFERENCES streams(id),
    user_id UUID NOT NULL REFERENCES users(id),
    message TEXT NOT NULL CHECK (char_length(message) BETWEEN 1 AND 500),
    sent_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_chat_messages_stream_sent ON chat_messages(stream_id, sent_at);

CREATE TABLE stream_viewers (
    stream_id UUID NOT NULL REFERENCES streams(id),
    user_id UUID REFERENCES users(id),
    client_id TEXT NOT NULL,
    first_seen TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (stream_id, client_id)
);

CREATE TABLE stream_analytics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    date DATE NOT NULL,
    total_seconds INTEGER NOT NULL DEFAULT 0,
    peak_viewers INTEGER NOT NULL DEFAULT 0,
    unique_viewers INTEGER NOT NULL DEFAULT 0,
    UNIQUE (user_id, date)
);
CREATE INDEX idx_stream_analytics_user_date ON stream_analytics(user_id, date DESC);
