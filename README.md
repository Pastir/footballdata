# Football Data Lab ⚽📊

Автоматизированный аналитический конвейер спортивных футбольных данных со сквозной генерацией инфографики, публикацией структурированных постов в Telegram-канал и интерактивным Telegram Mini App (Web App Match Center).

---

## 1. Стек технологий

* **Язык**: Go 1.21+ (Idiomatic Go, `pgx/v4`, `zap`, `ManageServer`)
* **Базы данных**: PostgreSQL (основное хранилище и провайдер данных), ClickHouse (историческая статистика)
* **Вёрстка и рендеринг**: HTML5, CSS3, `html/template`, Headless Chromium
* **Telegram**: Telegram Bot API (форматирование HTML-постов, deep links, silent pin, inline-клавиатуры)
* **Telegram Mini App**: Vanilla JS/CSS Single Page Application (таблица, предматчевый H2H/xG, составы, результаты, селектор лиг)

---

## 2. Быстрый старт

### 2.1. Сборка бинарника
```bash
go build -o lab ./cmd/lab/main.go
```

### 2.2. Запуск без параметров (справка)
```bash
./lab
```

---

## 3. Справочник консольных команд публикатора (`./lab`)

Все операции генерации аналитики, экспорта графики и публикаций в Telegram выполняются через единый CLI-интерфейс:

```bash
./lab -conf config.yaml -task PublishToTelegram -type <TYPE> [ФЛАГИ]
```

### 3.1. Основные команды

| Команда | Описание | Пример вызова |
|---|---|---|
| `-type table` | Генерация турнирной таблицы (PNG-инфографика + текст с акцентами + публикация) | `./lab -conf config.yaml -task PublishToTelegram -type table -leagueId 8 -publish true` |
| `-type roundFixtures` | Расписание следующего тура: котировки 1X2 с deep links, H2H, Hero-xG, Silent Pin | `./lab -conf config.yaml -task PublishToTelegram -type roundFixtures -leagueId 8 -publish true` |
| `-type roundResults` | Результаты тура: счёт, авторы голов, карточки, инлайн-кнопки в Mini App | `./lab -conf config.yaml -task PublishToTelegram -type roundResults -leagueId 8 -publish true` |
| `-type previewMatch` | Превью отдельного матча по fixtureId | `./lab -conf config.yaml -task PublishToTelegram -type previewMatch -leagueId 8 -fixturesId 18535518 -publish true` |
| `-type roundDigest` | Интерактивный дайджест тура с кнопкой открытия Mini App | `./lab -conf config.yaml -task PublishToTelegram -type roundDigest -leagueId 8 -publish true` |
| `-type catalog` | Экспорт активных лиг (`active_leagues.json`) и синхронизация Mini App | `./lab -conf config.yaml -task PublishToTelegram -type catalog` |

### 3.2. Локальный режим тестирования (Dry-Run)
Для локальной проверки рендеринга и формирования данных без отправки в Telegram используется флаг `-publish false`:
```bash
# Тест таблицы с сохранением в локальный файл:
./lab -conf config.yaml -task PublishToTelegram -type table -leagueId 8 -output test_table.png -publish false

# Тест расписания (генерация round_data.json и вывод текста поста в лог):
./lab -conf config.yaml -task PublishToTelegram -type roundFixtures -leagueId 8 -publish false

# Тест результатов тура:
./lab -conf config.yaml -task PublishToTelegram -type roundResults -leagueId 8 -publish false
```

---

## 4. Таблица CLI-флагов

| Флаг | Тип | По умолчанию | Описание |
|---|---|---|---|
| `-conf` | `string` | `./config.yaml` | Путь к конфигурационному файлу |
| `-task` | `string` | `""` | Имя задачи: `PublishToTelegram` |
| `-type` | `string` | `""` | Тип контента: `table`, `roundFixtures`, `roundResults`, `previewMatch`, `roundDigest`, `catalog` |
| `-leagueId` | `int` | `8` | ID лиги (`8` — АПЛ, `564` — Ла Лига, `384` — Серия А, `82` — Бундеслига, `301` — Лига 1) |
| `-roundId` | `int` | `0` | ID тура (при `0` автоматически определяется актуальный тур) |
| `-fixturesId` | `string` | `""` | ID матча (для задачи `previewMatch`) |
| `-publish` | `bool` | `false` | Отправка в Telegram (`true` — боевая публикация, `false` — только локальная генерация) |
| `-asDocument`| `bool` | `false` | Отправка инфографики несжатым файлом (`true` / `false`) |
| `-output` | `string` | `""` | Путь для сохранения локального PNG-файла |
| `-locale`, `-l` | `string` | `ru` | Локализация названий команд и текстов (`ru`, `en`, `ua`) |
| `-timezone` | `string` | `MSK` | Отображаемый часовой пояс начала матчей |

---

## 5. Коллекторы данных Sportmonks (`main.go`)

Фоновые команды сбора и обновления справочников и статистики из Sportmonks API:

```bash
# Обновление справочников
go run main.go -conf config.yaml -task upsertTypes
go run main.go -conf config.yaml -task upsertLeagues
go run main.go -conf config.yaml -task upsertSeasons
go run main.go -conf config.yaml -task upsertStages

# Синхронизация текущего сезона
go run main.go -conf config.yaml -task upsertRounds
go run main.go -conf config.yaml -task upsertSeasonsTeams -seasonIsCurrent true
go run main.go -conf config.yaml -task upsertStandingsSeasons -seasonIsCurrent true

# Сбор матчей, таймлайнов и статистики
go run main.go -conf config.yaml -task upsertFixturesSeasons -seasonIsCurrent true
go run main.go -conf config.yaml -task upsertFixturesSeasonsTimeline -seasonIsCurrent true
go run main.go -conf config.yaml -task upsertFixturesSeasonsStatistics -seasonIsCurrent true

# Сбор коэффициентов букмекеров
go run main.go -conf config.yaml -task upsertOdds -startDate YYYY-MM-DD -endDate YYYY-MM-DD

# Сбор ожидаемых голов (xG)
go run main.go -conf config.yaml -task upsertExpectedFixtures -startDate YYYY-MM-DD -endDate YYYY-MM-DD
```

---

## 6. Документация проекта

* [PROJECT_SPEC.md](PROJECT_SPEC.md) — Генеральная архитектурная спецификация Football Data Lab V2.
* [CLI_UNIFICATION_PLAN.md](CLI_UNIFICATION_PLAN.md) — Детальный план реализации и хронология этапов CLI-унификации.
* [AGENTS.md](AGENTS.md) — Правила и архитектурные ограничения для AI / Coding Agent.
