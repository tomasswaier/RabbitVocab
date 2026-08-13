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
  const langs = (await apiFetch('/languages')) || [];

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
    renderers[currentTab]?.();
  });
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

// Chance (1 in N) that a correct answer bumps the word's knowledge level.
const LEVEL_UP_CHANCE = 15;

async function maybeLevelUp(wordId, currentState) {
  if (Math.random() >= 1 / LEVEL_UP_CHANCE) return false;
  const next = nextState(currentState);
  if (!next) return false;
  await apiFetch(`/words/${wordId}/state`, {
    method: 'PATCH',
    body: JSON.stringify({ state: next }),
  });
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

    const body = { wordId, subject, form };
    if (tense) body.tense = tense;

    try {
      await apiFetch('/word-forms', { method: 'POST', body: JSON.stringify(body) });
      document.getElementById('form-result').textContent = 'Word form added.';
      e.target.reset();
      formsPage = 1;
      loadFormsList();
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
let wordQueueIdx = 0;

async function renderTestWords() {
  const el = document.getElementById('tab-testwords');
  el.className = 'tab-panel space-y-6';
  el.innerHTML = `
    <div>
      <h2 class="text-lg font-medium mb-3">Test words</h2>
      <form id="start-words-form" class="flex gap-2 mb-6">
        <input id="word-count" type="number" min="1" value="10"
          class="w-24 rounded-lg bg-slate-800 border border-slate-700 px-3 py-2" />
        <button class="bg-indigo-600 hover:bg-indigo-500 transition rounded-lg px-4 py-2">Start</button>
      </form>
      <div id="word-quiz"></div>
    </div>
  `;

  document.getElementById('start-words-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const count = Number(document.getElementById('word-count').value) || 10;
    const languageId = getSelectedLanguageId();
    if (!languageId) return;
    wordQueue = (await apiFetch(`/words/random?count=${count}&languageId=${languageId}`)) || [];
    wordQueueIdx = 0;
    showWordQuizItem();
  });
}

function showWordQuizItem() {
  const quizEl = document.getElementById('word-quiz');
  if (wordQueueIdx >= wordQueue.length) {
    quizEl.innerHTML = `<p class="text-slate-400">Done — no more words in this session.</p>`;
    return;
  }

  const w = wordQueue[wordQueueIdx];
  quizEl.innerHTML = `
    <div class="bg-slate-800 rounded-xl p-6 max-w-sm">
      <p class="text-sm text-slate-400 mb-1">Translate</p>
      <p class="text-2xl font-semibold mb-4">${escapeHtml(w.NativeWord)}</p>

      <label class="block text-sm mb-1 text-slate-300">Translation</label>
      <input id="answer-word" class="w-full rounded-lg bg-slate-700 border border-slate-600 px-3 py-2 mb-3" />

      <label class="block text-sm mb-1 text-slate-300">Article (if any)</label>
      <input id="answer-article" class="w-full rounded-lg bg-slate-700 border border-slate-600 px-3 py-2 mb-4" />

      <button id="check-word-btn" class="bg-indigo-600 hover:bg-indigo-500 transition rounded-lg px-4 py-2">Check</button>
      <p id="word-feedback" class="mt-3 text-sm"></p>
    </div>
  `;

  document.getElementById('check-word-btn').addEventListener('click', async () => {
    const answer = document.getElementById('answer-word').value.trim();
    const articleAnswer = document.getElementById('answer-article').value.trim();
    const feedback = document.getElementById('word-feedback');

    const correctWord = normalize(answer) === normalize(w.LearningWord);
    const correctArticle = !w.Article || normalize(articleAnswer) === normalize(w.Article);

    if (correctWord && correctArticle) {
      const leveledUp = await maybeLevelUp(w.ID, w.State);
      feedback.className = 'mt-3 text-sm text-emerald-400';
      feedback.textContent = leveledUp ? 'Correct! Knowledge level increased.' : 'Correct!';
    } else {
      feedback.className = 'mt-3 text-sm text-red-400';
      feedback.textContent = `Not quite. Correct answer: ${w.Article ? w.Article + ' ' : ''}${w.LearningWord}`;
    }

    setTimeout(() => {
      wordQueueIdx++;
      showWordQuizItem();
    }, 1500);
  });
}

// ---------- Test Verbs tab ----------

let formQueue = [];
let formQueueIdx = 0;

async function renderTestVerbs() {
  const el = document.getElementById('tab-testverbs');
  el.className = 'tab-panel space-y-6';
  el.innerHTML = `
    <div>
      <h2 class="text-lg font-medium mb-3">Test verb forms</h2>
      <form id="start-verbs-form" class="flex gap-2 mb-6">
        <input id="verb-count" type="number" min="1" value="10"
          class="w-24 rounded-lg bg-slate-800 border border-slate-700 px-3 py-2" />
        <button class="bg-indigo-600 hover:bg-indigo-500 transition rounded-lg px-4 py-2">Start</button>
      </form>
      <div id="verb-quiz"></div>
    </div>
  `;

  document.getElementById('start-verbs-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const count = Number(document.getElementById('verb-count').value) || 10;
    const languageId = getSelectedLanguageId();
    if (!languageId) return;
    formQueue = (await apiFetch(`/word-forms/random?count=${count}&languageId=${languageId}`)) || [];
    formQueueIdx = 0;
    showVerbQuizItem();
  });
}

function showVerbQuizItem() {
  const quizEl = document.getElementById('verb-quiz');
  if (formQueueIdx >= formQueue.length) {
    quizEl.innerHTML = `<p class="text-slate-400">Done — no more verb forms in this session.</p>`;
    return;
  }

  const wf = formQueue[formQueueIdx];
  quizEl.innerHTML = `
    <div class="bg-slate-800 rounded-xl p-6 max-w-sm">
      <p class="text-sm text-slate-400 mb-1">${escapeHtml(wf.NativeWord)}</p>
      <p class="text-2xl font-semibold mb-4">${escapeHtml(wf.Subject)}</p>

      <label class="block text-sm mb-1 text-slate-300">Conjugated form</label>
      <input id="answer-form" class="w-full rounded-lg bg-slate-700 border border-slate-600 px-3 py-2 mb-4" />

      <button id="check-verb-btn" class="bg-indigo-600 hover:bg-indigo-500 transition rounded-lg px-4 py-2">Check</button>
      <p id="verb-feedback" class="mt-3 text-sm"></p>
    </div>
  `;

  document.getElementById('check-verb-btn').addEventListener('click', async () => {
    const answer = document.getElementById('answer-form').value.trim();
    const feedback = document.getElementById('verb-feedback');

    const correct = normalize(answer) === normalize(wf.Form);

    if (correct) {
      feedback.className = 'mt-3 text-sm text-emerald-400';
      feedback.textContent = 'Correct!';
    } else {
      feedback.className = 'mt-3 text-sm text-red-400';
      feedback.textContent = `Not quite. Correct answer: ${wf.Form}`;
    }

    setTimeout(() => {
      formQueueIdx++;
      showVerbQuizItem();
    }, 1500);
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
  showTab('languages');
})();
