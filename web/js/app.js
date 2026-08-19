// ---------- Language selector ----------

const LANG_STORAGE_KEY = 'rabbitvocab_language_id';

function getSelectedLanguageId() {
  const v = localStorage.getItem(LANG_STORAGE_KEY);
  return v ? Number(v) : null;
}

function setSelectedLanguageId(id) {
  localStorage.setItem(LANG_STORAGE_KEY, String(id));
}

async function initLanguageSelector() {
  const select = document.getElementById('language-select');

  let langs = [];
  try {
    langs = (await withTimeout(apiFetch('/languages'), 4000)) || [];
  } catch (err) {
    return; // offline or unreachable — selector keeps whatever was last stored
  }

  if (langs.length === 0) {
    select.innerHTML = '<option value="">No languages yet</option>';
    return;
  }

  select.innerHTML = langs.map((l) => `<option value="${l.ID}">${escapeHtml(l.Name)}</option>`).join('');

  const stored = getSelectedLanguageId();
  const validStored = stored && langs.some((l) => l.ID === stored);
  select.value = validStored ? String(stored) : String(langs[0].ID);
  if (!validStored) setSelectedLanguageId(langs[0].ID);

  select.addEventListener('change', () => {
    setSelectedLanguageId(Number(select.value));
    syncAllData(getSelectedLanguageId());
    renderers[currentTab]?.();
  });
}

// ---------- Full dataset sync (runs on every load, language switch, and reconnect) ----------

function withTimeout(promise, ms) {
  return Promise.race([
    promise,
    new Promise((_, reject) => setTimeout(() => reject(new Error('timeout')), ms)),
  ]);
}

async function fetchAllPages(basePath, languageId) {
  const pageSize = 100;
  let page = 1;
  let all = [];

  while (true) {
    const result = await withTimeout(
      apiFetch(`${basePath}?page=${page}&pageSize=${pageSize}&languageId=${languageId}`),
      6000
    );
    if (!result || !result.items) break;
    all = all.concat(result.items);
    const totalPages = Math.max(1, Math.ceil((result.total || 0) / pageSize));
    if (page >= totalPages) break;
    page++;
  }

  return all;
}

// Pulls the FULL word and word-form set for a language into IndexedDB.
// This is what makes offline testing correct — testing always selects from
// this complete local set, never from whatever a single /random call
// happened to return last.
async function syncAllData(languageId) {
  if (!languageId || !navigator.onLine) return;

  try {
    const words = await fetchAllPages('/words', languageId);
    await saveFullWordSet(languageId, words);
  } catch (err) {
    console.error('word sync failed', err);
  }

  try {
    const forms = await fetchAllPages('/word-forms', languageId);
    await saveFullWordFormSet(languageId, forms);
  } catch (err) {
    console.error('word form sync failed', err);
  }
}

// ---------- Client-side weighted random selection (mirrors backend logic) ----------
// Excludes mastered items; weights new/learning/confident so less-known
// items surface more often. Runs entirely client-side so it works
// identically online or offline.

const STATE_WEIGHT = { new: 4, learning: 3, confident: 2 };

function weightedSample(items, count) {
  const pool = items.filter((i) => i.State !== 'mastered');
  const picked = [];

  while (pool.length && picked.length < count) {
    const totalWeight = pool.reduce((sum, i) => sum + (STATE_WEIGHT[i.State] || 1), 0);
    let r = Math.random() * totalWeight;
    let idx = 0;
    for (; idx < pool.length - 1; idx++) {
      r -= STATE_WEIGHT[pool[idx].State] || 1;
      if (r <= 0) break;
    }
    picked.push(pool.splice(idx, 1)[0]);
  }

  return picked;
}

// ---------- Tab switching ----------

const tabButtons = document.querySelectorAll('.tab-btn');
const tabPanels = document.querySelectorAll('.tab-panel');
let currentTab = 'languages';

function showTab(name) {
  currentTab = name;
  tabButtons.forEach((b) => b.classList.toggle('active', b.dataset.tab === name));
  tabPanels.forEach((p) => p.classList.toggle('hidden', p.id !== `tab-${name}`));
  renderers[name]?.();
}

tabButtons.forEach((btn) => btn.addEventListener('click', () => showTab(btn.dataset.tab)));

document.getElementById('logout-btn').addEventListener('click', async () => {
  await apiFetch('/auth/logout', { method: 'POST' });
  window.location.href = '/index.html';
});

// ---------- Online/offline UI gating ----------

