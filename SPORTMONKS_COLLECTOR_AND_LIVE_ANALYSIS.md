# Полный анализ подсистемы сбора данных Sportmonks (`api/handlers/*`) и архитектура Live-режима

**Дата анализа:** 2026-11-11  
**Объект аудита:** Пакет `api/handlers/*` (26 файлов), связанные зависимости (`client`, `db`, `db/queries`, `config`, `logs`, `api/statistics`, `main.go`), схема хранения `provider_*` в PostgreSQL и продакшен-скрипты `/opt/apisportmonks/cron/*.sh`.  
**Цель:** Оценить текущую архитектуру сбора и хранения данных, качество кода и спроектировать надёжный **Live-сбор данных** со Sportmonks API v3.

---

## 1. Архитектура текущего сбора данных и граф зависимостей

### 1.1. Поток данных (Data Flow)
Сейчас сбор данных со Sportmonks работает как **CLI-приложение пакетной загрузки (Batch Collector)**, собираемое из `main.go` в бинарник `prematch` и вызываемое по расписанию из системного `crontab`:

```text
Crontab (/opt/apisportmonks/cron/*.sh)
  │
  ▼
main.go (бинарник ./prematch -conf config.yaml -task <TaskName> ...)
  ├── ManageServer.ServiceWatcher инициализирует:
  │     1. config.Config   (чтение config.yaml + CLI-флагов)
  │     2. logs.Logs       (zap-логгеры access / error)
  │     3. db.DBx          (пул соединений pgxpool.Pool к PostgreSQL)
  │     4. db.Prov         (загрузка настроек провайдера из БД: таблицы provider и provider_endpoints)
  │
  ▼
handler.EndPoint(taskData) [api/handlers/endpoint.go]
  ├── Определяет цикл итерации (Cycle) из provider_endpoints:
  │     • "seasons"       → db/queries.GetSeasons()
  │     • "rounds"        → db/queries.GetLast3CurrentSeasonRoundsIDs()
  │     • "seasons,teams" → GetSeasons() × GetTeamsIdFromSeasons()
  │     • "date"          → {start_date}, {end_date}, {date}
  │     • "fixtures" / "players" / "coaches" / "teamsID" / "countries" / "stage"
  │     • "" (пустой)     → одиночный вызов RequestEndpoint()
  │
  ▼
handler.RequestEndpoint(taskData) [api/handlers/endpoint.go]
  ├── Формирует URL из шаблона provider_endpoints.endpoint + include + filters + api_token
  ├── Цикл пагинации (page=1..N):
  │     1. client.RequestToEndpoint(url) — HTTP GET (с поддержкой gzip / brotli и ожиданием при 429)
  │     2. fastjson.Parser.ParseBytes(resp) — парсинг JSON без аллокации Go-структур
  │     3. Вызов обработчика из мапы Handlers[endpointData.TaskName](dv, taskData)
  │     4. RateLimitHandler() — проверка rate_limit.remaining (sleep при 0)
  │     5. PaginationHandler() — переход на следующую страницу при pagination.has_more == true
  │
  ▼
TaskLastUpdate(taskData)
  └── UPDATE provider_endpoints SET updated_at = NOW() WHERE task_name = $1
```

---

### 1.2. Карта зависимостей пакета `api/handlers/*`

| Модуль / Пакет | От чего зависит | Роль в системе сбора |
|---|---|---|
| `main.go` | `config`, `logs`, `db`, `api/handlers`, `api/statistics` | Точка входа коллектора (`./prematch`). Инициализирует сервисы и маршрутизирует `-task` в `handler.EndPoint` (либо в `UpsertHead2Head` / `PrematchStatistics`). |
| `db/init_data_provider.go` (`db.Prov`) | `db.DBx`, `config` | При старте читает из БД таблицу `provider` (токен `api_token` в `settings->'get_params'`), таблицу `provider_endpoints` (46 маршрутов с шаблонами URL, `include`, `filters`, `cycle`) и справочник стран `provider_countries`. |
| `client/request.go` | `net/http`, `cbrotli`, `compress/gzip`, `logs` | Выполняет HTTP-запросы к `api.sportmonks.com/v3/...`, распаковывает Brotli/Gzip, обрабатывает `HTTP 429 Too Many Requests`. |
| `db/func.go` | `reflect`, `github.com/lib/pq` | Содержит функцию `db.CastInterfaceString(i interface{})`, которая через рефлексию и `pq.QuoteLiteral` превращает значения из JSON в строковые литералы для конкатенации сырых SQL-запросов `INSERT ... VALUES (...)`. |
| `db/queries/*` | `db.DBx`, `analytics` | Вспомогательные выборки ID для циклов обхода: активные сезоны (`GetSeasons`), последние 3 тура (`GetLast3CurrentSeasonRoundsIDs`), команды сезона (`GetTeamsIdFromSeasons`), страны (`GetListCountriesList`). |
| `api/handlers/function.go` | `db.DBx`, `regexp`, `fastjson` | Реестр `Handlers` (мапа `task_name -> func`) и механизм авторазрешения ошибок внешних ключей `HandlingViolatesFKError` (код PostgreSQL `23503`). |

