-- Migration: 001_create_telegram_publications.sql
-- Description: Create table and indexes for tracking Telegram publication history, idempotency, and scheduling.

CREATE TABLE IF NOT EXISTS telegram_publications (
    id BIGSERIAL PRIMARY KEY,
    
    -- Контекст публикации
    channel_id BIGINT NOT NULL,                  -- ID целевого Telegram-канала / чата
    message_id BIGINT,                           -- ID опубликованного сообщения в Telegram (для reply/edit)
    publication_type VARCHAR(50) NOT NULL,       -- 'table', 'round_results', 'round_fixtures', 'prediction', 'express', 'insight'
    
    -- Привязка к футбольным сущностям
    league_id INT,                               -- ID лиги
    season_id INT,                               -- ID сезона
    round_id INT,                                -- ID тура (если применимо)
    fixture_id INT,                              -- ID матча (для одиночных постов)
    
    -- Контент
    title VARCHAR(255) NOT NULL,                 -- Внутренний заголовок публикации
    text_content TEXT,                           -- Итоговый текст поста (Markdown/HTML для Telegram)
    media_url TEXT,                              -- Путь к сгенерированному PNG на диске / S3
    media_type VARCHAR(20) DEFAULT 'photo',      -- 'photo', 'album', 'text'
    metadata JSONB DEFAULT '{}'::jsonb,          -- Доп. параметры (prompt LLM, токены, параметры рендера)
    
    -- Жизненный цикл и планирование
    status VARCHAR(30) NOT NULL DEFAULT 'draft', -- 'draft', 'scheduled', 'rendering', 'ready', 'published', 'failed', 'cancelled'
    scheduled_at TIMESTAMPTZ,                    -- Запланированное время публикации
    published_at TIMESTAMPTZ,                    -- Фактическое время публикации
    
    -- Ошибки и повторные попытки
    retry_count INT DEFAULT 0,
    error_message TEXT,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tg_pub_status_scheduled ON telegram_publications(status, scheduled_at);
CREATE INDEX IF NOT EXISTS idx_tg_pub_league_round ON telegram_publications(league_id, round_id);
CREATE INDEX IF NOT EXISTS idx_tg_pub_type ON telegram_publications(publication_type);
