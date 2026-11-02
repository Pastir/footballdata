-- Migration: 005_create_press_articles.sql
-- Description: Таблица для дайджеста европейской прессы (RSS сбор, выжимки LLM, публикация в Telegram)

CREATE TABLE IF NOT EXISTS press_articles (
    id BIGSERIAL PRIMARY KEY,

    -- Источник и привязка к лиге
    source VARCHAR(64) NOT NULL,          -- 'bbc', 'theathletic', 'marca', 'bild', 'lequipe', 'gazzetta'
    league_id INT,                        -- Опционально: 8 (EPL), 564 (LaLiga), 82 (Bundesliga), 384 (Serie A), 301 (Ligue 1)

    -- Оригинальный материал
    original_url TEXT NOT NULL UNIQUE,    -- Уникальный URL (дедупликация)
    original_title TEXT NOT NULL,         -- Исходный заголовок
    original_lead TEXT,                   -- Исходный лид / анонс из RSS
    image_url TEXT,                       -- Ссылка на обложку / og:image

    -- Результат обработки LLM
    summary_ru TEXT,                      -- Готовая выжимка на русском для поста в Telegram

    -- Статусы и публикация
    status VARCHAR(32) NOT NULL DEFAULT 'draft', -- 'draft', 'ready', 'published', 'skipped'
    tg_message_id BIGINT,                 -- ID сообщения в Telegram-канале

    -- Временные метки
    published_at TIMESTAMPTZ,             -- Время публикации первоисточника
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Индексы для быстрой выборки новых/опубликованных статей
CREATE INDEX IF NOT EXISTS idx_press_articles_status
    ON press_articles (status);

CREATE INDEX IF NOT EXISTS idx_press_articles_published_at
    ON press_articles (published_at DESC);

CREATE INDEX IF NOT EXISTS idx_press_articles_source
    ON press_articles (source);