---

### 1.3. Пофайловый аудит `api/handlers/*` (все 26 файлов)

1. **`endpoint.go` (257 строк)** — Ядро диспетчеризации запросов:
   * Управляет циклами (`seasons`, `rounds`, `date` и др.), собирает GET-параметры (`include`, `filters`, `locale`, `page`), вызывает `client.RequestToEndpoint`, контролирует `rate_limit` и `pagination`.
   * **Проблемы:**
     * [x] **ИСПРАВЛЕНО:** На строке 169 при любой ошибке HTTP вызывался `os.Exit(0)` — заменено на `return`, чтобы не убивать процесс при циклических обходах и будущем Live-режиме.
     * [x] **ИСПРАВЛЕНО:** В `replaceToken` (строки 215–254) убран двойной вложенный цикл по мапе `replaceTokens`, опасные приведения типов и отладочный `fmt.Println("AAAAAA")`. Покрыто unit-тестом `TestReplaceToken` в `api/handlers/endpoint_test.go`.
2. **`function.go` (118 строк)** — Карта `Handlers` (36 задач) и `HandlingViolatesFKError`:
   * **Фича:** Если при вставке (например, состава или статистики) в БД нет нового игрока (`provider_players`), тренера (`provider_coachs`), команды (`provider_teams`) или матча (`provider_fixtures`), `HandlingViolatesFKError` парсит текст ошибки `pgErr.Detail` регуляркой, рекурсивно вызывает `EndPoint()` для докачки недостающего ID по API и повторяет `INSERT`.
   * [x] **ИСПРАВЛЕНО (Критический баг):** В `fixtures.go` на строках **543 (`UpsertFixturePlayersStatistics`) и 733 (`UpsertFixtureParticipants`)**, а также в `players.go` на строке **122 (`UpsertPlayersSeasons`)** в `HandlingViolatesFKError(err, query)` передавалась переменная `query` (шапка `INSERT INTO ... VALUES ` без данных!) вместо собранного запроса `queryItog` / `queryInsert`. Исправлено на передачу `queryItog` и `queryInsert`.
3. **`fixtures.go` (900 строк)** — Главный обработчик матчей и 12 связанных сущностей:
   * Содержит функции вставки в 12 таблиц:
     * `UpsertFixturesId` / `UpsertFixtures` $\rightarrow$ `provider_fixtures`
     * `UpsertFixtureScores` $\rightarrow$ `provider_fixtures_scores`
     * `UpsertFixtureLineups` $\rightarrow$ `provider_lineups` (партиционирована по `league_id`)
     * `UpsertFixtureStatistics` $\rightarrow$ `provider_fixtures_statistics` (партиционирована по `league_id`)
     * `UpsertFixtureWeatherReport` $\rightarrow$ `provider_weather_report`
     * `UpsertFixtureReferees` $\rightarrow$ `provider_fixtures_referees`
     * `UpsertFixturePeriods` $\rightarrow$ `provider_fixtures_periods`
     * `UpsertFixturePeriodsStatistics` $\rightarrow$ `provider_fixtures_periods_statistics`
     * `UpsertFixtureMeta` $\rightarrow$ `provider_fixtures_meta`
     * `UpsertFixturePlayersStatistics` $\rightarrow$ `provider_fixtures_players_statistics` (партиционирована по `league_id`)
     * `UpsertFixtureEvents` $\rightarrow$ `provider_events`
     * `UpsertFixtureTimeLines` $\rightarrow$ `provider_timelines`
     * `UpsertFixtureParticipants` $\rightarrow$ `provider_fixtures_participants`
     * `UpsertFixtureCoaches` $\rightarrow$ `provider_fixtures_coaches`
   * Диспетчеры: `UpsertFixturesSeasons` (вложенный массив `data.fixtures`) и `UpsertFixturesBetweenDate` (корневой массив `data`).
   * **Найденные баги в `fixtures.go`:**
     * [x] **ИСПРАВЛЕНО:** **Строка 410 (`UpsertFixturePeriods`):** читает `fx.GetBool("ticking")` у объекта матча вместо `v.GetBool("ticking")` у самого периода (`v`)! Из-за этого в БД флаг `ticking` (идёт ли сейчас тайм) всегда записывается как `false`! Для Live-режима это критично, так как именно `v.GetBool("ticking")` и `v.GetInt("minutes")` показывают текущую минуту живого матча.
     * [x] **ИСПРАВЛЕНО:** **Строка 875 (`UpsertFixturesBetweenDate`):** в комментарии указано `-include .metadata` (с точкой) $\rightarrow$ исправлено на `-include metadata`.
     * [x] **ИСПРАВЛЕНО:** **Отсутствие `UpsertFixtureParticipants` и `UpsertFixtureCoaches` в `default:` ветке** (строки 825–839 и 889–903): добавлены вызовы `UpsertFixtureParticipants` и `UpsertFixtureCoaches` в `default:` обеих функций (`UpsertFixturesSeasons` и `UpsertFixturesBetweenDate`).
