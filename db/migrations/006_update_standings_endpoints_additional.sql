-- Migration: 006_update_standings_endpoints_additional.sql
-- Description: Отключение неиспользуемого include=form в запросах турнирных таблиц (upsertStandings, upsertStandingsSeasons)

UPDATE provider_endpoints
SET additional = '{"include": "rule;details;group"}'::jsonb,
    updated_at = NOW()
WHERE task_name IN ('upsertStandings', 'upsertStandingsSeasons');

