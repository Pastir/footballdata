-- Migration: 002_create_predictions.sql
-- Description: Create tables and indexes for single & accumulator predictions, items, and settlement tracking.

CREATE TABLE IF NOT EXISTS predictions (
    id BIGSERIAL PRIMARY KEY,
    
    -- Тип ставки
    bet_type VARCHAR(30) NOT NULL,               -- 'single' (ординар) или 'accumulator' (экспресс)
    title VARCHAR(255) NOT NULL,                 -- e.g. "Двойник на субботний тур АПЛ"
    analysis_text TEXT,                          -- Аналитическое обоснование (сгенерировано AI)
    
    -- Финансово-математические показатели
    total_odds NUMERIC(8, 3) NOT NULL,           -- Итоговый коэффициент (произведение для экспресса)
    confidence NUMERIC(4, 2),                    -- Оценка вероятности модели (0.00 - 1.00)
    value_edge NUMERIC(5, 2),                    -- Положительный перевес над линией букмекера (%)
    recommended_stake NUMERIC(5, 2) DEFAULT 1.0, -- Рекомендуемый размер ставки в юнитах (e.g. 1.0u, 2.5u)
    
    -- Статус и расчёт исхода
    status VARCHAR(30) NOT NULL DEFAULT 'pending', 
    -- 'pending', 'won', 'lost', 'refund', 'half_won', 'half_lost', 'cancelled'
    
    profit_units NUMERIC(8, 3) DEFAULT 0.000,    -- Чистая прибыль в юнитах (+1.85u или -1.00u)
    settled_at TIMESTAMPTZ,                      -- Время закрытия прогноза
    
    -- Связь с публикацией
    publication_id BIGINT REFERENCES telegram_publications(id) ON DELETE SET NULL,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_predictions_status ON predictions(status);
CREATE INDEX IF NOT EXISTS idx_predictions_created ON predictions(created_at);

CREATE TABLE IF NOT EXISTS prediction_items (
    id BIGSERIAL PRIMARY KEY,
    prediction_id BIGINT NOT NULL REFERENCES predictions(id) ON DELETE CASCADE,
    
    -- Привязка к матчу Sportmonks
    fixture_id INT NOT NULL,
    league_id INT NOT NULL,
    home_team_id INT NOT NULL,
    away_team_id INT NOT NULL,
    match_starting_at TIMESTAMPTZ NOT NULL,
    
    -- Рынок и выбор ставки
    market_name VARCHAR(100) NOT NULL,           -- '1X2', 'Double Chance', 'Over/Under 2.5', 'BTTS', 'Handicap'
    market_key VARCHAR(50) NOT NULL,             -- Кодовое обозначение для автоматического матчинга
    selection VARCHAR(100) NOT NULL,            -- 'Home Win', 'Over 2.5', 'Yes', 'Away +1.0'
    odds NUMERIC(6, 3) NOT NULL,                -- Коэффициент на момент формирования
    bookmaker_name VARCHAR(50) DEFAULT 'Pinnacle',
    
    -- Фактический результат после свистка
    actual_score_ft VARCHAR(20),                 -- '2:1'
    actual_score_ht VARCHAR(20),                 -- '1:0'
    settlement_status VARCHAR(20) DEFAULT 'pending', -- 'pending', 'won', 'lost', 'refund'
    settlement_details TEXT,                     -- Причина расчёта: e.g. "Тотал голов 3 > 2.5 — Выигрыш"
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pred_items_fixture ON prediction_items(fixture_id);
CREATE INDEX IF NOT EXISTS idx_pred_items_status ON prediction_items(settlement_status);