function updateOnlineUI() {
  const online = navigator.onLine;

  document.querySelectorAll('.online-only-tab').forEach((btn) => {
    btn.classList.toggle('hidden', !online);
  });

  const banner = document.getElementById('offline-banner');
  if (banner) banner.classList.toggle('hidden', online);

  if (!online && ['languages', 'words', 'verbforms'].includes(currentTab)) {
    showTab('testwords');
  }
}

window.addEventListener('online', () => {
  updateOnlineUI();
  flushSyncQueue();
  syncAllData(getSelectedLanguageId());
});
window.addEventListener('offline', updateOnlineUI);

// ---------- Sync queue (pending level-ups made while offline) ----------

async function flushSyncQueue() {
  if (!navigator.onLine) return;

  const queued = await getQueuedUpdates();
  for (const item of queued) {
    try {
      await apiFetch(`/words/${item.wordId}/state`, {
        method: 'PATCH',
        body: JSON.stringify({ state: item.state }),
      });
      await deleteQueuedUpdate(item.id);
    } catch (err) {
      break; // retry the rest on the next 'online' event
    }
  }
}

// ---------- Word state ordering (for the "level up" chance in testing views) ----------

const STATE_ORDER = ['new', 'learning', 'confident', 'mastered'];

const STATE_STYLES = {
  new:       'bg-red-900 text-red-300',
  learning:  'bg-orange-900 text-orange-300',
  confident: 'bg-sky-900 text-sky-300',
  mastered:  'bg-emerald-900 text-emerald-300',
};

function nextState(current) {
  const idx = STATE_ORDER.indexOf(current);
  if (idx === -1 || idx === STATE_ORDER.length - 1) return null;
  return STATE_ORDER[idx + 1];
}

const LEVEL_UP_CHANCE = 15;

async function maybeLevelUp(wordId, currentState) {
  if (Math.random() >= 1 / LEVEL_UP_CHANCE) return false;
  const next = nextState(currentState);
  if (!next) return false;

  if (navigator.onLine) {
    try {
      await apiFetch(`/words/${wordId}/state`, {
        method: 'PATCH',
        body: JSON.stringify({ state: next }),
      });
      return true;
    } catch (err) {
      await queueStateUpdate(wordId, next);
      return true;
    }
  }

  await queueStateUpdate(wordId, next);
  return true;
}

// Flashcard level-up — same 1-in-N random chance style as write-mode, not
// guaranteed on every checkmark tap.
const FLASHCARD_LEVEL_UP_CHANCE = 20;

async function levelUpDirect(wordId, currentState) {
  if (Math.random() >= 1 / FLASHCARD_LEVEL_UP_CHANCE) return false;
  const next = nextState(currentState);
  if (!next) return false; // already mastered

  if (navigator.onLine) {
    try {
      await apiFetch(`/words/${wordId}/state`, {
        method: 'PATCH',
        body: JSON.stringify({ state: next }),
      });
      return true;
    } catch (err) {
      await queueStateUpdate(wordId, next);
      return true;
    }
  }

  await queueStateUpdate(wordId, next);
  return true;
}

// ---------- Languages tab ----------

async function renderLanguages() {
  const el = document.getElementById('tab-languages');
  el.innerHTML = `
    <div>
      <h2 class="text-lg font-medium mb-3">Your languages</h2>
      <ul id="lang-list" class="space-y-1 text-slate-300 mb-6"></ul>

      <form id="lang-form" class="flex gap-2">
        <input id="lang-name" required placeholder="e.g. German"
          class="flex-1 rounded-lg bg-slate-800 border border-slate-700 px-3 py-2" />
        <button class="bg-indigo-600 hover:bg-indigo-500 transition rounded-lg px-4 py-2">Add</button>
      </form>
    </div>
  `;

  const list = document.getElementById('lang-list');
  const langs = await apiFetch('/languages');
  list.innerHTML = (langs || [])
    .map((l) => `<li class="bg-slate-800 rounded-lg px-3 py-2">#${l.ID} — ${escapeHtml(l.Name)}</li>`)
    .join('') || '<li class="text-slate-500">No languages yet.</li>';

  document.getElementById('lang-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const name = document.getElementById('lang-name').value.trim();
    if (!name) return;
    await apiFetch('/languages', { method: 'POST', body: JSON.stringify({ name }) });
    await initLanguageSelector();
    renderLanguages();
  });
}

// ---------- Words tab (insert form + paginated list) ----------

let wordsPage = 1;
const WORDS_PAGE_SIZE = 10;