4. **`standings.go` (196 строк)** — Турнирные таблицы:
   * Заполняет `provider_standings`, `provider_standings_details` (с предварительным `DELETE ... WHERE standing_id IN (...)`), `provider_groups`, `provider_standings_position_rules`.
   * [x] **ИСПРАВЛЕНО:** Удалены 5 неиспользуемых функций (`GetFinishedRoundsOfSeason`, `GetPrevRounds`, `GetCurrentRounds`, `GetStageIdFromRounds`, `GetRoundsIdFromSeasonId`), исправлена метка лога в `UpsertStandingsDetails`, отключён неиспользуемый сбор `provider_standings_forms` (`UpsertStandingsForms` и `;form` в `provider_endpoints`).
5. **`odds.go` (78 строк)** — Коэффициенты букмекеров (`provider_odds`):
   * [x] **ИСПРАВЛЕНО:** Добавлена буферная переменная `qi` и проверка `if strings.Compare(qi, "") != 0` перед выполнением SQL-запроса (ранее при пустом массиве `odds` в PostgreSQL отправлялся невалидный запрос `"INSERT INTO provider_odds(...) VALUES  ON CONFLICT..."`).
6. **`expected.go` (48 строк)** — Ожидаемые голы xG (`provider_expected_fixtures`, партиционирована по `league_id`).
7. **`topscorers.go` (70 строк)** — Бомбардиры и ассистенты сезона (`provider_seasons_topscores`):
   * [x] **ИСПРАВЛЕНО:** Добавлена проверка `if len(data) == 0`, дедупликация в памяти по ключу `(season_id, player_id, type_id)` внутри батча и вставка через `SELECT ... FROM (VALUES ...) WHERE NOT EXISTS (...)`, исключающая дублирование игроков на стыке страниц пагинации и при повторном вызове из `HandlingViolatesFKError`.
8. **`players.go` (255 строк)** — Игроки и их сезонная статистика (`provider_players`, `provider_statistics_players_seasons`):
   * [x] **ИСПРАВЛЕНО:** На строке 122 исправлена передача `queryItog` вместо шапки `query` в `HandlingViolatesFKError(err, queryItog)` (на строке 249 `query` перезаписывается и передаётся корректно).
9. **`coachs.go` (290 строк)** — Тренеры (`provider_coachs`, `provider_teams_coaches`):
   * **Два критических бага в `UpsertCoaches` / `UpsertCoachesTeamsSeasons` / `UpsertCoachesIds` / `UpsertCoachesFixturesErrorFK`:**
     1. [x] **ИСПРАВЛЕНО:** **Перепутаны местами колонки `player_id` и `sport_id`!** В шапке SQL написано: `INSERT INTO provider_coachs(id, player_id, sport_id, ...)`, а в `VALUES` ранее передавался сначала `v.GetInt("sport_id")`, а вторым — `v.GetInt("player_id")` (во всех 4 функциях). Теперь порядок значений приведён в полное соответствие со списком колонок (`player_id`, затем `sport_id`).
     2. [x] **ИСПРАВЛЕНО:** В `UpsertCoachesIds` данные теперь читаются из `v := d.Get("data")` (с защитой `if v == nil || v.GetInt("id") == 0 { return }`), а не из корня `d.GetInt("id")`.
10. **`teams.go` (159 строк)** — Команды и командная статистика (`provider_teams`, `provider_statistics_teams_season`, `TranslateTeams`). (На строке 95 `query` перезаписывается строкой 90, поэтому передаётся полный запрос).
11. **`seasons.go` (124 строки)** — Сезоны (`provider_seasons`, `provider_statistics_seasons`, `provider_seasons_teams`).
    * [x] **ИСПРАВЛЕНО:** В `UpsertSeasons` добавлена буферная переменная `qi` и проверка `if strings.Compare(qi, "") != 0` перед `Exec`.
