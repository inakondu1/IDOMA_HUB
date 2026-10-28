const CACHE_NAME = "idoma-hub-static-v1";
const STATIC_ASSETS = [
  "/static/manifest.json",
  "/static/images/idoma-app-192.png",
  "/static/images/idoma-app-512.png"
];

self.addEventListener("install", (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then((cache) => cache.addAll(STATIC_ASSETS))
      .then(() => self.skipWaiting())
  );
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches.keys()
      .then((keys) => Promise.all(
        keys.filter((key) =>
          key.startsWith("idoma-hub-static-") && key !== CACHE_NAME
        ).map((key) => caches.delete(key))
      ))
      .then(() => self.clients.claim())
  );
});

self.addEventListener("fetch", (event) => {
  const request = event.request;
  const url = new URL(request.url);

  // Never cache non-GET requests, other origins, or application pages.
  if (
    request.method !== "GET" ||
    url.origin !== self.location.origin ||
    !url.pathname.startsWith("/static/")
  ) {
    return;
  }

  // Only use cache-first for the app's own icon and manifest files.
  if (!STATIC_ASSETS.includes(url.pathname)) {
    return;
  }

  event.respondWith(
    caches.match(request).then((cached) => {
      if (cached) return cached;
      return fetch(request);
    })
  );
});