async function renderWords() {
  const el = document.getElementById('tab-words');
  el.className = 'tab-panel grid grid-cols-1 md:grid-cols-2 gap-8';
  el.innerHTML = `
    <div>
      <h2 class="text-lg font-medium mb-3">Add a word</h2>
      <form id="word-form" class="space-y-3 max-w-sm">
        <div>
          <label class="block text-sm mb-1 text-slate-300">Native word</label>
          <input id="native-word" required class="w-full rounded-lg bg-slate-800 border border-slate-700 px-3 py-2" />
        </div>
        <div>
          <label class="block text-sm mb-1 text-slate-300">Learning word</label>
          <input id="learning-word" required class="w-full rounded-lg bg-slate-800 border border-slate-700 px-3 py-2" />
        </div>
        <div>
          <label class="block text-sm mb-1 text-slate-300">Article (optional)</label>
          <input id="article" placeholder="e.g. das" class="w-full rounded-lg bg-slate-800 border border-slate-700 px-3 py-2" />
        </div>
        <button class="bg-indigo-600 hover:bg-indigo-500 transition rounded-lg px-4 py-2">Add word</button>
      </form>
      <p id="word-result" class="mt-4 text-sm text-slate-400"></p>
    </div>

    <div>
      <h2 class="text-lg font-medium mb-3">Your words</h2>
      <div id="word-list" class="space-y-2"></div>
      <div id="word-pagination" class="flex items-center gap-3 mt-4 text-sm"></div>
    </div>
  `;

  document.getElementById('word-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const nativeWord = document.getElementById('native-word').value.trim();
    const learningWord = document.getElementById('learning-word').value.trim();
    const article = document.getElementById('article').value.trim();
    const languageId = getSelectedLanguageId();

    const body = { nativeWord, learningWord };
    if (article) body.article = article;
    if (languageId) body.languageId = languageId;

    try {
      await apiFetch('/words', { method: 'POST', body: JSON.stringify(body) });
      document.getElementById('word-result').textContent = 'Word added.';
      e.target.reset();
      wordsPage = 1;
      loadWordsList();
      syncAllData(languageId);
    } catch (err) {
      document.getElementById('word-result').textContent = `Error: ${err.message}`;
    }
  });

  wordsPage = 1;
  loadWordsList();
}

async function loadWordsList() {
  const listEl = document.getElementById('word-list');
  const pageEl = document.getElementById('word-pagination');
  const languageId = getSelectedLanguageId();
  if (!languageId) {
    listEl.innerHTML = '<p class="text-slate-500">Select a language first.</p>';
    pageEl.innerHTML = '';
    return;
  }

  const result = await apiFetch(`/words?page=${wordsPage}&pageSize=${WORDS_PAGE_SIZE}&languageId=${languageId}`);
  const items = result?.items || [];

  listEl.innerHTML = items.length
    ? items.map(wordRow).join('')
    : '<p class="text-slate-500">No words yet.</p>';

  listEl.querySelectorAll('[data-delete-word]').forEach((btn) => {
    btn.addEventListener('click', async () => {
      await apiFetch(`/words/${btn.dataset.deleteWord}`, { method: 'DELETE' });
      loadWordsList();
      syncAllData(languageId);
    });
  });

  const totalPages = Math.max(1, Math.ceil((result?.total || 0) / WORDS_PAGE_SIZE));
  pageEl.innerHTML = `
    <button id="word-prev" class="px-2 py-1 rounded bg-slate-800 disabled:opacity-40" ${wordsPage <= 1 ? 'disabled' : ''}>Prev</button>
    <span class="text-slate-400">Page ${wordsPage} of ${totalPages}</span>
    <button id="word-next" class="px-2 py-1 rounded bg-slate-800 disabled:opacity-40" ${wordsPage >= totalPages ? 'disabled' : ''}>Next</button>
  `;
  document.getElementById('word-prev')?.addEventListener('click', () => { wordsPage--; loadWordsList(); });
  document.getElementById('word-next')?.addEventListener('click', () => { wordsPage++; loadWordsList(); });
}

function wordRow(w) {
  const badge = STATE_STYLES[w.State] || 'bg-slate-700 text-slate-300';
  const article = w.Article ? `${escapeHtml(w.Article)} ` : '';
  return `
    <div class="flex items-center justify-between bg-slate-800 rounded-lg px-3 py-2">
      <div class="flex items-center gap-3">
        <span class="px-2 py-0.5 rounded text-xs font-medium ${badge}">${escapeHtml(w.State)}</span>
        <span>${escapeHtml(w.NativeWord)} — ${article}${escapeHtml(w.LearningWord)}</span>
      </div>
      <button data-delete-word="${w.ID}" class="text-red-400 hover:text-red-300 text-sm">Delete</button>
    </div>
  `;
}