12. **`stages.go` (153 строки)** — Стадии турниров (`provider_stages`, `provider_stages_name`, `provider_statistics_stages`).
13. **`rounds.go` (67 строк)** — Туры (`provider_rounds`).
14. **`leagues.go` (62 строки)** — Лиги (`provider_leagues`).
    * [x] **ИСПРАВЛЕНО:** Добавлена буферная переменная `qi`, проверка `if strings.Compare(qi, "") != 0` и исправлена метка лога `"UpsertLeagues"`.
15. **`countries.go`, `continents.go`, `bookmakers.go`, `markets.go`, `referees.go`, `rivals.go`, `states.go`, `types.go`, `venues.go`** — Справочники.
    * [x] **ИСПРАВЛЕНО:** В `venues.go` убран старый внешний логгер `github.com/Pastir/logger` в пользу `db.PGxConnError`, поштучный `INSERT` в цикле заменён на батчевую вставку, `national_team` читается через `v.GetBool("national_team")`. В `countries.go`, `bookmakers.go`, `markets.go`, `rivals.go`, `types.go` добавлена проверка `if strings.Compare(qi, "") != 0` (а в `countries.go` также исправлен порядок `longitude, latitude`).
16. **`meta.go` (132 строки)** — Обработчики `PaginationHandler` и `RateLimitHandler`.
17. **`head2head.go` (10 строк)** и **`tasks.go` (6 строк)** — [x] **УДАЛЕНЫ:** ненужные файлы-заглушки удалены из `api/handlers/`.

---

## 2. Оценка архитектуры сбора и хранения данных

### 2.1. Сильные стороны текущей архитектуры
1. **Высокая скорость парсинга (`valyala/fastjson`):** Использование `fastjson` без промежуточного `json.Unmarshal` в тысячи Go-структур позволяет парсить многомегабайтные ответы Sportmonks с минимальной нагрузкой на Garbage Collector.
2. **Конфигурируемость через БД (`provider_endpoints`):** Шаблоны URL, наборы `include` и `filters` хранятся в таблице `provider_endpoints` и могут корректироваться без перекомпиляции.
3. **Идемпотентность на уровне БД (`ON CONFLICT DO UPDATE`):** Почти все таблицы имеют уникальные ключи и корректно обновляют `updated_at = NOW()` при повторной заливке.
4. **Автоматическое разрешение внешних ключей (`HandlingViolatesFKError`):** Отличная архитектурная идея — не отключать FK-констрейнты в БД, а автоматически докачивать появившегося молодого игрока или нового тренера при ошибке `23503`.

---

### 2.2. Архитектурные узкие места и проблемы хранения

#### 1. Разделение тяжёлых `include` в `upsert_match_beetween_date.sh` (Защита от таймаутов Sportmonks)
В скрипте `/opt/apisportmonks/cron/upsert_match_beetween_date.sh` вызов `./prematch -task upsertFixturesBetweenDate` разбит на отдельные проходы по каждому `include` (`scores`, `participants`, `coaches`, `events`, `timeline`, `lineups`, `statistics`, `weatherReport`, `referees`, `periods`, `metadata`, `lineups.details`).
* **Причина (подтверждено на практике):** При объединении всех `include` (особенно сверхтяжёлого `lineups.details`, где на каждый матч приходится ~600 объектов статистики игроков, и `timeline`) на окне в 24 дня бэкенд Sportmonks упирается в лимит сложности запроса (Query Complexity / Gateway Timeout 504) и рвёт соединение.
* **Вывод для архитектуры (в том числе для Live!):** Нельзя складывать `lineups.details` и `timeline` в один общий тяжёлый запрос с оперативными данными (`scores`, `periods`, `events`). Группировать можно только лёгкие сущности (например, `scores;participants;periods;state`), а тяжёлые (`lineups.details`, `timeline`, `statistics`) обязательно запрашивать отдельно.

#### 2. Опечатка в продакшен-скрипте `/opt/apisportmonks/cron/odds_every_day.sh`
В файле `/opt/apisportmonks/cron/odds_every_day.sh` на строке 3 написано:
```bash
WORK_DIR="/opt/apisportmonks/footbaldata" # пропущена буква 'l'!
```
Из-за этого `cd $WORK_DIR` падает, и `./prematch` не находит директорию при запуске из `/opt/apisportmonks/cron`!

