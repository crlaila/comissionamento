-- Tabela de statements (extratos de comissão).
-- Um statement é o "resumo final" da comissão de um vendedor em um período.
-- É gerado pelo financeiro no fim do período e passa por aprovação.
--
-- Fluxo de status:
--   draft → pending_approval → approved → paid
--
--   1. draft: statement gerado pelo sistema, aguardando revisão
--   2. pending_approval: financeiro colocou na fila de aprovação
--   3. approved: financeiro aprovou, pronto para pagamento
--   4. paid: pagamento efetuado
--
-- Campo "total_amount": valor total da comissão em CENTAVOS.
-- Campo "attainment_pct": percentual de atingimento da meta (0.0 a 1.0+).
-- Campo "approved_by": qual usuário do financeiro aprovou.

CREATE TYPE statement_status AS ENUM ('draft', 'pending_approval', 'approved', 'paid');

CREATE TABLE statements (
    id              BIGSERIAL        PRIMARY KEY,
    rep_id          BIGINT           NOT NULL REFERENCES users(id),
    period_id       BIGINT           NOT NULL REFERENCES periods(id),
    total_amount    BIGINT           NOT NULL DEFAULT 0,  -- em centavos
    attainment_pct  DOUBLE PRECISION NOT NULL DEFAULT 0,
    status          statement_status NOT NULL DEFAULT 'draft',
    approved_by     BIGINT           REFERENCES users(id),
    approved_at     TIMESTAMPTZ,
    rejection_reason TEXT,
    created_at      TIMESTAMPTZ      NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ      NOT NULL DEFAULT now(),

    -- Um vendedor tem no máximo um statement por período.
    CONSTRAINT uq_statements_rep_period UNIQUE (rep_id, period_id),

    -- Valor de comissão não pode ser negativo.
    CONSTRAINT chk_statements_amount CHECK (total_amount >= 0)
);

CREATE INDEX idx_statements_period_id ON statements(period_id);
CREATE INDEX idx_statements_rep_id ON statements(rep_id);
CREATE INDEX idx_statements_status ON statements(status);
