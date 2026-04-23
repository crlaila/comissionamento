-- Tabela de refresh tokens para gerenciamento de sessão.
-- Armazena refresh tokens válidos e seus períodos de expiração.
-- Refresh tokens podem ser revogados manualmente (logout).

CREATE TABLE refresh_tokens (
    id        BIGSERIAL    PRIMARY KEY,
    user_id   BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token     TEXT         NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Índice para busca rápida por user_id (validação de token ao refresh)
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);

-- Índice para limpeza de tokens expirados
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);

-- Índice para busca por token durante logout/refresh
CREATE INDEX idx_refresh_tokens_token ON refresh_tokens(token);