#### 3. Партиционирование таблиц по `league_id` без `DEFAULT` партиции
Таблицы `provider_lineups`, `provider_fixtures_statistics`, `provider_fixtures_players_statistics`, `provider_expected_fixtures` разбиты на 29 партиций (`LIST (league_id)`), но **не имеют `DEFAULT` партиции**.
* Сейчас в `provider_leagues` активны 5 топ-лиг (`8, 82, 301, 384, 564`), для которых партиции есть.
* Но если вызвать `livescores` или `fixtures/between` без фильтра по лигам, и в выдачу попадёт кубковый матч или новая лига из подписки Sportmonks, чей `league_id` не входит в 29 партиций, PostgreSQL отменит весь батч `INSERT` с ошибкой `no partition of relation found for row`.
* **Решение:** Создать `DEFAULT`-партиции для этих 4 таблиц ИЛИ всегда передавать фильтр `filters=fixtureLeagues:8,82,301,384,564` при запросах.

#### 4. Формирование SQL через конкатенацию строк (`db.CastInterfaceString`)
Все хендлеры клеят одну гигантскую SQL-строку через `qi += fmt.Sprintf(...)`.
* `pq.QuoteLiteral` защищает от SQL-инъекций, но на больших выборках конкатенация `+=` в цикле создаёт квадратичную аллокацию памяти в Go (`O(N^2)` по памяти, вместо `strings.Builder` или `pgx.Batch` / `CopyFrom`).
* Кроме того, в `db/func.go:L41` для `*bool` при значении `true` возвращается `pq.QuoteLiteral(...) + "'t'"`, что даёт невалидный SQL `''true'''t'`.

#### 5. Два отдельных бинарника (`prematch` vs `lab`)
Исторически проект разделился на:
* `main.go` ($\rightarrow$ бинарник `prematch`) — старый коллектор Sportmonks;
* `cmd/lab/main.go` ($\rightarrow$ бинарник `lab`) — аналитика, рендеринг, Telegram-паблишер, планировщик и REST API сервер Mini App.
При переходе к Live-режиму это разделение мешает: `fdl-api` (работающий внутри `lab`) уже держит пул соединений с БД и `LivePoller`, но не может сам напрямую вызвать сбор live-данных со Sportmonks.

---

## 3. Проектирование Live-сбора данных со Sportmonks API v3

### 3.1. Возможности Sportmonks v3 для Live-режима (по документации `docs.sportmonks.com/v3`)

Sportmonks Football API v3 предоставляет 4 специализированных эндпоинта для реального времени:

| Эндпоинт Sportmonks v3 | Что возвращает | Когда использовать |
|---|---|---|
| `GET /v3/football/livescores/inplay` | Только матчи, которые **прямо сейчас идут на поле** (`INPLAY_1ST_HALF`, `HT`, `INPLAY_2ND_HALF`, `INPLAY_ET`, `INPLAY_PENALTIES`). | Основной высокочастотный поллинг во время матчей (каждые 15–30 сек). |
| `GET /v3/football/livescores` | Все матчи в окне **от 15 минут до стартового свистка до 15 минут после финального свистка (`FT`)**. | Идеально для захвата: 1) стартовых составов за 15 мин до игры, 2) момента перехода матча в `FT` (финальный свисток!). |
| `GET /v3/football/livescores/latest` | Только матчи, в которых **произошли изменения за последние 10 секунд**. | Сверхлёгкий дельта-поллинг (при опросе раз в 8–10 сек), но если пропустить окно в 10 сек — изменение не попадёт в ответ. |
| `GET /v3/football/standings/live/leagues/{league_id}` | Живая турнирная таблица лиги в реальном времени. | Опционально; но **собственный расчёт Live-таблицы на базе снапшота + текущих счетов** быстрее, не тратит лимиты API и даёт 100% консистентность со счётом на экране. |

#### Разделение `include` для `/v3/football/livescores` на «Быстрый контур» и «Тяжёлый контур»:
Учитывая, что Sportmonks тормозит и падает по таймауту при смешивании `lineups.details` и `timeline` с остальными данными (в субботу вечером одновременно идут 15–20 матчей!), в таблице `provider_endpoints` сейчас для `livescores` прописан слишком тяжёлый набор:
`lineups.details;scores;participants;lineups;events;timeline;periods;statistics;metadata;state` (причём `lineups` продублирован дважды).

