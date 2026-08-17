// Thin IndexedDB wrapper. Two object stores:
//   testBatches - last-fetched quiz data, keyed by `${type}:${languageId}`
//                 (type is 'words' or 'verbs'), so testing works offline.
//   syncQueue   - pending word-state updates made while offline, flushed
//                 once connectivity returns.

const DB_NAME = 'rabbitvocab';
const DB_VERSION = 1;

function openDB() {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, DB_VERSION);

    req.onupgradeneeded = () => {
      const db = req.result;
      if (!db.objectStoreNames.contains('testBatches')) {
        db.createObjectStore('testBatches', { keyPath: 'key' });
      }
      if (!db.objectStoreNames.contains('syncQueue')) {
        db.createObjectStore('syncQueue', { keyPath: 'id', autoIncrement: true });
      }
    };

    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error);
  });
}

async function withStore(storeName, mode, fn) {
  const db = await openDB();
  return new Promise((resolve, reject) => {
    const tx = db.transaction(storeName, mode);
    const store = tx.objectStore(storeName);
    const result = fn(store);

    tx.oncomplete = () => resolve(result);
    tx.onerror = () => reject(tx.error);
  });
}

// ---------- Test batch cache ----------

async function saveTestBatch(type, languageId, items) {
  const key = `${type}:${languageId}`;
  await withStore('testBatches', 'readwrite', (store) => {
    store.put({ key, items, savedAt: Date.now() });
  });
}

async function loadTestBatch(type, languageId) {
  const key = `${type}:${languageId}`;
  const db = await openDB();
  return new Promise((resolve, reject) => {
    const tx = db.transaction('testBatches', 'readonly');
    const req = tx.objectStore('testBatches').get(key);
    req.onsuccess = () => resolve(req.result ? req.result.items : null);
    req.onerror = () => reject(req.error);
  });
}

// ---------- Full offline mirror (all words/forms, for client-side random selection) ----------

async function saveFullMirror(type, languageId, items) {
  const key = `${type}-full:${languageId}`;
  await withStore('testBatches', 'readwrite', (store) => {
    store.put({ key, items, savedAt: Date.now() });
  });
}

async function loadFullMirror(type, languageId) {
  const key = `${type}-full:${languageId}`;
  const db = await openDB();
  return new Promise((resolve, reject) => {
    const tx = db.transaction('testBatches', 'readonly');
    const req = tx.objectStore('testBatches').get(key);
    req.onsuccess = () => resolve(req.result ? req.result.items : null);
    req.onerror = () => reject(req.error);
  });
}

// ---------- Full local dataset (synced on every load, not just last batch) ----------
// Stored under the same `testBatches` store, keyed distinctly from the old
// per-request batch cache, so the whole known word/word-form set for a
// language is available for client-side random selection at any time,
// online or offline.

async function saveFullWordSet(languageId, words) {
  await withStore('testBatches', 'readwrite', (store) => {
    store.put({ key: `words-full:${languageId}`, items: words, savedAt: Date.now() });
  });
}

async function loadFullWordSet(languageId) {
  return loadTestBatch('words-full', languageId);
}

async function saveFullWordFormSet(languageId, forms) {
  await withStore('testBatches', 'readwrite', (store) => {
    store.put({ key: `verbs-full:${languageId}`, items: forms, savedAt: Date.now() });
  });
}

async function loadFullWordFormSet(languageId) {
  return loadTestBatch('verbs-full', languageId);
}

// ---------- Sync queue (pending word-state updates) ----------

async function queueStateUpdate(wordId, state) {
  await withStore('syncQueue', 'readwrite', (store) => {
    store.add({ wordId, state, queuedAt: Date.now() });
  });
}

async function getQueuedUpdates() {
  const db = await openDB();
  return new Promise((resolve, reject) => {
    const tx = db.transaction('syncQueue', 'readonly');
    const req = tx.objectStore('syncQueue').getAll();
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error);
  });
}

async function deleteQueuedUpdate(id) {
  await withStore('syncQueue', 'readwrite', (store) => {
    store.delete(id);
  });
}
