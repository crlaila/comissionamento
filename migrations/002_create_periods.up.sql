-- Tabela de períodos de comissão.
-- Um período é o intervalo de tempo no qual as metas são medidas
-- e as comissões calculadas (ex: "Abril 2026", "Q2 2026").
--
-- Campo "status":
--   - open: período ativo, metas podem ser editadas, eventos sendo contados
--   - closed: período encerrado, statements gerados para aprovação
--   - archived: tudo finalizado e pago, guardado para histórico
--
-- Regra de negócio: metas só podem ser editadas em períodos "open".
-- Quando o período fecha, as metas são "travadas" para garantir
-- que o cálculo de comissão não mude depois.

CREATE TYPE period_status AS ENUM ('open', 'closed', 'archived');

CREATE TABLE periods (
    id         BIGSERIAL     PRIMARY KEY,
    name       VARCHAR(100)  NOT NULL,
    start_date DATE          NOT NULL,
    end_date   DATE          NOT NULL,
    status     period_status NOT NULL DEFAULT 'open',
    created_at TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ   NOT NULL DEFAULT now(),

    -- Garante que a data de início é antes da data de fim.
    CONSTRAINT chk_period_dates CHECK (start_date < end_date)
);