Для безотказной работы в Live нужно разделить опрос на **2 контура**:
1. **Быстрый контур (Fast Live Tick — каждые 20–30 сек):**
   ```text
   include=scores;participants;periods;events;state
   ```
   * Вес ответа минимальный (несколько килобайт), время ответа Sportmonks $< 200\text{ мс}$.
   * Мгновенно доставляет главное: **голы, текущий счёт, минуту матча (`periods`), карточки, замены и финальный свисток (`FT`)**.
2. **Тяжёлый контур (Heavy Live / Post-Match Tick — раз в 3–5 минут + один раз при `HT` и `FT`):**
   * Запрос командной статистики: `include=statistics`
   * Запрос составов и статистики игроков: `include=lineups.details` (или только `include=lineups` за 15 мин до старта, а `lineups.details` — в перерыве и по `FT`).

#### Поддерживаемые `filters` для `/v3/football/livescores`:
Чтобы не тянуть лишние лиги, дополнительно ускорить ответ и не падать на партициях:
```text
filters=fixtureLeagues:8,82,301,384,564
```

---

### 3.2. Что уже подготовлено в базе данных проекта

В таблице `provider_endpoints` уже создана строка (провайдер `2`):
* **`name`**: `livescores`
* **`task_name`**: `UpsertFixturesLive`
* **`endpoint`**: `https://api.sportmonks.com/v3/football/livescores`
* **`additional`**: `{"include": "lineups.details;scores;participants;lineups;events;timeline;periods;statistics;metadata;state"}`
* **`update_frequency`**: `60`

Чего не хватает в коде прямо сейчас:
1. В `api/handlers/function.go` в мапе `Handlers` нет ключа `"UpsertFixturesLive"`.
2. В `api/handlers/fixtures.go` нужно реализовать функцию `UpsertFixturesLive(d *fastjson.Value, taskData config.TaskData)`.
3. Нужно исправить баг `fx.GetBool("ticking")` $\rightarrow$ `v.GetBool("ticking")` в `UpsertFixturePeriods` (`fixtures.go:L410`), иначе минуты живого матча и статус идущего тайма не сохраняются корректно.
4. Нужно учесть **отмену голов / событий после VAR** в Live-режиме!

---

### 3.3. Важный нюанс Live-данных: проблема VAR и отменённых голов

В пакетном режиме (после матча) все события финальные. Но в **Live-режиме** происходит следующее:
1. На 65-й минуте забивается гол $\rightarrow$ Sportmonks присылает событие `id = 999123` в массиве `events` и счёт `2:1` в `scores`. Мы делаем `INSERT INTO provider_events ... ON CONFLICT DO UPDATE`.
2. На 67-й минуте **VAR отменяет гол** $\rightarrow$ Sportmonks **удаляет** событие `999123` из массива `events` и возвращает счёт `1:1` в `scores`.
3. Если мы делаем только `INSERT ... ON CONFLICT DO UPDATE`, то счёт в `provider_fixtures_scores` обновится обратно на `1:1`, а **отменённый гол `999123` навсегда останется висеть в таблице `provider_events` и `provider_timelines`**!

**Как решить проблему VAR в `UpsertFixturesLive`:**
Для каждого активного `fixture_id`, пришедшего в `livescores`, перед (или в рамках транзакции) вставкой `events` и `timeline` удалять старые события этого матча, которых больше нет в актуальном ответе (либо очищать `DELETE FROM provider_events WHERE fixture_id IN (...)` и вставлять актуальный массив заново — для 5–10 живых матчей это занимает $< 1$ мс!).

---

### 3.4. Реализация `UpsertFixturesLive` в `api/handlers/fixtures.go`

Поскольку структура JSON-ответа `/v3/football/livescores` идентична `/v3/football/fixtures/between/{start}/{end}` (корневой массив `data` содержит объекты матчей с вложенными `scores`, `participants`, `periods`, `events`, `timeline`, `statistics`, `lineups`), функция `UpsertFixturesLive` переиспользует все существующие функции `fixtures.go`:

```go
// UpsertFixturesLive обрабатывает live-матчи из эндпоинта /v3/football/livescores
func UpsertFixturesLive(d *fastjson.Value, taskData config.TaskData) {
    data := d.Get("data")
    if data == nil || len(data.GetArray()) == 0 {
        return
    }

    // 1. Обновляем саму запись матча (state_id меняется: NS -> 1st -> HT -> 2nd -> FT)
    UpsertFixtures(taskData.Name(), data)

    // 2. Обновляем участников (winner / result по мере завершения)
    UpsertFixtureParticipants(taskData.Name(), data)

    // 3. Обновляем текущий счёт и счёт по таймам (CURRENT, 1ST_HALF, 2ND_HALF)
    UpsertFixtureScores(taskData.Name(), data)

    // 4. Обновляем периоды и текущую минуту матча (ticking, minutes, time_added)
    UpsertFixturePeriods(taskData.Name(), data)

    // 5. Синхронизируем события и таймлайн (с защитой от отменённых VAR голов)
    UpsertFixtureEventsLive(taskData.Name(), data)
    UpsertFixtureTimeLines(taskData.Name(), data)

    // 6. Обновляем командную статистику (удары, владение, угловые, сейвы)
    UpsertFixtureStatistics(taskData.Name(), data)

    // 7. Обновляем составы и оценки/статистику игроков
    UpsertFixtureLineups(taskData.Name(), data)
    UpsertFixturePlayersStatistics(taskData.Name(), data)
    UpsertFixtureMeta(taskData.Name(), data)
}
```

