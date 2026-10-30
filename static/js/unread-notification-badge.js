(function () {
    const badges = document.querySelectorAll(".unread-notification-badge");

    let checking = false;

    async function refreshUnreadNotificationBadges() {
        if (checking || document.hidden || badges.length === 0) return;
        checking = true;

        try {
            const response = await fetch("/notifications/count", {
                method: "GET",
                credentials: "same-origin",
                headers: { "Accept": "application/json" },
                cache: "no-store"
            });

            if (!response.ok) return;

            const data = await response.json();
            const count = Math.max(0, Number(data.count) || 0);

            badges.forEach(function (badge) {
                badge.textContent = count > 99 ? "99+" : String(count);
                badge.hidden = count === 0;
            });
        } catch (_) {
            // Retry on the next polling interval.
        } finally {
            checking = false;
        }
    }

    refreshUnreadNotificationBadges();
    window.setInterval(refreshUnreadNotificationBadges, 20000);
    document.addEventListener("visibilitychange", refreshUnreadNotificationBadges);
})();