// ---------- Verb Forms tab (insert form + paginated list) ----------

let formsPage = 1;
const FORMS_PAGE_SIZE = 10;

async function renderVerbForms() {
  const el = document.getElementById('tab-verbforms');
  el.className = 'tab-panel grid grid-cols-1 md:grid-cols-2 gap-8';
  el.innerHTML = `
    <div>
      <h2 class="text-lg font-medium mb-3">Add a word form</h2>
      <p class="text-sm text-slate-400 mb-3">
        Reference an existing word by ID (find it in the Words tab list on the right).
      </p>
      <form id="form-form" class="space-y-3 max-w-sm">
        <div>
          <label class="block text-sm mb-1 text-slate-300">Word ID</label>
          <input id="word-id" type="number" required class="w-full rounded-lg bg-slate-800 border border-slate-700 px-3 py-2" />
        </div>
        <div>
          <label class="block text-sm mb-1 text-slate-300">Subject</label>
          <input id="subject" required placeholder="e.g. je" class="w-full rounded-lg bg-slate-800 border border-slate-700 px-3 py-2" />
        </div>
        <div>
          <label class="block text-sm mb-1 text-slate-300">Form</label>
          <input id="form-value" required placeholder="e.g. fais" class="w-full rounded-lg bg-slate-800 border border-slate-700 px-3 py-2" />
        </div>
        <div>
          <label class="block text-sm mb-1 text-slate-300">Tense (optional)</label>
          <input id="tense" placeholder="e.g. present" class="w-full rounded-lg bg-slate-800 border border-slate-700 px-3 py-2" />
        </div>
        <button class="bg-indigo-600 hover:bg-indigo-500 transition rounded-lg px-4 py-2">Add form</button>
      </form>
      <p id="form-result" class="mt-4 text-sm text-slate-400"></p>
    </div>

    <div>
      <h2 class="text-lg font-medium mb-3">Your word forms</h2>
      <div id="form-list" class="space-y-2"></div>
      <div id="form-pagination" class="flex items-center gap-3 mt-4 text-sm"></div>
    </div>
  `;

  document.getElementById('form-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const wordId = Number(document.getElementById('word-id').value);
    const subject = document.getElementById('subject').value.trim();
    const form = document.getElementById('form-value').value.trim();
    const tense = document.getElementById('tense').value.trim();
    const languageId = getSelectedLanguageId();

    const body = { wordId, subject, form };
    if (tense) body.tense = tense;

    try {
      await apiFetch('/word-forms', { method: 'POST', body: JSON.stringify(body) });
      document.getElementById('form-result').textContent = 'Word form added.';
      e.target.reset();
      formsPage = 1;
      loadFormsList();
      syncAllData(languageId);
    } catch (err) {
      document.getElementById('form-result').textContent = `Error: ${err.message}`;
    }
  });

  formsPage = 1;
  loadFormsList();
}

async function loadFormsList() {
  const listEl = document.getElementById('form-list');
  const pageEl = document.getElementById('form-pagination');
  const languageId = getSelectedLanguageId();
  if (!languageId) {
    listEl.innerHTML = '<p class="text-slate-500">Select a language first.</p>';
    pageEl.innerHTML = '';
    return;
  }

  const result = await apiFetch(`/word-forms?page=${formsPage}&pageSize=${FORMS_PAGE_SIZE}&languageId=${languageId}`);
  const items = result?.items || [];

  listEl.innerHTML = items.length
    ? items.map(formRow).join('')
    : '<p class="text-slate-500">No word forms yet.</p>';

  listEl.querySelectorAll('[data-delete-form]').forEach((btn) => {
    btn.addEventListener('click', async () => {
      await apiFetch(`/word-forms/${btn.dataset.deleteForm}`, { method: 'DELETE' });
      loadFormsList();
      syncAllData(languageId);
    });
  });

  const totalPages = Math.max(1, Math.ceil((result?.total || 0) / FORMS_PAGE_SIZE));
  pageEl.innerHTML = `
    <button id="form-prev" class="px-2 py-1 rounded bg-slate-800 disabled:opacity-40" ${formsPage <= 1 ? 'disabled' : ''}>Prev</button>
    <span class="text-slate-400">Page ${formsPage} of ${totalPages}</span>
    <button id="form-next" class="px-2 py-1 rounded bg-slate-800 disabled:opacity-40" ${formsPage >= totalPages ? 'disabled' : ''}>Next</button>
  `;
  document.getElementById('form-prev')?.addEventListener('click', () => { formsPage--; loadFormsList(); });
  document.getElementById('form-next')?.addEventListener('click', () => { formsPage++; loadFormsList(); });
}