---

### 3.5. Архитектура запуска: «Умный Live-коллектор» (Smart Live Polling)

Тупо дергать Sportmonks API каждые 20 секунд 24/7 нельзя и бессмысленно:
* В будние дни с 00:00 до 18:00 UTC матчей в Топ-5 лигах вообще нет (мы просто сожжём лимит запросов впустую).
* Наоборот, в субботу с 14:00 до 22:00 UTC одновременно идут по 15–20 матчей, и данные нужны каждые **20–30 секунд**.

#### Алгоритм работы Умного Live-Коллектора (внутри `lab` / `fdl-api`):

Мы уже знаем расписание всех матчей в таблице `provider_fixtures`!
Поэтому перед тем, как делать внешний HTTP-запрос в Sportmonks, воркер делает бесплатный локальный SQL-запрос к нашей БД (раз в 30 секунд):

```sql
SELECT COUNT(*)
FROM provider_fixtures pf
JOIN provider_leagues pl ON pl.id = pf.league_id
WHERE pl.active = true
  AND (
      -- 1. Матч должен начаться в ближайшие 15 минут или уже начался в последние 3 часа,
      --    и ещё не перешёл в финальный статус (FT=5, AET=6, FT_PEN=7, POSTP=10, CANC=12)
      (pf.starting_at BETWEEN NOW() AT TIME ZONE 'UTC' - INTERVAL '150 minutes'
                          AND NOW() AT TIME ZONE 'UTC' + INTERVAL '15 minutes'
       AND COALESCE(pf.state_id, 1) NOT IN (5, 6, 7, 10, 11, 12, 13, 14, 15, 17))
      OR
      -- 2. Или числится в in-play статусе (1st, HT, 2nd, ET, PEN и т.д.)
      pf.state_id IN (2, 3, 4, 9, 21, 22, 23, 25)
  );
```

1. **Если `COUNT(*) == 0` (режим сна / Idle Mode):**
   * Ни один матч сейчас не идёт и не начинается в ближайшие 15 минут.
   * **0 запросов к Sportmonks API!**
2. **Если `COUNT(*) > 0` (режим Live / Active Window):**
   * Вызывается `https://api.sportmonks.com/v3/football/livescores?filters=fixtureLeagues:8,82,301,384,564&include=...` (1 запрос сразу возвращает все идущие матчи всех 5 топ-лиг!).
   * Данные сохраняются в БД через `UpsertFixturesLive`.
   * Наш `LivePoller` мгновенно подхватывает обновлённые счета и события в `LiveStore` для Mini App.

#### Расход лимитов Sportmonks API при таком подходе:
* В Sportmonks стандартный лимит — **3 000 запросов в час** на эндпоинт.
* При опросе раз в **30 секунд** только во время активных матчей тратится всего **120 запросов в час** (4% от лимита!), а в часы без матчей — **0 запросов**.
* При этом 1 запрос `/v3/football/livescores` покрывает сразу все 5 лиг (так как в один момент времени одновременно идёт максимум 15–25 матчей, что помещается на 1 страницу пагинации без доп. запросов!).

---

### 3.6. Единая цепочка событий (Event Triggers) от Live-коллектора

Когда Умный Live-коллектор получает свежий пакет из `/v3/football/livescores`, он сравнивает старый и новый статус/счёт каждого матча в памяти:

