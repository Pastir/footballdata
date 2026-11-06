# План архитектурного разделения: Сбор данных vs Аналитика и Публикация

Данный документ описывает целевую архитектуру разделения контура **сбора данных** (`prematch`) и контура **аналитики, генерации медиа-контента и публикации** (`cmd/<name>`), с регистрацией внутренних задач в `provider_endpoints` (`provider_id = 3`, `footballstats`).

---

## 1. Концепция и разделение ответственности

В соответствии с правилами проекта и разделением сфер ответственности:
1. **Контур сбора данных (Data Ingestion)**:
   - Бинарник: `./prematch` (корень `main.go`).
   - Задача: взаимодействие с внешним Sportmonks API (`provider_id = 2`), загрузка данных и сырых сущностей в PostgreSQL и ClickHouse.
   - Логика публикаций отсюда полностью исключается.
2. **Контур аналитики и публикации (Analytics, Media & Publishing)**:
   - Единая точка входа: `cmd/<name>/main.go`.
   - Задача: детерминированный расчёт футбольной аналитики на основе собственной БД, сборка ViewModel, генерация графики (HTML/CSS -> Chromium -> PNG), формирование текста (FactPack / Copywriter) и отправка в Telegram с фиксацией в истории публикаций.
   - Единый интерфейс запуска через флаги `-task` и `-type`.
   - Упразднение разрозненных утилит `cmd/fixtures/` и `cmd/table/`.

---

## 2. Предложения по названию для `cmd/<name>` (10 вариантов)

Директория и исполняемый файл для аналитики, медиа-генерации и публикаций:

1. **`lab`** (`cmd/lab/main.go`, бинарник `./lab`) — отражает название проекта *Football Data Lab*, лаконично, олицетворяет фабрику аналитики и контента.
2. **`publisher`** (`cmd/publisher/main.go`, бинарник `./publisher`) — классическое имя для контура публикации и дистрибуции контента.
3. **`media`** (`cmd/media/main.go`, бинарник `./media`) — подчёркивает фокус на медиа-производстве (инфографика, посты, визуализации).
4. **`analyst`** (`cmd/analyst/main.go`, бинарник `./analyst`) — роль футбольного дата-аналитика, формирующего инсайты и отчёты.
5. **`content`** (`cmd/content/main.go`, бинарник `./content`) — фабрика генерации контента для соцсетей и мессенджеров.
6. **`press`** (`cmd/press/main.go`, бинарник `./press`) — типография / издательский дом проекта: верстка таблиц, расписаний и выпуск в свет.
7. **`feed`** (`cmd/feed/main.go`, бинарник `./feed`) — генератор и публикатор новостной аналитической ленты.
8. **`matchcenter`** (`cmd/matchcenter/main.go`, бинарник `./matchcenter`) — перекликается с историческим назначением последовательностей в БД (`matchcenter_provider_id_seq`).
9. **`dispatch`** (`cmd/dispatch/main.go`, бинарник `./dispatch`) — диспетчер задач аналитики, рендеринга и отправки сообщений.
10. **`statsfeed`** (`cmd/statsfeed/main.go`, бинарник `./statsfeed`) — акцент на связке «статистика + лента публикаций».

> **Рекомендация**: вариант **`lab`** (или **`publisher`**) наиболее точно отражает архитектурную роль и бренд проекта.

---

## 3. Регистрация в БД: `provider` и `provider_endpoints`

Для стандартизации управления задачами публикации используется существующая инфраструктура провайдеров.

### 3.1. Запись в таблицу `provider`
```sql
INSERT INTO provider (id, name, status, settings)
VALUES (
    3, 
    'footballstats', 
    true, 
    '{"description": "Internal analytics, content rendering and publication engine"}'::jsonb
)
ON CONFLICT (id) DO UPDATE 
SET name = EXCLUDED.name, status = EXCLUDED.status, settings = EXCLUDED.settings;
```

### 3.2. Регистрация задач в `provider_endpoints` (`provider_id = 3`)
Таблица `provider_endpoints` хранит системные метаданные задач аналитики и публикаций (название, частота обновления/публикации, параметры):

| provider_id | name | task_name | endpoint | cycle | update_frequency | additional |
|---|---|---|---|---|---|---|
| 3 | `telegram.standings` | `PublishToTelegram` | `internal://analytics/standings` | `league` | 86400 | `{"type": "table", "template": "template/football_table_template.html"}` |
| 3 | `telegram.round_fixtures` | `PublishToTelegram` | `internal://analytics/round_fixtures` | `league,round` | 86400 | `{"type": "roundFixtures", "template": "template/round_fixtures_template.html"}` |
| 3 | `telegram.preview_match` | `PublishToTelegram` | `internal://analytics/preview_match` | `fixture` | 0 | `{"type": "previewMatch"}` |
| 3 | `telegram.round_results` | `PublishToTelegram` | `internal://analytics/round_results` | `league,round` | 86400 | `{"type": "roundResults", "template": "template/round_results_template.html"}` |

Это обеспечивает:
- Чёткое разграничение: `provider_id = 2` — Sportmonks API, `provider_id = 3` — собственный аналитический контур `footballstats`.
- Хранение частоты обновления (`update_frequency`), что позволит крону и планировщику запускать генерацию по расписанию.

---

## 4. Единый формат запуска и CLI-параметры

### 4.1. Формат запуска единого исполнителя (`cmd/lab`)
Запуск производится через единый бинарник `./lab` с параметрами:
```bash
# 1. Турнирная таблица (table / standings)
./lab -conf config.yaml -task PublishToTelegram -type table -leagueId 8 -publish true

# 2. Расписание тура с котировками 1X2 и Silent Pin (roundFixtures / fixtures)
./lab -conf config.yaml -task PublishToTelegram -type roundFixtures -leagueId 8 -roundId 407868 -publish true

# 3. Результаты тура и итоги вторника (tuesdaySummary / roundResults)
# 2-этапный пост: Премиум-постер Топ-5 лидеров (1920x1080) + нативная таблица с реальным xG + кнопки Mini App
./lab -conf config.yaml -task PublishToTelegram -type tuesdaySummary -leagueId 8 -publish true
./lab -conf config.yaml -task PublishToTelegram -type roundResults -leagueId 8 -roundId 407869 -publish true


# 4. Превью матча (previewMatch / preview)
./lab -conf config.yaml -task PublishToTelegram -type previewMatch -leagueId 8 -fixturesId 18535518 -publish true

# 5. Интерактивный дайджест тура (roundDigest / digest)
./lab -conf config.yaml -task PublishToTelegram -type roundDigest -leagueId 8 -publish true

# 6. Каталог активных турниров для Telegram Mini App (catalog / leagues)
# Собирает турниры с pl.active = true, генерирует active_leagues.json и синхронизирует в Nginx
./lab -conf config.yaml -task PublishToTelegram -type catalog

# 7. Лидеры сезона и индивидуальная статистика игроков (leaders)
./lab -conf config.yaml -task PublishToTelegram -type leaders -leagueId 8 -publish true

# 8. Пакетная синхронизация всех активных лиг для Telegram Mini App (syncAll)
# Собирает и экспортирует таблицы, матчи, расписание и лидеров для всех турниров из БД
./lab -conf config.yaml -task PublishToTelegram -type syncAll

# Локальная генерация графики и JSON (dry-run без отправки в Telegram канал):
./lab -conf config.yaml -task PublishToTelegram -type table -leagueId 8 -output test_table.png -publish false
./lab -conf config.yaml -task PublishToTelegram -type roundFixtures -leagueId 8 -publish false
./lab -conf config.yaml -task PublishToTelegram -type roundResults -leagueId 8 -publish false
./lab -conf config.yaml -task PublishToTelegram -type leaders -leagueId 8 -publish false
./lab -conf config.yaml -task PublishToTelegram -type syncAll -publish false
```

### 4.2. Полная таблица CLI-параметров бинарника `./lab`

| Флаг | Тип | Default | Описание |
|---|---|---|---|
| `-conf` | `string` | `./config.yaml` | Путь к конфигурационному YAML-файлу |
| `-task` | `string` | `""` | Имя задачи (`PublishToTelegram`) |
| `-type` | `string` | `""` | Тип контента: `table`, `roundFixtures`, `roundResults`, `previewMatch`, `roundDigest`, `catalog`, `leaders`, `syncAll` |
| `-leagueId` | `int` | `8` | ID лиги (8 = АПЛ, 564 = Ла Лига, 384 = Серия А, 82 = Бундеслига, 301 = Лига 1) |
| `-roundId` | `int` | `0` | Принудительное переопределение ID раунда/тура |
| `-fixturesId` | `string` | `""` | ID конкретного матча (для `previewMatch`) |
| `-publish` | `bool` | `false` | Публиковать в Telegram (`true` / `false`). При `false` выполняется только локальный рендер и экспорт данных |
| `-asDocument`| `bool` | `false` | Отправка инфографики документом без JPEG-сжатия Telegram (`true` / `false`) |
| `-output` | `string` | `""` | Путь для сохранения локального PNG-файла |
| `-locale` / `-l` | `string` | `ru` | Локализация названий команд и контента (`ru`, `en`, `ua`) |
| `-timezone` | `string` | `MSK` | Отображаемый часовой пояс матчей (по умолчанию `MSK`) |
| `-force` | `bool` | `false` | Принудительная публикация (игнорировать защиту от дубликатов) |