function formRow(wf) {
  return `
    <div class="flex items-center justify-between bg-slate-800 rounded-lg px-3 py-2">
      <span>${escapeHtml(wf.Subject)} — ${escapeHtml(wf.Form)}</span>
      <button data-delete-form="${wf.ID}" class="text-red-400 hover:text-red-300 text-sm">Delete</button>
    </div>
  `;
}

// ---------- Test Words tab ----------

let wordQueue = [];
let lastWordError = null;

async function renderTestWords() {
  const el = document.getElementById('tab-testwords');
  el.className = 'tab-panel space-y-6';
  el.innerHTML = `
    <div>
      <h2 class="text-lg font-medium mb-3">Test words</h2>

      <label class="block text-sm mb-1 text-slate-300">How many words?</label>
      <input id="word-count" type="number" min="1" value="10"
        class="w-full max-w-xs rounded-lg bg-slate-800 border border-slate-700 px-3 py-2 mb-4" />

      <div class="flex gap-2 mb-6">
        <button id="start-write-btn" type="button"
          class="bg-indigo-600 hover:bg-indigo-500 transition rounded-lg px-4 py-2">Write</button>
        <button id="start-flashcards-btn" type="button"
          class="bg-indigo-600 hover:bg-indigo-500 transition rounded-lg px-4 py-2">Flashcards</button>
      </div>

      <div id="word-quiz"></div>
    </div>
  `;

  document.getElementById('start-write-btn').addEventListener('click', () => startWordsTest('write'));
  document.getElementById('start-flashcards-btn').addEventListener('click', () => startWordsTest('flashcards'));
}

async function startWordsTest(mode) {
  const count = Number(document.getElementById('word-count').value) || 10;
  const languageId = getSelectedLanguageId();
  if (!languageId) return;

  const quizEl = document.getElementById('word-quiz');
  quizEl.innerHTML = '<p class="text-slate-400">Loading…</p>';
  lastWordError = null;

  if (navigator.onLine) {
    try {
      await withTimeout(syncAllData(languageId), 5000);
    } catch (err) {
      // best-effort refresh; fall through to whatever's already cached
    }
  }

  const allWords = (await loadFullWordSet(languageId)) || [];
  if (allWords.length === 0) {
    quizEl.innerHTML = '<p class="text-slate-400">No words available yet. Connect to the internet at least once to download your words for offline testing.</p>';
    return;
  }

  if (mode === 'flashcards') {
    wordFlashQueue = weightedSample(allWords, count);
    if (wordFlashQueue.length === 0) {
      quizEl.innerHTML = '<p class="text-slate-400">All your words are already mastered — nothing to test.</p>';
      return;
    }
    wordFlashReversed = false;
    wordFlashLapRemaining = wordFlashQueue.length;
    renderWordFlashcard();
    return;
  }

  wordQueue = weightedSample(allWords, count);
  if (wordQueue.length === 0) {
    quizEl.innerHTML = '<p class="text-slate-400">All your words are already mastered — nothing to test.</p>';
    return;
  }

  showWordQuizItem();
}

