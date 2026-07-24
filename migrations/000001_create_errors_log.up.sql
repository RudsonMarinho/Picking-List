CREATE TABLE errors_log (
    error_id        VARCHAR(12) PRIMARY KEY,
    request_id      UUID NOT NULL,
    tenant_id       UUID NULL,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    endpoint        VARCHAR(255),
    http_method     VARCHAR(10),
    http_status     INTEGER,
    exception_type  VARCHAR(255),
    exception_msg   TEXT,
    traceback       TEXT,
    request_payload JSONB,   -- mascarar password/token/secret ANTES do INSERT
    user_context    VARCHAR(255),
    environment     VARCHAR(20) DEFAULT 'production'
);
CREATE INDEX ix_errors_log_occurred ON errors_log(occurred_at);

-- C4: append-only para o runtime.
REVOKE UPDATE, DELETE ON errors_log FROM invtech_app;
-- Retenção 90 dias: expurgo com role de manutenção, nunca invtech_app.
