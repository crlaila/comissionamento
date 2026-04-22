-- Tabela de logs de auditoria.
-- Registra TUDO que acontece no sistema: quem fez o quê e quando.
-- Essencial para comissões — se alguém questionar um valor,
-- o financeiro pode rastrear exatamente como ele foi calculado.
--
-- Campo "action": o que foi feito (ex: "statement.approved", "goal.updated")
-- Campo "entity_type": qual tipo de entidade foi afetada (ex: "statement", "goal")
-- Campo "entity_id": o ID da entidade afetada
-- Campo "details": um JSON livre com detalhes extras.
--   Ex: {"old_status": "pending", "new_status": "approved", "reason": "Valores conferidos"}
--
-- Usamos JSONB (não JSON) porque:
--   - JSONB é mais rápido para queries (armazena em formato binário)
--   - Permite criar índices nos campos internos do JSON
--   - Suporta operadores de busca como @>, ?, etc.
--
-- Esta tabela é APPEND-ONLY — nunca editamos ou deletamos logs.

CREATE TABLE audit_logs (
    id          BIGSERIAL   PRIMARY KEY,
    user_id     BIGINT      REFERENCES users(id),
    action      VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50)  NOT NULL,
    entity_id   BIGINT       NOT NULL,
    details     JSONB        NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- Índice para buscar logs de uma entidade específica.
-- Ex: "me mostra todos os logs do statement #42"
CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id);

-- Índice para buscar logs de um usuário específico.
-- Ex: "o que o João fez hoje?"
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);

-- Índice para buscar por data (relatórios de auditoria).
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