function showWordQuizItem() {
  const quizEl = document.getElementById('word-quiz');
  if (wordQueue.length === 0) {
    quizEl.innerHTML = `<p class="text-slate-400">Done — no more words in this session.</p>`;
    return;
  }

  const w = wordQueue[0];
  const articleField = w.Article
    ? `
      <label class="block text-sm mb-1 text-slate-300">Article</label>
      <input id="answer-article" class="w-full rounded-lg bg-slate-700 border border-slate-600 px-3 py-2 mb-3" />
    `
    : '';

  const errorBanner = lastWordError
    ? `<div class="mt-4 text-sm text-red-300 bg-red-950/50 border border-red-900 rounded-lg px-3 py-2">
         <span class="font-semibold">Last mistake:</span> ${escapeHtml(lastWordError)}
       </div>`
    : '';

  quizEl.innerHTML = `
    <div class="bg-slate-800 rounded-xl p-6 max-w-sm">
      <p class="text-sm text-slate-400 mb-1">Translate</p>
      <p class="text-2xl font-semibold mb-4">${escapeHtml(w.NativeWord)}</p>

      ${articleField}

      <label class="block text-sm mb-1 text-slate-300">Translation</label>
      <input id="answer-word" class="w-full rounded-lg bg-slate-700 border border-slate-600 px-3 py-2 mb-4" />

      <button id="check-word-btn" class="bg-indigo-600 hover:bg-indigo-500 transition rounded-lg px-4 py-2">Check</button>
      <p id="word-feedback" class="mt-3 text-sm"></p>
      <p class="mt-2 text-xs text-slate-500">${wordQueue.length} word${wordQueue.length === 1 ? '' : 's'} left</p>
    </div>
    ${errorBanner}
  `;

  document.getElementById('check-word-btn').addEventListener('click', async () => {
    const answer = document.getElementById('answer-word').value.trim();
    const articleAnswer = w.Article ? document.getElementById('answer-article').value.trim() : '';
    const feedback = document.getElementById('word-feedback');

    const correctWord = normalize(answer) === normalize(w.LearningWord);
    const correctArticle = !w.Article || normalize(articleAnswer) === normalize(w.Article);

    if (correctWord && correctArticle) {
      const leveledUp = await maybeLevelUp(w.ID, w.State);
      feedback.className = 'mt-3 text-sm text-emerald-400';
      feedback.textContent = leveledUp ? 'Correct! Knowledge level increased.' : 'Correct!';
      wordQueue.shift();
    } else {
      feedback.className = 'mt-3 text-sm text-red-400';
      feedback.textContent = 'Not quite — see below.';
      lastWordError = `"${w.NativeWord}" — correct answer: ${w.Article ? w.Article + ' ' : ''}${w.LearningWord}`;
      wordQueue.push(wordQueue.shift());
    }

    setTimeout(showWordQuizItem, 1500);
  });
}

// ---------- Test Words: flashcards mode ----------

let wordFlashQueue = [];
let wordFlashLapRemaining = 0;
let wordFlashReversed = false;

function advanceWordFlashcard() {
  wordFlashLapRemaining--;
  if (wordFlashLapRemaining <= 0) {
    wordFlashReversed = !wordFlashReversed;
    wordFlashLapRemaining = wordFlashQueue.length;
  }
  renderWordFlashcard();
}

function renderWordFlashcard() {
  const quizEl = document.getElementById('word-quiz');
  if (wordFlashQueue.length === 0) {
    quizEl.innerHTML = `<p class="text-slate-400">Done — no more cards in this deck.</p>`;
    return;
  }

  const w = wordFlashQueue[0];

  const reversed = wordFlashReversed;
  const topText = reversed ? `${w.Article ? w.Article + ' ' : ''}${w.LearningWord}` : w.NativeWord;
  const hiddenText = reversed ? w.NativeWord : `${w.Article ? w.Article + ' ' : ''}${w.LearningWord}`;
  const markIcon = reversed ? '✕' : '✓';

  quizEl.innerHTML = `
    <div class="max-w-sm">
      <div class="bg-slate-800 rounded-t-xl px-6 py-10 text-center">
        <p class="text-2xl font-semibold">${escapeHtml(topText)}</p>
      </div>
      <div id="flash-reveal" class="relative w-full bg-slate-900 hover:bg-slate-950 transition rounded-b-xl px-6 py-10 text-center border-t border-slate-700 cursor-pointer select-none">
        <span id="flash-hidden-text" class="text-xl text-slate-500">Tap to reveal</span>
        <button id="flash-mark-btn" type="button"
          class="absolute bottom-3 right-3 w-9 h-9 flex items-center justify-center rounded-full bg-slate-700 hover:bg-emerald-700 text-lg transition">
          ${markIcon}
        </button>
      </div>
      <div class="flex justify-between items-center mt-3">
        <span class="text-xs text-slate-500">${wordFlashQueue.length} card${wordFlashQueue.length === 1 ? '' : 's'} left</span>
        <button id="flash-next-btn" type="button" class="text-slate-300 hover:text-white text-xl leading-none">→</button>
      </div>
    </div>
  `;

  let revealed = false;
  document.getElementById('flash-reveal').addEventListener('click', () => {
    revealed = !revealed;
    document.getElementById('flash-hidden-text').textContent = revealed ? hiddenText : 'Tap to reveal';
  });

  document.getElementById('flash-mark-btn').addEventListener('click', async (e) => {
    e.stopPropagation();
    await levelUpDirect(w.ID, w.State);
    wordFlashQueue.shift();
    advanceWordFlashcard();
  });

  document.getElementById('flash-next-btn').addEventListener('click', (e) => {
    e.stopPropagation();
    wordFlashQueue.push(wordFlashQueue.shift());
    advanceWordFlashcard();
  });
}

