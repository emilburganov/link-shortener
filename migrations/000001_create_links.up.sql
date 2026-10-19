CREATE TABLE links (
    id         BIGSERIAL PRIMARY KEY,
    code       TEXT        NOT NULL,
    url        TEXT        NOT NULL,
    clicks     BIGINT      NOT NULL DEFAULT 0,
    max_clicks BIGINT,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT links_code_key UNIQUE (code),
    CONSTRAINT links_url_check CHECK (url <> ''),
    CONSTRAINT links_clicks_check CHECK (clicks >= 0),
    CONSTRAINT links_max_clicks_check CHECK (max_clicks IS NULL OR max_clicks > 0)
);
