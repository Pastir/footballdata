-- 008_add_live_standings_endpoint.sql
-- Добавление эндпоинта live-таблицы Sportmonks (standings/live/leagues/{leagueID})
-- и актуализация параметров UpsertFixturesLive (удаление дубля lineups и фильтр по активным лигам).

INSERT INTO provider_endpoints (provider_id, endpoint, name, task_name, additional, update_frequency, cycle)
SELECT 2,
       'https://api.sportmonks.com/v3/football/standings/live/leagues/{leagueID}',
       'standingsLive',
       'upsertStandingsLive',
       '{"include": "rule;details;group"}'::jsonb,
       60,
       'leagues'
WHERE NOT EXISTS (
    SELECT 1 FROM provider_endpoints WHERE provider_id = 2 AND task_name = 'upsertStandingsLive'
);

INSERT INTO provider_endpoints (provider_id, endpoint, name, task_name, additional, update_frequency, cycle)
SELECT 2,
       'https://api.sportmonks.com/v3/football/livescores',
       'livescores',
       'UpsertFixturesLive',
       '{"include": "lineups.details;scores;participants;events;timeline;periods;statistics;metadata;state", "filters": "fixtureLeagues:8,82,301,384,564"}'::jsonb,
       60,
       NULL
WHERE NOT EXISTS (
    SELECT 1 FROM provider_endpoints WHERE provider_id = 2 AND task_name = 'UpsertFixturesLive'
);

UPDATE provider_endpoints
SET additional = '{"include": "lineups.details;scores;participants;events;timeline;periods;statistics;metadata;state", "filters": "fixtureLeagues:8,82,301,384,564"}'::jsonb
WHERE provider_id = 2 AND task_name = 'UpsertFixturesLive';