// ---------- Test Verbs tab ----------

let formQueue = [];
let lastVerbError = null;

async function renderTestVerbs() {
  const el = document.getElementById('tab-testverbs');
  el.className = 'tab-panel space-y-6';
  el.innerHTML = `
    <div>
      <h2 class="text-lg font-medium mb-3">Test verb forms</h2>

      <label class="block text-sm mb-1 text-slate-300">How many forms?</label>
      <input id="verb-count" type="number" min="1" value="10"
        class="w-full max-w-xs rounded-lg bg-slate-800 border border-slate-700 px-3 py-2 mb-4" />

      <div class="flex gap-2 mb-6">
        <button id="start-verb-write-btn" type="button"
          class="bg-indigo-600 hover:bg-indigo-500 transition rounded-lg px-4 py-2">Write</button>
        <button id="start-verb-flashcards-btn" type="button"
          class="bg-indigo-600 hover:bg-indigo-500 transition rounded-lg px-4 py-2">Flashcards</button>
      </div>

      <div id="verb-quiz"></div>
    </div>
  `;

  document.getElementById('start-verb-write-btn').addEventListener('click', () => startVerbsTest('write'));
  document.getElementById('start-verb-flashcards-btn').addEventListener('click', () => startVerbsTest('flashcards'));
}

async function startVerbsTest(mode) {
  const count = Number(document.getElementById('verb-count').value) || 10;
  const languageId = getSelectedLanguageId();
  if (!languageId) return;

  const quizEl = document.getElementById('verb-quiz');
  quizEl.innerHTML = '<p class="text-slate-400">Loading…</p>';
  lastVerbError = null;

  if (navigator.onLine) {
    try {
      await withTimeout(syncAllData(languageId), 5000);
    } catch (err) {
      // best-effort refresh; fall through to whatever's already cached
    }
  }

  const allForms = (await loadFullWordFormSet(languageId)) || [];
  if (allForms.length === 0) {
    quizEl.innerHTML = '<p class="text-slate-400">No verb forms available yet. Connect to the internet at least once to download them for offline testing.</p>';
    return;
  }

  if (mode === 'flashcards') {
    formFlashQueue = weightedSample(allForms, count);
    if (formFlashQueue.length === 0) {
      quizEl.innerHTML = '<p class="text-slate-400">All related words are already mastered — nothing to test.</p>';
      return;
    }
    formFlashReversed = false;
    formFlashLapRemaining = formFlashQueue.length;
    renderVerbFlashcard();
    return;
  }

  formQueue = weightedSample(allForms, count);
  if (formQueue.length === 0) {
    quizEl.innerHTML = '<p class="text-slate-400">All related words are already mastered — nothing to test.</p>';
    return;
  }

  showVerbQuizItem();
}

function showVerbQuizItem() {
  const quizEl = document.getElementById('verb-quiz');
  if (formQueue.length === 0) {
    quizEl.innerHTML = `<p class="text-slate-400">Done — no more verb forms in this session.</p>`;
    return;
  }

  const wf = formQueue[0];
  const tenseLine = wf.Tense
    ? `<p class="text-xs text-slate-500 mb-1">${escapeHtml(wf.Tense)}</p>`
    : '';

  const errorBanner = lastVerbError
    ? `<div class="mt-4 text-sm text-red-300 bg-red-950/50 border border-red-900 rounded-lg px-3 py-2">
         <span class="font-semibold">Last mistake:</span> ${escapeHtml(lastVerbError)}
       </div>`
    : '';

  quizEl.innerHTML = `
    <div class="bg-slate-800 rounded-xl p-6 max-w-sm">
      <p class="text-sm text-slate-400 mb-1">${escapeHtml(wf.NativeWord)}</p>
      ${tenseLine}
      <p class="text-2xl font-semibold mb-4">${escapeHtml(wf.Subject)}</p>

      <label class="block text-sm mb-1 text-slate-300">Conjugated form</label>
      <input id="answer-form" class="w-full rounded-lg bg-slate-700 border border-slate-600 px-3 py-2 mb-4" />

      <button id="check-verb-btn" class="bg-indigo-600 hover:bg-indigo-500 transition rounded-lg px-4 py-2">Check</button>
      <p id="verb-feedback" class="mt-3 text-sm"></p>
      <p class="mt-2 text-xs text-slate-500">${formQueue.length} form${formQueue.length === 1 ? '' : 's'} left</p>
    </div>
    ${errorBanner}
  `;

  document.getElementById('check-verb-btn').addEventListener('click', async () => {
    const answer = document.getElementById('answer-form').value.trim();
    const feedback = document.getElementById('verb-feedback');

    const correct = normalize(answer) === normalize(wf.Form);

    if (correct) {
      feedback.className = 'mt-3 text-sm text-emerald-400';
      feedback.textContent = 'Correct!';
      formQueue.shift();
    } else {
      feedback.className = 'mt-3 text-sm text-red-400';
      feedback.textContent = 'Not quite — see below.';
      lastVerbError = `"${wf.Subject} ${wf.NativeWord}" — correct answer: ${wf.Form}`;
      formQueue.push(formQueue.shift());
    }

    setTimeout(showVerbQuizItem, 1500);
  });
}