```text
[Пакет из Sportmonks /v3/football/livescores (раз в 30 сек)]
        │
        ├──► 1. Сменился счёт (Гол!) или минута
        │       └──► Обновляем БД + LiveStore в памяти Mini App
        │       └──► Пересчитываем Виртуальную Live-таблицу лиги (базовый снапшот + текущие очки live-матчей)
        │
        ├──► 2. Матч перешёл в статус FT (state_id == 5, Финальный свисток!)
        │       └──► Докачиваем xG матча (upsertExpectedFixtures для даты матча)
        │       └──► Фиксируем очки матча в официальном снапшоте турнирной таблицы тура
        │       └──► Проверяем матч в analytics.DetectMatchAnomalies:
        │            если SeverityScore >= 90 (суперкамбэк 90+, дикое xG-ограбление) —
        │            можно сразу генерировать карточку по горячим следам!
        │
        └──► 3. Все матчи тура перешли в FT (Тур полностью завершён!)
                └──► Замораживаем таблицу тура N (is_final = true)
                └──► Клонируем таблицу в тур N+1 (чтобы следующий тур сразу имел идеальную таблицу)
                └──► Ставим в очередь утренний отчёт тура на 07:00 UTC
```

---

## 4. Резюме и пошаговый план действий

### Оценка текущей системы сбора данных: **7.0 / 10**
* **Плюсы:** Быстрый парсинг на `fastjson`, богатая схема БД в PostgreSQL (охватывает всё от погоды и судей до координат игроков и xG), гибкая настройка через `provider_endpoints`, автодокачка внешних ключей, практичное разделение тяжёлых `include` для обхода таймаутов Sportmonks.
* **Минусы:** Пакетный характер (только по крону раз в сутки), критические баги в `HandlingViolatesFKError` (потеря запроса `query`), перепутанные колонки в `coachs.go`, баг `fx.GetBool("ticking")` в `periods`, и опечатка пути в `odds_every_day.sh`.

### Предлагаемый план внедрения (от быстрых фиксов к полному Live):

1. **Шаг 1 (Быстрые исправления коллектора без ломки архитектуры):**
   * [x] Исправить `os.Exit(0)` $\rightarrow$ `return` и упростить `replaceToken` в `api/handlers/endpoint.go`.
   * [x] Исправить передачу полного SQL-запроса (`queryItog` / `queryInsert`) в `HandlingViolatesFKError` (`fixtures.go:L543, L733`, `players.go:L122`).
   * [x] Исправить опечатку `footbaldata` $\rightarrow$ `footballdata` в `/opt/apisportmonks/cron/odds_every_day.sh`.
   * [x] Исправить `fx.GetBool("ticking")` $\rightarrow$ `v.GetBool("ticking")` в `api/handlers/fixtures.go:L410`, а также добавить `UpsertFixtureParticipants` и `UpsertFixtureCoaches` в `default:` ветки `UpsertFixturesSeasons` и `UpsertFixturesBetweenDate`.
   * [x] Исправить чтение `d.Get("data")` и порядок колонок `player_id, sport_id` в `api/handlers/coachs.go`.
   * [x] Добавить проверку на пустой массив в `odds.go`, `seasons.go`, `leagues.go`, `countries.go`, `bookmakers.go`, `markets.go`, `rivals.go`, `types.go` и батчевую вставку в `venues.go`.
   * [x] Исправить `CastInterfaceString` для `*bool` и `nil`-указателей в `db/func.go`.
2. **Шаг 2 (Включение `UpsertFixturesLive` и `upsertStandingsLive`, единый бинарник `lab`):**
   * [x] Объединить `prematch` (`main.go`) и `lab` (`cmd/lab/main.go`) в единый бинарник `./lab` с инициализацией `&db.Prov` и маршрутизацией всех задач коллектора из `provider_endpoints`.
   * [x] Зарегистрировать `"UpsertFixturesLive"` и `"upsertStandingsLive"` в `api/handlers/function.go`.
   * [x] Реализовать `UpsertFixturesLive` в `api/handlers/fixtures.go` с очисткой отменённых VAR событий и фильтром по активным лигам `filters=fixtureLeagues:8,82,301,384,564`.
   * [x] Реализовать `UpsertStandingsLive` (`GET /v3/football/standings/live/leagues/{leagueID}`) в `api/handlers/standings.go` с записью актуальной таблицы в активный `round_id` без перезаписи `id` прошлых туров.
3. **Шаг 3 (Умный Live-поллер с интервалом 1 минута и триггером Live-таблицы):**
   * [x] Встроить вызов Live-синхронизации (`UpsertFixturesLive`, интервал по умолчанию `1m`) в `LivePoller` (`server/poller.go`), который активируется только тогда, когда в БД есть матчи в активном временном окне (`NOW() - 150m .. NOW() + 15m` и статус $\ne$ `FT`).
   * [x] При изменении статуса или счёта матча в лиге автоматически вызывать `upsertStandingsLive` (`standings/live/leagues/{leagueID}`) для соответствующей лиги и тура, а при переходе матча в `FT` — докачивать xG (`upsertExpectedFixtures`).

