-- Migration: 003_create_analytics_match_of_round.sql
-- Description: Таблица для кэширования результата выбора матча тура.
--              Позволяет не пересчитывать алгоритм при каждом запросе.

CREATE TABLE IF NOT EXISTS analytics_match_of_round (
    id BIGSERIAL PRIMARY KEY,

    -- Ключ уникальности: один матч тура на лигу
    league_id  INT NOT NULL,
    season_id  INT NOT NULL,
    round_id   INT NOT NULL,

    -- Результат выбора
    fixture_id      BIGINT      NOT NULL,
    team_home_name  VARCHAR(100) NOT NULL,
    team_away_name  VARCHAR(100) NOT NULL,

    -- Скоринг (для аудита и отладки алгоритма)
    strength_score   NUMERIC(6, 2),
    form_score       NUMERIC(6, 2),
    base_score       NUMERIC(6, 2),
    importance_score NUMERIC(6, 2),
    importance_bonus NUMERIC(6, 2),
    final_score      NUMERIC(6, 2),

    -- Причины (title race, relegation battle, etc.)
    reasons TEXT[] DEFAULT '{}',

    -- Входные параметры для воспроизводимости
    team_count         INT,
    total_team_matches INT,

    -- Метаданные
    computed_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    invalidated_at TIMESTAMPTZ,          -- NULL = актуально; NOT NULL = устаревший кэш

    CONSTRAINT uq_analytics_mor UNIQUE (league_id, season_id, round_id)
);

-- Основной индекс для GetCachedMatchOfRound
CREATE INDEX IF NOT EXISTS idx_analytics_mor_lookup
    ON analytics_match_of_round (league_id, round_id)
    WHERE invalidated_at IS NULL;

-- Индекс для очистки устаревших записей
CREATE INDEX IF NOT EXISTS idx_analytics_mor_invalidated
    ON analytics_match_of_round (invalidated_at)
    WHERE invalidated_at IS NOT NULL;
