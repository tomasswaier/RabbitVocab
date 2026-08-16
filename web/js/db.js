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
