-- Migration: 007_fix_provider_coachs_columns.sql
-- Description: Исправление перепутанных местами колонок player_id и sport_id в provider_coachs и удаление ошибочной записи с id = 0

DELETE FROM provider_coachs WHERE id = 0;

UPDATE provider_coachs
SET player_id = sport_id,
    sport_id = player_id,
    updated_at = NOW()
WHERE player_id = 1 AND sport_id <> 1;

