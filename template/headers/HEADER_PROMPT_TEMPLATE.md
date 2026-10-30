# Football Data Lab — Единый стандарт и промпт шапок сезонов

Документ фиксирует утверждённую концепцию и универсальный шаблон промпта для генерации сезонных шапок лиг (`header.png`).

---

## 1. Архитектура и хранение

Каждая шапка генерируется под конкретный сезон лиги с **запечённым годом сезона** (`SEASON 2026/27`).

### Пути сохранения (Data Flow):
* **Основной путь сезона**: `/extssd/footballstats/public/media/{LeagueID}/{SeasonID}/header.png`
* **Фолбек на уровне лиги**: `/extssd/footballstats/public/media/{LeagueID}/header.png`
* **Шаблоны проекта**: `template/headers/{LeagueID}.png`

### Технические параметры:
* **Разрешение**: строго **`2043 × 770` px** (соотношение ~2.65:1, рассчитано под HTML-контейнер `1920 × 724` в `football_table_template.html`, `round_fixtures_template.html` и др.).
* **Формат**: PNG, 8-bit RGB, sRGB.

---

## 2. 5-зонная сетка композиции

```text
┌────────────────────────────────────────────────────────────────────────────────────────┐
│ [ЗОНА 1: ЛЕВО]          [ЗОНА 2: ЦЕНТР]                     [ЗОНА 3 & 4: ПРАВО & ВВЕРХ]│
│                                                                                        │
│  ┌──────────┐            {COUNTRY} {FLAG}                          FOOTBALL DATA LAB   │
│  │  ЛЕВЫЙ   │             {LEAGUE_NAME}                            ─────────────────   │
│  │  БЕЙДЖ   │           {LEAGUE_SUBTITLE}                         MORE THAN RESULTS    │
│  │  ЛИГИ    │          SEASON {SEASON_YEAR}                                            │
│  │ [Свечение│          ════════════════════                  [АРХИТЕКТУРНЫЕ СИМВОЛЫ]   │
│  │   неона] │                                                [ОФИЦИАЛЬНЫЙ КУБОК ЛИГИ]  │
│  └──────────┘                                                [ШЕЛКОВЫЙ ФЛАГ/ЛЕНТЫ]     │
│                                                                                        │
│ ┌────────────────────────────────────────────────────────────────────────────────────┐ │
│ │ [ЗОНА 5: КАПСУЛА СТАТ-БАРА]                                                        │ │
│ │ 🏟 {TEAMS} КОМАНД │ 📅 {ROUNDS} ТУРОВ │ 🏆 {TITLE_NAME} │ 📈 STATS & INSIGHTS (FDL)│ │
│ └────────────────────────────────────────────────────────────────────────────────────┘ │
│ [НЕОНОВАЯ ЛАЗЕРНАЯ ЛИНИЯ ГОРИЗОНТА ПО САМОМУ НИЗУ]                                    │
└────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Универсальный Master-Prompt

Используйте этот шаблон для генерации шапки любой новой лиги, заменяя переменные в `{ФИГУРНЫХ_СКОБКАХ}`:

```text
A cinematic, ultra-wide panoramic sports broadcast header graphic for {LEAGUE_NAME} Season {SEASON_YEAR}, designed for Football Data Lab.

1. LEFT LOGO:
   The official {LEAGUE_NAME} emblem/crest floating inside a rounded frosted glass card with a glowing {NEON_ACCENT_COLOR} rim halo. Crisp official sponsor branding ({SPONSOR_NAME}).

2. CENTER TYPOGRAPHY:
   - Subtle top country tag: '{COUNTRY_NAME_UPPER} {COUNTRY_FLAG_EMOJI}' with delicate horizontal divider lines.
   - Massive bold 3D metallic title: '{LEAGUE_NAME_UPPER}' in polished chrome with {NEON_ACCENT_COLOR} side bevels.
   - Elegant subtitle: '{LEAGUE_SUBTITLE_UPPER}' with wide letter-spacing.
   - Prominent season text: 'SEASON {SEASON_YEAR}' in bold clean white typography.
   - Sleek glowing accent underline bar in {COUNTRY_OR_LEAGUE_COLORS} gradient.

3. BACKGROUND & RIGHT:
   - Background: Panoramic night skyline featuring iconic architectural landmarks of {COUNTRY_NAME} ({LANDMARKS_LIST}) illuminated by volumetric stadium floodlights, lens flares, and deep midnight blue/purple stadium haze.
   - Right foreground: The authentic official championship trophy of {LEAGUE_NAME} ({TROPHY_NAME}) with metallic golden/silver shine, engraved details, and official tournament ribbons.
   - A dynamic waving silk {COUNTRY_NAME} national flag (or league silk banner) flowing gracefully across the scene behind the trophy.

4. TOP RIGHT CORNER:
   Crisp, minimalist luxury typography:
   'FOOTBALL DATA LAB'
   ──────────────────
   'MORE THAN JUST RESULTS'