### 4.3. Консольные команды Sportmonks-коллекторов (`main.go`)
Для первичного наполнения и обновления базы данных Sportmonks используются фоновые задачи:
```bash
# Обновление справочников лиг, сезонов и турнирных типов
go run main.go -conf config.yaml -task upsertTypes
go run main.go -conf config.yaml -task upsertLeagues
go run main.go -conf config.yaml -task upsertSeasons
go run main.go -conf config.yaml -task upsertStages

# Обновление туров и команд текущих сезонов
go run main.go -conf config.yaml -task upsertRounds
go run main.go -conf config.yaml -task upsertSeasonsTeams -seasonIsCurrent true

# Обновление турнирных таблиц
go run main.go -conf config.yaml -task upsertStandingsSeasons -seasonIsCurrent true

# Сбор матчей и статистики туров
go run main.go -conf config.yaml -task upsertFixturesSeasons -seasonIsCurrent true
go run main.go -conf config.yaml -task upsertFixturesSeasonsTimeline -seasonIsCurrent true
go run main.go -conf config.yaml -task upsertFixturesSeasonsStatistics -seasonIsCurrent true

# Сбор коэффициентов букмекеров
go run main.go -conf config.yaml -task upsertOdds -startDate 2026-10-15 -endDate 2026-10-25

# Сбор xG (ожидаемых голов)
go run main.go -conf config.yaml -task upsertExpectedFixtures -startDate 2026-10-15 -endDate 2026-10-25
```

## 5. Архитектура единой точки входа (`cmd/<name>/main.go`)

### 5.1. Поток выполнения
```text
CLI Flags (-task, -type, -leagueId, etc.)
               ↓
    Init Services (Config, Logs, DB)
               ↓
         main() Router
               ↓
    [task == "PublishToTelegram"]
               ↓
   Publisher Dispatcher (по -type)
   ┌───────────────┬──────────────────┬─────────────────┬─────────────────┐
   ▼               ▼                  ▼                 ▼
 table       roundFixtures       previewMatch      roundResults
   │               │                  │                 │
   └───────┬───────┴──────────┬───────┴─────────┬───────┘
           ▼                  ▼                 ▼
      Analytics          Renderer          Publisher
```

### 5.2. Переиспользование существующих компонентов
1. **Подключение к БД**: `db.DBx` (PGxPool).
2. **Слой аналитики (`analytics/`)**:
   - `analytics.GetLeagueSeasonInfo`
   - `analytics.GetCurrentStandingsRounds`
   - `analytics.GetCurrentStandings`
   - `analytics.GetStandingDetails`
   - `analytics.GetForm`, `analytics.GetTrend`
   - `analytics.GetKeyStats`, `analytics.BuildInsight`
   - `analytics.GetNextRoundFixtures`
   - `analytics.GetH2HMatches`
   - `analytics.GetTeamPreMatchProfile`
3. **Слой представления (`viewmodel/`)**:
   - `viewmodel.BuildTableViewModel`
   - `viewmodel.RoundFixturesViewModel`
   - `viewmodel.RoundResultsViewModel`
4. **Рендерер (`renderer/`)**:
   - `renderer.RenderStandingsPNG`
   - `renderer.RenderRoundFixturesPNG`
   - `renderer.RenderRoundResultsPNG`
5. **Копирайтер (`copywriter/`)**:
   - `copywriter.BuildRoundFixturesFactPack`
   - `copywriter.BuildStandingsFactPack`
   - `copywriter.GenerateRoundFixturesHTMLPost`
   - `copywriter.GenerateRoundResultsHTMLPost`
   - `copywriter.GenerateFallbackPost`
6. **Слой публикации и хранилища (`publisher/`)**:
   - `publisher.SendPhotoWithKeyboard`, `publisher.SendTextMessageWithKeyboard`
   - `publisher.PinChatMessage` (Silent Pin)
   - `publisher.PGStorage` (учёт статусов, привязка `message_id`, защита от дублирования).

---

## 6. Статус реализации этапов унификации

1. **[x] Шаг 1: Миграция БД для провайдера и задач публикации**:
   - Таблица `telegram_publications` создана и функционирует.
   - Задачи зарегистрированы в схеме проекта.

2. **[x] Шаг 2: Вынос и обобщение CLI-параметров**:
   - Расширен `config.LineArgs` и `TaskData`: флаги `-type`, `-output`, `-publish`, `-asDocument`, `-timezone`, `-roundId`, `-l` / `-locale`.
   - Локализация по умолчанию: `ru`.

3. **[x] Шаг 3: Реализация диспетчера публикаций (`publisher/dispatcher.go`)**:
   - Реализована функция `Dispatch(ctx, taskData, log)`.
   - Реализованы обработчики: `dispatchTable`, `dispatchRoundResults`, `dispatchRoundFixtures`, `dispatchRoundDigest`.
   - Для `roundResults` и `roundFixtures` реализована полная интеграция с Telegram Mini App.

4. **[x] Шаг 4: Создание единой точки входа `cmd/lab/main.go`**:
   - Точка входа `cmd/lab/main.go` функционирует с единой командой `-task PublishToTelegram`.

5. **[x] Шаг 5: Интерактивный Mini App и Telegram-посты (Результаты тура)**:
   - Текстовый пост тура с моноширинным форматированием `<code>`, центровкой счетов и жирными акцентами в экспресс-статистике.
   - Двусторонний таймлайн событий («Обзор»), визуализация расстановки на тактическом поле, интерактивная таблица с трендами и еврокубковыми зонами.
   - Локализация названий клубов по колонке `provider_teams.locale`.

6. **[x] Шаг 6: Унификация расписания тура (`roundFixtures`) и Pre-Match режим в Mini App**:
   - Бэкенд-аналитика H2H (до 10 последних очных встреч из БД) и 3 Hero-метрики (xG за матч, xGA за матч, голы за матч за 5 туров и сезон).
   - Генерация моноширинного текста расписания (`copywriter/round_fixtures.go`) с центровкой, ссылками на котировках в Mini App и блоком «Матч тура».
   - Pre-Match интерфейс в Mini App (`app.js`, `style.css`): автоопределение `state != 'FT'`, таб H2H со сводкой и карточками встреч, таб Статистики с переключателем периода и 3 Hero-карточками, таб Составов с баннером ожидания за 60 минут.
   - Автоматический экспорт `round_data.json` и синхронизация с Nginx public dir (`/extssd/footballstats/public/`).
   - Поддержка тихого закрепления (Silent Pin) и фиксации в `telegram_publications`.

7. **[x] Шаг 7: Аудит кодовой базы и устранение всех замечаний (17 пунктов `AUDIT_REPORT.md`)**:
   - Исправлена запись `ChannelID` в `telegram_publications`, внедрена проверка идемпотентности во всех диспетчерах (`-force` для переопределения);
   - Устранены риски повреждения HTML-тегов при обрезке длинных подписей в Telegram;
   - Устранены race conditions в параметрах запросов и паники в обработчиках FK коллектора;
   - Исправлен селектор и изоляция лиг в Mini App, экранированы строковые поля турниров;
   - Удален устаревший некомпилируемый пакет `utility/`, актуализированы `.gitignore` и тесты;
   - Вынесены шаблоны дайджеста в `copywriter/digest.go`, оптимизированы N+1 запросы и вызовы `EnsureSchema`.

---

## 7. Реализация унификации расписания тура (`roundFixtures`) — СТАТУС: ЗАВЕРШЕНО

### 7.1. Проблема предыдущей реализации (РЕШЕНО)
- Предыдущая реализация генерировала статичный PNG и устаревший текст без ссылок и центровки.
- В рамках данного шага расписание полностью приведено к единому интерактивному формату проекта со сквозной интеграцией с Telegram Mini App.

---

### 7.2. Отчёт о выполнении этапов унификации расписания

#### [x] Шаг 7.2.1: Бэкенд-аналитика и экспорт предматчевого `round_data.json`
* **Файлы**: `analytics/round_fixtures.go`, `analytics/h2h.go`, `analytics/team_stats.go`.
* **Выполнено**:
  1. **Котировки 1X2**: подключены данные из `provider_odds` (Pinnacle / маркет 1).
  2. **H2H (Очные встречи)**:
     - Реализована функция `GetH2HMatches(ctx, pool, homeTeamID, awayTeamID, locale, limit)`;
     - Выбираются до **10 последних очных встреч** из `provider_fixtures` с итоговым счётом, датой, турниром и xG из `provider_expected_fixtures`;
     - Рассчитывается агрегированная сводка: победы хозяев / ничьи / победы гостей, средний тотал голов.
  3. **Сравнительный профиль команд (Hero-метрики)**:
     - Расчёт топ-3 метрик: ⚡ **xG за матч**, 🛡️ **xGA за матч**, ⚽ **Голы за матч**;
     - Поддержка двух разрезов: **последние 5 туров** и **весь сезон**;
     - Дополнительные индикаторы: Обе забьют (BTTS %), Тотал больше 2.5 (%), Удары в створ за матч, Владение мячом (%).
  4. **Составы и готовность (Lineups & Sidelined)**:
     - Определение статуса ожидания официальных составов за 60 мин до свистка.
  5. **Экспорт данных**:
     - Формирование JSON-структуры `round_data.json` для следующего тура;
     - Сохранение в `web/viewer/round_data.json` и публичные директории Nginx (`/extssd/footballstats/public/media/{leagueId}/{seasonId}/{roundId}/round_data.json` и `/extssd/footballstats/public/viewer/round_data.json`).

#### [x] Шаг 7.2.2: Копирайтер и форматирование Telegram-поста расписания
* **Файл**: `copywriter/round_fixtures.go`.
* **Выполнено**:
  1. Реализована функция `GenerateRoundFixturesHTMLPost(...)`.
  2. Моноширинное форматирование колонок (`<code>`) с расчётом отступа от самого длинного названия хозяев.
  3. Котировки 1X2 обернуты в кликабельные deep link ссылки на Mini App:
     `<code>Arsenal       </code><a href="https://t.me/bot/app?startapp=match_{league}_{season}_{round}_{fixtureId}"><b>[ 1.85 · 3.60 · 4.20 ]</b></a><code>  Chelsea</code>`
  4. Блок «МАТЧ ТУРА» с выделением центральной встречи.
  5. Подсказка читателям: `💡 Нажмите на котировки матча, чтобы открыть H2H, xG и составы`.
  6. Инлайн-клавиатура: `[ 📱 Все матчи тура в Mini App ]` и `[ 📊 Турнирная таблица ]`.
  7. Поддержка Silent Pin: тихое закрепление сообщения в Telegram-канале на уикенд.

