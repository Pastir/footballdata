-- Migration: 004_create_analytics_streaks.sql
-- Description: Справочники и основная таблица для хранения серий и рекордов команд, тренеров, игроков и H2H.

-- 1. Справочник типов серий с поддержкой многоязычной локализации (locale JSONB)
CREATE TABLE IF NOT EXISTS analytics_streak_types (
    code       VARCHAR(64) PRIMARY KEY,
    category   VARCHAR(32) NOT NULL,          -- 'team', 'coach', 'player', 'h2h'
    icon       VARCHAR(16) NOT NULL DEFAULT '⚽',
    locale     JSONB       NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Начальное наполнение типов серий
INSERT INTO analytics_streak_types (code, category, icon, locale) VALUES
  ('team_win', 'team', '🔥', '{
    "ru": {"short": "Победы", "long": "Победная серия", "description": "Победы команды подряд"},
    "en": {"short": "Wins", "long": "Winning Streak", "description": "Consecutive team victories"}
  }'::jsonb),
  ('team_unbeaten', 'team', '🛡️', '{
    "ru": {"short": "Непобедимость", "long": "Беспроигрышная серия", "description": "Матчи команды без поражений подряд"},
    "en": {"short": "Unbeaten", "long": "Unbeaten Streak", "description": "Consecutive matches without defeat"}
  }'::jsonb),
  ('team_no_win', 'team', '📉', '{
    "ru": {"short": "Без побед", "long": "Серия без побед", "description": "Матчи команды подряд без побед"},
    "en": {"short": "Winless", "long": "Winless Streak", "description": "Consecutive matches without winning"}
  }'::jsonb),
  ('team_scoring', 'team', '⚽', '{
    "ru": {"short": "Голы", "long": "Голевая серия", "description": "Матчи команды с забитыми голами подряд"},
    "en": {"short": "Scoring", "long": "Scoring Streak", "description": "Consecutive matches with goals scored"}
  }'::jsonb),
  ('team_clean_sheet', 'team', '🧤', '{
    "ru": {"short": "Сухие матчи", "long": "Сухая серия обороны", "description": "Матчи команды подряд без пропущенных голов"},
    "en": {"short": "Clean Sheets", "long": "Clean Sheet Streak", "description": "Consecutive matches without conceding"}
  }'::jsonb),
  ('coach_win', 'coach', '👔', '{
    "ru": {"short": "Победы тренера", "long": "Победная серия тренера", "description": "Победы подряд под руководством тренера"},
    "en": {"short": "Coach Wins", "long": "Manager Winning Streak", "description": "Consecutive wins under the manager"}
  }'::jsonb),
  ('coach_unbeaten', 'coach', '🛡️', '{
    "ru": {"short": "Тренер непобедим", "long": "Беспроигрышная серия тренера", "description": "Матчи подряд без поражений под руководством тренера"},
    "en": {"short": "Coach Unbeaten", "long": "Manager Unbeaten Streak", "description": "Consecutive matches without defeat under the manager"}
  }'::jsonb),
  ('player_goal', 'player', '🎯', '{
    "ru": {"short": "Голы игрока", "long": "Голевая серия игрока", "description": "Матчи подряд, в которых игрок забивал гол"},
    "en": {"short": "Goalscoring", "long": "Player Scoring Streak", "description": "Consecutive matches where player scored"}
  }'::jsonb),
  ('player_g_a', 'player', '⚡', '{
    "ru": {"short": "Гол+Пас", "long": "Серия результативности", "description": "Матчи подряд с результативными действиями (гол или ассист)"},
    "en": {"short": "G+A Streak", "long": "Goal Contribution Streak", "description": "Consecutive matches with a goal or assist"}
  }'::jsonb),
  ('h2h_unbeaten', 'h2h', '⚔️', '{
    "ru": {"short": "H2H Доминация", "long": "Серия личных встреч без поражений", "description": "Очные матчи между соперниками без поражений"},
    "en": {"short": "H2H Dominance", "long": "Head-to-Head Unbeaten", "description": "Consecutive head-to-head matches without defeat"}
  }'::jsonb),
  ('h2h_no_win', 'h2h', '🔒', '{
    "ru": {"short": "H2H Проклятие", "long": "Серия личных встреч без побед", "description": "Очные матчи, в которых команда не может победить соперника"},
    "en": {"short": "H2H Curse", "long": "Head-to-Head Winless Streak", "description": "Consecutive head-to-head matches without a win"}
  }'::jsonb)
ON CONFLICT (code) DO UPDATE SET
  category = EXCLUDED.category,
  icon = EXCLUDED.icon,
  locale = EXCLUDED.locale;

-- 2. Справочник областей действия серий (Scope) с поддержкой локализации
CREATE TABLE IF NOT EXISTS analytics_streak_scopes (
    code       VARCHAR(32) PRIMARY KEY,
    locale     JSONB       NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Начальное наполнение областей действия
INSERT INTO analytics_streak_scopes (code, locale) VALUES
  ('league', '{
    "ru": {"short": "Чемпионат", "long": "В чемпионате", "description": "Только матчи национального первенства (напр. АПЛ)"},
    "en": {"short": "League", "long": "In the League", "description": "Domestic league fixtures only (e.g. Premier League)"}
  }'::jsonb),
  ('all', '{
    "ru": {"short": "Все турниры", "long": "Во всех турнирах", "description": "Матчи с учётом кубков и еврокубков"},
    "en": {"short": "All Comps", "long": "All Competitions", "description": "Matches across all official competitions"}
  }'::jsonb),
  ('home', '{
    "ru": {"short": "Дома", "long": "На своём поле", "description": "Только домашние матчи (домашняя крепость)"},
    "en": {"short": "Home", "long": "At Home", "description": "Home fixtures only"}
  }'::jsonb),
  ('away', '{
    "ru": {"short": "В гостях", "long": "На выезде", "description": "Только выездные матчи"},
    "en": {"short": "Away", "long": "Away", "description": "Away fixtures only"}
  }'::jsonb)
ON CONFLICT (code) DO UPDATE SET
  locale = EXCLUDED.locale;

-- 3. Основная таблица серий и рекордов
CREATE TABLE IF NOT EXISTS analytics_streaks (
    id BIGSERIAL PRIMARY KEY,

    -- Контекст турнира
    league_id INT NOT NULL REFERENCES provider_leagues(id) ON UPDATE CASCADE ON DELETE CASCADE,

    -- Субъект серии
    entity_type         VARCHAR(32) NOT NULL, -- 'team', 'coach', 'player', 'h2h'
    entity_id           BIGINT      NOT NULL, -- team_id, coach_id или player_id
    secondary_entity_id BIGINT,               -- соперник для H2H дуэлей
    coach_id            INT REFERENCES provider_coachs(id) ON UPDATE CASCADE ON DELETE SET NULL, -- тренер во время серии

    -- Классификация
    streak_type VARCHAR(64) NOT NULL REFERENCES analytics_streak_types(code) ON UPDATE CASCADE,
    scope       VARCHAR(32) NOT NULL DEFAULT 'league' REFERENCES analytics_streak_scopes(code) ON UPDATE CASCADE,

    -- Показатели серии
    length           INT     NOT NULL,
    is_active        BOOLEAN NOT NULL DEFAULT true,
    is_record_club   BOOLEAN NOT NULL DEFAULT false,
    is_record_league BOOLEAN NOT NULL DEFAULT false,

    -- Точки отсчёта серии (старт)
    start_fixture_id BIGINT REFERENCES provider_fixtures(id) ON UPDATE CASCADE ON DELETE SET NULL,
    start_season_id  INT REFERENCES provider_seasons(id) ON UPDATE CASCADE ON DELETE SET NULL,
    start_round_id   INT,
    start_date       TIMESTAMPTZ NOT NULL,

    -- Точки окончания/текущего состояния серии
    end_fixture_id BIGINT REFERENCES provider_fixtures(id) ON UPDATE CASCADE ON DELETE SET NULL,
    end_season_id  INT REFERENCES provider_seasons(id) ON UPDATE CASCADE ON DELETE SET NULL,
    end_round_id   INT,
    end_date       TIMESTAMPTZ NOT NULL,

    -- Матч, прервавший серию (для завершённых серий)
    breaking_fixture_id BIGINT REFERENCES provider_fixtures(id) ON UPDATE CASCADE ON DELETE SET NULL,

    -- История матчей серии (для отрисовки кружков W-W-W и детального просмотра в Mini App)
    fixtures_history JSONB NOT NULL DEFAULT '[]'::jsonb,

    -- Дополнительные статистические агрегаты
    metrics JSONB NOT NULL DEFAULT '{}'::jsonb,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Индексы для мгновенных выборок
CREATE INDEX IF NOT EXISTS idx_streaks_active 
    ON analytics_streaks(league_id, is_active, streak_type, scope);

CREATE INDEX IF NOT EXISTS idx_streaks_entity_lookup
    ON analytics_streaks(entity_type, entity_id, streak_type, scope, is_active);

CREATE INDEX IF NOT EXISTS idx_streaks_club_records 
    ON analytics_streaks(entity_id, streak_type, scope, length DESC);

CREATE INDEX IF NOT EXISTS idx_streaks_league_records 
    ON analytics_streaks(league_id, streak_type, scope, length DESC);

CREATE INDEX IF NOT EXISTS idx_streaks_coach_lookup
    ON analytics_streaks(coach_id, streak_type, scope, length DESC)
    WHERE coach_id IS NOT NULL;
