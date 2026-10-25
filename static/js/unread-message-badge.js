(function () {
    const badges = document.querySelectorAll(".unread-message-badge");
    if (!badges.length) return;

    let checking = false;

    async function refreshUnreadMessageBadges() {
        if (checking || document.hidden) return;
        checking = true;

        try {
            const response = await fetch("/messages/unread-count", {
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
            // Ignore temporary network failures; the next poll will retry.
        } finally {
            checking = false;
        }
    }

    refreshUnreadMessageBadges();
    window.setInterval(refreshUnreadMessageBadges, 20000);
    document.addEventListener("visibilitychange", refreshUnreadMessageBadges);
})();