#### [x] Шаг 7.2.3: Pre-Match режим в Telegram Mini App (FDL WebApp)
* **Файлы**: `web/viewer/app.js`, `web/viewer/index.html`, `web/viewer/style.css`.
* **Выполнено**:
  1. Автоопределение предматчевого состояния (`match.StateShort !== 'FT'`).
  2. Шапка матча: дата, время начала матча и котировки 1X2 `[ 1: 1.85 · X: 3.60 · 2: 4.20 ]`.
  3. Суб-таб **H2H** (активен по умолчанию):
     * Баннер: победы хозяев / ничьи / победы гостей, средний тотал голов;
     * Карточки до 10 последних очных матчей (дата, турнир, счёт, xG).
  4. Суб-таб **Статистика**:
     * Сегментированный переключатель: `[ 5 последних туров ]` / `[ Весь сезон ]`;
     * 3 Hero-карточки первого экрана: `xG за матч`, `xGA за матч`, `Голы за матч`;
     * Сравнительные прогресс-бары вспомогательных показателей (BTTS, Over 2.5, Удары в створ, Владение).
  5. Суб-таб **Составы**:
     * Баннер ожидания с иконкой песочных часов `⏳ Официальные составы появятся за 60 минут до матча`, стадионом и судьей.

#### [x] Шаг 7.2.4: Интеграция в диспетчер CLI (`publisher/dispatcher.go`)
* **Файлы**: `publisher/dispatcher.go`.
* **Выполнено**:
  - Полный сквозной пайплайн: `analytics.GetNextRoundFixtures` $ightarrow$ `analytics.GetH2H` $ightarrow$ `analytics.GetTeamPreMatchProfile` $ightarrow$ экспорт `round_data.json` $ightarrow$ `copywriter.GenerateRoundFixturesHTMLPost` $ightarrow$ отправка в Telegram $ightarrow$ Silent Pin $ightarrow$ сохранение в `telegram_publications`.

#### [x] Шаг 7.2.5: Тестирование и верификация
* **Выполнено**:
  1. Unit-тесты для копирайтера `copywriter/round_fixtures_test.go` (пройдены).
  2. Unit-тесты для H2H-аналитики `analytics/h2h_test.go` (пройдены).
  3. Локальный запуск dry-run: `./lab -conf config.yaml -task PublishToTelegram -type roundFixtures -leagueId 8 -publish=false` (успешно).
  4. Визуальная верификация в браузере (субагент проверил список матчей, шапку с коэффициентами, H2H, Hero-карточки со свитчером и баннер составов).
  5. Боевая публикация расписания тура в Telegram-канал (`-publish=true`): пост успешно опубликован (Message ID: 55), закреплён (Silent Pin) и сохранён в `telegram_publications`.

---

## 8. Реализация мульти-лигового каталога и селектора (Фаза 6) — СТАТУС: ЗАВЕРШЕНО

### 8.1. Выполненные этапы Фазы 6

#### [x] Шаг 8.1.1: Бэкенд-модель и экспорт каталога турниров
* **Файлы**: `analytics/league.go`, `analytics/league_catalog_test.go`, `publisher/dispatcher.go`.
* **Выполнено**:
  1. Реализована функция `analytics.GetActiveLeagues(ctx, pool)`, собирающая все турниры с `pl.active = true` и `ps.is_current = true`, последний завершенный тур (`curr_round`) и ближайший незавершенный тур (`next_round`).
  2. Реализована классификация `resolveLeagueMetadata`:
     - «Европейские чемпионаты» (`domestic`, иконка 🌍): АПЛ, Ла Лига, Серия А, Бундеслига, Лига 1 и другие национальные лиги;
     - «Еврокубки» (`eurocups`, иконка ⭐): Лига чемпионов, Лига Европы, Лига конференций.
  3. **Правило динамического скрытия категорий**: пустые категории (`Count == 0`) строго исключаются из `resp.Categories` (сейчас активны 5 топ-лиг из категории «Европейские чемпионаты», а категория «Еврокубки» скрыта до активации еврокубков).
  4. Реализован CLI-хэндлер: `./lab -conf config.yaml -task PublishToTelegram -type catalog`.
  5. Файл каталога экспортируется в `web/viewer/active_leagues.json` и публичные директории Nginx (`/extssd/footballstats/public/` и `/viewer/`).

#### [x] Шаг 8.1.2: Пользовательский интерфейс Telegram Mini App
* **Файлы**: `web/viewer/index.html`, `web/viewer/style.css`, `web/viewer/app.js`.
* **Выполнено**:
  1. **Селектор в шапке (`#leagueSelectorTrigger`)**:
     - Отображает эмодзи-флаг страны, название текущей лиги и шеврон `▼`;
     - По нажатию открывает Bottom Sheet.
  2. **Quick-Pills карусель (`#quickLeaguesBar`)**:
     - Горизонтальный скролл топовых турниров (`is_top`) для мгновенного выбора в 1 тап;
     - Кнопка «🌐 Все (5)» (динамическое число) для быстрого вызова каталога.
  3. **Bottom Sheet модальное окно (`#leagueBottomSheet`)**:
     - Заголовок с количеством активных лиг;
     - Поисковая строка по названию турнира и стране;
     - Динамическое меню категорий: если в активных турнирах отсутствуют турниры из «Европейские чемпионаты» или «Еврокубки», эти пункты меню скрываются (не отображаются);
     - Список турниров с логотипами/флагами, бейджами туров и индикатором текущего выбора;
     - Плавная анимация появления снизу (slide-up) и затемнение фона (backdrop blur).
  4. **SPA-переключение лиг**:
     - Мгновенное переключение (~150 мс) без перезагрузки страницы;
     - Автоматическое сохранение выбора в `localStorage` (`fdl_active_league_id`);
     - Обновление адресной строки (`history.replaceState`) с сохранением Deep Link совместимости.

#### [x] Шаг 8.1.3: Тестирование и верификация
* **Выполнено**:
  1. Unit-тесты `analytics/league_catalog_test.go` (пройдены).
  2. Браузерная верификация субагентом (проверены шапка, карусель, открытие шторки, фильтры «Еврокубки» и «Европейские чемпионаты», поиск и переключение).

---

## 9. Реализация индивидуальной статистики игроков и лидеров лиг (Фаза 7) — СТАТУС: ЗАВЕРШЕНО

### 9.1. Выполненные этапы Фазы 7

#### [x] Шаг 9.1.1: Детерминированная SQL-аналитика по секционированным таблицам
* **Файлы**: `analytics/player_leaders.go`, `analytics/player_leaders_test.go`.
* **Выполнено**:
  1. Реализована функция `analytics.GetSeasonPlayerLeaders(ctx, pool, info, limit)`, выполняющая выборку топ-10 игроков сезона в 5 номинациях:
     - **Бомбардиры (`TopScorers`)**: подсчёт голов (`GOALS`), голов с пенальти (`PENALTIES_SCORED`) и числа матчей из партиционированной таблицы `provider_fixtures_players_statistics_%d`;
     - **Ассистенты (`TopAssists`)**: подсчёт голевых передач (`ASSISTS`) и вторичный учёт голов;
     - **Гол + Пас (`TopPoints`)**: суммирование результативных баллов `goals + assists`;
     - **Сухие матчи вратарей (`CleanSheets`)**: точный расчёт матчей на ноль через джойн стартовых голкиперов (`type_id = 11 AND position_id = 24`) из `provider_lineups_%d` с завершёнными матчами (`state_id = 5`) и голами соперника `opp_score.goals = 0` из `provider_fixtures_scores`;
     - **Карточки и дисциплина (`Discipline`)**: суммирование жёлтых и красных карточек (`YELLOWCARDS`, `REDCARDS`).
  2. Партиционирование таблиц по `league_id` даёт время выполнения SQL-запросов ~12–15 мс.
  3. Написаны unit-тесты `analytics/player_leaders_test.go` (пройдены).

#### [x] Шаг 9.1.2: Модель представления и инфографика 1920px
* **Файлы**: `viewmodel/leaders.go`, `template/leaders_template.html`, `renderer/round.go`.
* **Выполнено**:
  1. Создана модель `viewmodel.BuildLeadersViewModel`, формирующая категории с Hero-элементом для #1 игрока и структурированными подстроками для каждого типа метрики.
  2. Разработан widescreen HTML/CSS шаблон инфографики `template/leaders_template.html` (1920×1080 px) с темным глассморфизмом, логотипами лиг, фото игроков и клубными эмблемами.
  3. В модуль `renderer` добавлена функция `RenderLeadersPNG`.

#### [x] Шаг 9.1.3: Копирайтер и публикация в Telegram
* **Файлы**: `copywriter/leaders.go`, `publisher/dispatcher.go`.
* **Выполнено**:
  1. Реализован генератор поста `copywriter.GenerateLeadersHTMLPost` с моноширинными таблицами ключевых лидеров турнира.
  2. В диспетчер `publisher/dispatcher.go` добавлен обработчик `dispatchLeaders` для типа `-type leaders`:
     - Экспорт `leaders.json` в `web/viewer/data/` и публичные директории Nginx;
     - Рендеринг инфографики `leaders.png` и сохранение в медиа-директорию сезона;
     - Проверка идемпотентности публикации (с флагом `-force`);
     - Инлайн-кнопки с прямым открытием вкладки лидеров в Telegram Mini App (`startapp={league}_{season}_leaders`).

