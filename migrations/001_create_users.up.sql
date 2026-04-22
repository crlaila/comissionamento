-- Tabela de usuários do sistema.
-- Armazena vendedores, gestores, financeiro e admins.
--
-- Campo "role": controla o que o usuário pode ver e fazer.
--   - rep: vendedor, vê só os próprios dados
--   - manager: gestor, vê a equipe dele
--   - finance: financeiro, vê tudo e aprova statements
--   - admin: administrador, gerencia usuários e configurações
--
-- Campo "manager_id": referência ao próprio usuário que é gestor deste rep.
-- Isso cria a hierarquia de equipe (gestor → vendedores).

CREATE TYPE user_role AS ENUM ('admin', 'manager', 'rep', 'finance');

CREATE TABLE users (
    id            BIGSERIAL    PRIMARY KEY,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    name          VARCHAR(255) NOT NULL,
    role          user_role    NOT NULL DEFAULT 'rep',
    manager_id    BIGINT       REFERENCES users(id),
    active        BOOLEAN      NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- Índice para busca por email no login (precisa ser rápido).
CREATE INDEX idx_users_email ON users(email);

-- Índice para buscar os vendedores de um gestor.
CREATE INDEX idx_users_manager_id ON users(manager_id);
