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


// Display private, generic push notifications from IDOMA HUB.
self.addEventListener("push", (event) => {
  event.waitUntil((async () => {
    let data = {};
    try {
      data = event.data ? event.data.json() : {};
    } catch (_) {}

    const title = typeof data.title === "string" ? data.title : "IDOMA HUB";
    const body = typeof data.body === "string"
      ? data.body
      : "You have a new notification.";

    // Only allow a local path; never navigate to an external site.
    let target = "/dashboard";
    if (typeof data.url === "string" && data.url.startsWith("/") &&
        !data.url.startsWith("//")) {
      try {
        const parsed = new URL(data.url, self.location.origin);
        if (parsed.origin === self.location.origin) {
          target = parsed.pathname + parsed.search + parsed.hash;
        }
      } catch (_) {}
    }

    await self.registration.showNotification(title, {
      body,
      icon: "/static/images/idoma-app-192.png",
      badge: "/static/images/idoma-app-192.png",
      data: { url: target }
    });
  })());
});

// Close only push notifications belonging to the conversation currently open.
self.addEventListener("message", (event) => {
  const message = event.data;
  if (!message || message.type !== "CLOSE_CONVERSATION_NOTIFICATIONS") {
    return;
  }

  const supplied = message.url;
  if (typeof supplied !== "string" || !supplied.startsWith("/") ||
      supplied.startsWith("//")) {
    return;
  }

  let target;
  try {
    const parsed = new URL(supplied, self.location.origin);
    if (parsed.origin !== self.location.origin) return;
    target = parsed.pathname + parsed.search;
  } catch (_) {
    return;
  }

  event.waitUntil((async () => {
    const notifications = await self.registration.getNotifications();
    for (const notification of notifications) {
      const notificationURL = notification.data && notification.data.url;
      if (typeof notificationURL !== "string") continue;

      try {
        const parsed = new URL(notificationURL, self.location.origin);
        if (parsed.origin === self.location.origin &&
            parsed.pathname + parsed.search === target) {
          notification.close();
        }
      } catch (_) {}
    }
  })());
});

self.addEventListener("notificationclick", (event) => {
  event.notification.close();

  let target = "/dashboard";
  const supplied = event.notification.data &&
    event.notification.data.url;

  if (typeof supplied === "string" && supplied.startsWith("/") &&
      !supplied.startsWith("//")) {
    try {
      const parsed = new URL(supplied, self.location.origin);
      if (parsed.origin === self.location.origin) {
        target = parsed.href;
      }
    } catch (_) {}
  }

  event.waitUntil((async () => {
    const windows = await self.clients.matchAll({
      type: "window",
      includeUncontrolled: true
    });

    for (const client of windows) {
      if (new URL(client.url).origin === self.location.origin) {
        if ("navigate" in client) await client.navigate(target);
        return client.focus();
      }
    }

    return self.clients.openWindow(target);
  })());
});