5. BOTTOM STAT BAR:
   A full-width translucent dark glass capsule pill container with subtle neon {NEON_ACCENT_COLOR} edge glow, containing 4 clean metric blocks with glowing icons:
   - [Stadium icon]: '{TEAM_COUNT} КОМАНД' (subtitle: '{LEAGUE_LOCAL_NAME}')
   - [Calendar icon]: '{ROUND_COUNT} ТУРОВ' (subtitle: 'РЕГУЛЯРНЫЙ СЕЗОН')
   - [Trophy icon]: '{TROPHY_SHORT_NAME}' (subtitle: 'БОРЬБА ЗА ТИТУЛ')
   - [Analytics graph icon]: 'STATS & INSIGHTS' (subtitle: 'FOOTBALL DATA LAB')

6. LIGHTING & COLOR GRADING:
   {COLOR_PALETTE_DESCRIPTION}, intense nocturnal stadium floodlight beams, volumetric atmosphere, vibrant neon laser beam along the bottom edge. Broadcast TV package quality, zero distortion, razor-sharp typography.
```

---

## 4. Готовые пресеты для топ-лиг

### 🇮🇹 Серия А (Италия) — `league_id: 384`
* `{LEAGUE_NAME}`: `SERIE A`
* `{SPONSOR_NAME}`: `enilive` (шестиногий пес)
* `{COUNTRY_NAME}`: `ITALY`, флаг 🇮🇹
* `{LEAGUE_SUBTITLE}`: `CAMPIONATO ITALIANO`
* `{LANDMARKS}`: `Roman Colosseum, Duomo di Milano`
* `{TROPHY_NAME}`: `Coppa Campioni d'Italia (Scudetto Trophy)`
* `{COUNTRY_COLORS}`: `Italian tricolor (green, white, red)`
* `{TEAMS}`: `20`, `{ROUNDS}`: `38`, `{TROPHY_SHORT_NAME}`: `SCUDETTO`
* `{NEON_ACCENT_COLOR}`: `cyan and azure blue (#008fd7)`

### 🏴󠁧󠁢󠁥󠁮󠁧󠁿 Английская Премьер-лига — `league_id: 8`
* `{LEAGUE_NAME}`: `PREMIER LEAGUE`
* `{COUNTRY_NAME}`: `ENGLAND`, флаг 🏴󠁧󠁢󠁥󠁮󠁧󠁿
* `{LEAGUE_SUBTITLE}`: `THE GREATEST SHOW ON EARTH`
* `{LANDMARKS}`: `Tower Bridge, Big Ben, London stadium lights`
* `{TROPHY_NAME}`: `Official Premier League Trophy with golden crown and purple ribbons`
* `{COUNTRY_COLORS}`: `Premier League royal purple (#38003c) and electric cyan`
* `{TEAMS}`: `20`, `{ROUNDS}`: `38`, `{TROPHY_SHORT_NAME}`: `CHAMPIONS`
* `{NEON_ACCENT_COLOR}`: `royal purple and neon cyan`

### 🇪🇸 Ла Лига (Испания) — `league_id: 564`
* `{LEAGUE_NAME}`: `LALIGA`
* `{SPONSOR_NAME}`: `EA SPORTS`
* `{COUNTRY_NAME}`: `SPAIN`, флаг 🇪🇸
* `{LEAGUE_SUBTITLE}`: `EL PODER DE NUESTRO FÚTBOL`
* `{LANDMARKS}`: `Plaza de Cibeles Madrid, Sagrada Família Barcelona`
* `{TROPHY_NAME}`: `Trofeo de la Liga (Silver Cup with handles)`
* `{COUNTRY_COLORS}`: `Spanish red and golden yellow`
* `{TEAMS}`: `20`, `{ROUNDS}`: `38`, `{TROPHY_SHORT_NAME}`: `CAMPEÓN`
* `{NEON_ACCENT_COLOR}`: `crimson red and warm gold`

### 🇩🇪 Бундеслига (Германия) — `league_id: 82`
* `{LEAGUE_NAME}`: `BUNDESLIGA`
* `{COUNTRY_NAME}`: `GERMANY`, флаг 🇩🇪
* `{LEAGUE_SUBTITLE}`: `DIE LIGA DER MEISTER`
* `{LANDMARKS}`: `Brandenburg Gate, Berlin, iconic modern German stadiums`
* `{TROPHY_NAME}`: `Meisterschale (Championship Shield plate with gemstones)`
* `{COUNTRY_COLORS}`: `German black, red, and gold`
* `{TEAMS}`: `18`, `{ROUNDS}`: `34`, `{TROPHY_SHORT_NAME}`: `MEISTERSCHALE`
* `{NEON_ACCENT_COLOR}`: `Bundesliga vibrant red (#d20515) and pure white`

### 🇫🇷 Лига 1 (Франция) — `league_id: 301`
* `{LEAGUE_NAME}`: `LIGUE 1`
* `{SPONSOR_NAME}`: `McDonald's`
* `{COUNTRY_NAME}`: `FRANCE`, флаг 🇫🇷
* `{LEAGUE_SUBTITLE}`: `LE CHAMPIONNAT DE FRANCE`
* `{LANDMARKS}`: `Eiffel Tower, Arc de Triomphe illuminated at night`
* `{TROPHY_NAME}`: `L'Hexagoal (Trophy with hexagonal modern design)`
* `{COUNTRY_COLORS}`: `French blue, white, and red`
* `{TEAMS}`: `18`, `{ROUNDS}`: `34`, `{TROPHY_SHORT_NAME}`: `CHAMPIONNAT`
* `{NEON_ACCENT_COLOR}`: `lime-cyan neon and French royal blue`
