const CACHE_NAME = "idoma-hub-static-v1";
const CACHE_PREFIX = "idoma-hub-static-";

self.addEventListener("install", event => {
    self.skipWaiting();
});

self.addEventListener("activate", event => {
    event.waitUntil(
        caches.keys().then(keys =>
            Promise.all(
                keys
                    .filter(key => key.startsWith(CACHE_PREFIX) && key !== CACHE_NAME)
                    .map(key => caches.delete(key))
            )
        ).then(() => self.clients.claim())
    );
});

self.addEventListener("fetch", event => {
    const request = event.request;
    const url = new URL(request.url);
    const path = url.pathname.toLowerCase();

    const isStaticAsset =
        path.startsWith("/static/") &&
        !path.startsWith("/static/uploads/") &&
        /\.(css|js|png|jpg|jpeg|webp|svg|woff|woff2)$/.test(path);

    if (
        request.method !== "GET" ||
        url.origin !== self.location.origin ||
        !isStaticAsset
    ) {
        return;
    }

    event.respondWith(
        caches.match(request).then(cached => {
            if (cached) return cached;

            return fetch(request).then(response => {
                if (response.ok) {
                    const copy = response.clone();

                    caches.open(CACHE_NAME).then(cache =>
                        cache.put(request, copy)
                    );
                }

                return response;
            });
        })
    );
});
