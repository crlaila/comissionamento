-- Tabela de eventos de associados (vindos da API da Hinova).
-- Cada registro representa um evento: um novo associado captado
-- ou um associado que renovou.
--
-- Campo "hinova_id": o ID do evento no sistema da Hinova.
-- É UNIQUE porque usamos ele para DEDUPLICAÇÃO — quando o worker
-- de sync puxa dados da Hinova, ele pode receber o mesmo evento
-- mais de uma vez. O hinova_id garante que não criamos duplicatas.
--
-- Campo "event_type":
--   - acquisition: novo associado captado pelo vendedor
--   - renewal: associado existente renovou com o vendedor
--
-- Campo "rep_id": o vendedor responsável pelo evento.
-- Este evento conta para a meta dele.

CREATE TYPE event_type AS ENUM ('acquisition', 'renewal');

CREATE TABLE member_events (
    id          BIGSERIAL   PRIMARY KEY,
    hinova_id   VARCHAR(100) NOT NULL UNIQUE,  -- ID externo para deduplicação
    rep_id      BIGINT       NOT NULL REFERENCES users(id),
    event_type  event_type   NOT NULL,
    member_name VARCHAR(255) NOT NULL,
    event_date  DATE         NOT NULL,
    synced_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),  -- quando foi importado da Hinova
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- Índice para contar eventos por vendedor e tipo (usado no cálculo de comissão).
CREATE INDEX idx_member_events_rep_type ON member_events(rep_id, event_type);

-- Índice para filtrar eventos por data (usado para associar ao período).
CREATE INDEX idx_member_events_date ON member_events(event_date);

-- Índice para busca por hinova_id (usado na deduplicação do sync).
CREATE INDEX idx_member_events_hinova_id ON member_events(hinova_id);