// ---------- Test Verbs: flashcards mode ----------

let formFlashQueue = [];
let formFlashLapRemaining = 0;
let formFlashReversed = false;

function advanceVerbFlashcard() {
  formFlashLapRemaining--;
  if (formFlashLapRemaining <= 0) {
    formFlashReversed = !formFlashReversed;
    formFlashLapRemaining = formFlashQueue.length;
  }
  renderVerbFlashcard();
}

function renderVerbFlashcard() {
  const quizEl = document.getElementById('verb-quiz');
  if (formFlashQueue.length === 0) {
    quizEl.innerHTML = `<p class="text-slate-400">Done — no more cards in this deck.</p>`;
    return;
  }

  const wf = formFlashQueue[0];

  const reversed = formFlashReversed;
  const tenseLine = wf.Tense
    ? `<p class="text-xs text-slate-500 mb-1">${escapeHtml(wf.Tense)}</p>`
    : '';

  const topText = reversed
    ? escapeHtml(wf.Form)
    : `${escapeHtml(wf.NativeWord)} (${escapeHtml(wf.Subject)})`;
  const hiddenText = reversed
    ? `${escapeHtml(wf.NativeWord)} (${escapeHtml(wf.Subject)})`
    : escapeHtml(wf.Form);
  const markIcon = reversed ? '✕' : '✓';

  quizEl.innerHTML = `
    <div class="max-w-sm">
      <div class="bg-slate-800 rounded-t-xl px-6 py-10 text-center">
        ${tenseLine}
        <p class="text-2xl font-semibold">${topText}</p>
      </div>
      <div id="flash-reveal" class="relative w-full bg-slate-900 hover:bg-slate-950 transition rounded-b-xl px-6 py-10 text-center border-t border-slate-700 cursor-pointer select-none">
        <span id="flash-hidden-text" class="text-xl text-slate-500">Tap to reveal</span>
        <button id="flash-mark-btn" type="button"
          class="absolute bottom-3 right-3 w-9 h-9 flex items-center justify-center rounded-full bg-slate-700 hover:bg-emerald-700 text-lg transition">
          ${markIcon}
        </button>
      </div>
      <div class="flex justify-between items-center mt-3">
        <span class="text-xs text-slate-500">${formFlashQueue.length} card${formFlashQueue.length === 1 ? '' : 's'} left</span>
        <button id="flash-next-btn" type="button" class="text-slate-300 hover:text-white text-xl leading-none">→</button>
      </div>
    </div>
  `;

  let revealed = false;
  document.getElementById('flash-reveal').addEventListener('click', () => {
    revealed = !revealed;
    document.getElementById('flash-hidden-text').textContent = revealed ? hiddenText : 'Tap to reveal';
  });

  document.getElementById('flash-mark-btn').addEventListener('click', async (e) => {
    e.stopPropagation();
    await levelUpDirect(wf.WordID, wf.State);
    formFlashQueue.shift();
    advanceVerbFlashcard();
  });

  document.getElementById('flash-next-btn').addEventListener('click', (e) => {
    e.stopPropagation();
    formFlashQueue.push(formFlashQueue.shift());
    advanceVerbFlashcard();
  });
}

// ---------- helpers ----------

function normalize(s) {
  return (s || '').trim().toLowerCase();
}

function escapeHtml(s) {
  const div = document.createElement('div');
  div.textContent = s;
  return div.innerHTML;
}

// ---------- init ----------

const renderers = {
  languages: renderLanguages,
  words: renderWords,
  verbforms: renderVerbForms,
  testwords: renderTestWords,
  testverbs: renderTestVerbs,
};

(async () => {
  await initLanguageSelector();
  updateOnlineUI();
  flushSyncQueue();
  syncAllData(getSelectedLanguageId()); // refresh local dataset on every PWA load
  showTab(navigator.onLine ? 'languages' : 'testwords');
})();
