/**
 * Football Data Lab — Telegram Mini App Viewer Logic
 * Handles Telegram WebApp lifecycle, URL parameters, tab switching,
 * touch pan/zoom gestures, and interactive Match Center.
 */

(function () {
  'use strict';

  // 1. Инициализация Telegram WebApp SDK (если доступен)
  const tg = window.Telegram?.WebApp;
  if (tg) {
    tg.ready();
    tg.expand();
    try {
      tg.enableClosingConfirmation();
    } catch (_) {}
  }

  // 2. Чтение URL параметров и Telegram start_param
  const urlParams = new URLSearchParams(window.location.search);
  const hashRaw = window.location.hash.startsWith('#') ? window.location.hash.substring(1) : window.location.hash;
  const hashParams = new URLSearchParams(hashRaw);

  const startParam = (tg?.initDataUnsafe?.start_param ||
                     urlParams.get('tgWebAppStartParam') ||
                     hashParams.get('tgWebAppStartParam') ||
                     urlParams.get('startapp') ||
                     hashParams.get('startapp') ||
                     urlParams.get('start_param') ||
                     hashParams.get('start_param') || '').trim();

  let leagueId = urlParams.get('league') || urlParams.get('league_id') || '';
  let seasonId = urlParams.get('season') || urlParams.get('season_id') || '';
  let roundId = urlParams.get('round') || urlParams.get('round_id') || '';
  let initialTab = urlParams.get('tab') || urlParams.get('view') || '';
  let targetMatchId = urlParams.get('match') || urlParams.get('fixture_id') || '';

  // startParam имеет НАИВЫСШИЙ приоритет, так как он задается при клике на кнопку поста или матча
  if (startParam) {
    if (startParam.startsWith('fixture_')) {
      const rest = startParam.replace('fixture_', '');
      const parts = rest.split('_');
      if (parts.length >= 4) {
        leagueId = parts[0];
        seasonId = parts[1];
        roundId = parts[2];
        targetMatchId = parts[3];
      } else {
        targetMatchId = rest;
      }
      initialTab = 'fixtures';
    } else if (startParam.startsWith('match_')) {
      const rest = startParam.replace('match_', '');
      const parts = rest.split('_');
      if (parts.length >= 4) {
        leagueId = parts[0];
        seasonId = parts[1];
        roundId = parts[2];
        targetMatchId = parts[3];
      } else {
        targetMatchId = rest;
      }
      initialTab = 'fixtures';
    } else {
      const parts = startParam.split('_');
      if (parts.length >= 4) {
        leagueId = parts[0];
        seasonId = parts[1];
        roundId = parts[2];
        initialTab = parts[3];
      } else if (parts.length === 3) {
        // Формат: <league_id>_<season_id>_<tab> (например, 8_28083_table, 8_28083_matches)
        leagueId = parts[0];
        seasonId = parts[1];
        if (parts[2] === 'table' || parts[2] === 'matches' || parts[2] === 'fixtures' || parts[2] === 'results' || parts[2] === 'leaders') {
          initialTab = (parts[2] === 'fixtures' || parts[2] === 'results') ? 'matches' : parts[2];
        } else {
          roundId = parts[2];
          initialTab = 'matches';
        }
      } else if (parts.length === 2) {
        // Формат: <league_id>_<tab>
        leagueId = parts[0];
        if (parts[1] === 'table' || parts[1] === 'matches' || parts[1] === 'fixtures' || parts[1] === 'results' || parts[1] === 'leaders') {
          initialTab = (parts[1] === 'fixtures' || parts[1] === 'results') ? 'matches' : parts[1];
        } else {
          roundId = parts[1];
          initialTab = 'matches';
        }
      } else if (parts.length === 1) {
        if (parts[0] === 'table' || parts[0] === 'matches' || parts[0] === 'leaders') {
          initialTab = parts[0];
        } else if (parts[0] === 'fixtures' || parts[0] === 'results') {
          initialTab = 'matches';
        } else if (/^\d+$/.test(parts[0])) {
          leagueId = parts[0];
          initialTab = 'matches';
        } else if (parts[0].startsWith('league')) {
          leagueId = parts[0].replace('league', '').replace('-', '').replace('_', '');
          initialTab = 'matches';
        }
      }
    }
  }

  if (!leagueId) {
    try {
      leagueId = localStorage.getItem('fdl_active_league_id') || '';
    } catch (_) {}
  }
  if (!leagueId) leagueId = '8';
  if (!seasonId) seasonId = '28083';
  if (!initialTab) initialTab = targetMatchId ? 'matches' : 'matches';

  const mediaBase = urlParams.get('media_base') || '/media';

  const LEAGUE_DEFAULTS = {
    '8': 'АПЛ',
    '9': 'Чемпионшип',
    '564': 'Ла Лига',
    '82': 'Бундеслига',
    '384': 'Серия А',
    '301': 'Лига 1',
    '2': 'Лига чемпионов',
    '5': 'Лига Европы'
  };

  const rawLeagueName = urlParams.get('league_name') || urlParams.get('league_title') || urlParams.get('league_label');
  let displayLeague = (rawLeagueName || LEAGUE_DEFAULTS[leagueId] || `Лига ${leagueId}`).trim();

  const rawRoundName = urlParams.get('round_name') || urlParams.get('round_title') || urlParams.get('round_num');
  let displayRound = '';
  if (rawRoundName) {
    const cleanRound = rawRoundName.trim();
    if (/тур/i.test(cleanRound)) {
      displayRound = cleanRound;
    } else if (/^\d+$/.test(cleanRound)) {
      displayRound = `${cleanRound} тур`;
    } else {
      displayRound = cleanRound;
    }
  } else if (/^\d{1,2}$/.test(roundId)) {
    displayRound = `${roundId} тур`;
  } else {
    displayRound = `Тур ${roundId}`;
  }

  // 3. Элементы DOM
  const viewport = document.getElementById('viewport');
  const stage = document.getElementById('stage');
  const img = document.getElementById('infographicImg');
  const loader = document.getElementById('loader');
  const errorOverlay = document.getElementById('errorOverlay');
  const errorMessage = document.getElementById('errorMessage');
  const btnRetry = document.getElementById('btnRetry');
  const headerContext = document.getElementById('headerContext');
  const zoomLevelIndicator = document.getElementById('zoomLevelIndicator');
  const zoomActions = document.querySelector('.zoom-actions');
  const btnZoomIn = document.getElementById('btnZoomIn');
  const btnZoomOut = document.getElementById('btnZoomOut');
  const btnZoomReset = document.getElementById('btnZoomReset');
  const tabButtons = document.querySelectorAll('.tab-btn');

  // Элементы выбора лиги и Bottom Sheet
  const headerLeagueFlag = document.getElementById('headerLeagueFlag');
  const headerLeagueName = document.getElementById('headerLeagueName');
  const leagueSelectorTrigger = document.getElementById('leagueSelectorTrigger');
  const quickLeaguesBar = document.getElementById('quickLeaguesBar');
  const quickLeaguesScroll = document.getElementById('quickLeaguesScroll');
  const sheetBackdrop = document.getElementById('sheetBackdrop');
  const leagueBottomSheet = document.getElementById('leagueBottomSheet');
  const sheetCloseBtn = document.getElementById('sheetCloseBtn');
  const leagueSearchInput = document.getElementById('leagueSearchInput');
  const sheetSearchClear = document.getElementById('sheetSearchClear');
  const sheetCategoryTabs = document.getElementById('sheetCategoryTabs');
  const sheetLeaguesList = document.getElementById('sheetLeaguesList');
  const sheetSubtitle = document.getElementById('sheetSubtitle');

  // Элементы архивного селектора сезонов и туров (Phase 9)
  const archiveSelectorTrigger = document.getElementById('archiveSelectorTrigger');
  const headerRoundBadge = document.getElementById('headerRoundBadge');
  const archiveSheetBackdrop = document.getElementById('archiveSheetBackdrop');
  const archiveBottomSheet = document.getElementById('archiveBottomSheet');
  const archiveSheetCloseBtn = document.getElementById('archiveSheetCloseBtn');
  const archiveSheetSubtitle = document.getElementById('archiveSheetSubtitle');
  const archiveSeasonsBar = document.getElementById('archiveSeasonsBar');
  const archiveRoundsGrid = document.getElementById('archiveRoundsGrid');
  const mcLiveBadge = document.getElementById('mcLiveBadge');

  let activeArchiveData = null;
  let selectedArchiveSeasonId = null;
  let livePollInterval = null;

  // Элементы интерактивного режима
  const interactiveStage = document.getElementById('interactiveStage');
  const roundResultsView = document.getElementById('roundResultsView');
  const matchCenterView = document.getElementById('matchCenterView');
  const standingsView = document.getElementById('standingsView');
  const standingsContainer = document.getElementById('standingsContainer');
  const roundSummaryCard = document.getElementById('roundSummaryCard');
  const roundMatchesList = document.getElementById('roundMatchesList');
  const btnBackToResults = document.getElementById('btnBackToResults');
  const mcLeagueBadge = document.getElementById('mcLeagueBadge');
  const mcHomeLogo = document.getElementById('mcHomeLogo');
  const mcHomeName = document.getElementById('mcHomeName');
  const mcAwayLogo = document.getElementById('mcAwayLogo');
  const mcAwayName = document.getElementById('mcAwayName');
  const mcScore = document.getElementById('mcScore');
  const mcHTScore = document.getElementById('mcHTScore');
  const mcStatus = document.getElementById('mcStatus');
  const mcVenue = document.getElementById('mcVenue');
  const mcReferee = document.getElementById('mcReferee');
  const mcSubtabButtons = document.querySelectorAll('.mc-subtab-btn');
  const eventsTimeline = document.getElementById('eventsTimeline');
  const statsBarsContainer = document.getElementById('statsBarsContainer');
  const pitchHomePlayers = document.getElementById('pitchHomePlayers');
  const pitchAwayPlayers = document.getElementById('pitchAwayPlayers');
  const mcCoachesCard = document.getElementById('mcCoachesCard');
  const mcBenchCard = document.getElementById('mcBenchCard');
  const h2hContainer = document.getElementById('h2hContainer');
  const leadersView = document.getElementById('leadersView');
  const leadersContainer = document.getElementById('leadersContainer');
  const leadersCategoryNav = document.getElementById('leadersCategoryNav');

  // Элементы навигатора туров ‹ 9 ТУР ›
  const roundPager = document.getElementById('roundPager');
  const pagerPrev = document.getElementById('pagerPrev');
  const pagerNext = document.getElementById('pagerNext');
  const pagerRoundTitle = document.getElementById('pagerRoundTitle');
  const pagerRoundNum = document.getElementById('pagerRoundNum');
  const pagerRoundTag = document.getElementById('pagerRoundTag');
  const pagerGoCurrent = document.getElementById('pagerGoCurrent');

  const TAB_FILES = {
    table: 'standings.png',
    matches: 'round_results.png',
    fixtures: 'round_results.png',
    results: 'round_results.png',
    leaders: 'leaders.png'
  };

  const TAB_TITLES = {
    table: 'ТУРНИРНАЯ ТАБЛИЦА',
    matches: 'МАТЧИ ТУРА',
    fixtures: 'МАТЧИ ТУРА',
    results: 'МАТЧИ ТУРА',
    leaders: 'ЛИДЕРЫ СЕЗОНА'
  };

  let currentTab = initialTab;
  if (currentTab === 'fixtures' || currentTab === 'results') {
    currentTab = 'matches';
  }
  if (!TAB_FILES[currentTab]) {
    currentTab = 'matches';
  }

  // Кэш загруженных данных туров (key: `${leagueId}_${roundId}`) и лидеров
  const roundDataCache = new Map();
  let activeRoundData = null;
  let leadersDataCache = null;
  const leadersDataCacheMap = new Map();
  let currentLeaderCategory = 'scorers';

  function getActiveLeague() {
    return leaguesCatalog.find(l => String(l.league_id) === String(leagueId)) || null;
  }

  function formatRoundName(nameOrNum) {
    if (!nameOrNum) return '';
    const s = String(nameOrNum).trim();
    if (/тур/i.test(s)) return s;
    if (/^\d+$/.test(s)) return `${s} тур`;
    return `Тур ${s}`;
  }

  function getDefaultRoundId() {
    const active = getActiveLeague();
    if (!active) return roundId || '';
    // По умолчанию отображается текущий тур (или следующий, если он еще не начался)
    return String(active.next_round_id || active.curr_round_id || roundId || '');
  }

  function getRoundIdForTab(tabKey) {
    if (roundId) return String(roundId);
    return getDefaultRoundId();
  }

  // 4. Состояние масштабирования и панорамирования (Pan & Zoom для PNG)
  const state = {
    scale: 1,
    minScale: 0.8,
    maxScale: 4.0,
    posX: 0,
    posY: 0,
    isDragging: false,
    startX: 0,
    startY: 0,
    lastTouchDist: 0,
    lastTapTime: 0
  };

  function updateTransform() {
    stage.style.transform = `translate3d(${state.posX}px, ${state.posY}px, 0) scale(${state.scale})`;
    if (zoomLevelIndicator) {
      zoomLevelIndicator.textContent = `${Math.round(state.scale * 100)}%`;
    }
  }

  function resetZoom() {
    state.scale = 1;
    state.posX = 0;
    state.posY = 0;
    updateTransform();
  }

  function setZoom(newScale, centerX = 0, centerY = 0) {
    const clampedScale = Math.min(Math.max(newScale, state.minScale), state.maxScale);
    if (clampedScale === state.scale) return;

    const ratio = clampedScale / state.scale;
    state.posX = centerX - (centerX - state.posX) * ratio;
    state.posY = centerY - (centerY - state.posY) * ratio;
    state.scale = clampedScale;

    if (state.scale <= 1.05) {
      state.posX = 0;
      state.posY = 0;
    }
    updateTransform();
  }

  // =========================================================================
  // Каталог лиг и логика селектора (Phase 6)
  // =========================================================================
  let leaguesCatalog = [];
  let currentCategoryFilter = 'all';
  let currentSearchQuery = '';

  async function loadLeaguesCatalog() {
    const urls = [
      '/api/leagues',
      'active_leagues.json',
      'viewer/active_leagues.json',
      `${mediaBase}/active_leagues.json`
    ];

    let data = null;
    const buster = `?_cb=${Date.now()}`;
    for (const u of urls) {
      try {
        const fetchUrl = u.includes('?') ? `${u}&${buster.slice(1)}` : `${u}${buster}`;
        const resp = await fetch(fetchUrl, { cache: 'no-cache' });
        if (resp.ok) {
          data = await resp.json();
          break;
        }
      } catch (_) {}
    }

    if (!data || !Array.isArray(data.leagues)) {
      return;
    }

    leaguesCatalog = data.leagues;
    if (sheetSubtitle) {
      sheetSubtitle.textContent = `${data.total || leaguesCatalog.length} активных лиг`;
    }

    updateHeaderLeagueInfo();
    renderQuickLeaguesCarousel();
    renderCategoryMenu();
    renderLeaguesList();
  }

  function updateHeaderLeagueInfo() {
    const current = leaguesCatalog.find(l => String(l.league_id) === String(leagueId));
    if (current) {
      if (headerLeagueFlag) headerLeagueFlag.textContent = current.country_flag || '⚽';
      if (headerLeagueName) headerLeagueName.textContent = current.display_name || current.short_name || 'Лига';
      displayLeague = current.display_name || current.league_name;
    } else {
      if (headerLeagueFlag) headerLeagueFlag.textContent = '⚽';
      if (headerLeagueName) headerLeagueName.textContent = displayLeague;
    }

    if (headerRoundBadge) {
      headerRoundBadge.textContent = displayRound.toUpperCase();
    }
  }

  function renderQuickLeaguesCarousel() {
    if (!quickLeaguesScroll || !quickLeaguesBar) return;
    quickLeaguesScroll.innerHTML = '';

    const topLeagues = leaguesCatalog.filter(l => l.is_top);
    if (topLeagues.length === 0) {
      quickLeaguesBar.style.display = 'none';
      return;
    }

    quickLeaguesBar.style.display = 'flex';

    topLeagues.forEach(l => {
      const chip = document.createElement('button');
      chip.className = `quick-chip ${String(l.league_id) === String(leagueId) ? 'active' : ''}`;
      chip.innerHTML = `<span>${l.country_flag || '⚽'}</span><span>${escapeHtml(l.display_name || '')}</span>`;
      chip.onclick = () => {
        if (String(l.league_id) !== String(leagueId)) {
          switchLeague(l.league_id);
        }
      };
      quickLeaguesScroll.appendChild(chip);
    });

    // Кнопка "Все лиги +"
    const moreBtn = document.createElement('button');
    moreBtn.className = 'quick-chip quick-chip-more';
    moreBtn.innerHTML = `<span>🌐</span><span>Все (${leaguesCatalog.length})</span>`;
    moreBtn.onclick = () => openBottomSheet();
    quickLeaguesScroll.appendChild(moreBtn);
  }

  /**
   * Рендер меню категорий в Bottom Sheet:
   * СТРОГО соблюдаем правило:
   * «если в активных лигах нет из «Европейские чемпионаты», «Еврокубки»; меню не отображать или просто задисейблить»
   */
  function renderCategoryMenu() {
    if (!sheetCategoryTabs) return;
    sheetCategoryTabs.innerHTML = '';

    const domesticCount = leaguesCatalog.filter(l => l.category === 'domestic').length;
    const eurocupsCount = leaguesCatalog.filter(l => l.category === 'eurocups').length;

    const validTabs = [];
    // Если в активных лигах нет «Европейские чемпионаты» — этот пункт меню не отображаем!
    if (domesticCount > 0) {
      validTabs.push({ key: 'domestic', title: '🌍 Европейские чемпионаты', count: domesticCount });
    }
    // Если в активных лигах нет «Еврокубки» — этот пункт меню не отображаем!
    if (eurocupsCount > 0) {
      validTabs.push({ key: 'eurocups', title: '⭐ Еврокубки', count: eurocupsCount });
    }

    // Если нет категорий или доступна только 1 категория — меню категорий скрываем
    if (validTabs.length <= 1) {
      sheetCategoryTabs.style.display = 'none';
      return;
    }

    sheetCategoryTabs.style.display = 'flex';

    // Вкладка "Все"
    const allBtn = document.createElement('button');
    allBtn.className = `sheet-cat-btn ${currentCategoryFilter === 'all' ? 'active' : ''}`;
    allBtn.dataset.category = 'all';
    allBtn.innerHTML = `<span>Все</span> <span class="cat-count">${leaguesCatalog.length}</span>`;
    allBtn.onclick = () => {
      currentCategoryFilter = 'all';
      updateCategoryTabsActive();
      renderLeaguesList();
    };
    sheetCategoryTabs.appendChild(allBtn);

    // Добавляем только активные категории
    validTabs.forEach(tab => {
      const btn = document.createElement('button');
      btn.className = `sheet-cat-btn ${currentCategoryFilter === tab.key ? 'active' : ''}`;
      btn.dataset.category = tab.key;
      btn.innerHTML = `<span>${tab.title}</span> <span class="cat-count">${tab.count}</span>`;
      btn.onclick = () => {
        currentCategoryFilter = tab.key;
        updateCategoryTabsActive();
        renderLeaguesList();
      };
      sheetCategoryTabs.appendChild(btn);
    });
  }

  function updateCategoryTabsActive() {
    if (!sheetCategoryTabs) return;
    const buttons = sheetCategoryTabs.querySelectorAll('.sheet-cat-btn');
    buttons.forEach(btn => {
      const cat = btn.dataset.category || 'all';
      if (cat === currentCategoryFilter) {
        btn.classList.add('active');
      } else {
        btn.classList.remove('active');
      }
    });
  }

  function renderLeaguesList() {
    if (!sheetLeaguesList) return;
    sheetLeaguesList.innerHTML = '';

    let filtered = leaguesCatalog;
    if (currentCategoryFilter !== 'all') {
      filtered = filtered.filter(l => l.category === currentCategoryFilter);
    }

    if (currentSearchQuery) {
      const q = currentSearchQuery.toLowerCase().trim();
      filtered = filtered.filter(l =>
        (l.display_name && l.display_name.toLowerCase().includes(q)) ||
        (l.league_name && l.league_name.toLowerCase().includes(q)) ||
        (l.country_name && l.country_name.toLowerCase().includes(q)) ||
        (l.short_name && l.short_name.toLowerCase().includes(q))
      );
    }

    if (filtered.length === 0) {
      sheetLeaguesList.innerHTML = '<div class="sheet-empty">Турниры не найдены</div>';
      return;
    }

    // Если режим "Все" и нет поиска — группируем по категориям
    if (currentCategoryFilter === 'all' && !currentSearchQuery) {
      const domesticLeagues = filtered.filter(l => l.category === 'domestic');
      const eurocupsLeagues = filtered.filter(l => l.category === 'eurocups');

      // Отображаем секцию «Европейские чемпионаты» ТОЛЬКО если в ней есть лиги
      if (domesticLeagues.length > 0) {
        const secHeader = document.createElement('div');
        secHeader.className = 'sheet-section-header';
        secHeader.textContent = '🌍 Европейские чемпионаты';
        sheetLeaguesList.appendChild(secHeader);

        domesticLeagues.sort((a, b) => a.order - b.order).forEach(l => {
          sheetLeaguesList.appendChild(createLeagueRow(l));
        });
      }

      // Отображаем секцию «Еврокубки» ТОЛЬКО если в ней есть лиги
      if (eurocupsLeagues.length > 0) {
        const secHeader = document.createElement('div');
        secHeader.className = 'sheet-section-header';
        secHeader.textContent = '⭐ Еврокубки';
        sheetLeaguesList.appendChild(secHeader);

        eurocupsLeagues.sort((a, b) => a.order - b.order).forEach(l => {
          sheetLeaguesList.appendChild(createLeagueRow(l));
        });
      }
    } else {
      // Иначе плоский список
      filtered.sort((a, b) => a.order - b.order).forEach(l => {
        sheetLeaguesList.appendChild(createLeagueRow(l));
      });
    }
  }

  function createLeagueRow(l) {
    const item = document.createElement('div');
    const isSelected = String(l.league_id) === String(leagueId);
    item.className = `sheet-league-item ${isSelected ? 'active' : ''}`;

    const iconHtml = l.league_logo
      ? `<img src="${l.league_logo}" class="sheet-league-logo" alt="${l.display_name}" onerror="this.outerHTML='<span class=\\'sheet-league-flag\\'>${l.country_flag || '⚽'}</span>'">`
      : `<span class="sheet-league-flag">${l.country_flag || '⚽'}</span>`;

    const topTag = l.is_top ? '<span class="sheet-top-tag">ТОП</span>' : '';

    let roundBadge = '';
    if (l.next_round) {
      roundBadge = `<span class="sheet-round-badge has-round">Тур ${l.next_round}</span>`;
    } else if (l.curr_round) {
      roundBadge = `<span class="sheet-round-badge">Тур ${l.curr_round}</span>`;
    }

    item.innerHTML = `
      ${iconHtml}
      <div class="sheet-league-info">
        <div class="sheet-league-row-top">
          <span class="sheet-league-name">${escapeHtml(l.display_name || '')}</span>
          ${topTag}
        </div>
        <div class="sheet-league-sub">${escapeHtml(l.country_name || l.league_name || '')} · ${escapeHtml(l.season_name || '')}</div>
      </div>
      <div class="sheet-league-meta">
        ${roundBadge}
        <span class="sheet-active-check">✓</span>
      </div>
    `;

    item.onclick = () => {
      closeBottomSheet();
      if (String(l.league_id) !== String(leagueId)) {
        switchLeague(l.league_id);
      }
    };

    return item;
  }

  function switchLeague(targetLeagueId) {
    const target = leaguesCatalog.find(l => String(l.league_id) === String(targetLeagueId));
    if (!target) return;

    leagueId = String(target.league_id);
    seasonId = String(target.season_id);
    roundId = String(target.next_round_id || target.curr_round_id || '');

    try {
      localStorage.setItem('fdl_active_league_id', leagueId);
    } catch (_) {}

    // Обновляем параметры адресной строки без перезагрузки
    const newUrl = new URL(window.location.href);
    newUrl.searchParams.set('league', leagueId);
    newUrl.searchParams.set('season', seasonId);
    newUrl.searchParams.set('round', roundId);
    newUrl.searchParams.delete('match');
    newUrl.searchParams.delete('fixture_id');
    window.history.replaceState({}, '', newUrl.toString());

    // Очищаем кэш тура и лидеров для новой лиги
    roundDataCache.clear();
    activeRoundData = null;
    leadersDataCache = null;

    // Сбрасываем содержимое контейнеров, чтобы не было наложения данных прошлой лиги
    if (roundMatchesList) roundMatchesList.innerHTML = '';
    if (standingsContainer) standingsContainer.innerHTML = '';
    if (leadersContainer) leadersContainer.innerHTML = '';
    if (roundSummaryCard) roundSummaryCard.style.display = 'none';

    updateHeaderLeagueInfo();
    renderQuickLeaguesCarousel();
    renderLeaguesList();

    // Единый вызов отображения текущей вкладки для выбранной лиги
    selectTab(currentTab);
  }

  function openBottomSheet() {
    if (!sheetBackdrop || !leagueBottomSheet) return;
    sheetBackdrop.classList.remove('hidden');
    leagueBottomSheet.classList.remove('hidden');
    if (leagueSelectorTrigger) leagueSelectorTrigger.classList.add('open');
    if (leagueSearchInput) {
      leagueSearchInput.value = '';
      currentSearchQuery = '';
      if (sheetSearchClear) sheetSearchClear.classList.add('hidden');
    }
    renderLeaguesList();
  }

  function closeBottomSheet() {
    if (!sheetBackdrop || !leagueBottomSheet) return;
    sheetBackdrop.classList.add('hidden');
    leagueBottomSheet.classList.add('hidden');
    if (leagueSelectorTrigger) leagueSelectorTrigger.classList.remove('open');
  }

  // Привязка обработчиков Bottom Sheet
  if (leagueSelectorTrigger) {
    leagueSelectorTrigger.onclick = openBottomSheet;
  }
  if (sheetCloseBtn) {
    sheetCloseBtn.onclick = closeBottomSheet;
  }
  if (sheetBackdrop) {
    sheetBackdrop.onclick = closeBottomSheet;
  }
  if (leagueSearchInput) {
    leagueSearchInput.oninput = (e) => {
      currentSearchQuery = e.target.value || '';
      if (sheetSearchClear) {
        if (currentSearchQuery) sheetSearchClear.classList.remove('hidden');
        else sheetSearchClear.classList.add('hidden');
      }
      renderLeaguesList();
    };
  }
  if (sheetSearchClear) {
    sheetSearchClear.onclick = () => {
      if (leagueSearchInput) {
        leagueSearchInput.value = '';
        currentSearchQuery = '';
        sheetSearchClear.classList.add('hidden');
        renderLeaguesList();
      }
    };
  }

  // ==========================================================================
  // Логика архивного селектора сезонов и туров (Phase 9)
  // ==========================================================================
  async function openArchiveBottomSheet() {
    if (!archiveSheetBackdrop || !archiveBottomSheet) return;
    archiveSheetBackdrop.classList.remove('hidden');
    archiveBottomSheet.classList.remove('hidden');
    if (archiveSelectorTrigger) archiveSelectorTrigger.classList.add('open');

    const active = getActiveLeague();
    if (active && active.rounds && active.rounds.length > 0) {
      if (archiveSheetSubtitle) {
        archiveSheetSubtitle.textContent = `${active.display_name || active.league_name} · ${active.season_name}`;
      }
      renderArchiveRounds({
        id: active.season_id,
        name: active.season_name,
        is_current: true,
        rounds: active.rounds.map(r => ({
          id: r.id,
          name: r.name,
          is_current: String(r.id) === String(active.next_round_id || active.curr_round_id),
          total_matches: r.total_matches || 10,
          finished_count: r.finished_count !== undefined ? r.finished_count : (r.finished ? (r.total_matches || 10) : 0)
        }))
      });
      return;
    }

    if (archiveSheetSubtitle) {
      archiveSheetSubtitle.textContent = 'Загрузка сезонов и туров...';
    }
    await loadLeagueArchive(leagueId);
  }

  function closeArchiveBottomSheet() {
    if (!archiveSheetBackdrop || !archiveBottomSheet) return;
    archiveSheetBackdrop.classList.add('hidden');
    archiveBottomSheet.classList.add('hidden');
    if (archiveSelectorTrigger) archiveSelectorTrigger.classList.remove('open');
  }

  async function loadLeagueArchive(lid) {
    try {
      const resp = await fetch(`/api/leagues/${lid}/archive?_cb=${Date.now()}`);
      if (!resp.ok) {
        if (archiveSheetSubtitle) archiveSheetSubtitle.textContent = 'Архив временно формируется';
        if (archiveRoundsGrid) archiveRoundsGrid.innerHTML = '<div class="sheet-loading-spinner">Нет доступных архивных данных</div>';
        return;
      }
      const data = await resp.json();
      if (!data || !Array.isArray(data.seasons) || data.seasons.length === 0) {
        if (archiveSheetSubtitle) archiveSheetSubtitle.textContent = 'Сезоны не найдены';
        return;
      }

      activeArchiveData = data;
      if (archiveSheetSubtitle) {
        archiveSheetSubtitle.textContent = `${data.league_name} · ${data.seasons.length} сезонов`;
      }

      renderArchiveSeasons(data.seasons);
    } catch (_) {
      if (archiveSheetSubtitle) archiveSheetSubtitle.textContent = 'Архив временно недоступен';
    }
  }

  function renderArchiveSeasons(seasons) {
    if (!archiveSeasonsBar) return;
    archiveSeasonsBar.innerHTML = '';

    let currentSeason = seasons.find(s => String(s.id) === String(seasonId)) ||
                        seasons.find(s => s.is_current) ||
                        seasons[0];
    selectedArchiveSeasonId = currentSeason.id;

    seasons.forEach(s => {
      const chip = document.createElement('button');
      chip.className = `archive-season-chip ${s.id === selectedArchiveSeasonId ? 'active' : ''}`;
      chip.textContent = s.name;
      chip.onclick = () => {
        selectedArchiveSeasonId = s.id;
        document.querySelectorAll('.archive-season-chip').forEach(c => c.classList.remove('active'));
        chip.classList.add('active');
        renderArchiveRounds(s);
      };
      archiveSeasonsBar.appendChild(chip);
    });

    renderArchiveRounds(currentSeason);
  }

  function renderArchiveRounds(season) {
    if (!archiveRoundsGrid) return;
    archiveRoundsGrid.innerHTML = '';

    if (!season.rounds || season.rounds.length === 0) {
      archiveRoundsGrid.innerHTML = '<div class="sheet-loading-spinner">Нет туров для данного сезона</div>';
      return;
    }

    season.rounds.forEach(r => {
      const card = document.createElement('div');
      const isSelected = String(r.id) === String(roundId);
      card.className = `archive-round-card ${isSelected ? 'active' : ''} ${r.is_current ? 'is-current' : ''}`;
      card.innerHTML = `
        <div class="archive-round-num">${escapeHtml(r.name)} ТУР</div>
        <div class="archive-round-status">${r.finished_count}/${r.total_matches} завершено</div>
      `;
      card.onclick = () => {
        selectArchiveRound(r.id, r.name, season.id, season.name);
      };
      archiveRoundsGrid.appendChild(card);
    });
  }

  function selectArchiveRound(rId, rName, sId, sName) {
    closeArchiveBottomSheet();
    navigateToRound(rId);
  }

  if (archiveSelectorTrigger) {
    archiveSelectorTrigger.onclick = openArchiveBottomSheet;
  }
  if (archiveSheetCloseBtn) {
    archiveSheetCloseBtn.onclick = closeArchiveBottomSheet;
  }
  if (archiveSheetBackdrop) {
    archiveSheetBackdrop.onclick = closeArchiveBottomSheet;
  }

  // 5. Загрузка статического изображения
  function getMediaUrl(tabKey) {
    const fileName = TAB_FILES[tabKey] || TAB_FILES.table;
    if (tabKey === 'leaders') {
      return `${mediaBase}/${leagueId}/${seasonId}/leaders.png`;
    }
    const targetRound = getRoundIdForTab(tabKey);
    return `${mediaBase}/${leagueId}/${seasonId}/${targetRound}/${fileName}`;
  }

  function loadCurrentTabImage() {
    const url = getMediaUrl(currentTab);
    loader.classList.remove('hidden');
    errorOverlay.classList.add('hidden');
    img.style.opacity = '0';

    if (currentTab === 'leaders') {
      headerContext.textContent = `${displayLeague.toUpperCase()} • ${TAB_TITLES.leaders}`;
      document.title = `${displayLeague} — Лидеры сезона | Football Data Lab`;
    } else {
      const active = getActiveLeague();
      let roundLabel = displayRound;
      if (active?.next_round) {
        roundLabel = formatRoundName(active.next_round);
      } else if (active?.curr_round) {
        roundLabel = formatRoundName(active.curr_round);
      }
      headerContext.textContent = `${displayLeague.toUpperCase()} • ${roundLabel.toUpperCase()} • ${TAB_TITLES[currentTab] || ''}`;
      document.title = `${displayLeague} — ${roundLabel} | Football Data Lab`;
    }

    const testImg = new Image();
    testImg.onload = () => {
      img.src = url;
      img.style.opacity = '1';
      loader.classList.add('hidden');
      resetZoom();
    };

    testImg.onerror = () => {
      loader.classList.add('hidden');
      errorOverlay.classList.remove('hidden');
      errorMessage.innerHTML = `Данные для <b>${escapeHtml(displayLeague)}</b> еще формируются.<br><span style="font-size:12px;opacity:0.8;">Расписание, результаты и статистика тура появятся перед началом матчей.</span>`;
    };

    testImg.src = url;
  }

  // 6. Интерактивный режим: Загрузка данных тура (round_data.json)
  async function loadRoundData(targetRoundId, callback) {
    if (typeof targetRoundId === 'function') {
      callback = targetRoundId;
      targetRoundId = getRoundIdForTab(currentTab);
    }
    if (!targetRoundId) targetRoundId = roundId;
    if (!targetRoundId) {
      const active = getActiveLeague();
      if (active) {
        targetRoundId = String(active.next_round_id || active.curr_round_id || '');
        if (!roundId) roundId = targetRoundId;
      }
    }

    if (!targetRoundId || targetRoundId === 'undefined' || targetRoundId === 'null') {
      targetRoundId = '';
    }

    const roundApiUrl = targetRoundId ? `/api/rounds/${targetRoundId}?locale=ru` : `/api/rounds?league_id=${leagueId}&locale=ru`;

    const cacheKey = `${leagueId}_${targetRoundId || 'default'}`;
    if (roundDataCache.has(cacheKey)) {
      const cached = roundDataCache.get(cacheKey);
      if (callback) callback(cached);
      else handleRoundDataLoaded(cached);
      return;
    }

    loader.classList.remove('hidden');
    errorOverlay.classList.add('hidden');

    const urls = [
      roundApiUrl,
      targetRoundId ? `data/${leagueId}_${seasonId}_${targetRoundId}.json` : null,
      targetRoundId ? `viewer/data/${leagueId}_${seasonId}_${targetRoundId}.json` : null,
      targetRoundId ? `${mediaBase}/${leagueId}/${seasonId}/${targetRoundId}/round_data.json` : null
    ].filter(Boolean);

    let loadedData = null;
    const buster = `?_cb=${Date.now()}`;
    for (const u of urls) {
      try {
        const fetchUrl = u.includes('?') ? `${u}&${buster.slice(1)}` : `${u}${buster}`;
        const resp = await fetch(fetchUrl, { cache: 'no-cache' });
        if (resp.ok) {
          const json = await resp.json();
          // Валидируем, что загруженные данные соответствуют запрошенной лиге
          if (!json.league_id || String(json.league_id) === String(leagueId)) {
            loadedData = json;
            break;
          }
        }
      } catch (_) {}
    }

    loader.classList.add('hidden');

    if (loadedData) {
      roundDataCache.set(cacheKey, loadedData);
      activeRoundData = loadedData;
      if (callback) callback(loadedData);
      else handleRoundDataLoaded(loadedData);
    } else {
      if (callback) {
        callback(null);
      } else if (currentTab === 'results' || currentTab === 'fixtures') {
        stage.classList.remove('hidden');
        interactiveStage.classList.add('hidden');
        if (zoomActions) zoomActions.style.display = 'flex';
        loadCurrentTabImage();
      }
    }
  }

  function handleRoundDataLoaded(data) {
    if (!data) return;

    if (currentTab === 'table') {
      if (data.standings && data.standings.length > 0) {
        renderStandingsView(data.standings);
      } else {
        interactiveStage.classList.add('hidden');
        stage.classList.remove('hidden');
        if (zoomActions) zoomActions.style.display = 'flex';
        loadCurrentTabImage();
      }
      return;
    }

    currentTab = 'matches';
    tabButtons.forEach(btn => {
      const isActive = btn.dataset.tab === 'matches';
      btn.classList.toggle('active', isActive);
      btn.setAttribute('aria-selected', isActive ? 'true' : 'false');
    });

    if (data.round_name) {
      displayRound = formatRoundName(data.round_name);
    }
    headerContext.textContent = `${displayLeague.toUpperCase()} • ${displayRound.toUpperCase()} • ${TAB_TITLES.matches}`;
    document.title = `${displayLeague} — ${displayRound} | Football Data Lab`;

    const btnBackText = document.getElementById('btnBackText');
    if (btnBackText) {
      btnBackText.textContent = 'К матчам тура';
    }

    renderRoundResultsView(data);
    updateRoundPager(data);

    if (targetMatchId) {
      openMatchCenter(targetMatchId);
    } else {
      showRoundResults();
    }
  }

  // 7. Отрисовка списка матчей тура
  function renderRoundResultsView(data) {
    activeRoundData = data;
    const summary = data.summary || {};
    const matches = data.matches || [];

    // Экспресс-статистика тура (отображается при наличии сыгранных матчей)
    if (summary.TotalMatches > 0 && (summary.TotalGoals > 0 || summary.HomeWins > 0 || summary.AwayWins > 0)) {
      roundSummaryCard.innerHTML = `
        <div class="rs-title">Экспресс-статистика тура</div>
        <div class="rs-grid">
          <div class="rs-metric">
            <div class="rs-metric-val">${summary.TotalGoals || 0}</div>
            <div class="rs-metric-label">Голов (${summary.AvgGoals || 0}/м)</div>
          </div>
          <div class="rs-metric">
            <div class="rs-metric-val">${summary.HomeWins || 0}·${summary.Draws || 0}·${summary.AwayWins || 0}</div>
            <div class="rs-metric-label">1 · X · 2</div>
          </div>
          <div class="rs-metric">
            <div class="rs-metric-val">${summary.FavoritesWon || 0}/${(summary.FavoritesWon || 0) + (summary.FavoritesFailed || 0)}</div>
            <div class="rs-metric-label">Фавориты</div>
          </div>
          <div class="rs-metric">
            <div class="rs-metric-val">${summary.TotalYellowCards || 0}🟨 · ${summary.TotalRedCards || 0}🟥</div>
            <div class="rs-metric-label">Карточки</div>
          </div>
        </div>
      `;
      roundSummaryCard.style.display = 'flex';
    } else {
      roundSummaryCard.style.display = 'none';
    }

    // Группировка матчей по датам
    const dateGroups = {};
    matches.forEach(m => {
      const d = new Date(m.FixtureStart);
      const dateKey = d.toISOString().split('T')[0];
      if (!dateGroups[dateKey]) {
        dateGroups[dateKey] = {
          date: d,
          matches: []
        };
      }
      dateGroups[dateKey].matches.push(m);
    });

    let html = '';
    const sortedDates = Object.keys(dateGroups).sort();
    sortedDates.forEach(k => {
      const g = dateGroups[k];
      const dateHeader = formatJsRussianDate(g.date);
      html += `<div class="day-header">📅 ${dateHeader}</div>`;

      g.matches.forEach(m => {
        const isFinished = isMatchFinished(m.StateShort);
        const isLive = isMatchLive(m.StateShort);
        let scoreBoxHtml = '';
        if (isFinished) {
          const homeScore = m.Scores?.HomeCurrent ?? '-';
          const awayScore = m.Scores?.AwayCurrent ?? '-';
          const homeHT = m.Scores?.HomeHT;
          const awayHT = m.Scores?.AwayHT;
          const htText = (homeHT != null && awayHT != null && homeHT !== 'null' && awayHT !== 'null') ? `(${homeHT}:${awayHT})` : '';
          scoreBoxHtml = `
            <div class="rm-score">${homeScore} : ${awayScore}</div>
            ${htText ? `<div class="rm-ht-score">${htText}</div>` : ''}
          `;
        } else if (isLive) {
          const homeScore = m.Scores?.HomeCurrent ?? 0;
          const awayScore = m.Scores?.AwayCurrent ?? 0;
          const homeHT = m.Scores?.HomeHT;
          const awayHT = m.Scores?.AwayHT;
          const htText = (homeHT != null && awayHT != null && homeHT !== 'null' && awayHT !== 'null') ? `(${homeHT}:${awayHT})` : '';
          const liveLabel = formatLiveStateLabel(m.StateShort);
          scoreBoxHtml = `
            <div class="rm-score" style="color:#ef4444;font-weight:800;">${homeScore} : ${awayScore}</div>
            <div class="rm-ht-score" style="color:#ef4444;font-weight:700;">
              <span class="live-dot" style="display:inline-block;width:6px;height:6px;background:#ef4444;border-radius:50%;margin-right:4px;vertical-align:middle;animation:pulse-glow 1.5s infinite;"></span>
              ${liveLabel} ${htText}
            </div>
          `;
        } else {
          const kTime = formatMatchTime(m.FixtureStart);
          const oddsText = m.Odds?.HomeOdds ? `${m.Odds.HomeOdds} · ${m.Odds.DrawOdds} · ${m.Odds.AwayOdds}` : 'VS';
          scoreBoxHtml = `
            <div class="rm-score" style="font-size:14px;letter-spacing:0.5px;">${kTime}</div>
            <div class="rm-ht-score" style="color:#38bdf8;font-weight:700;">${oddsText}</div>
          `;
        }

        const homeName = m.TeamHomeName || 'TBD';
        const awayName = m.TeamAwayName || 'TBD';

        html += `
          <div class="round-match-card" data-fixture-id="${m.FixtureID}">
            <div class="rm-team home">
              ${m.TeamHomeLogo ? `<img class="rm-logo" src="${m.TeamHomeLogo}" alt="">` : ''}
              <span class="rm-name">${escapeHtml(homeName)}</span>
            </div>
            <div class="rm-score-box">
              ${scoreBoxHtml}
            </div>
            <div class="rm-team away">
              <span class="rm-name">${escapeHtml(awayName)}</span>
              ${m.TeamAwayLogo ? `<img class="rm-logo" src="${m.TeamAwayLogo}" alt="">` : ''}
            </div>
            <div class="rm-arrow">›</div>
          </div>
        `;
      });
    });

    roundMatchesList.innerHTML = html;

    // Клик по карточке матча
    roundMatchesList.querySelectorAll('.round-match-card').forEach(card => {
      card.addEventListener('click', () => {
        const fid = card.dataset.fixtureId;
        openMatchCenter(fid);
      });
    });
  }

  // Вспомогательные функции определения состояния матчей
  function isMatchLive(state) {
    if (!state) return false;
    return ['1st', '2nd', 'HT', 'ET', 'et', '2et', 'PEN', 'LIVE', '1H', '2H', 'INPLAY_1ST_HALF', 'INPLAY_2ND_HALF', 'BREAK', 'BRK', 'ETB', 'PENB', 'INT'].includes(state);
  }

  function isMatchFinished(state) {
    if (!state) return false;
    return ['FT', 'AET', 'FTP', 'AWAR', 'WO'].includes(state);
  }

  function formatLiveStateLabel(state) {
    if (state === 'HT') return 'Перерыв';
    if (state === '1st' || state === '1H' || state === 'INPLAY_1ST_HALF') return '1-й тайм';
    if (state === '2nd' || state === '2H' || state === 'INPLAY_2ND_HALF') return '2-й тайм';
    if (state === 'et' || state === 'ET' || state === 'INPLAY_ET') return 'Экстра-тайм';
    if (state === '2et' || state === 'INPLAY_ET_2ND_HALF') return '2-й экстра-тайм';
    if (state === 'PEN' || state === 'INPLAY_PENALTIES') return 'Пенальти';
    if (state === 'BRK' || state === 'BREAK') return 'Перерыв';
    return 'LIVE';
  }

  // Обновление состояния навигатора туров ‹ 9 ТУР ›
  function updateRoundPager(data) {
    if (!roundPager || !data) return;

    const active = getActiveLeague();
    const currentRId = String(data.round_id || roundId);
    const rName = formatRoundName(data.round_name || '');

    if (pagerRoundNum) {
      pagerRoundNum.textContent = rName ? rName.toUpperCase() : 'ТУР';
    }

    // Определение статуса тура (Текущий, LIVE, Прошлый тур, Архив, Предстоящий)
    let tag = 'Архив';
    let tagClass = 'pager-round-tag past';
    let isCurrentRound = false;

    if (active) {
      if (String(active.next_round_id) === currentRId) {
        tag = 'Текущий';
        tagClass = 'pager-round-tag';
        isCurrentRound = true;
      } else if (String(active.curr_round_id) === currentRId) {
        tag = 'Прошлый тур';
        tagClass = 'pager-round-tag past';
      }
    }

    const hasLive = data.matches?.some(m => isMatchLive(m.StateShort));
    if (hasLive) {
      tag = 'LIVE';
      tagClass = 'pager-round-tag live';
    }

    if (pagerRoundTag) {
      pagerRoundTag.textContent = tag;
      pagerRoundTag.className = tagClass;
    }

    if (pagerGoCurrent) {
      if (isCurrentRound) {
        pagerGoCurrent.classList.add('hidden');
      } else {
        pagerGoCurrent.classList.remove('hidden');
      }
    }

    // Навигация влево/вправо по упорядоченному списку туров сезона
    const rounds = active?.rounds || [];
    if (rounds.length > 0) {
      const idx = rounds.findIndex(r => String(r.id) === currentRId);
      if (idx !== -1) {
        // Предыдущий тур (листаем влево до 1-го тура)
        if (idx > 0) {
          if (pagerPrev) {
            pagerPrev.disabled = false;
            pagerPrev.dataset.targetId = rounds[idx - 1].id;
            pagerPrev.title = `К ${rounds[idx - 1].name} туру`;
          }
        } else {
          if (pagerPrev) {
            pagerPrev.disabled = true;
            pagerPrev.dataset.targetId = '';
          }
        }

        // Следующий тур (листаем вправо)
        if (idx < rounds.length - 1) {
          if (pagerNext) {
            pagerNext.disabled = false;
            pagerNext.dataset.targetId = rounds[idx + 1].id;
            pagerNext.title = `К ${rounds[idx + 1].name} туру`;
          }
        } else {
          if (pagerNext) {
            pagerNext.disabled = true;
            pagerNext.dataset.targetId = '';
          }
        }
        return;
      }
    }

    // Резервный переход по prev_round_id / next_round_id из данных тура
    if (pagerPrev) {
      if (data.prev_round_id) {
        pagerPrev.disabled = false;
        pagerPrev.dataset.targetId = data.prev_round_id;
      } else {
        pagerPrev.disabled = true;
        pagerPrev.dataset.targetId = '';
      }
    }

    if (pagerNext) {
      if (data.next_round_id) {
        pagerNext.disabled = false;
        pagerNext.dataset.targetId = data.next_round_id;
      } else {
        pagerNext.disabled = true;
        pagerNext.dataset.targetId = '';
      }
    }
  }

  function navigateToRound(targetId) {
    if (!targetId) return;
    const nextRId = String(targetId);
    roundId = nextRId;

    try {
      const u = new URL(window.location.href);
      u.searchParams.set('tab', currentTab || 'matches');
      u.searchParams.set('round', roundId);
      window.history.replaceState({}, '', u.toString());
    } catch (_) {}

    loadRoundData(roundId, (data) => {
      handleRoundDataLoaded(data);
    });
  }

  if (pagerPrev) {
    pagerPrev.addEventListener('click', (e) => {
      e.stopPropagation();
      const tid = pagerPrev.dataset.targetId;
      if (tid) navigateToRound(tid);
    });
  }

  if (pagerNext) {
    pagerNext.addEventListener('click', (e) => {
      e.stopPropagation();
      const tid = pagerNext.dataset.targetId;
      if (tid) navigateToRound(tid);
    });
  }

  if (pagerGoCurrent) {
    pagerGoCurrent.addEventListener('click', (e) => {
      e.stopPropagation();
      const active = getActiveLeague();
      const tid = active?.next_round_id || active?.curr_round_id;
      if (tid) navigateToRound(tid);
    });
  }

  if (pagerRoundTitle) {
    pagerRoundTitle.addEventListener('click', () => {
      openArchiveBottomSheet();
    });
  }

  function formatMatchTime(dateStr) {
    if (!dateStr) return '  -  ';
    const d = new Date(dateStr);
    const hh = String(d.getHours()).padStart(2, '0');
    const mm = String(d.getMinutes()).padStart(2, '0');
    return `${hh}:${mm}`;
  }

  function switchSubtab(tabName) {
    mcSubtabButtons.forEach(b => {
      b.classList.toggle('active', b.dataset.subtab === tabName);
    });
    document.querySelectorAll('.mc-tab-pane').forEach(p => p.classList.remove('active'));
    if (tabName === 'overview') document.getElementById('paneOverview')?.classList.add('active');
    if (tabName === 'stats') document.getElementById('paneStats')?.classList.add('active');
    if (tabName === 'lineups') document.getElementById('paneLineups')?.classList.add('active');
    if (tabName === 'h2h') document.getElementById('paneH2H')?.classList.add('active');
  }

  // 8. Открытие Match Center
  function openMatchCenter(fixtureId) {
    let match = null;
    if (activeRoundData?.matches) {
      match = activeRoundData.matches.find(m => String(m.FixtureID) === String(fixtureId));
    }
    if (!match) {
      for (const d of roundDataCache.values()) {
        if (d.matches) {
          match = d.matches.find(m => String(m.FixtureID) === String(fixtureId));
          if (match) break;
        }
      }
    }
    if (!match) return;

    targetMatchId = fixtureId;

    // Заполнение шапки матча
    mcLeagueBadge.textContent = (match.LeagueName || displayLeague).toUpperCase();
    mcHomeName.textContent = match.TeamHomeName;
    mcHomeLogo.src = match.TeamHomeLogo || '';
    mcHomeLogo.style.display = match.TeamHomeLogo ? 'block' : 'none';

    mcAwayName.textContent = match.TeamAwayName;
    mcAwayLogo.src = match.TeamAwayLogo || '';
    mcAwayLogo.style.display = match.TeamAwayLogo ? 'block' : 'none';

    mcVenue.textContent = match.VenueName || 'Стадион';
    mcReferee.textContent = match.RefereeName ? `Судья: ${match.RefereeName}` : (match.StateShort === 'FT' ? 'Матч завершён' : 'Матч ожидает начала');

    const isLive = isMatchLive(match.StateShort);
    const isFinished = isMatchFinished(match.StateShort);
    const isPreMatch = !isLive && !isFinished;
    currentTab = 'matches';

    // Синхронизируем состояние верхних табов навигации
    tabButtons.forEach(btn => {
      const isActive = btn.dataset.tab === 'matches';
      btn.classList.toggle('active', isActive);
      btn.setAttribute('aria-selected', isActive ? 'true' : 'false');
    });
    headerContext.textContent = `${displayLeague.toUpperCase()} • ${displayRound.toUpperCase()} • ${TAB_TITLES.matches}`;
    const btnBackText = document.getElementById('btnBackText');
    if (btnBackText) {
      btnBackText.textContent = 'К матчам тура';
    }

    const overviewBtn = document.querySelector('[data-subtab="overview"]');

    if (isPreMatch) {
      mcScore.textContent = formatMatchTime(match.FixtureStart);
      mcScore.style.fontSize = '24px';
      mcScore.style.letterSpacing = '1px';
      mcHTScore.innerHTML = match.Odds?.HomeOdds ? `
        <span class="mc-prematch-odds-pill">
          1: <b>${match.Odds.HomeOdds}</b> · X: <b>${match.Odds.DrawOdds}</b> · 2: <b>${match.Odds.AwayOdds}</b>
        </span>
      ` : '';
      mcStatus.textContent = formatJsRussianDate(new Date(match.FixtureStart));
      if (mcLiveBadge) mcLiveBadge.classList.add('hidden');
      stopLiveMatchPolling();

      if (overviewBtn) {
        overviewBtn.classList.add('disabled');
        overviewBtn.style.display = 'none';
      }
      switchSubtab('h2h');
    } else {
      mcScore.textContent = `${match.Scores?.HomeCurrent ?? 0} : ${match.Scores?.AwayCurrent ?? 0}`;
      mcScore.style.fontSize = '';
      mcScore.style.letterSpacing = '';
      const hHT = match.Scores?.HomeHT;
      const aHT = match.Scores?.AwayHT;
      mcHTScore.textContent = (hHT != null && aHT != null && hHT !== 'null' && aHT !== 'null') ? `(${hHT}:${aHT})` : '';

      if (isLive) {
        mcStatus.textContent = formatLiveStateLabel(match.StateShort);
        if (mcLiveBadge) mcLiveBadge.classList.remove('hidden');
        startLiveMatchPolling(fixtureId);
      } else {
        mcStatus.textContent = 'Матч завершён';
        if (mcLiveBadge) mcLiveBadge.classList.add('hidden');
        stopLiveMatchPolling();
      }

      if (overviewBtn) {
        overviewBtn.classList.remove('disabled');
        overviewBtn.style.display = '';
      }
      switchSubtab('overview');
    }

    // Рендер суб-табов
    renderOverviewTimeline(match.Events || [], match);
    renderStatsBars(match.Stats || {}, match);
    renderLineupsPitch(match.Lineups || {}, match.Coaches || {}, match);
    renderH2H(match);

    // Показываем экран матча
    if (standingsView) standingsView.classList.add('hidden');
    if (leadersView) leadersView.classList.add('hidden');
    roundResultsView.classList.add('hidden');
    matchCenterView.classList.remove('hidden');

    // Telegram BackButton
    if (tg?.BackButton) {
      tg.BackButton.show();
      tg.BackButton.onClick(showRoundResults);
    }
  }

  function startLiveMatchPolling(fixtureId) {
    stopLiveMatchPolling();
    livePollInterval = setInterval(async () => {
      if (!targetMatchId || String(targetMatchId) !== String(fixtureId)) {
        stopLiveMatchPolling();
        return;
      }
      try {
        const resp = await fetch(`/api/fixtures/${fixtureId}/live?_cb=${Date.now()}`);
        if (!resp.ok) return;
        const liveMatch = await resp.json();
        if (!liveMatch || !liveMatch.FixtureID) return;

        if (liveMatch.Scores) {
          const newScore = `${liveMatch.Scores.HomeCurrent ?? 0} : ${liveMatch.Scores.AwayCurrent ?? 0}`;
          if (mcScore.textContent !== newScore) {
            mcScore.textContent = newScore;
          }
          const hHT = liveMatch.Scores.HomeHT;
          const aHT = liveMatch.Scores.AwayHT;
          mcHTScore.textContent = (hHT != null && aHT != null && hHT !== 'null' && aHT !== 'null') ? `(${hHT}:${aHT})` : '';
        }

        if (isMatchFinished(liveMatch.StateShort)) {
          mcStatus.textContent = 'Матч завершён';
          if (mcLiveBadge) mcLiveBadge.classList.add('hidden');
          stopLiveMatchPolling();
        } else {
          mcStatus.textContent = formatLiveStateLabel(liveMatch.StateShort);
        }

        if (liveMatch.Events) renderOverviewTimeline(liveMatch.Events, liveMatch);
        if (liveMatch.Stats) renderStatsBars(liveMatch.Stats, liveMatch);
      } catch (_) {}
    }, 20000);
  }

  function stopLiveMatchPolling() {
    if (livePollInterval) {
      clearInterval(livePollInterval);
      livePollInterval = null;
    }
  }

  function showRoundResults() {
    stopLiveMatchPolling();
    targetMatchId = '';
    if (standingsView) standingsView.classList.add('hidden');
    if (leadersView) leadersView.classList.add('hidden');
    matchCenterView.classList.add('hidden');
    roundResultsView.classList.remove('hidden');

    currentTab = 'matches';
    tabButtons.forEach(btn => {
      const isActive = btn.dataset.tab === 'matches';
      btn.classList.toggle('active', isActive);
      btn.setAttribute('aria-selected', isActive ? 'true' : 'false');
    });
    headerContext.textContent = `${displayLeague.toUpperCase()} • ${displayRound.toUpperCase()} • ${TAB_TITLES.matches}`;

    if (tg?.BackButton) {
      tg.BackButton.hide();
    }
  }

  if (btnBackToResults) {
    btnBackToResults.addEventListener('click', showRoundResults);
  }

  // 8.1. Рендеринг интерактивной турнирной таблицы
  function renderStandingsView(standings) {
    if (!standingsContainer) return;
    if (!standings || standings.length === 0) return;

    stage.classList.add('hidden');
    interactiveStage.classList.remove('hidden');
    if (roundResultsView) roundResultsView.classList.add('hidden');
    if (matchCenterView) matchCenterView.classList.add('hidden');
    if (leadersView) leadersView.classList.add('hidden');
    if (standingsView) standingsView.classList.remove('hidden');
    if (zoomActions) zoomActions.style.display = 'none';

    let html = `
      <div class="standings-card">
        <div class="st-table-wrapper">
          <table class="st-table">
            <thead>
              <tr>
                <th style="width: 36px;">#</th>
                <th style="text-align: left;">Клуб</th>
                <th>И</th>
                <th>В</th>
                <th>Н</th>
                <th>П</th>
                <th>З-П</th>
                <th>РМ</th>
                <th>Форма</th>
                <th>О</th>
              </tr>
            </thead>
            <tbody>
    `;

    standings.forEach(row => {
      const pos = row.Position;
      const trend = row.Trend || 0;
      let trendHtml = '<span class="st-trend same">–</span>';
      if (trend > 0) trendHtml = `<span class="st-trend up">▲${trend}</span>`;
      else if (trend < 0) trendHtml = `<span class="st-trend down">▼${Math.abs(trend)}</span>`;

      let zoneClass = '';
      const devName = (row.RuleDevName || '').toLowerCase();
      if (devName.includes('champions_league') || pos <= 4) {
        zoneClass = 'zone-cl';
      } else if (devName.includes('europa_league') || pos === 5) {
        zoneClass = 'zone-el';
      } else if (devName.includes('conference') || pos === 6) {
        zoneClass = 'zone-conf';
      } else if (devName.includes('relegation') || pos >= 18) {
        zoneClass = 'zone-rel';
      }

      let formHtml = '';
      if (row.Form && row.Form.length > 0) {
        formHtml = '<div class="st-form-dots">' + row.Form.map(f => {
          const l = f.toUpperCase();
          let cls = 'w';
          if (l === 'D') cls = 'd';
          else if (l === 'L') cls = 'l';
          return `<span class="st-dot ${cls}">${l}</span>`;
        }).join('') + '</div>';
      } else {
        formHtml = '<span style="color:var(--text-muted);font-size:10px;">–</span>';
      }

      const gd = row.GD || 0;
      const gdStr = gd > 0 ? `+${gd}` : `${gd}`;

      html += `
        <tr class="st-row ${zoneClass}">
          <td>
            <div class="st-pos-box">
              <span>${pos}</span>
              ${trendHtml}
            </div>
          </td>
          <td>
            <div class="st-team-cell">
              ${row.TeamLogoURL ? `<img class="st-team-logo" src="${row.TeamLogoURL}" alt="">` : ''}
              <span class="st-team-name">${escapeHtml(row.TeamName)}</span>
            </div>
          </td>
          <td class="st-num">${row.Played || 0}</td>
          <td class="st-num">${row.Won || 0}</td>
          <td class="st-num">${row.Drawn || 0}</td>
          <td class="st-num">${row.Lost || 0}</td>
          <td class="st-num">${row.GF || 0}:${row.GA || 0}</td>
          <td class="st-num" style="font-weight:700;">${gdStr}</td>
          <td>${formHtml}</td>
          <td class="st-pts">${row.Points || 0}</td>
        </tr>
      `;
    });

    html += `
            </tbody>
          </table>
        </div>
        <div class="st-legend">
          <div class="st-legend-item"><span class="st-legend-dot" style="background:#3b82f6;"></span>Лига чемпионов</div>
          <div class="st-legend-item"><span class="st-legend-dot" style="background:#f97316;"></span>Лига Европы</div>
          <div class="st-legend-item"><span class="st-legend-dot" style="background:#10b981;"></span>Лига конференций</div>
          <div class="st-legend-item"><span class="st-legend-dot" style="background:#ef4444;"></span>Зона вылета</div>
        </div>
      </div>
    `;

    standingsContainer.innerHTML = html;
  }

  // 8.2. Интерактивные лидеры турнира (Phase 7)
  async function loadLeadersData(callback) {
    const cacheKey = `${leagueId}_${seasonId}`;
    if (leadersDataCacheMap.has(cacheKey)) {
      const cached = leadersDataCacheMap.get(cacheKey);
      if (Date.now() - cached.timestamp < 600000) { // 10 мин in-memory кэш на клиенте
        leadersDataCache = cached.data;
        if (callback) callback(cached.data);
        return;
      }
    }

    loader.classList.remove('hidden');
    errorOverlay.classList.add('hidden');

    const urls = [
      `/api/leaders?league_id=${leagueId}&season_id=${seasonId}&locale=ru`,
      `data/${leagueId}_${seasonId}_leaders.json`,
      `viewer/data/${leagueId}_${seasonId}_leaders.json`,
      `${mediaBase}/${leagueId}/${seasonId}/leaders.json`
    ];

    let loadedData = null;
    for (const u of urls) {
      try {
        const resp = await fetch(u);
        if (resp.ok) {
          const json = await resp.json();
          if (!json.league_id || String(json.league_id) === String(leagueId)) {
            loadedData = json;
            break;
          }
        }
      } catch (_) {}
    }

    loader.classList.add('hidden');

    if (loadedData) {
      leadersDataCache = loadedData;
      leadersDataCacheMap.set(cacheKey, { data: loadedData, timestamp: Date.now() });
      if (callback) callback(loadedData);
    } else {
      if (callback) {
        callback(null);
      } else {
        interactiveStage.classList.add('hidden');
        stage.classList.remove('hidden');
        if (zoomActions) zoomActions.style.display = 'flex';
        loadCurrentTabImage();
      }
    }
  }

  function getLeaderSubText(catKey, item) {
    if (catKey === 'scorers') {
      return item.secondary_val > 0 ? `${item.secondary_val} пен. · ${item.matches} матчей` : `${item.matches} матчей`;
    } else if (catKey === 'assists') {
      return `${item.matches} матчей`;
    } else if (catKey === 'points') {
      const g = item.secondary_val || 0;
      const a = (item.primary_val || 0) - g;
      return `${g} г + ${a} пас`;
    } else if (catKey === 'cleansheets') {
      return `пропущено: ${item.secondary_val} / ${item.matches} матчей`;
    } else if (catKey === 'discipline') {
      if (item.secondary_val > 0) {
        return `${item.secondary_val} 🟥 · ${item.primary_val} 🟨`;
      }
      return `${item.primary_val} 🟨 · ${item.matches} матчей`;
    }
    return `${item.matches} матчей`;
  }

  function renderLeadersView(data, requestedCategory) {
    if (!leadersContainer) return;
    if (!data) return;

    if (requestedCategory) {
      currentLeaderCategory = requestedCategory;
    }

    stage.classList.add('hidden');
    interactiveStage.classList.remove('hidden');
    if (roundResultsView) roundResultsView.classList.add('hidden');
    if (matchCenterView) matchCenterView.classList.add('hidden');
    if (standingsView) standingsView.classList.add('hidden');
    if (leadersView) leadersView.classList.remove('hidden');
    if (zoomActions) zoomActions.style.display = 'none';

    if (leadersCategoryNav) {
      const chips = leadersCategoryNav.querySelectorAll('.leader-chip');
      chips.forEach(ch => {
        ch.classList.toggle('active', ch.dataset.category === currentLeaderCategory);
      });
    }

    let items = [];
    if (currentLeaderCategory === 'scorers') items = data.top_scorers || [];
    else if (currentLeaderCategory === 'assists') items = data.top_assists || [];
    else if (currentLeaderCategory === 'points') items = data.top_points || [];
    else if (currentLeaderCategory === 'cleansheets') items = data.clean_sheets || [];
    else if (currentLeaderCategory === 'discipline') items = data.discipline || [];

    if (!items || items.length === 0) {
      leadersContainer.innerHTML = '<div class="sheet-empty" style="padding:40px 16px;">Данные по выбранной категории отсутствуют</div>';
      return;
    }

    const defaultAvatarSvg = "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='%2364748b'%3E%3Cpath d='M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z'/%3E%3C/svg%3E";

    const topPlayer = items[0];
    const runnersUp = items.slice(1);

    const topSubText = getLeaderSubText(currentLeaderCategory, topPlayer);
    const topAvatar = topPlayer.player_image || defaultAvatarSvg;
    const topTeamLogo = topPlayer.team_logo ? `<img src="${topPlayer.team_logo}" alt="">` : '';

    let html = `
      <div class="leader-hero-card">
        <div class="leader-hero-badge">👑 ТОП-1</div>
        <img class="leader-hero-avatar" src="${topAvatar}" alt="${escapeHtml(topPlayer.player_name)}" onerror="this.src='${defaultAvatarSvg}'">
        <div class="leader-hero-info">
          <div class="leader-hero-name">${escapeHtml(topPlayer.player_name)}</div>
          <div class="leader-hero-team">
            ${topTeamLogo}
            <span>${escapeHtml(topPlayer.team_name || topPlayer.team_code || '')}</span>
          </div>
        </div>
        <div class="leader-hero-score">
          <div class="leader-hero-val">${topPlayer.primary_val}</div>
          <div class="leader-hero-sub">${topSubText}</div>
        </div>
      </div>
    `;

    runnersUp.forEach(p => {
      const subText = getLeaderSubText(currentLeaderCategory, p);
      const avatar = p.player_image || defaultAvatarSvg;
      const rankCls = p.rank === 2 ? 'rank-2' : (p.rank === 3 ? 'rank-3' : '');
      const teamLogo = p.team_logo ? `<img src="${p.team_logo}" alt="">` : '';

      html += `
        <div class="leader-row-card">
          <div class="leader-row-rank ${rankCls}">${p.rank}</div>
          <img class="leader-row-avatar" src="${avatar}" alt="${escapeHtml(p.player_name)}" onerror="this.src='${defaultAvatarSvg}'">
          <div class="leader-row-info">
            <div class="leader-row-name">${escapeHtml(p.player_name)}</div>
            <div class="leader-row-team">
              ${teamLogo}
              <span>${escapeHtml(p.team_name || p.team_code || '')}</span>
            </div>
          </div>
          <div class="leader-row-score">
            <div class="leader-row-val">${p.primary_val}</div>
            <div class="leader-row-sub">${subText}</div>
          </div>
        </div>
      `;
    });

    leadersContainer.innerHTML = html;
  }

  // Навешиваем клик на чипы категорий лидеров
  if (leadersCategoryNav) {
    const chips = leadersCategoryNav.querySelectorAll('.leader-chip');
    chips.forEach(chip => {
      chip.addEventListener('click', () => {
        const cat = chip.dataset.category;
        if (cat && cat !== currentLeaderCategory) {
          currentLeaderCategory = cat;
          if (leadersDataCache) {
            renderLeadersView(leadersDataCache, currentLeaderCategory);
          } else {
            loadLeadersData(data => renderLeadersView(data, currentLeaderCategory));
          }
        }
      });
    });
  }

  // 9. Рендеринг вкладок Match Center
  // 9.1. Обзор (Two-Sided Timeline Mockup B)
  function renderOverviewTimeline(events, match) {
    if (!events || events.length === 0) {
      eventsTimeline.innerHTML = '<div class="hint-text" style="padding:24px;justify-content:center;">События матча не зафиксированы</div>';
      return;
    }

    const homeTeamId = match?.TeamHomeID;
    const awayTeamId = match?.TeamAwayID;

    // Сортировка событий по времени (minute ASC, extra_minute ASC)
    const sortedEvents = [...events].sort((a, b) => {
      const mA = a.minute ?? 0;
      const mB = b.minute ?? 0;
      if (mA !== mB) return mA - mB;
      return (a.extra_minute || 0) - (b.extra_minute || 0);
    });

    const firstHalfEvents = sortedEvents.filter(ev => (ev.minute || 0) <= 45);
    const secondHalfEvents = sortedEvents.filter(ev => (ev.minute || 0) > 45);

    const hHT = match?.Scores?.HomeHT ?? 0;
    const aHT = match?.Scores?.AwayHT ?? 0;
    const hFT = match?.Scores?.HomeCurrent ?? 0;
    const aFT = match?.Scores?.AwayCurrent ?? 0;

    function renderEventRow(ev) {
      const isHome = homeTeamId ? (ev.team_id === homeTeamId) : true;
      const type = (ev.type || '').toUpperCase();
      let icon = '⚽';
      const shortName = ev.player ? getShortPlayerName(ev.player) : '';
      let title = shortName;
      let subtitle = '';
      let isGoal = false;
      let isCard = false;
      let cardClass = '';

      if (type.includes('YELLOW') && type.includes('RED')) {
        icon = '🟨🟥';
        isCard = true;
        cardClass = 'card-red';
        subtitle = 'Вторая жёлтая';
      } else if (type.includes('YELLOW')) {
        icon = '🟨';
        isCard = true;
        cardClass = 'card-yellow';
        subtitle = '';
      } else if (type.includes('RED')) {
        icon = '🟥';
        isCard = true;
        cardClass = 'card-red';
        subtitle = 'Красная карточка';
      } else if (type.includes('SUBST')) {
        icon = '🔄';
        if (ev.related_player) {
          const relShort = getShortPlayerName(ev.related_player);
          if (relShort) subtitle = `вместо: ${relShort}`;
        }
      } else if (type.includes('VAR')) {
        icon = '🖥️';
        if (ev.addition) subtitle = ev.addition;
      } else if (type.includes('GOAL') || type.includes('PENALTY') || type.includes('OWNGOAL')) {
        icon = '⚽';
        isGoal = true;
        if (type.includes('PENALTY')) subtitle = 'Пенальти';
        else if (type.includes('OWNGOAL')) subtitle = 'Автогол';
        else if (ev.related_player) {
          const relShort = getShortPlayerName(ev.related_player);
          if (relShort) subtitle = `пас: ${relShort}`;
        }
      }

      if (!title) {
        if (isCard) title = 'Предупреждение';
        else title = 'Игрок';
      }

      const minuteStr = ev.extra_minute ? `${ev.minute}+${ev.extra_minute}'` : `${ev.minute}'`;
      const scoreStr = ev.result ? `[ ${ev.result} ]` : '';
      const cardClassType = isGoal ? 'goal' : (isCard ? 'card' : 'other');

      let rowInner = '';
      if (isHome) {
        rowInner = `
          <div class="timeline-card home ${cardClassType}">
            <div class="tc-main">
              <span class="tc-player">${escapeHtml(title)}</span>
              <span class="tc-icon ${cardClass}">${icon}</span>
              ${scoreStr ? `<span class="tc-score">${escapeHtml(scoreStr)}</span>` : ''}
            </div>
            ${subtitle ? `<div class="tc-sub">${escapeHtml(subtitle)}</div>` : ''}
          </div>
          <div class="timeline-node">
            <span class="node-min">${minuteStr}</span>
          </div>
          <div class="timeline-spacer"></div>
        `;
      } else {
        rowInner = `
          <div class="timeline-spacer"></div>
          <div class="timeline-node">
            <span class="node-min">${minuteStr}</span>
          </div>
          <div class="timeline-card away ${cardClassType}">
            <div class="tc-main">
              ${scoreStr ? `<span class="tc-score">${escapeHtml(scoreStr)}</span>` : ''}
              <span class="tc-icon ${cardClass}">${icon}</span>
              <span class="tc-player">${escapeHtml(title)}</span>
            </div>
            ${subtitle ? `<div class="tc-sub">${escapeHtml(subtitle)}</div>` : ''}
          </div>
        `;
      }

      return `<div class="timeline-row ${isHome ? 'home' : 'away'}">${rowInner}</div>`;
    }

    let html = '<div class="timeline-container">';

    // 1-й тайм
    html += `
      <div class="timeline-period-badge">
        <span class="period-icon">⏱</span>
        <span>1-й тайм</span>
      </div>
      <div class="timeline-stream">
        <div class="timeline-spine"></div>
    `;

    if (firstHalfEvents.length > 0) {
      firstHalfEvents.forEach(ev => {
        html += renderEventRow(ev);
      });
    } else {
      html += '<div class="timeline-empty-period">Без ключевых событий в 1-м тайме</div>';
    }

    html += `</div>`;

    // Перерыв (HT)
    html += `
      <div class="timeline-period-badge halftime">
        <span>Перерыв (${hHT} : ${aHT})</span>
      </div>
      <div class="timeline-stream">
        <div class="timeline-spine"></div>
    `;

    // 2-й тайм
    if (secondHalfEvents.length > 0) {
      secondHalfEvents.forEach(ev => {
        html += renderEventRow(ev);
      });
    } else {
      html += '<div class="timeline-empty-period">Без ключевых событий во 2-м тайме</div>';
    }

    html += `</div>`;

    // Матч окончен (FT)
    html += `
      <div class="timeline-period-badge fulltime">
        <span>🏁 Матч окончен (${hFT} : ${aFT})</span>
      </div>
    </div>`;

    eventsTimeline.innerHTML = html;
  }

  // 9.2. Статистика (Comparison Bars & Pre-Match Profiles)
  let preMatchStatsActivePeriod = 'last_5';

  function renderStatsBars(stats, match) {
    if (match && match.StateShort !== 'FT' && match.PreMatchStats) {
      renderPreMatchStats(match.PreMatchStats, match);
      return;
    }

    const keys = Object.keys(stats || {});
    if (keys.length === 0) {
      statsBarsContainer.innerHTML = '<div class="hint-text" style="padding:20px;justify-content:center;">Статистика матча недоступна</div>';
      return;
    }

    // Приоритетные метрики для верха
    const priority = ['EXPECTED_GOALS', 'BALL_POSSESSION', 'SHOTS_TOTAL', 'SHOTS_ON_TARGET', 'CORNERS', 'FOULS', 'SAVES'];
    const sortedKeys = keys.sort((a, b) => {
      const idxA = priority.indexOf(a);
      const idxB = priority.indexOf(b);
      if (idxA !== -1 && idxB !== -1) return idxA - idxB;
      if (idxA !== -1) return -1;
      if (idxB !== -1) return 1;
      return a.localeCompare(b);
    });

    let html = '';
    sortedKeys.forEach(k => {
      const s = stats[k];
      const hVal = s.home !== undefined && s.home !== null ? Number(s.home) : 0;
      const aVal = s.away !== undefined && s.away !== null ? Number(s.away) : 0;
      const total = (hVal + aVal) || 1;

      const hPercent = Math.round((hVal / total) * 100);
      const aPercent = 100 - hPercent;

      const title = s.name || k.replace(/_/g, ' ');

      html += `
        <div class="stat-item">
          <div class="stat-header">
            <span class="stat-val home">${s.home ?? 0}</span>
            <span class="stat-title">${escapeHtml(title)}</span>
            <span class="stat-val away">${s.away ?? 0}</span>
          </div>
          <div class="stat-bars-dual">
            <div class="stat-bar-half home">
              <div class="stat-bar-fill home" style="width: ${hPercent}%"></div>
            </div>
            <div class="stat-bar-half away">
              <div class="stat-bar-fill away" style="width: ${aPercent}%"></div>
            </div>
          </div>
        </div>
      `;
    });

    statsBarsContainer.innerHTML = html;
  }

  function renderPreMatchStats(preStats, match) {
    const home = preStats.home;
    const away = preStats.away;
    if (!home || !away) {
      statsBarsContainer.innerHTML = '<div class="hint-text" style="padding:20px;justify-content:center;">Предматчевая статистика недоступна</div>';
      return;
    }

    const pH = (preMatchStatsActivePeriod === 'season') ? (home.season || home.last_5) : (home.last_5 || home.season);
    const pA = (preMatchStatsActivePeriod === 'season') ? (away.season || away.last_5) : (away.last_5 || away.season);

    let html = `
      <div class="prematch-filter-bar">
        <div class="segmented-toggle">
          <button class="segmented-btn ${preMatchStatsActivePeriod === 'last_5' ? 'active' : ''}" id="btnPeriodLast5">5 последних туров</button>
          <button class="segmented-btn ${preMatchStatsActivePeriod === 'season' ? 'active' : ''}" id="btnPeriodSeason">Весь сезон</button>
        </div>
      </div>

      <div class="hero-stats-grid">
        <div class="hero-stat-card">
          <span class="hero-stat-badge">⚡</span>
          <span class="hero-stat-label">xG за матч</span>
          <div class="hero-stat-values">
            <span class="hero-val home">${(pH.xg_per_match || 0).toFixed(2)}</span>
            <span class="hero-val-divider">:</span>
            <span class="hero-val away">${(pA.xg_per_match || 0).toFixed(2)}</span>
          </div>
        </div>
        <div class="hero-stat-card">
          <span class="hero-stat-badge">🛡️</span>
          <span class="hero-stat-label">xGA за матч</span>
          <div class="hero-stat-values">
            <span class="hero-val home">${(pH.xga_per_match || 0).toFixed(2)}</span>
            <span class="hero-val-divider">:</span>
            <span class="hero-val away">${(pA.xga_per_match || 0).toFixed(2)}</span>
          </div>
        </div>
        <div class="hero-stat-card">
          <span class="hero-stat-badge">⚽</span>
          <span class="hero-stat-label">Голы за матч</span>
          <div class="hero-stat-values">
            <span class="hero-val home">${(pH.goals_scored_avg || 0).toFixed(1)}</span>
            <span class="hero-val-divider">:</span>
            <span class="hero-val away">${(pA.goals_scored_avg || 0).toFixed(1)}</span>
          </div>
        </div>
      </div>

      <div class="prematch-bars-list">
    `;

    const secMetrics = [
      {
        title: 'Обе забьют (BTTS %)',
        hVal: pH.btts_percent || 0,
        aVal: pA.btts_percent || 0,
        suffix: '%'
      },
      {
        title: 'Тотал больше 2.5 (%)',
        hVal: pH.over25_percent || 0,
        aVal: pA.over25_percent || 0,
        suffix: '%'
      },
      {
        title: 'Удары в створ (в ср.)',
        hVal: (pH.shots_on_target_avg || 0).toFixed(1),
        aVal: (pA.shots_on_target_avg || 0).toFixed(1),
        suffix: ''
      },
      {
        title: 'Владение мячом (%)',
        hVal: Math.round(pH.possession_avg || 50),
        aVal: Math.round(pA.possession_avg || 50),
        suffix: '%'
      }
    ];

    secMetrics.forEach(m => {
      const hN = Number(m.hVal) || 0;
      const aN = Number(m.aVal) || 0;
      const total = (hN + aN) || 1;
      const hPercent = Math.round((hN / total) * 100);
      const aPercent = 100 - hPercent;

      html += `
        <div class="stat-item">
          <div class="stat-header">
            <span class="stat-val home">${m.hVal}${m.suffix}</span>
            <span class="stat-title">${escapeHtml(m.title)}</span>
            <span class="stat-val away">${m.aVal}${m.suffix}</span>
          </div>
          <div class="stat-bars-dual">
            <div class="stat-bar-half home">
              <div class="stat-bar-fill home" style="width: ${hPercent}%"></div>
            </div>
            <div class="stat-bar-half away">
              <div class="stat-bar-fill away" style="width: ${aPercent}%"></div>
            </div>
          </div>
        </div>
      `;
    });

    html += `</div>`;
    statsBarsContainer.innerHTML = html;

    document.getElementById('btnPeriodLast5')?.addEventListener('click', () => {
      preMatchStatsActivePeriod = 'last_5';
      renderPreMatchStats(preStats, match);
    });
    document.getElementById('btnPeriodSeason')?.addEventListener('click', () => {
      preMatchStatsActivePeriod = 'season';
      renderPreMatchStats(preStats, match);
    });
  }

  // 9.3. Составы и Тактическое поле
  function renderLineupsPitch(lineups, coaches, match) {
    if (match && match.StateShort !== 'FT' && (!lineups.home || lineups.home.length === 0)) {
      pitchContainer.style.display = 'none';
      mcCoachesCard.style.display = 'none';
      mcBenchCard.style.display = 'none';

      let waitingCard = document.getElementById('lineupsWaitingCard');
      if (!waitingCard) {
        waitingCard = document.createElement('div');
        waitingCard.id = 'lineupsWaitingCard';
        document.getElementById('paneLineups')?.appendChild(waitingCard);
      }
      waitingCard.className = 'lineups-waiting-card';
      waitingCard.style.display = 'flex';
      waitingCard.innerHTML = `
        <div class="lw-icon">⏳</div>
        <div class="lw-title">Официальные составы команд</div>
        <div class="lw-desc">
          Стартовые составы и тактическая расстановка на поле появятся за 60 минут до стартового свистка.
        </div>
        <div class="lw-info-pills">
          <span class="lw-pill">🏟️ ${escapeHtml(match.VenueName || 'Стадион')}</span>
          ${match.RefereeName ? `<span class="lw-pill">⚖️ ${escapeHtml(match.RefereeName)}</span>` : ''}
          ${match.IsRival ? `<span class="lw-pill" style="color:#f87171;">🔥 Принципиальное дерби</span>` : ''}
        </div>
      `;
      return;
    } else {
      pitchContainer.style.display = 'flex';
      mcCoachesCard.style.display = 'block';
      mcBenchCard.style.display = 'block';
      const waitingCard = document.getElementById('lineupsWaitingCard');
      if (waitingCard) waitingCard.style.display = 'none';
    }

    const homePlayers = lineups.home || [];
    const awayPlayers = lineups.away || [];

    // Гарантируем уникальность игроков и ровно 11 стартеров без дублей
    const homeStarters = filterUniqueStarters(homePlayers);
    const awayStarters = filterUniqueStarters(awayPlayers);

    // Рассчитываем аккуратные, не накладывающиеся координаты
    const homePositions = layoutTeamStarters(homeStarters, true);
    let homeHtml = '';
    homeStarters.forEach((p, idx) => {
      const pos = homePositions[idx] || { top: 75, left: 50 };
      const isGK = p.position_id === 24 || p.formation_field === '1:1' || p.formation_position === 1;
      homeHtml += `
        <div class="pitch-player home${isGK ? ' gk' : ''}" style="top: ${pos.top}%; left: ${pos.left}%;">
          <div class="pitch-number">${p.number || ''}</div>
          <div class="pitch-name" title="${escapeHtml(p.name)}">${escapeHtml(getShortPlayerName(p.name))}</div>
        </div>
      `;
    });
    pitchHomePlayers.innerHTML = homeHtml;

    const awayPositions = layoutTeamStarters(awayStarters, false);
    let awayHtml = '';
    awayStarters.forEach((p, idx) => {
      const pos = awayPositions[idx] || { top: 25, left: 50 };
      const isGK = p.position_id === 24 || p.formation_field === '1:1' || p.formation_position === 1;
      awayHtml += `
        <div class="pitch-player away${isGK ? ' gk' : ''}" style="top: ${pos.top}%; left: ${pos.left}%;">
          <div class="pitch-number">${p.number || ''}</div>
          <div class="pitch-name" title="${escapeHtml(p.name)}">${escapeHtml(getShortPlayerName(p.name))}</div>
        </div>
      `;
    });
    pitchAwayPlayers.innerHTML = awayHtml;

    // Главные тренеры
    const homeCoach = coaches.home?.name || 'Главный тренер';
    const awayCoach = coaches.away?.name || 'Главный тренер';
    mcCoachesCard.innerHTML = `
      <div class="coaches-title">Главные тренеры</div>
      <div class="coaches-row">
        <div class="coach-item home">
          ${match?.TeamHomeLogo ? `<img class="coach-club-logo" src="${match.TeamHomeLogo}" alt="">` : '<span>👔</span>'}
          <div class="coach-meta">
            <div class="coach-role">ТРЕНЕР</div>
            <div class="coach-name">${escapeHtml(homeCoach)}</div>
          </div>
        </div>
        <div class="coach-item away">
          <div class="coach-meta">
            <div class="coach-role">ТРЕНЕР</div>
            <div class="coach-name">${escapeHtml(awayCoach)}</div>
          </div>
          ${match?.TeamAwayLogo ? `<img class="coach-club-logo" src="${match.TeamAwayLogo}" alt="">` : '<span>👔</span>'}
        </div>
      </div>
    `;

    // Скамейка запасных
    const homeBench = homePlayers.filter(p => !p.is_starter);
    const awayBench = awayPlayers.filter(p => !p.is_starter);
    const maxBench = Math.max(homeBench.length, awayBench.length);

    // Минуты выхода на замену из событий матча
    const subEvents = (match?.Events || []).filter(e => (e.type || '').toUpperCase().includes('SUBST'));
    const subbedMinutes = {};
    subEvents.forEach(e => {
      if (e.player) {
        subbedMinutes[e.player.toLowerCase().trim()] = e.minute;
      }
    });

    let benchHtml = '<div class="bench-title">Запасные игроки</div><div class="bench-list">';
    for (let i = 0; i < maxBench; i++) {
      const hp = homeBench[i];
      const ap = awayBench[i];

      const hSubMin = hp ? subbedMinutes[hp.name?.toLowerCase().trim()] : null;
      const aSubMin = ap ? subbedMinutes[ap.name?.toLowerCase().trim()] : null;

      const hSubTag = hSubMin ? `<span class="bench-sub-badge">${hSubMin}'</span>` : '';
      const aSubTag = aSubMin ? `<span class="bench-sub-badge">${aSubMin}'</span>` : '';

      const hText = hp ? `<span class="bench-num">${hp.number || ''}</span> ${escapeHtml(getShortPlayerName(hp.name))} ${hSubTag}` : '';
      const aText = ap ? `<span class="bench-num">${ap.number || ''}</span> ${escapeHtml(getShortPlayerName(ap.name))} ${aSubTag}` : '';

      benchHtml += `
        <div class="bench-player home">${hText}</div>
        <div class="bench-player away">${aText}</div>
      `;
    }
    benchHtml += '</div>';
    mcBenchCard.innerHTML = benchHtml;
  }

  // 9.4. H2H (Личные встречи)
  function renderH2H(match) {
    if (!match.H2H || !match.H2H.summary || match.H2H.summary.total_matches === 0) {
      h2hContainer.innerHTML = `
        <div class="round-summary-card" style="text-align:center;padding:24px;">
          <div class="rs-title">История противостояний</div>
          <div style="font-size:13px;color:var(--text-secondary);margin-top:8px;">
            В базе данных пока нет предыдущих очных встреч этих команд.
          </div>
          ${match.IsRival ? '<div style="margin-top:10px;color:#f87171;font-weight:700;font-size:12px;">🔥 Принципиальное дерби</div>' : ''}
        </div>
      `;
      return;
    }

    const s = match.H2H.summary;
    let html = `
      <div class="h2h-summary-banner">
        <div class="h2h-banner-title">ОЧНЫЕ ВСТРЕЧИ В БАЗЕ ДАННЫХ</div>
        <div class="h2h-banner-counts">
          <div class="h2h-count-item">
            <span class="h2h-count-val home">${s.home_wins}</span>
            <span class="h2h-count-label">${escapeHtml(match.TeamHomeName)}</span>
          </div>
          <div class="h2h-count-item">
            <span class="h2h-count-val draw">${s.draws}</span>
            <span class="h2h-count-label">Ничьи</span>
          </div>
          <div class="h2h-count-item">
            <span class="h2h-count-val away">${s.away_wins}</span>
            <span class="h2h-count-label">${escapeHtml(match.TeamAwayName)}</span>
          </div>
        </div>
        <div class="h2h-banner-footer">
          Всего матчей: <b>${s.total_matches}</b> • Средний тотал голов: <b>${(s.avg_goals || 0).toFixed(1)}</b>
          ${match.IsRival ? ' • <span style="color:#f87171;font-weight:700;">🔥 Дерби</span>' : ''}
        </div>
      </div>
      <div class="h2h-matches-list">
    `;

    (match.H2H.matches || []).forEach(m => {
      const homeXG = m.home_xg != null ? `<span class="h2h-mc-xg-pill">${Number(m.home_xg).toFixed(2)}</span>` : '';
      const awayXG = m.away_xg != null ? `<span class="h2h-mc-xg-pill">${Number(m.away_xg).toFixed(2)}</span>` : '';
      const xgHtml = (homeXG || awayXG) ? `<div class="h2h-mc-xg-row">xG: ${homeXG || '–'} — ${awayXG || '–'}</div>` : '';

      html += `
        <div class="h2h-match-card">
          <div class="h2h-mc-header">
            <span>${escapeHtml(m.date_str)}</span>
            <span class="h2h-mc-tournament">${escapeHtml(m.league_name)}</span>
          </div>
          <div class="h2h-mc-teams-row">
            <div class="h2h-mc-team home">
              <span class="h2h-mc-name">${escapeHtml(m.home_team_name)}</span>
              ${m.home_team_logo ? `<img class="h2h-mc-logo" src="${m.home_team_logo}" alt="">` : ''}
            </div>
            <div class="h2h-mc-score">${m.home_score} : ${m.away_score}</div>
            <div class="h2h-mc-team away">
              ${m.away_team_logo ? `<img class="h2h-mc-logo" src="${m.away_team_logo}" alt="">` : ''}
              <span class="h2h-mc-name">${escapeHtml(m.away_team_name)}</span>
            </div>
          </div>
          ${xgHtml}
        </div>
      `;
    });

    html += `</div>`;
    h2hContainer.innerHTML = html;
  }

  // Фильтрация уникальных 11 стартеров (без повторов одного игрока)
  function filterUniqueStarters(players) {
    const seen = new Set();
    const starters = [];
    players.forEach(p => {
      if (p.is_starter && !seen.has(p.player_id)) {
        seen.add(p.player_id);
        starters.push(p);
      }
    });
    if (starters.length < 11) {
      players.forEach(p => {
        if (!seen.has(p.player_id) && starters.length < 11) {
          seen.add(p.player_id);
          starters.push(p);
        }
      });
    }
    return starters.slice(0, 11);
  }

  // Раскладка игроков на тактическом поле: группировка по линиям с равными отступами
  function layoutTeamStarters(starters, isHome) {
    const items = starters.map((p, originalIndex) => {
      let row = 0;
      let col = 0;
      if (p.formation_field && typeof p.formation_field === 'string') {
        const parts = p.formation_field.split(':');
        row = parseInt(parts[0], 10) || 0;
        col = parseInt(parts[1], 10) || 0;
      }
      if (!row) {
        if (p.position_id === 24) row = 1;
        else if (p.position_id === 25) row = 2;
        else if (p.position_id === 26) row = 3;
        else if (p.position_id === 27) row = 4;
        else row = 2;
      }
      if (!col) {
        col = p.formation_position || (originalIndex + 1);
      }
      return { p, originalIndex, row, col };
    });

    const rowsMap = {};
    items.forEach(it => {
      if (!rowsMap[it.row]) rowsMap[it.row] = [];
      rowsMap[it.row].push(it);
    });

    Object.keys(rowsMap).forEach(r => {
      rowsMap[r].sort((a, b) => a.col - b.col || a.originalIndex - b.originalIndex);
    });

    const maxRow = Math.max(...items.map(it => it.row), 1);
    let rowTops = {};
    if (isHome) {
      if (maxRow <= 4) {
        rowTops = { 1: 92, 2: 78, 3: 67, 4: 57 };
      } else {
        rowTops = { 1: 92, 2: 80, 3: 70, 4: 62, 5: 56 };
      }
    } else {
      if (maxRow <= 4) {
        rowTops = { 1: 8, 2: 22, 3: 33, 4: 43 };
      } else {
        rowTops = { 1: 8, 2: 20, 3: 30, 4: 38, 5: 44 };
      }
    }
    const resultPositions = new Array(starters.length);

    Object.keys(rowsMap).forEach(rStr => {
      const r = parseInt(rStr, 10);
      const rowItems = rowsMap[r];
      const count = rowItems.length;
      const top = rowTops[r] || (isHome ? Math.max(51, 91 - (r - 1) * 10) : Math.min(49, 9 + (r - 1) * 10));

      rowItems.forEach((it, indexInRow) => {
        let left = 50;
        if (count > 1) {
          const padding = count === 2 ? 26 : (count === 3 ? 18 : 12);
          const minX = padding;
          const maxX = 100 - padding;
          left = minX + (indexInRow / (count - 1)) * (maxX - minX);
        }
        resultPositions[it.originalIndex] = {
          top: Math.round(top * 10) / 10,
          left: Math.round(left * 10) / 10
        };
      });
    });

    return resultPositions;
  }

  function getShortPlayerName(name) {
    if (!name) return '';
    const cleaned = name.replace(/[\u00a0\s]+/g, ' ').trim();
    if (!cleaned) return '';
    const parts = cleaned.split(' ');
    if (parts.length > 1) {
      const first = parts[0];
      const initial = first.endsWith('.') ? first : `${first[0]}.`;
      return `${initial} ${parts[parts.length - 1]}`;
    }
    return cleaned;
  }

  function escapeHtml(str) {
    if (!str) return '';
    return str.replace(/[&<>"']/g, m => ({
      '&': '&amp;',
      '<': '&lt;',
      '>': '&gt;',
      '"': '&quot;',
      "'": '&#39;'
    })[m]);
  }

  function formatJsRussianDate(d) {
    const months = ['Янв.', 'Фев.', 'Мар.', 'Апр.', 'Мая', 'Июн.', 'Июл.', 'Авг.', 'Сен.', 'Окт.', 'Ноя.', 'Дек.'];
    const days = ['Воскресенье', 'Понедельник', 'Вторник', 'Среда', 'Четверг', 'Пятница', 'Суббота'];
    return `${d.getDate()} ${months[d.getMonth()]} ${d.getFullYear()}, ${days[d.getDay()]}`;
  }

  // 10. Переключение суб-вкладок Match Center (Overview / Stats / Lineups / H2H)
  mcSubtabButtons.forEach(btn => {
    btn.addEventListener('click', () => {
      const targetSubtab = btn.dataset.subtab;
      mcSubtabButtons.forEach(b => b.classList.toggle('active', b === btn));

      document.querySelectorAll('.mc-tab-pane').forEach(pane => {
        pane.classList.remove('active');
      });

      if (targetSubtab === 'overview') document.getElementById('paneOverview')?.classList.add('active');
      if (targetSubtab === 'stats') document.getElementById('paneStats')?.classList.add('active');
      if (targetSubtab === 'lineups') document.getElementById('paneLineups')?.classList.add('active');
      if (targetSubtab === 'h2h') document.getElementById('paneH2H')?.classList.add('active');
    });
  });

  // 11. Переключение главных табов (Таблица / Матчи / Лидеры)
  function selectTab(tabKey) {
    if (!TAB_FILES[tabKey]) return;
    if (tabKey === 'fixtures' || tabKey === 'results') tabKey = 'matches';
    currentTab = tabKey;

    tabButtons.forEach(btn => {
      const isActive = btn.dataset.tab === tabKey;
      btn.classList.toggle('active', isActive);
      btn.setAttribute('aria-selected', isActive ? 'true' : 'false');
    });

    if (tabKey === 'matches') {
      stage.classList.add('hidden');
      interactiveStage.classList.remove('hidden');
      if (standingsView) standingsView.classList.add('hidden');
      if (leadersView) leadersView.classList.add('hidden');
      if (roundResultsView) roundResultsView.classList.remove('hidden');
      if (matchCenterView) matchCenterView.classList.add('hidden');
      if (zoomActions) zoomActions.style.display = 'none';

      const targetRoundId = getRoundIdForTab('matches');
      const active = getActiveLeague();
      let initialRoundLabel = displayRound;
      if (active?.next_round) {
        initialRoundLabel = formatRoundName(active.next_round);
      } else if (active?.curr_round) {
        initialRoundLabel = formatRoundName(active.curr_round);
      }

      headerContext.textContent = `${displayLeague.toUpperCase()} • ${initialRoundLabel.toUpperCase()} • ${TAB_TITLES.matches}`;
      document.title = `${displayLeague} — ${initialRoundLabel} | ${TAB_TITLES.matches}`;

      const btnBackText = document.getElementById('btnBackText');
      if (btnBackText) {
        btnBackText.textContent = 'К матчам тура';
      }

      // Обновляем адресную строку с актуальным туром для выбранной вкладки
      try {
        const u = new URL(window.location.href);
        u.searchParams.set('tab', 'matches');
        if (targetRoundId) u.searchParams.set('round', targetRoundId);
        window.history.replaceState({}, '', u.toString());
      } catch (_) {}

      loadRoundData(targetRoundId, data => {
        if (!data || !data.matches || data.matches.length === 0) {
          interactiveStage.classList.add('hidden');
          stage.classList.remove('hidden');
          if (zoomActions) zoomActions.style.display = 'flex';
          loadCurrentTabImage();
          return;
        }

        if (data.round_name) {
          const rn = formatRoundName(data.round_name);
          headerContext.textContent = `${displayLeague.toUpperCase()} • ${rn.toUpperCase()} • ${TAB_TITLES.matches}`;
          document.title = `${displayLeague} — ${rn} | ${TAB_TITLES.matches}`;
        }

        renderRoundResultsView(data);
        updateRoundPager(data);
        if (targetMatchId) {
          openMatchCenter(targetMatchId);
        } else {
          showRoundResults();
        }
      });
    } else if (tabKey === 'table') {
      if (roundResultsView) roundResultsView.classList.add('hidden');
      if (matchCenterView) matchCenterView.classList.add('hidden');
      if (leadersView) leadersView.classList.add('hidden');
      headerContext.textContent = `${displayLeague.toUpperCase()} • ${TAB_TITLES.table}`;
      document.title = `${displayLeague} — ${TAB_TITLES.table} | Football Data Lab`;
      
      const targetRoundId = getRoundIdForTab('table');
      loadRoundData(targetRoundId, data => {
        if (data && data.standings && data.standings.length > 0) {
          renderStandingsView(data.standings);
        } else {
          interactiveStage.classList.add('hidden');
          stage.classList.remove('hidden');
          if (zoomActions) zoomActions.style.display = 'flex';
          loadCurrentTabImage();
        }
      });
    } else if (tabKey === 'leaders') {
      if (roundResultsView) roundResultsView.classList.add('hidden');
      if (matchCenterView) matchCenterView.classList.add('hidden');
      if (standingsView) standingsView.classList.add('hidden');
      headerContext.textContent = `${displayLeague.toUpperCase()} • ${TAB_TITLES.leaders}`;
      document.title = `${displayLeague} — ${TAB_TITLES.leaders} | Football Data Lab`;
      loadLeadersData(data => {
        if (data) {
          renderLeadersView(data, currentLeaderCategory);
        } else {
          interactiveStage.classList.add('hidden');
          stage.classList.remove('hidden');
          if (zoomActions) zoomActions.style.display = 'flex';
          loadCurrentTabImage();
        }
      });
    }
  }

  tabButtons.forEach(btn => {
    btn.addEventListener('click', (e) => {
      e.preventDefault();
      selectTab(btn.dataset.tab);
    });
  });

  btnRetry.addEventListener('click', () => {
    if (currentTab === 'matches' || currentTab === 'table') {
      const targetRoundId = getRoundIdForTab(currentTab);
      roundDataCache.delete(`${leagueId}_${targetRoundId}`);
      selectTab(currentTab);
    } else if (currentTab === 'leaders') {
      leadersDataCache = null;
      selectTab('leaders');
    } else {
      loadCurrentTabImage();
    }
  });

  // 12. Кнопки масштабирования в шапке (для PNG)
  if (btnZoomIn) btnZoomIn.addEventListener('click', () => setZoom(state.scale * 1.3));
  if (btnZoomOut) btnZoomOut.addEventListener('click', () => setZoom(state.scale / 1.3));
  if (btnZoomReset) btnZoomReset.addEventListener('click', resetZoom);

  // 13. Жесты Touch: Pinch-to-zoom и Pan для PNG режима
  viewport.addEventListener('touchstart', (e) => {
    if (currentTab === 'matches' || currentTab === 'table' || currentTab === 'leaders') return; // В интерактивном режиме скроллится нативно
    if (e.touches.length === 1) {
      const now = Date.now();
      if (now - state.lastTapTime < 300) {
        if (state.scale > 1.2) {
          resetZoom();
        } else {
          setZoom(2.2, e.touches[0].clientX - viewport.clientWidth / 2, e.touches[0].clientY - viewport.clientHeight / 2);
        }
        state.lastTapTime = 0;
        return;
      }
      state.lastTapTime = now;
      state.isDragging = true;
      state.startX = e.touches[0].clientX - state.posX;
      state.startY = e.touches[0].clientY - state.posY;
    } else if (e.touches.length === 2) {
      state.isDragging = false;
      state.lastTouchDist = Math.hypot(
        e.touches[0].clientX - e.touches[1].clientX,
        e.touches[0].clientY - e.touches[1].clientY
      );
    }
  }, { passive: true });

  viewport.addEventListener('touchmove', (e) => {
    if (currentTab === 'matches' || currentTab === 'table' || currentTab === 'leaders') return;
    if (e.touches.length === 1 && state.isDragging) {
      state.posX = e.touches[0].clientX - state.startX;
      state.posY = e.touches[0].clientY - state.startY;
      updateTransform();
    } else if (e.touches.length === 2) {
      const dist = Math.hypot(
        e.touches[0].clientX - e.touches[1].clientX,
        e.touches[0].clientY - e.touches[1].clientY
      );
      if (state.lastTouchDist > 0) {
        const factor = dist / state.lastTouchDist;
        setZoom(state.scale * factor);
      }
      state.lastTouchDist = dist;
    }
  }, { passive: true });

  viewport.addEventListener('touchend', (e) => {
    if (currentTab === 'matches' || currentTab === 'table' || currentTab === 'leaders') return;
    if (e.touches.length < 2) state.lastTouchDist = 0;
    if (e.touches.length === 0) state.isDragging = false;
  });

  // 14. Стартовый запуск
  (async () => {
    await loadLeaguesCatalog();
    if (!roundId) {
      const active = getActiveLeague();
      if (active) {
        roundId = String(active.next_round_id || active.curr_round_id || '');
      }
    }
    selectTab(currentTab);
  })();
})();
