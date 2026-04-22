-- Tabela de metas de comissão.
-- Define quanto cada vendedor precisa atingir em um período
-- e quanto ele ganha se atingir 100% da meta.
--
-- Exemplo: Vendedor João, Abril 2026
--   acquisition_target = 10  (captar 10 novos associados)
--   renewal_target = 20      (renovar 20 associados)
--   commission_value = 500000 (R$ 5.000,00 se atingir 100%)
--
-- Se João atingir 80% da meta, ganha R$ 4.000,00 (proporcional).
--
-- IMPORTANTE: commission_value é em CENTAVOS (int64).
-- R$ 5.000,00 = 500000 centavos.
-- Usamos centavos para evitar erros de arredondamento com float.
-- Nunca use float para dinheiro!
--
-- A constraint unique garante que um vendedor tem no máximo
-- uma meta por período — evita duplicatas acidentais.

CREATE TABLE goals (
    id                 BIGSERIAL   PRIMARY KEY,
    rep_id             BIGINT      NOT NULL REFERENCES users(id),
    period_id          BIGINT      NOT NULL REFERENCES periods(id),
    acquisition_target INT         NOT NULL DEFAULT 0,
    renewal_target     INT         NOT NULL DEFAULT 0,
    commission_value   BIGINT      NOT NULL DEFAULT 0,  -- em centavos
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- Um vendedor só pode ter uma meta por período.
    CONSTRAINT uq_goals_rep_period UNIQUE (rep_id, period_id),

    -- Metas devem ser valores positivos.
    CONSTRAINT chk_goals_positive CHECK (
        acquisition_target >= 0 AND
        renewal_target >= 0 AND
        commission_value >= 0
    )
);

CREATE INDEX idx_goals_period_id ON goals(period_id);
CREATE INDEX idx_goals_rep_id ON goals(rep_id);