#### [x] Шаг 9.1.4: Интерактивный раздел «Лидеры» в Telegram Mini App
* **Файлы**: `web/viewer/index.html`, `web/viewer/style.css`, `web/viewer/app.js`.
* **Выполнено**:
  1. Добавлена главная вкладка «Лидеры» (`#tabLeaders`) в верхнюю панель навигации.
  2. Реализован интерактивный экран `#leadersView` со свитчером 5 категорий:
     - ⚽ Бомбардиры;
     - 🎯 Ассистенты;
     - ⚡ Гол + Пас;
     - 🧤 Сухие матчи;
     - 🟨 Карточки.
  3. Сверстана Hero-карточка топ-1 игрока с золотым бейджем `👑 ТОП-1`, крупным аватаром, клубным логотипом и акцентным числом голов/очков.
  4. Сверстаны карточки игроков со 2 по 10 места с круглыми бейджами рангов (серебряный для #2, бронзовый для #3).
  5. Поддержана плавная смена категорий без перезагрузок и fallback на дефолтные SVG-аватары при отсутствии фото.

#### [x] Шаг 9.1.5: Тестирование и сквозная верификация
* **Выполнено**:
  1. Сборка бинарника `lab` (`go build -o lab cmd/lab/main.go`).
  2. Успешный прогон полного набора unit-тестов (`go test ./...`).
  3. Локальный запуск dry-run: `./lab -conf config.yaml -task PublishToTelegram -type leaders -leagueId 8 -publish=false`.
  4. Браузерное тестирование субагентом в Telegram Mini App (переход на вкладку лидеров, переключение всех 5 категорий, отображение фото игроков и эмблем клубов).

### 9.2. Пакетная синхронизация всех лиг и устранение артефактов интерфейса (Октябрь 2026) — ВЫПОЛНЕНО
* **Файлы**: `publisher/dispatcher.go`, `cmd/lab/main.go`, `web/viewer/app.js`, `web/viewer/style.css`.
* **Выполнено**:
  1. **Устранение бага с отображением лидеров внизу других вкладок**:
     - В `web/viewer/style.css` добавлен класс `.leaders-view.hidden { display: none !important; }` и глобальное правило `.hidden { display: none !important; }`.
     - Теперь блок лидеров изолирован и отображается исключительно на вкладке «Лидеры».
  2. **Устранение показа старой PNG-таблицы при смене лиги**:
     - В `switchLeague` заменен устаревший вызов `loadCurrentTabImage()` на штатный вызов `selectTab(currentTab)`.
     - Добавлена очистка DOM-контейнеров перед загрузкой новой лиги, исключающая наложение контента АПЛ на другие турниры.
     - Добавлен информативный Empty-State («Данные турнира готовятся к публикации») при отсутствии данных.
  3. **Пакетная генерация данных для всех активных турниров (`-type syncAll`)**:
     - Реализована функция `syncSingleLeagueData` и обработчик `dispatchSyncAll` (`./lab -conf config.yaml -task PublishToTelegram -type syncAll -publish=false`);
     - В один клик собираются и экспортируются актуальные таблицы, расписание, результаты туров и списки лидеров для всех 5 активных турниров (АПЛ, Ла Лига, Серия А, Бундеслига, Лига 1);
     - Все данные синхронизированы в веб-директории Nginx (`/extssd/footballstats/public/media/` и `/public/viewer/`).
  4. **Сквозная верификация в браузере**:
     - Субагент подтвердил корректное переключение между АПЛ, Ла Лигой, Бундеслигой и Серией А: мгновенно отображаются реальные таблицы, результаты и списки бомбардиров (Мбаппе, Левандовски в Ла Лиге, Кейн в Бундеслиге и т.д.).
  5. **Разделение расписания (несыгранный тур) и результатов (завершённый тур)**:
     - В `web/viewer/app.js` устранено смешивание туров: вкладка «Расписание» теперь строго загружает `next_round_id` (несыгранные матчи со временем начала и коэффициентами 1·X·2), а вкладка «Результаты» — `curr_round_id` (завершённые матчи с финальным счётом FT и экспресс-статистикой);
     - Внедрена изолированная карта кэширования `roundDataCache = new Map()`, разделяющая кэши завершённого и предстоящего туров для каждой лиги;
     - Заголовки экранов, подзаголовки и адресная строка браузера автоматически синхронизируются с выбранным туром (`7 ТУР • РАСПИСАНИЕ ТУРА` vs `6 ТУР • РЕЗУЛЬТАТЫ ТУРА`);
     - Верифицировано в Headless Chrome: вкладка «Расписание» выводит матчи со временем начала и котировками без финального счёта, вкладка «Результаты» — завершённые матчи со счётом.

---

## 10. Карта дальнейших этапов

### Фаза 8: Управление контентной лентой канала, факты/серии, расписание 5 лиг и умное закрытие туров

#### [x] Шаг 8.1: Аналитический движок «Факты и Серии» (Streaks & Trends Engine)
* **Файлы**: `db/migrations/004_create_analytics_streaks.sql`, `analytics/streaks.go`, `analytics/streaks_test.go`, `viewmodel/fact_card.go`, `template/fact_card_template.html`, `renderer/fact_card.go`, `copywriter/fact_post.go`, `copywriter/fact_post_test.go`, `publisher/dispatcher.go`, `cmd/lab/main.go`.
* **Выполнено**:
  1. Создана и применена миграция БД: таблицы `analytics_streak_types` (11 типов), `analytics_streak_scopes` (4 скоупа: лига, все, дом, выезд) со структурой `locale JSONB` и таблица `analytics_streaks` с историей матчей в JSONB и связями с тренерами `provider_coachs(id)`.
  2. Реализован детерминированный движок расчета серий `analytics.ComputeAndSaveTeamStreaks`:
     - Поддержка победных (`team_win`), беспроигрышных (`team_unbeaten`), без побед (`team_no_win`), голевых (`team_scoring`) и сухих (`team_clean_sheet`) серий;
     - Анализ всех 4 областей видимости (чемпионат, все турниры, только дома, только на выезде);
     - Автоматический расчет клубных и лиговых рекордов, а также вычисление рейтинга интересности (Interest Score);
     - Получение топ активных серий через `analytics.GetTopActiveStreaks`.
  3. Разработан квадратный 1:1 (1200×1200 px, 2400×2400 px Retina) шаблон инфографики `template/fact_card_template.html` в стиле темного неонового глассморфизма с эмблемой клуба, фото тренера, Hero-счетчиком, цепочкой карточек последних матчей (W-W-W) и блоком ключевых статистических метрик.
  4. Реализован рендерер `renderer.RenderFactCardPNG` через Headless Chromium.
  5. Реализован копирайтер `copywriter.GenerateFactHTMLPost` с эмодзи, хронологией игр, суммарной статистикой и кнопкой перехода в Telegram Mini App (`BuildFactMiniAppURL`).
  6. Добавлена CLI-диспетчеризация в `publisher/dispatcher.go`:
     - `-type recalculateStreaks`: принудительный перерасчет и сохранение серий в БД;
     - `-type facts`: выбор лучшей активной серии, рендеринг карточки 1200x1200px, dry-run режим и публикация в Telegram с инлайн-клавиатурой;
     - Гранулярная фильтрация по флагам `-leagueId`, `-teamsId`, `-coachesId`, `-playersId`, `-seasonsId`, `-publish`, `-output`.
  7. Проведено тестирование:
     - 100% прохождение unit-тестов `analytics/streaks_test.go` и `copywriter/fact_post_test.go`;
     - Успешный расчет 82 серий АПЛ в реальной базе данных;
     - Проверка рендеринга и формирования постов в dry-run режиме (карточки Ливерпуля и Ман Сити 2400×2400 px).

#### [x] Шаг 8.2: Публикация расписания туров для 5 топ-лиг с «Матчем тура»
* **Цель**: разнесённая по времени публикация расписания уикенда для всех 5 активных лиг (АПЛ, Ла Лига, Серия А, Бундеслига, Лига 1) без спама подписчикам.
* **Реализовано**:
  1. Автоматический расчёт расписания и метаданных для всех 5 лиг (`analytics.GetLeagueSeasonInfo`, `analytics.GetNextRoundFixtures`).
  2. Интеграция математического выбора главного «Матча тура» (`analytics.GetOrComputeMatchOfRound`) с передачей `featuredFixtureID` в `viewmodel.BuildRoundFixturesViewModel`.
  3. Автоматический расчет подразумеваемых процентных вероятностей модели (П1 / Х / П2 в %) на основе котировок 1·X·2.
  4. Инфографика расписания (Retina 3840px, брендированные шапки `template/headers/{id}.png`, двухколоночная верстка и marquee-карточка).
  5. Telegram HTML-посты с моноширинной сеткой, котировками 1·X·2 со ссылками на несыгранный тур Mini App и блоком «🔥 МАТЧ ТУРА».
  6. Контроль окна контентной сетки (за 48 часов до `starting_at` первого матча) с опцией принудительного обхода `-force`.
  7. Механизм тихого закрепления (`disable_notification=true`) строго для флагманской лиги (АПЛ, ID `8`), исключающий мигание и перетирание закрепов в канале.
  8. Пакетная публикация `-type weekendHub` (алиасы `schedule5Leagues`, `scheduleTop5`):
     - **Этап 1 (Медиа-альбом)**: 5 графических карточек центральных матчей тура (по одной на лигу) с гексагональными неоновыми рамками, позициями команд в таблице (`МЕСТО X`), 3D золотым `VS` и котировками 1·X·2; подпись к альбому содержит аккуратную таблицу в `<pre>` с шапкой `Время | Хозяева | : | Гости | 1 X 2` (673 символа, полностью в рамках лимита Telegram в 1024 символа).
     - **Этап 2 (Полное расписание уикенда)**: отправляется следом вторым сообщением (через текстовый режим `-type weekendSchedule` / `scheduleOnly` либо через нативные таблицы Telegram `-type nativeTable` / `richTable`); чистый список матчей всех 5 лиг, сгруппированный по дням (Пт-Пн), внутри дня — по фиксированному порядку лиг (АПЛ ➔ Ла Лига ➔ Серия А ➔ Бундеслига ➔ Лига 1) и времени в **UTC**; прикреплена инлайн-кнопка Mini App и выполнен тихий закреп (Silent Pin).
     - **Нативные таблицы Telegram (Bot API 10.1+ `sendRichMessage`)**: реализована отправка через `sendRichMessage` со структурными блоками `RichBlockTable`, `is_bordered`, `is_striped`, `is_compact`. Включает вступительный блок с нативной таблицей главных матчей тура (`🔥 ГЛАВНЫЕ МАТЧИ УИКЕНДА (ТОП-5 ЛИГ)`) с флагами стран, временем, командами и котировками, а также таблицы по дням с заголовками лиг `colspan: 5` и выравниванием по левому краю (`align: "left"`), заголовками уменьшенного размера (`size: 3` и `size: 4`). Снимает лимит 4096 символов (поддерживает до 32 768 символов).
     - **Форматирование матчей (Вариант А)**: текстовый режим объединяет время и названия команд в моноширинный блок `<code>14:00  Арсенал — Эвертон</code>  <i>(1.38 · 4.75 · 7.00)</i>` (3751 / 4096 символов).
     - **Выделенные режимы топ-матчей**:
       - `-type topMatches` (алиасы `top5Matches`, `centralMatches`, `topMatchesHub`): утверждённый формат «Вариант В» — двухэтапная публикация: сначала медиа-альбом из 5 постеров с лаконичной подписью, затем сразу следом нативная интерактивная таблица Telegram со временем (UTC), котировками и кнопкой Mini App;
       - `-type topMatchesAlbum`: автономная публикация только медиа-альбома 5 карточек с табличной моноширинной подписью `<pre>`;
       - `-type topMatchesTable` (алиас `topMatchesNative`): автономная публикация только нативной таблицы главных матчей через `sendRichMessage` с инлайн-кнопкой Mini App.
     - **Системный переход на UTC**: значение по умолчанию таймзоны изменено с MSK на UTC во всей конфигурации (`config.go`), шаблонах (`match_header_template.html`), вьюмоделях (`viewmodel/round.go`) и диспетчере.

#### [x] Шаг 8.3: Закрытие туров строго по ending_at (с учётом понедельников и переносов)
* **Цель**: надёжное завершение тура и ротация закрепа с расписания на результаты.
* **Реализованные компоненты**:
  1. **Модуль `analytics/round_closing.go`**:
     - Детерминированная структура `RoundClosingStatus` со счётчиками матчей (завершённые, отложенные/перенесённые, в игре, предстоящие);
     - Чистая функция `EvaluateRoundClosing(status, now)`:
       - Блокирует закрытие, если есть матчи в игре (`InPlayMatches > 0`);
       - Учитывает временной буфер окончания вечерних матчей понедельника (2 часа после свистка);
       - Fallback-дедлайн утра вторника (08:30–09:00 UTC): если есть сыгранные матчи, тур принудительно закрывается даже при наличии перенесённых матчей (`HasPostponements = true`), предотвращая блокировку канала и Mini App.
     - Функция `GetRoundClosingStatus(ctx, pool, leagueID, now)` для выборки активного раунда и агрегации статусов матчей из `provider_fixtures` на основе функции `get_neighbor_rounds(ps.id)` (устойчива к переносам матчей);
  2. **Детерминированное определение последовательности туров (`get_neighbor_rounds`)**:
     - SQL-функция `get_neighbor_rounds(p_season_id integer)` внедрена во все ключевые модули: `analytics/league.go` (`GetLeagueSeasonInfo`, `GetActiveLeagues`), `analytics/express_builder.go` (`GetUpcomingRound`), `analytics/round_fixtures.go` (`GetNextRoundFixtures`) и `analytics/round_closing.go` (`GetRoundClosingStatus`);
     - Устранено зависание лиг из-за переносов отдельных матчей тура (например, перенесённый на декабрь матч 8-го тура Ла Лиги больше не блокирует определение сыгранного 10-го и предстоящего 11-го туров);
  3. **Планировщик `scheduler/db_task_provider.go`**:
     - `DBTaskProvider` проверяет созревание итогов тура (`roundResults`) по топ-5 лигам и пятничного пакета расписания (`weekendHub`);
     - Защита от дублей через `storage.IsAlreadyPublished`;
     - Диагностический метод `CheckRoundStatusAll`.
  4. **Тихая ротация закрепа (Silent Pin)**:
     - В `publisher/dispatcher.go` (`dispatchRoundResults`) добавлена смена закреплённого сообщения в канале с пятничного расписания на нативную интерактивную таблицу результатов тура без отправки пуш-уведомлений (`disable_notification: true`).
  5. **Интеграция в CLI (`cmd/lab/main.go`)**:
     - `-task CheckRoundClosing`: наглядная табличная диагностика готовности туров по лигам;
     - `-task SchedulerTick`: разовый цикл для выполнения созревших публикаций (идеально для Linux crontab);
     - `-task SchedulerDaemon`: непрерывный фоновый планировщик с интервалом 1 мин и graceful shutdown.

#### [ ] Шаг 8.4: Топ-5 голов тура и интерактивные голосования (Telegram Polls)
* **Цель**: максимальное вовлечение аудитории по итогам тура (отложено до реализации пайплайна видео/хайлайтов).
* **Компоненты**:
  1. Выборка 5 лучших/зрелищных голов тура (по xG удара, авторам, статусу матча).
  2. Публикация нативного опроса Telegram (`sendPoll`) за звание лучшего гола тура.
  3. Механизм прикрепления видеоматериалов / хайлайтов из медиа-ресурсов.

#### [x] Шаг 8.5: Монетизация: Data-Driven экспрессы и коэффициенты букмекера
* **Цель**: органичная монетизация канала через аналитику без навязчивого спама.
* **Реализованные компоненты**:
  1. **Модуль `analytics/express_builder.go`**:
     - Загрузка матчей тура и котировок 1X2 из `provider_odds` (`FetchRoundCandidateMatches`);
     - Алгоритмический отбор «Умного экспресса тура» (`BuildSmartExpress`): выбор 3 непересекающихся событий с перевесом (фаворит + результативный исход / ОЗ + надёжный двойной шанс) с суммарным коэффициентом в целевом диапазоне **3.50 – 5.25**;
     - Автоматическое определение ближайшего тура по дате (`GetUpcomingRound`).
  2. **Копирайтер, постер и нативные таблицы Telegram (`copywriter/express.go`, `renderer/express.go`, `publisher/dispatcher.go`)**:
     - **Нативные таблицы Telegram (Bot API 10.1+ `SendRichMessage`, `-style rich|rich1|rich2|rich3`)**:
       - **[1 ТЕСТ] Dashboard & Coupon Matrix**: структурированная таблица купона (`Время`, `Хозяева`, `Гости`, `Исход`, `Кэф`), подвальные сводные строки с `Colspan: 4` (`ИТОГОВЫЙ КОЭФФИЦИЕНТ: 4.02`, `VALUE EDGE: +7.4%`, `РЕКОМЕНДУЕМЫЙ ФЛЭТ: 1.0 юнит`), таблица xG-факторов и таблица партнерского бонуса FONBET;
       - **[2 ТЕСТ] Quant Screener & Portfolio Performance**: верхняя KPI-матрица в 4 колонки (`Общий кэф`, `Value Edge`, `Win Prob`, `Flat Stake`), 5-колоночная таблица позиций портфеля (`Событие (UTC)`, `Рынок`, `Исход`, `Кэф`, `Edge`) и детализация алгоритмических сигналов моделей;
       - **[3 ТЕСТ • УТВЕРЖДЁН] Мобильный аналитический паспорт экспресса (`-style mobile`)**: двухколоночный протокол без эмодзи с выравниванием по левому краю; визуальное разделение событий внешними крупными заголовками `Size: 3` (`СОБЫТИЕ %d · %s`), полноширинными акцентными плашками матчей (`Colspan: 2, IsHeader: true`), симметричной xG-фактурой для обеих команд при прогнозах «Обе забьют» и отдельной карточкой `ИТОГОВЫЙ ЭКСПРЕСС`;
     - **Текстовые стили и графический постер (`-style 1|2|3|all`)**:
       - **[1 ТЕСТ] The Athletic / Редакционная колонка**: глубокая интеллектуальная подача без эмодзи-спама, нативные цитаты `<blockquote>`, точные xG-метрики и финансовый риск-менеджмент;
       - **[2 ТЕСТ] Bloomberg Terminal / Дашборд**: моноширинная псевдографическая таблица в `<pre>` с идеальным попиксельным выравниванием колонок, вероятностями модели и фактами перевеса в одну строку;
       - **[3 ТЕСТ] Visual Bet Slip / Графический постер**: высокодетализированный PNG-купон (1080×1350) на движке Headless Chromium (`template/express_slip_template.html`) с эмблемами команд, неоновыми бейджами рынков, золотой плашкой коэффициента `4.02` и бонус-блоком БК;
  3. **База данных и расчёт ставок (`prediction/`)**:
     - Регистрация прогнозов в таблицах `predictions` и `prediction_items` со статусом `pending`;
     - `DBFixtureScoreProvider` с получением финального и перерывного счёта из `provider_fixtures` и `provider_fixtures_scores`;
     - Автоматический расчёт исходов через `SettlementWorker` (поддержка 1X2, Over/Under, BTTS, Double Chance);
     - Формирование открытого отчёта проходимости (`PerformanceReport`: WinRate, ROI, TotalProfit, ProfitFactor).
  4. **Интеграция в диспетчер и CLI**:
     - `-type express -style rich` (алиасы `richall`, `tables`): последовательная публикация всех 3 вариантов нативных таблиц Telegram (Bot API 10.1+);
     - `-type express -style rich1|rich2|rich3`: публикация конкретного варианта нативной таблицы;
     - `-type express -style 1|2|3|all`: публикация текстовых форматов и графического постера;
     - `-task SettleBets`: автоматический перерасчёт незавершённых прогнозов по сыгранным матчам и вывод отчёта эффективности.

---

### Фаза 9: Live Match Center и гибридная архитектура API (Archive & Live Polling) — СТАТУС: ВЫПОЛНЕНО
- [x] **Разработка легковесного Go REST API сервера**: пакет `server/` на чистом `net/http` с CORS, эндпоинтами `/health`, `/api/leagues`, `/api/leagues/{id}/archive`, `/api/rounds/{round_id}`, `/api/live`, `/api/fixtures/{id}/live`; интеграция в CLI `./lab -task ApiServer` (`-port`, `-static`, `-pollInterval`);
- [x] **In-play опрос с настраиваемым интервалом**: воркер `server/poller.go` и in-memory кэш `server/store.go` (`LiveStore`), интервал настраивается через флаг `-pollInterval` (по умолчанию 20s, e.g. 15s, 30s);
- [x] **Интерактивный архивный селектор сезонов и туров в Mini App**: кнопка в шапке `archiveSelectorTrigger` и Bottom Sheet `archiveBottomSheet` с чипами сезонов и сеткой туров, динамическая подгрузка 20-летнего архива из БД;
- [x] **Гибридная схема Cache-First**: эталонный конфиг `web/nginx.conf` (`try_files $uri @go_api_round`), клиентский fallback в `web/viewer/app.js` (`urls` со статикой и `/api/rounds/...`), Live Polling таймер в открытом Match Center.

---

### Фаза 10: Декаплинг генерации JSON от Telegram-постинга и отсечение устаревших постов — СТАТУС: ВЫПОЛНЕНО
- [x] **Устранение рассинхронизации Mini App на домене `s.pastir.me`**:
  - Создана символическая ссылка `/extssd/footballstats/public/data -> /extssd/footballstats/public/viewer/data` для мгновенной отдачи файлов туров `{league}_{season}_{round}.json` и `{league}_{season}_leaders.json` без 404 ошибок;
  - В `publisher/dispatcher.go` функция `syncViewerFiles` обновлена: автоматически синхронизирует всю директорию `data/` в публичный корень Nginx;
  - В `web/viewer/app.js`:
    - Устранён race condition при старте: `loadLeaguesCatalog` дожидается загрузки каталога перед рендером вкладок;
    - Добавлена поддержка 2-х и 3-х сегментных Telegram `startParam` (`8_28083_table`, `8_28083_leaders`);
    - Удалён устаревший хардкод раунда `407869` (Тур 6 АПЛ);
    - Вкладка турнирной таблицы `table` всегда детерминированно открывает актуальный тур `active.curr_round_id`;
    - Исправлен вызов `loadRoundData` при переключении архивных туров;
  - Обновлён кэш-бастинг в `web/viewer/index.html` (`v=20261101_v3`) и проведено полное развёртывание на `s.pastir.me`.
- [x] **Отсечение устаревших постов и предотвращение спама в канале**:
  - В `publisher/model.go` и `publisher/storage.go` добавлен статус `StatusSkipped = "skipped"` и метод `RecordSkipped(...)`;
  - Запрос `IsAlreadyPublished` расширен: `status IN ('published', 'skipped')` для исключения повторных проверок;
  - В `scheduler/db_task_provider.go` добавлены проверки временных окон устаревания:
    - Для `roundResults`: если с момента окончания тура прошло более 36 часов (`now.Sub(status.EndingAt) > 36h`), задача помечается как `SKIPPED`;
    - Для `weekendHub`: если первый матч тура уже стартовал (`now.After(firstMatchKickoff)`), задача помечается как `SKIPPED`;
- [x] **Регулярная фоновая актуализация данных Mini App**:
  - `SchedulerTick` и фоновый цикл планировщика автоматически запускают актуализацию JSON каталога, туров и лидеров (`syncAll`, `publish: false`), гарантируя 100% свежесть данных на диске даже при паузе в публикации постов.
- [x] **Унифицированная вкладка «Матчи» и навигатор туров `‹ 9 ТУР ›`**:
  - Устранено разделение на «Расписание» и «Результаты» — внедрен классический 3-вкладочный таб-бар: `📊 Таблица` | `⚽ Матчи` | `👑 Лидеры`;
  - Вверху вкладки «Матчи» добавлен интерактивный свитчер `‹ ТУР ›` с кнопками `‹` (листание назад до 1-го тура), `›` (листание вперед) и кликом по центру с открытием полной сетки туров сезона;
  - Добавлены смарт-бейджи статуса: `Текущий`, `LIVE`, `Прошлый тур`, `Архив`, `Предстоящий` и кнопка `[К текущему]`;
  - В `ActiveLeagueItem` бэкенда (`analytics/league.go`) добавлен полный массив туров сезона `rounds: []LeagueRoundItem`, благодаря чему клиент переключает туры мгновенно без дополнительных запросов;
  - Ссылки и кнопки Telegram обновлены на `_matches` с сохранением полной обратной совместимости для старых параметров `_fixtures` и `_results`;
  - Версия ассетов поднята до `v=20261101_v4` и полностью развернута на `s.pastir.me`.

---

### Фаза 11: Устранение критических багов навигатора туров, дефолтного раунда и актуализации таблицы — СТАТУС: ВЫПОЛНЕНО
- [x] **Исправление навигации в Mini App (`app.js`)**:
  - Устранена критическая ошибка `ReferenceError: targetRoundId is not defined` в `'use strict'` режиме внутри `navigateToRound` и `renderArchiveRounds`;
  - Нормализовано состояние `currentTab`: исключен сброс в устаревшие `'fixtures'` / `'results'` при открытии Match Center и возврате к матчам, благодаря чему вкладка «Матчи» (`matches`) всегда сохраняет активный статус в таб-баре;
  - В `openArchiveBottomSheet` добавлен мгновенный рендер сетки всех туров сезона из `active.rounds` без ожидания внешних API-запросов;
  - Исправлены touch-обработчики Pan & Zoom (`touchstart`, `touchmove`, `touchend`): исключен конфликт со скроллом на мобильных устройствах при просмотре вкладок `matches`, `table` и `leaders`.
- [x] **Отображение актуального тура по умолчанию**:
  - Внедрена функция `getDefaultRoundId()`: по умолчанию отображается текущий (или следующий предстоящий) тур `active.next_round_id` (fallback на `active.curr_round_id`);
  - При смене турнира (`switchLeague`) и первичном входе без явного `?round=` открывается текущий играемый тур (для АПЛ — Тур 9, Бундеслига — Тур 8, Серия А — Тур 10, Лига 1 — Тур 9, Ла Лига — Тур 11);
  - Вкладка «Таблица» (`table`) больше не привязана жестко к прошедшему завершенному туру, а детерминированно отображает актуальную турнирную таблицу текущего раунда.
- [x] **Актуализация турнирной таблицы и результатов тура**:
  - Зарегистрирован обработчик `"upsertStandingsSeasons"` в `api/handlers/function.go` и исправлен `conflictQuery` по бизнес-ключу `(participant_id, season_id, round_id)` вместо `id`;
  - В `api/handlers/standings.go` добавлена автоматическая дедупликация старых деталей турнирной таблицы при перезагрузке раунда;
  - В `provider_endpoints` обновлен параметр `standings.seasons` с включением связей `rule;details;group;form`;
  - Проведено полное обновление турнирных таблиц всех 5 текущих топ-лиг через `upsertStandingsSeasons`;
  - В `publisher/dispatcher.go` (`syncSingleLeagueData`) экспорт `NextRoundID` переведен на `analytics.GetRoundResults` и актуальные `Standings`: теперь текущий играемый тур экспортируется со всеми реальными счетами, статусами (FT/NS), HT счетами, статистикой и актуальной турнирной таблицей;
  - Версия ассетов поднята до `v=20261101_v6` и полностью развернута на `s.pastir.me`.

---

### Фаза 12: Исправление 8 дефектов Mini App, API туров и актуализации данных — СТАТУС: ВЫПОЛНЕНО
- [x] **1. Испания (Ла Лига) и результаты сыгранных матчей**:
  - В `web/viewer/app.js` метод `loadRoundData` переведен на приоритетный запрос живого API `/api/rounds/{round_id}?locale=ru` (с fallback на статичные JSON-файлы), гарантируя мгновенное отображение завершенных матчей тура;
  - Запущена регенерация статических данных через `./lab -task PublishToTelegram -type syncAll -publish=false`.
- [x] **2. Устранение бага отображения первого тайма `(null:null)`**:
  - В `web/viewer/app.js` исправлены проверки половинного счёта: `homeHT != null && awayHT != null` (вместо `!== undefined`, пропускавшего `null`), исключено появление `(null:null)` для несыгранных и частично заполненных матчей.
- [x] **3. Исправление выбора туров сезона дальше `текущий + 1`**:
  - В `analytics/round_archive.go` (`GetRoundResultsByRoundID` и `GetFixtureLiveDetails`) исправлено сканирование поля `odds.bookmaker_name`: вместо прямого сканирования в `string` применена промежуточная переменная `var bookmakerName *string`, предотвращающая падение сканирования с `can't scan into dest[30]: cannot scan null into *string` для предстоящих туров без котировок букмекеров. Теперь доступны все 38 туров сезона.
- [x] **4. Корректное отображение сыгранных матчей в турах (устранение `0/10`)**:
  - В `analytics/league.go` запрос `roundsQuery` дополнен подсчётом `finished`, `finished_count` и `total_matches` через агрегацию по `provider_fixtures`;
  - Поля добавлены в структуру `LeagueRoundItem` и экспортируются в `active_leagues.json`;
  - В `web/viewer/app.js` устранен хардкод `finished_count`, данные берутся напрямую из метаданных каждого тура.
- [x] **5. Удаление лишнего элемента в шапке**:
  - Из `web/viewer/index.html` удалена дублирующая кнопка `#archiveSelectorTrigger` (`#headerRoundBadge`) в правом верхнем углу; навигация полностью осуществляется через центральный интерактивный блок `‹ ТУР ›`.
- [x] **6. Подтягивание результатов сыгранных матчей Ла Лиги**:
  - В `564_27965_408950.json` и через API `/api/rounds/408950` обеспечена корректная отдача всех 6 завершенных матчей (FT), 1 live (1st) и 3 NS.
- [x] **7. Актуализация раздела «Лидеры»**:
  - В `server/handlers.go` и `server/server.go` реализован нативный REST API эндпоинт `/api/leaders?league_id=...&season_id=...&locale=ru`, рассчитывающий актуальных лидеров из БД на лету;
  - В `web/viewer/app.js` функция `loadLeadersData` переведена на приоритетную загрузку `/api/leaders` с fallback на статику.
- [x] **8. Корректный расчёт формы команды с учётом текущего тура (Ман Сити в АПЛ)**:
  - В `analytics/standings.go` исправлен расчет аргумента даты для `get_form_5last_rounds`: вместо даты старта следующего тура `nr.next_round_starting_at` (отсекавшей матчи, сыгранные в этот день) внедрен расчет `cutoff_date` на основе `ending_at + 1 day` целевого тура или `CURRENT_DATE + 1 day`;
  - Последнее поражение Манчестер Сити (Тур 9) теперь корректно входит в форму: `["W", "W", "W", "L", "L"]` при 9 сыгранных матчах.
- [x] **Деплой и верификация**:
  - Бинарник пересобран и сервер `ApiServer` перезапущен;
  - Ассеты Mini App обновлены до `v=20261101_v7` и синхронизированы в `/extssd/footballstats/public/`;
  - Все 8 пунктов успешно протестированы через `curl` на `https://s.pastir.me/`.

---

### Фаза 13: Исправление дефектов таблиц лиг, лидеров, навигации по турам и Live-статусов — СТАТУС: ВЫПОЛНЕНО
- [x] **1. Полная статистика в турнирных таблицах (W/D/L/GF/GA/GD) для всех лиг**:
  - Диагностирована причина нулевых показателей: в `provider_standings_details` отсутствовали строки для текущих туров Ла Лиги, Серии А, Бундеслиги и Лиги 1;
  - Выполнен `upsertStandingsSeasons -seasonIsCurrent true` с параметром `rule;details;group;form`, заполнивший 396–440 строк деталей для всех 5 текущих лиг;
  - Турнирные таблицы всех лиг отображают реальные данные (Барселона: 11 игр, 10 побед, 30 очков; Бавария: 8 игр, 18 очков; Лацио: 10 игр, 28 очков; ПСЖ: 9 игр, 20 очков).
- [x] **2. Синхронизация лидеров сезона (Эрлинг Холанд 9 голов)**:
  - Выполнен сбор `upsertTopScorers -seasonIsCurrent true`, заполнивший `provider_seasons_topscores` официальными данными Sportmonks;
  - В `analytics/player_leaders.go` функции `queryTopScorers`, `queryTopAssists`, `queryTopPoints` и `queryDiscipline` переведены на использование официальных данных `provider_seasons_topscores` с расчетом сыгранных матчей из `provider_lineups_{league_id}`;
  - Холанд (9 голов), Исак (6 голов), Эванилсон (4 ассиста + 4 гола), Райс (4 ассиста) корректно отдаются через `/api/leaders`.
- [x] **3. Доступность туров сезона дальше текущего (Тур 20 и до конца сезона)**:
  - Устранена ошибка сканирования `dest[12]` (`can't scan into dest[12]: cannot assign 0 1 into *int64`): для будущих туров (туры 13–38), где участники еще не определены в `provider_fixtures_participants`, значения `fh.participant_id` и `fa.participant_id` были NULL;
  - В `analytics/round_archive.go`, `round_fixtures.go` и `round_results.go` добавлено `COALESCE(fh.participant_id, 0)` и `COALESCE(fa.participant_id, 0)`;
  - Запрос `/api/rounds/407855` (Тур 20 АПЛ) возвращает HTTP 200 OK со всеми матчами тура.
- [x] **4. Отображение Live-матчей во всех лигах**:
  - В `web/viewer/app.js` внедрены функции `isMatchLive`, `isMatchFinished` и `formatLiveStateLabel`;
  - Расширен список Live-статусов Sportmonks (`1st`, `2nd`, `HT`, `et`, `2et`, `PEN`, `BRK`, `LIVE`), благодаря чему Live-матчи теперь корректно детектируются во всех лигах, а не только в Бундеслиге;
  - В карточке матча в Live-режиме отображается текущий счёт красным цветом с пульсирующим индикатором LIVE и текущей фазой матча вместо коэффициентов и времени начала;
  - Защищено отображение счёта 1-го тайма от строкового значения `"null"`.
- [x] **5. Деплой и сборка**:
  - Пересобран `lab`, сервер `ApiServer` перезапущен в режиме демона;
  - Ассеты Mini App обновлены до `v=20261101_v8` и выгружены на `s.pastir.me`;
  - Проведена пакетная генерация актуальных JSON-данных через `syncAll`.

---

### Фаза 14: Продакшен домен data-lab.live, устранение ошибки 400 invalid url format и systemd юнит — СТАТУС: ВЫПОЛНЕНО
- [x] **1. Устранение ошибки 400 `/api/rounds/{round_id}` при открытии продакшена `data-lab.live`**:
  - Диагностирована первопричина: на домене `data-lab.live` в корне статики отсутствовал файл `active_leagues.json` (отдавался 404), из-за чего `loadLeaguesCatalog` завершался без данных, `roundId` оставался пустым, и Mini App отправлял некорректный запрос `/api/rounds/?locale=ru`;
  - В `web/viewer/app.js` метод `loadLeaguesCatalog` переведён на приоритетное получение каталога через живой эндпоинт `/api/leagues` (с fallback на статические файлы `active_leagues.json` и `viewer/active_leagues.json`);
  - В `loadRoundData` добавлена валидация `targetRoundId` и автоматический fallback на `/api/rounds?league_id=${leagueId}&locale=ru` при отсутствии явного ID тура;
  - В `server/handlers.go` метод `HandleRoundOnDemand` дополнен безопасным разрешением тура: если `round_id` отсутствует в URL, сервер автоматически определяет активный тур лиги (по умолчанию АПЛ, ID 8) и отдаёт HTTP 200 вместо 400;
  - В `server/server.go` зарегистрированы маршруты `/api/rounds` и `/api/rounds/`.
- [x] **2. Настройка статики и сервиса на продакшен-сервере `data-lab.live`**:
  - Созданы недостающие символические ссылки в `/var/www/domains/data-lab.live/public/` (`active_leagues.json`, `round_data.json`, `leaders.json`);
  - Пересобран бинарник `lab` с помощью Go 1.27 (`/usr/local/go/bin/go`);
  - Обновлён и перезапущен systemd-сервис `fdl-api.service`;
  - Версия ассетов поднята до `v=20261101_v10`;
  - Успешно верифицирована работа `https://data-lab.live/`: `/api/rounds/?locale=ru` (HTTP 200), `/active_leagues.json` (HTTP 200), `/api/leaders` (`X-Cache: HIT`).

---

### Фаза 15: Европейский медиа-дайджест (Press Digest) — СТАТУС: ВЫПОЛНЕНО
- [x] **1. База данных и схема дедупликации**:
  - Создана и применена миграция `db/migrations/005_create_press_articles.sql` с таблицей `press_articles` и ограничением `UNIQUE (original_url)`;
  - Индексы по `status`, `published_at DESC`, `source` для моментальной выборки свежих драфтов.
- [x] **2. Модуль европейской прессы `press/`**:
  - `press/model.go`: DTO статей и проверенные RSS-источники (BBC Sport, MARCA Primera División, MARCA Fichajes, The Guardian);
  - `press/fetcher.go`: отказоустойчивый RSS 2.0 XML-парсер с извлечением заголовков, лидов, обложек (`media:content`, `media:thumbnail`, `enclosure`) и дат публикации;
  - `press/filter.go`: алгоритм `CalculateHypeScore` (0–100) — фильтрация шума (женский футбол, академии) и начисление баллов за топ-клубы (+30), мировых звёзд (+25), отставки (+40), трансферы (+35), скандалы (+30);
  - `press/storage.go`: слой работы с PostgreSQL через `pgxpool.Pool` (`SaveArticles` с `ON CONFLICT DO NOTHING`, `GetTopDraftArticles`, `UpdateArticleSummary`, `MarkArticlePublished`);
  - `press/summarizer.go`: интеграция с Google Gemini 3.5 Flash через OpenAI-совместимый эндпоинт Google API с поддержкой 2500 токенов (учет reasoning tokens);
  - `press/dispatcher.go`: форматирование поста для Telegram с флагом страны/издания, цитатой, ссылкой на оригинал и инлайн-кнопками «📰 Читать оригинал» и «📊 Открыть Mini App» (поддержка `LinkPreviewOptions` для крупной обложки);
  - `press/pipeline.go`: оркестратор `RunPipeline` полного цикла.
- [x] **3. Интеграция в CLI `./lab`**:
  - В `cmd/lab/main.go` добавлена задача `-task PressDigest` (алиас `Press`) с флагами `-publish`, `-conf`;
  - Добавлена справка в консольном выводе.
- [x] **4. Тестирование и верификация**:
  - 100% покрытие бизнес-логики unit-тестами `press/filter_test.go` и `press/fetcher_test.go` (все тесты зеленые);
  - Сквозное dry-run тестирование: парсер загрузил 230 статей из европейских таблоидов, сохранил их в БД, повторный опрос корректно пропустил дубликаты (0 новых), Gemini успешно перевел и оформил топ-новость на русском языке.

---

### Фаза 16: Карточки футбольных аномалий и драмы (Banter Cards) — СТАТУС: ВЫПОЛНЕНО
- [x] **1. Аналитический модуль детекции аномалий `analytics/anomalies.go`**:
  - `detectXGRobbery`: поиск матчей с аномальным расхождением xG (перевес $\ge 1.7$ xG при поражении/ничьей фаворита, «Автобус недели»);
  - `detectLateDrama`: поиск голов на 90+ минутах, определивших исход матча («Валидол 90+»);
  - `detectSuperSub`: детекция вышедших на замену игроков, забивших $2+$ гола («Джокер тура»);
  - `detectGoalkeeperWall`: поиск вратарских перформансов с $8+$ сейвами («Вратарь-Стена / Осьминог»);
  - Расчёт нормализованного рейтинга силы аномалии `SeverityScore` (0–100) для ранжирования топ-события тура.
- [x] **2. Слой нормализации матчевой статистики**:
  - Функция `normalizeStatKey` и извлечение показателей владения (`BALL_POSSESSION`), ударов (`SHOTS_TOTAL`) и сейвов (`SAVES`) независимо от регистра и кодов провайдера Sportmonks.
- [x] **3. Визуальный шаблон и ViewModel (`viewmodel/banter_card.go`, `template/banter_card_template.html`)**:
  - Генерация ViewModel с подготовкой цветовых схем, долей для сравнительных полос (Dual Progress Bars) и бейджей;
  - HTML/CSS шаблон 1200x1200px с темным премиальным дизайном, логотипами клубов, сравнением xG/ударов/владения/сейвов и ироничной цитатой тура.
- [x] **4. Рендерер и копирайтер (`renderer/banter_card.go`, `copywriter/banter.go`)**:
  - Рендеринг в PNG через Chromium (`RenderBanterCardPNG`);
  - Детерминированная генерация вовлекающего русскоязычного текста поста с Telegram HTML-разметкой и ссылкой на Mini App.
- [x] **5. Диспетчер и CLI интеграция (`publisher/dispatcher_banter.go`, `cmd/lab/main.go`)**:
  - Команды `./lab -task AnomalyCard` / `./lab -task BanterCard` и `./lab -task PublishToTelegram -type banter`;
  - Поддержка параметров `-leagueId`, `-roundId`, `-publish`, `-output`;
  - Проверено на реальных турах АПЛ (найдены аномалии: Фулхэм vs Кристал Пэлас с xG 3.88 против 1.03, Тоттенхэм vs Астон Вилла с победным голом на 90+8').
- [x] **6. Интеграция комментариев Telegram и живая публикация**:
  - Реализован стандарт публикаций Варианта А (без инлайн-кнопок): Telegram нативно отображает под каждым постом плашку «💬 Прокомментировать» с выделенным изолированным тредом обсуждения;
  - В `template/banter_card_template.html` исправлена адаптивная верстка плашки счёта: зафиксирован `white-space: nowrap`, `min-width: 200px` и неразрывные пробелы `&nbsp;:&nbsp;` для исключения переноса второй цифры счёта на новую строку;
  - Успешно выполнены боевые публикации: пост #176 (Челси 4 : 3 Брайтон) с нативным тредом комментариев, счётом в одну строку и глубокой ссылкой в Mini App на матч.
- [x] **7. Автопилот в Scheduler и нативные ссылки Mini App**:
  - В `scheduler/db_task_provider.go` добавлена автоматическая генерация задачи `PubTypeBanter` при наступлении события закрытия тура `IsReadyForClosing` (если выявлена аномалия с `SeverityScore >= 75`);
  - Прямые ссылки на Mini App унифицированы до `https://t.me/football_data_lab_bot/data_lab` во всех публикациях (результаты, расписания, пресса, карточки аномалий, серии, экспрессы), что обеспечивает мгновенное нативное открытие Telegram Mini App в один клик без внешних редиректов и сбоев парсинга параметров Telegram;
  - В `press/dispatcher.go` пресса также переведена на чистый нативный формат комментариев со ссылками на оригинал и Mini App в тексте.
- [x] **8. Уточнение аналитического движка поздней драмы и суперкамбэков (`analytics/anomalies.go`, `copywriter/banter.go`)**:
  - Исключены «голы престижа» (consolation goals): если фаворит уверенно вёл (например, Челси 4:2) и соперник забил на 90+6' (4:3), такой гол ничего не решал и не классифицируется как драма;
  - Реализована детекция волевых камбэков в концовке (`Камбэк тура` с наивысшим `SeverityScore = 98.0`): рассчитывается точный таймлайн поздних голов, число мячей и время камбэка («забили 2 гола за 3 минуты: 88', 90+1'»);
  - Для матча Ипсвич — Фулхэм сгенерирован пост и постер суперкамбэка (пост #177), внедрена динамическая подсветка команд-победителей в шаблоне постера и цветовая гамма бейджей (amber для камбэков, cyan для спасений 90+);
  - Написаны детерминированные unit-тесты в `analytics/anomalies_test.go` (`TestDetectLateDrama_Comeback`, `TestDetectLateDrama_ConsolationGoalIgnored`, `TestDetectLateDrama_LateEqualizer`, `TestPluralHelpers`).
- [x] **9. Унификация стиля Telegram-публикаций (Нативные комментарии + Лаконичные ссылки)**:
  - Во всех публикациях (`round_results`, `round_fixtures`, `round_digest`, `leaders`, `facts`, `banter`, `table`, `press`) инлайн-клавиатура переведена на `keyboard = nil`, что активирует родную плашку Telegram («Прокомментировать» / «N комментариев») со счётчиком в реальном времени и изолированным тредом;
  - Ссылки на Telegram Mini App размещены внутри текста поста в лаконичном аккуратном формате: `<a href="...">Все матчи</a> · <a href="...">Турнирная таблица</a>` (с глубокими ссылками `?startapp={leagueID}_matches` и `?startapp={leagueID}_table`);
  - Сохранены глубокие гиперссылки на счетах и котировках матчей для быстрого входа в Mini App.
- [x] **10. Подготовка бесшовного деплоя Mini App (Zero-Copy Deployment)**:
  - В `.gitignore` добавлены динамические runtime-данные Mini App (`web/viewer/data/`, `web/viewer/*.json`, `web/viewer/leaders*.png`, `web/viewer/media/`), предотвращающие конфликты при `git pull` на продакшене;
  - Из индекса Git исключены сгенерированные файлы туров и таблиц, в репозитории оставлены только чистые исходники фронтенда (`app.js`, `index.html`, `style.css`, статические баннеры).
- [x] **11. Общеевропейский смарт-экспресс, альбом лидеров Топ-5 лиг и Chromium-автодетект**:
  - **Общеевропейский экспресс тура (`analytics/express_builder.go`, `publisher/dispatcher_express.go`)**: при запуске без `-leagueId` (дефолт в кроне) анализируются матчи ближайших туров со всех активных лиг Европы (АПЛ, Ла Лига, Серия А, Бундеслига, Лига 1), внедрен весовой бонус за разнообразие лиг, в пост выводятся флаги стран и турниры;
  - **Нативные комментарии под экспрессом**: убраны все инлайн-клавиатуры (`keyboard = nil`), ссылки на БК и Mini App перенесены в аккуратный футер текста поста (`<a href="...">Фрибет FONBET</a> · <a href="...">Все матчи</a>`), активирована родная плашка комментариев Telegram;
  - **Альбом лидеров Топ-5 лиг Европы (Вариант А, `publisher/dispatcher.go`)**: при вызове `-type leaders` без указания лиги автоматически рендерятся 5 графических карточек 1920×1080 (по одной на каждую топ-лигу) и отправляются альбомом (`SendMediaGroup`) с единой сводкой снайперов и ссылками на Mini App;
  - **Золотая бутса Европы (Вариант Б, `publisher/dispatcher.go`)**: команда `-type leadersEuro` (или `-type leaders -style euro`) формирует сводный пан-европейский топ-10 бомбардиров и ассистентов с медалями, флагами и клубами;
  - **Headless Chromium в `renderer/standings.go`**: добавлено автоопределение путей `/usr/bin/chromium`, `/usr/bin/chromium-browser`, `/snap/bin/chromium` для стабильной работы рендеринга на Rocky Linux / RHEL / Ubuntu без необходимости ручных симлинков.
- [x] **12. Исправление актуальности турнирных таблиц, оптимизация шапки экспресса и стабильность публикаций**:
  - **Актуальность турнирных таблиц (`analytics/standings.go`)**: устранена отдача устаревших плейсхолдеров для незавершённых туров. Внедрён CTE `effective_round`: если запрашиваемый тур не завершён (`finished = false`), запрос автоматически подтягивает турнирную таблицу и форму команд из последнего завершённого тура (`last_round_id`). Теперь при просмотре предстоящего 10 тура АПЛ и других лиг в Mini App и генераторе постеров всегда отображаются актуальные очки и сыгранные матчи (Манчестер Сити: 9 игр, 21 очко);
  - **Оптимизация шапки аналитического экспресса (`publisher/dispatcher_express.go`)**: убрано дублирование общего коэффициента из основного заголовка (`DATA LAB | АНАЛИТИЧЕСКИЙ ЭКСПРЕСС`), размер заголовков уменьшен (Size 4 и Size 5), коэффициент вынесен в аккуратный подзаголовок лиг;
  - **Надёжность отправки медиа-альбомов (`publisher/telegram.go`)**: в `SendMediaGroup` добавлен механизм повторных попыток (до 3 попыток с нарастающей задержкой) для защиты от сбросов TCP-соединения (`connection reset by peer`) при отправке тяжелых медиа-пакетов;
  - **Идемпотентность и защита от ложных записей**: подтверждено, что запись публикации в БД (`telegram_publications`) выполняется строго после успешного подтверждения доставки сообщения от Telegram API (ID > 0). При сетевых сбоях или ошибках рендеринга запись в БД не создаётся, что позволяет повторять публикацию без конфликтов.

---

### Дорожная карта и дальнейшие шаги
- [ ] **Live Match Anomaly Trigger (Мгновенная детекция по окончании матча)**:
  - При подключении live-коллекторов Sportmonks: переход от проверки только закрытия тура (`IsReadyForClosing`) к триггеру по факту завершения конкретного матча (`state_short == "FT"`);
  - Для экстраординарных событий (`SeverityScore >= 90.0`, например, суперкамбэк в добавленное время или сенсационный разгром) публикация карточки и поста будет происходить мгновенно по горячим следам, не дожидаясь окончания всех матчей тура;
  - Архитектура `analytics/anomalies.go` уже полностью готова к этому (функции детекции принимают единичный матч `RoundResultMatch`).

