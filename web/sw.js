const CACHE_NAME = 'rabbitvocab-shell-v4';

const SHELL_FILES = [
  '/index.html',
  '/register.html',
  '/app.html',
  '/manifest.json',
  '/icons/icon.svg',
  '/js/api.js',
  '/js/app.js',
  '/js/login.js',
  '/js/register.js',
  '/js/db.js',
];

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME).then((cache) => cache.addAll(SHELL_FILES))
  );
  self.skipWaiting();
});

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((keys) =>
      Promise.all(keys.filter((k) => k !== CACHE_NAME).map((k) => caches.delete(k)))
    )
  );
  self.clients.claim();
});

// Only the app shell (HTML/CSS/JS/icons/manifest) is cached here.
// API requests (/languages, /words, etc.) always go straight to the network
// so data stays fresh — offline testing is handled separately via IndexedDB
// in app.js, not by caching API responses in the service worker.
self.addEventListener('fetch', (event) => {
  const url = new URL(event.request.url);

  if (event.request.method !== 'GET' || url.origin !== self.location.origin) {
    return;
  }

  const isShellFile =
    SHELL_FILES.includes(url.pathname) ||
    url.pathname === '/' ||
    url.pathname.startsWith('/js/') ||
    url.pathname.startsWith('/icons/');

  if (!isShellFile) return;

  // Network-first: always try to get the latest shell file when online
  // (and refresh the cache), only falling back to the cached copy when
  // there's genuinely no connection. This avoids serving stale JS/HTML
  // after deploys, at the small cost of one network round-trip per file
  // when online.
  event.respondWith(
    fetch(event.request)
      .then((response) => {
        const clone = response.clone();
        caches.open(CACHE_NAME).then((cache) => cache.put(event.request, clone));
        return response;
      })
      .catch(() => caches.match(event.request))
  );
});
