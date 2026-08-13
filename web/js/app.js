// ---------- Tab switching ----------

const tabButtons = document.querySelectorAll('.tab-btn');
const tabPanels = document.querySelectorAll('.tab-panel');

function showTab(name) {
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
    .map((l) => `<li class="bg-slate-800 rounded-lg px-3 py-2">#${l.ID} — ${l.Name}</li>`)
    .join('') || '<li class="text-slate-500">No languages yet.</li>';

  document.getElementById('lang-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const name = document.getElementById('lang-name').value.trim();
    if (!name) return;
    await apiFetch('/languages', { method: 'POST', body: JSON.stringify({ name }) });
    renderLanguages();
  });
}

// ---------- Words tab (management: insert only) ----------

async function renderWords() {
  const el = document.getElementById('tab-words');
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
        <div>
          <label class="block text-sm mb-1 text-slate-300">Language ID (optional if you have only one)</label>
          <input id="language-id" type="number" class="w-full rounded-lg bg-slate-800 border border-slate-700 px-3 py-2" />
        </div>
        <button class="bg-indigo-600 hover:bg-indigo-500 transition rounded-lg px-4 py-2">Add word</button>
      </form>
      <p id="word-result" class="mt-4 text-sm text-slate-400"></p>
    </div>
  `;

  document.getElementById('word-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const nativeWord = document.getElementById('native-word').value.trim();
    const learningWord = document.getElementById('learning-word').value.trim();
    const article = document.getElementById('article').value.trim();
    const languageIdRaw = document.getElementById('language-id').value.trim();

    const body = { nativeWord, learningWord };
    if (article) body.article = article;
    if (languageIdRaw) body.languageId = Number(languageIdRaw);

    try {
      const word = await apiFetch('/words', { method: 'POST', body: JSON.stringify(body) });
      document.getElementById('word-result').textContent =
        `Created word #${word.ID} — use this ID in the Verb Forms tab to add conjugations.`;
      e.target.reset();
    } catch (err) {
      document.getElementById('word-result').textContent = `Error: ${err.message}`;
    }
  });
}

// ---------- Verb Forms tab (management: insert only) ----------

async function renderVerbForms() {
  const el = document.getElementById('tab-verbforms');
  el.innerHTML = `
    <div>
      <h2 class="text-lg font-medium mb-3">Add a word form</h2>
      <p class="text-sm text-slate-400 mb-3">
        Reference an existing word by ID (shown after creating it in the Words tab).
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
        <button class="bg-indigo-600 hover:bg-indigo-500 transition rounded-lg px-4 py-2">Add form</button>
      </form>
      <p id="form-result" class="mt-4 text-sm text-slate-400"></p>
    </div>
  `;

  document.getElementById('form-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const wordId = Number(document.getElementById('word-id').value);
    const subject = document.getElementById('subject').value.trim();
    const form = document.getElementById('form-value').value.trim();

    try {
      const wf = await apiFetch('/word-forms', {
        method: 'POST',
        body: JSON.stringify({ wordId, subject, form }),
      });
      document.getElementById('form-result').textContent = `Created word form #${wf.ID}.`;
      e.target.reset();
    } catch (err) {
      document.getElementById('form-result').textContent = `Error: ${err.message}`;
    }
  });
}

// ---------- Test Words tab ----------

let wordQueue = [];
let wordQueueIdx = 0;

async function renderTestWords() {
  const el = document.getElementById('tab-testwords');
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
    wordQueue = (await apiFetch(`/words/random?count=${count}`)) || [];
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
    formQueue = (await apiFetch(`/word-forms/random?count=${count}`)) || [];
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
      // Word forms don't carry the parent word's current state, so we can't
      // safely resolve a "next" state client-side here. If you want the
      // level-up mechanic on verb testing too, GET /word-forms/random needs
      // to include the parent word's State field alongside NativeWord/LearningWord.
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

showTab('languages');
